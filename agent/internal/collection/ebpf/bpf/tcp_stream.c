// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE program: plaintext TCP byte stream on client and accepted sockets.
// fentry stashes the iov_iter start. The iterator advances before fexit.
// fexit copies min(ret, cap) across those segments. ret is not a length into iov[0].
// sendfile, splice, tcp_sendpage, and tcp_read_sock do not pass these hooks.
// Regenerate: go generate ./internal/collection/ebpf

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>
#include "dest.h"
#include "iov.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define AF_INET 2
#define AF_INET6 10
#define MSG_PEEK 2
#define STREAM_CAP 1024
#define STREAM_DIR_RECV 0
#define STREAM_DIR_SEND 1

struct stream_stash {
	struct iov_seg segs[IOV_STASH_SEGS];
	__u8 nsegs;
	__u8 pad[7];
};

struct stream_hdr {
	__u32 pid;
	__u32 len;
	__u64 cgroup_id;
	__u16 family;
	__u16 sport;
	__u16 dport;
	__u8 dir;
	__u8 pad;
	__u8 saddr[16];
	__u8 daddr[16];
} __attribute__((packed));

struct event {
	struct stream_hdr hdr;
	__u8 data[STREAM_CAP];
} __attribute__((packed));

_Static_assert(sizeof(struct stream_hdr) == 56, "stream header drifted");
_Static_assert(sizeof(struct event) == 56 + STREAM_CAP, "stream event drifted");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct stream_stash);
	__uint(max_entries, 8192);
} pending SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 22);
} events SEC(".maps");

static __always_inline int remote_ok(struct sock *sk)
{
	__u16 family = BPF_CORE_READ(sk, __sk_common.skc_family);
	__u16 dport = BPF_CORE_READ(sk, __sk_common.skc_dport);
	__u8 daddr[16] = {};

	if (dport == 0)
		return 0;
	if (family == AF_INET) {
		__u32 d = BPF_CORE_READ(sk, __sk_common.skc_daddr);
		__builtin_memcpy(daddr, &d, 4);
		return v4_dst_ok(daddr);
	}
	if (family == AF_INET6) {
		BPF_CORE_READ_INTO(&daddr, sk, __sk_common.skc_v6_daddr);
		return v6_dst_ok(daddr);
	}
	return 0;
}

static __always_inline void fill_tuple(struct sock *sk, struct stream_hdr *hdr)
{
	__u16 family = BPF_CORE_READ(sk, __sk_common.skc_family);
	__u16 dport = BPF_CORE_READ(sk, __sk_common.skc_dport);

	hdr->family = family;
	hdr->sport = BPF_CORE_READ(sk, __sk_common.skc_num);
	hdr->dport = bpf_ntohs(dport);
	__builtin_memset(hdr->saddr, 0, sizeof(hdr->saddr));
	__builtin_memset(hdr->daddr, 0, sizeof(hdr->daddr));
	if (family == AF_INET) {
		__u32 s = BPF_CORE_READ(sk, __sk_common.skc_rcv_saddr);
		__u32 d = BPF_CORE_READ(sk, __sk_common.skc_daddr);
		__builtin_memcpy(hdr->saddr, &s, 4);
		__builtin_memcpy(hdr->daddr, &d, 4);
		return;
	}
	if (family == AF_INET6) {
		BPF_CORE_READ_INTO(&hdr->saddr, sk, __sk_common.skc_v6_rcv_saddr);
		BPF_CORE_READ_INTO(&hdr->daddr, sk, __sk_common.skc_v6_daddr);
	}
}

static __always_inline int tls_bytes(__u8 hdr[3])
{
	if (hdr[0] < 0x14 || hdr[0] > 0x17)
		return 0;
	if (hdr[1] != 0x03)
		return 0;
	if (hdr[2] < 0x01 || hdr[2] > 0x04)
		return 0;
	return 1;
}

static __always_inline int read_user_byte(__u8 *dst, __u64 base, __u32 off)
{
	if (!base)
		return -1;
	return bpf_probe_read_user(dst, 1, (void *)(base + off));
}

/* Header split across segments. Probe size stays 1. A variable length into
 * hdr+got is not a bound the verifier accepts. */
static __always_inline int read_tls_split(struct stream_stash *stash, __u8 hdr[3])
{
	__u32 filled = 0;
	int i;

#pragma unroll
	for (i = 0; i < IOV_STASH_SEGS; i++) {
		__u32 len = stash->segs[i].len;
		__u64 base = stash->segs[i].base;

		if (i >= stash->nsegs || filled >= 3)
			break;
		if (len == 0 || base == 0)
			continue;
		if (filled == 0) {
			if (read_user_byte(&hdr[0], base, 0) < 0)
				return -1;
			filled = 1;
			if (len < 2)
				continue;
			if (read_user_byte(&hdr[1], base, 1) < 0)
				return -1;
			filled = 2;
			if (len < 3)
				continue;
			if (read_user_byte(&hdr[2], base, 2) < 0)
				return -1;
			filled = 3;
			continue;
		}
		if (filled == 1) {
			if (read_user_byte(&hdr[1], base, 0) < 0)
				return -1;
			filled = 2;
			if (len < 2)
				continue;
			if (read_user_byte(&hdr[2], base, 1) < 0)
				return -1;
			filled = 3;
			continue;
		}
		if (read_user_byte(&hdr[2], base, 0) < 0)
			return -1;
		filled = 3;
	}
	if (filled < 3)
		return -1;
	return 0;
}

