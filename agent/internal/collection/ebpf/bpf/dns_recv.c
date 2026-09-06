// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE program: capture client DNS responses (UDP/TCP port 53) for FQDN correlation.
// fentry stashes the user buffer. fexit copies the filled payload (iov_iter advances).
// Parse happens in userspace. Regenerate: go generate ./internal/collection/ebpf

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define DNS_PORT 53
#define DNS_PAYLOAD_MAX 1232

struct stash {
	__u64 buf;
};

struct event {
	__u32 pid;
	__u32 len;
	__u64 cgroup_id;
	__u8 is_tcp;
	__u8 pad[3];
	__u8 data[DNS_PAYLOAD_MAX];
};

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct stash);
	__uint(max_entries, 8192);
} pending SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 22);
} events SEC(".maps");

/* Kernel 6.4+ renamed iov_iter.iov → __iov. Flavor structs for CO-RE. */
struct iov_iter___old {
	const struct iovec *iov;
} __attribute__((preserve_access_index));

struct iov_iter___new {
	const struct iovec *__iov;
} __attribute__((preserve_access_index));

/* Kernel 6.0+ ITER_UBUF: single-buffer path uses ubuf instead of iov. */
struct iov_iter___ubuf {
	void *ubuf;
	u8 iter_type;
} __attribute__((preserve_access_index));

static __always_inline int dns_ports_ok(struct sock *sk)
{
	__u16 dport;
	__u16 sport;

	dport = BPF_CORE_READ(sk, __sk_common.skc_dport);
	if (bpf_ntohs(dport) != DNS_PORT)
		return 0;
	sport = BPF_CORE_READ(sk, __sk_common.skc_num);
	if (sport == DNS_PORT)
		return 0;
	return 1;
}

static __always_inline void *msg_user_buf(struct msghdr *msg)
{
	struct iov_iter *iter = &msg->msg_iter;
	struct iov_iter___ubuf *ubuf_iter = (void *)iter;
	struct iov_iter___new *new_iter = (void *)iter;
	struct iov_iter___old *old_iter = (void *)iter;
	const struct iovec *iov = NULL;
	void *buf = NULL;

	if (bpf_core_field_exists(ubuf_iter->ubuf) &&
	    bpf_core_enum_value_exists(enum iter_type, ITER_UBUF)) {
		u8 type = BPF_CORE_READ(ubuf_iter, iter_type);
		if (type == bpf_core_enum_value(enum iter_type, ITER_UBUF))
			return BPF_CORE_READ(ubuf_iter, ubuf);
	}

	if (bpf_core_field_exists(new_iter->__iov))
		iov = BPF_CORE_READ(new_iter, __iov);
	else if (bpf_core_field_exists(old_iter->iov))
		iov = BPF_CORE_READ(old_iter, iov);
	if (!iov)
		return NULL;
	buf = BPF_CORE_READ(iov, iov_base);
	return buf;
}

static __always_inline int stash_recv(struct sock *sk, struct msghdr *msg)
{
	__u64 id;
	struct stash s = {};
	void *buf;

	if (!dns_ports_ok(sk))
		return 0;
	buf = msg_user_buf(msg);
	if (!buf)
		return 0;
	s.buf = (__u64)buf;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&pending, &id, &s, BPF_ANY);
	return 0;
}

static __always_inline int emit_recv(int ret, __u8 is_tcp)
{
	__u64 id;
	struct stash *s;
	struct event *e;
	__u32 copy_len;
	__u32 pid;
	__u64 cgroup_id;
	void *buf;

	id = bpf_get_current_pid_tgid();
	s = bpf_map_lookup_elem(&pending, &id);
	if (!s) {
		return 0;
	}
	bpf_map_delete_elem(&pending, &id);
	buf = (void *)s->buf;
	if (!buf)
		return 0;

	if (ret <= 0)
		return 0;

	if (is_tcp) {
		__u16 be_len = 0;
		__u32 need;

		if (ret < 2)
			return 0;
		if (bpf_probe_read_user(&be_len, sizeof(be_len), buf) < 0)
			return 0;
		need = 2 + (__u32)bpf_ntohs(be_len);
		if ((__u32)ret < need || need > DNS_PAYLOAD_MAX)
			return 0;
		copy_len = need;
	} else {
		copy_len = (__u32)ret;
		if (copy_len > DNS_PAYLOAD_MAX)
			copy_len = DNS_PAYLOAD_MAX;
	}

	pid = (__u32)(id >> 32);
	cgroup_id = bpf_get_current_cgroup_id();
	if (pid == 0 && cgroup_id == 0)
		return 0;

	e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		return 0;

	e->pid = pid;
	e->cgroup_id = cgroup_id;
	e->is_tcp = is_tcp;
	e->len = copy_len;
	__builtin_memset(e->pad, 0, sizeof(e->pad));
	if (bpf_probe_read_user(e->data, copy_len, buf) < 0) {
		bpf_ringbuf_discard(e, 0);
		return 0;
	}
	bpf_ringbuf_submit(e, 0);
	return 0;
}

SEC("fentry/udp_recvmsg")
int BPF_PROG(mochi_udp_recvmsg_enter, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags)
{
	return stash_recv(sk, msg);
}

SEC("fexit/udp_recvmsg")
int BPF_PROG(mochi_udp_recvmsg_exit, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags, int ret)
{
	return emit_recv(ret, 0);
}

SEC("fentry/udpv6_recvmsg")
int BPF_PROG(mochi_udpv6_recvmsg_enter, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags)
{
	return stash_recv(sk, msg);
}

SEC("fexit/udpv6_recvmsg")
int BPF_PROG(mochi_udpv6_recvmsg_exit, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags, int ret)
{
	return emit_recv(ret, 0);
}

SEC("fentry/tcp_recvmsg")
int BPF_PROG(mochi_tcp_recvmsg_enter, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags)
{
	return stash_recv(sk, msg);
}

SEC("fexit/tcp_recvmsg")
int BPF_PROG(mochi_tcp_recvmsg_exit, struct sock *sk, struct msghdr *msg,
	     size_t len, int flags, int ret)
{
	return emit_recv(ret, 1);
}