static __always_inline int is_tls_record(struct stream_stash *stash)
{
	__u8 hdr[3] = {};
	int i;

	/* The first usable segment usually holds the record header. */
#pragma unroll
	for (i = 0; i < IOV_STASH_SEGS; i++) {
		if (i >= stash->nsegs)
			break;
		if (stash->segs[i].len == 0 || stash->segs[i].base == 0)
			continue;
		if (stash->segs[i].len < 3)
			break;
		if (bpf_probe_read_user(hdr, 3, (void *)stash->segs[i].base) < 0)
			return 0;
		return tls_bytes(hdr);
	}
	if (read_tls_split(stash, hdr) < 0)
		return 0;
	return tls_bytes(hdr);
}

static __always_inline __u32 copy_segs(struct bpf_dynptr *dst, struct stream_stash *stash, __u32 remain)
{
	__u32 off = 0;
	int i;

	if (remain > STREAM_CAP)
		remain = STREAM_CAP;

#pragma unroll
	for (i = 0; i < IOV_STASH_SEGS; i++) {
		__u32 chunk;

		if (i >= stash->nsegs || remain == 0 || off >= STREAM_CAP)
			break;
		chunk = stash->segs[i].len;
		if (chunk == 0 || stash->segs[i].base == 0)
			continue;
		if (chunk > remain)
			chunk = remain;
		/* A bare mask of 1023 would turn a 1024-byte chunk into 0.
		 * The other arm still needs a constant bound the verifier can see. */
		if (chunk >= STREAM_CAP)
			chunk = STREAM_CAP;
		else
			chunk &= STREAM_CAP - 1;
		if (chunk == 0 || off >= STREAM_CAP)
			continue;
		if (off >= STREAM_CAP - chunk)
			chunk = STREAM_CAP - off;
		if (bpf_probe_read_user_dynptr(dst, sizeof(struct stream_hdr) + off, chunk,
					       (const void *)stash->segs[i].base) < 0)
			break;
		off += chunk;
		remain -= chunk;
	}
	return off;
}

static __always_inline int stash_iov(struct sock *sk, struct msghdr *msg)
{
	__u64 id;
	struct stream_stash st = {};
	int nsegs;

	if (!remote_ok(sk))
		return 0;
	nsegs = iov_stash(msg, st.segs);
	if (nsegs <= 0)
		return 0;
	st.nsegs = (__u8)nsegs;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&pending, &id, &st, BPF_ANY);
	return 0;
}

static __always_inline int emit_chunk(struct sock *sk, int flags, int ret, __u8 dir)
{
	__u64 id;
	struct stream_stash *found;
	struct stream_stash st;
	struct bpf_dynptr ptr;
	struct stream_hdr hdr = {};
	__u32 copied;
	__u32 pid;
	__u64 cgroup_id;

	id = bpf_get_current_pid_tgid();
	found = bpf_map_lookup_elem(&pending, &id);
	if (!found)
		return 0;
	st = *found;
	bpf_map_delete_elem(&pending, &id);

	if (ret <= 0)
		return 0;
	if (dir == STREAM_DIR_RECV && (flags & MSG_PEEK))
		return 0;
	if (is_tls_record(&st))
		return 0;

	pid = (__u32)(id >> 32);
	cgroup_id = bpf_get_current_cgroup_id();
	if (pid == 0 && cgroup_id == 0)
		return 0;

	/* The verifier treats the dynptr as acquired even when reserve fails. */
	if (bpf_ringbuf_reserve_dynptr(&events, sizeof(struct event), 0, &ptr)) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}
	copied = copy_segs(&ptr, &st, (__u32)ret);
	if (copied == 0) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}
	hdr.pid = pid;
	hdr.cgroup_id = cgroup_id;
	hdr.len = copied;
	hdr.dir = dir;
	fill_tuple(sk, &hdr);
	if (bpf_dynptr_write(&ptr, 0, &hdr, sizeof(hdr), 0)) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}
	bpf_ringbuf_submit_dynptr(&ptr, 0);
	return 0;
}

SEC("fentry/tcp_sendmsg")
int BPF_PROG(mochi_stream_send_enter, struct sock *sk, struct msghdr *msg, size_t size)
{
	return stash_iov(sk, msg);
}

SEC("fexit/tcp_sendmsg")
int BPF_PROG(mochi_stream_send_exit, struct sock *sk, struct msghdr *msg, size_t size, int ret)
{
	return emit_chunk(sk, 0, ret, STREAM_DIR_SEND);
}

SEC("fentry/tcp_recvmsg")
int BPF_PROG(mochi_stream_recv_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags)
{
	return stash_iov(sk, msg);
}

SEC("fexit/tcp_recvmsg")
int BPF_PROG(mochi_stream_recv_exit, struct sock *sk, struct msghdr *msg, size_t len, int flags, int ret)
{
	return emit_chunk(sk, flags, ret, STREAM_DIR_RECV);
}
