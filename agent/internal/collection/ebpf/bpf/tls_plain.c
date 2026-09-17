// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE program: TLS plaintext from OpenSSL and Go crypto/tls uprobes.
// Nested tcp_sendmsg / tcp_recvmsg is ciphertext. It only snapshots the
// 4-tuple while struct sock is live. Emit copies the user buffer at return.
// A held sock pointer would be use-after-free on a later buffered SSL_read.
// amd64 only. Go is entry plus decoded RET uprobes, never uretprobe.
// Cookies in this file must match tls.go.
// Regenerate: go generate ./internal/collection/ebpf

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>
#include "dest.h"
#include "stream_hdr.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define TLS_COOKIE_SSL_WRITE 1
#define TLS_COOKIE_SSL_READ 2
#define TLS_COOKIE_SSL_WRITE_EX 3
#define TLS_COOKIE_SSL_READ_EX 4
#define TLS_COOKIE_GO_WRITE 5
#define TLS_COOKIE_GO_READ 6
#define TLS_COOKIE_GO_RET 7

struct inflight_stash {
	__u64 obj;
	__u64 buf;
	__u64 aux;
	__u32 bound;
	__u8 dir;
	__u8 kind;
	__u8 ex;
	__u8 pad;
};

struct go_key {
	__u32 tgid;
	__u32 pad;
	__u64 g;
};

struct go_stash {
	__u64 obj;
	__u64 buf;
	__u64 tid;
	__u32 bound;
	__u8 dir;
	__u8 pad[3];
};

struct obj_key {
	__u32 tgid;
	__u32 pad;
	__u64 obj;
};

struct conn_tuple {
	__u16 family;
	__u16 sport;
	__u16 dport;
	__u8 has_tuple;
	__u8 pad;
	__u8 saddr[16];
	__u8 daddr[16];
};

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct inflight_stash);
	__uint(max_entries, 8192);
} inflight SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct go_key);
	__type(value, struct go_stash);
	__uint(max_entries, 8192);
} go_active SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct obj_key);
	__type(value, struct conn_tuple);
	__uint(max_entries, 32768);
} tuples SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 22);
} events SEC(".maps");

static __always_inline __u32 clamp_copy(__u32 n)
{
	if (n == 0)
		return 0;
	if (n >= STREAM_CAP)
		return STREAM_CAP;
	return n;
}

static __always_inline void snapshot_tuple(struct sock *sk, struct conn_tuple *dst)
{
	struct stream_hdr hdr = {};

	if (!remote_ok(sk))
		return;
	fill_tuple(sk, &hdr);
	dst->family = hdr.family;
	dst->sport = hdr.sport;
	dst->dport = hdr.dport;
	__builtin_memcpy(dst->saddr, hdr.saddr, sizeof(dst->saddr));
	__builtin_memcpy(dst->daddr, hdr.daddr, sizeof(dst->daddr));
	dst->has_tuple = 1;
}

static __always_inline int fill_tuple_map(struct sock *sk)
{
	__u64 id;
	struct inflight_stash *act;
	struct obj_key key = {};
	struct conn_tuple next = {};
	struct conn_tuple *cur;

	if (!sk)
		return 0;
	id = bpf_get_current_pid_tgid();
	act = bpf_map_lookup_elem(&inflight, &id);
	if (!act || !act->obj)
		return 0;

	key.tgid = id >> 32;
	key.obj = act->obj;
	cur = bpf_map_lookup_elem(&tuples, &key);
	if (cur)
		next = *cur;
	snapshot_tuple(sk, &next);
	if (!next.has_tuple)
		return 0;
	bpf_map_update_elem(&tuples, &key, &next, BPF_ANY);
	return 0;
}

static __always_inline int emit_obj(__u32 tgid, __u64 obj, __u64 buf, __u32 n, __u8 dir, __u8 kind)
{
	struct obj_key key = {};
	struct conn_tuple *cs;
	struct conn_tuple snap;
	struct bpf_dynptr ptr;
	struct stream_hdr hdr = {};
	__u32 chunk;
	__u32 pid;
	__u64 cgroup_id;

	if (!obj || !buf || n == 0)
		return 0;
	key.tgid = tgid;
	key.obj = obj;
	cs = bpf_map_lookup_elem(&tuples, &key);
	if (!cs || !cs->has_tuple)
		return 0;
	snap = *cs;

	chunk = clamp_copy(n);
	if (chunk == 0)
		return 0;

	pid = tgid;
	cgroup_id = bpf_get_current_cgroup_id();
	if (pid == 0 && cgroup_id == 0)
		return 0;

	/* The verifier treats the dynptr as acquired even when reserve fails. */
	if (bpf_ringbuf_reserve_dynptr(&events, sizeof(struct stream_event), 0, &ptr)) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}
	if (bpf_probe_read_user_dynptr(&ptr, sizeof(struct stream_hdr), chunk, (const void *)buf) < 0) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}

	hdr.pid = pid;
	hdr.cgroup_id = cgroup_id;
	hdr.len = chunk;
	hdr.dir = dir;
	hdr.kind = kind;
	hdr.family = snap.family;
	hdr.sport = snap.sport;
	hdr.dport = snap.dport;
	__builtin_memcpy(hdr.saddr, snap.saddr, sizeof(hdr.saddr));
	__builtin_memcpy(hdr.daddr, snap.daddr, sizeof(hdr.daddr));
	if (bpf_dynptr_write(&ptr, 0, &hdr, sizeof(hdr), 0)) {
		bpf_ringbuf_discard_dynptr(&ptr, 0);
		return 0;
	}
	bpf_ringbuf_submit_dynptr(&ptr, 0);
	return 0;
}

static __always_inline int ssl_enter(struct pt_regs *ctx, __u8 dir, __u8 ex)
{
	__u64 id;
	struct inflight_stash act = {};

	act.obj = ctx->di;
	act.buf = ctx->si;
	if (!act.obj || !act.buf)
		return 0;
	act.dir = dir;
	act.kind = STREAM_KIND_OPENSSL;
	act.ex = ex;
	if (ex)
		act.aux = ctx->cx;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&inflight, &id, &act, BPF_ANY);
	return 0;
}

static __always_inline int ssl_ret(struct pt_regs *ctx)
{
	__u64 id;
	struct inflight_stash *found;
	struct inflight_stash act;
	int ret;
	__u32 bytes = 0;
	__u64 n = 0;

	id = bpf_get_current_pid_tgid();
	found = bpf_map_lookup_elem(&inflight, &id);
	if (!found)
		return 0;
	act = *found;
	bpf_map_delete_elem(&inflight, &id);

	ret = (int)ctx->ax;
	if (act.ex) {
		if (ret != 1 || !act.aux)
			return 0;
		if (bpf_probe_read_user(&n, sizeof(n), (const void *)act.aux) < 0)
			return 0;
		if (n == 0 || n > 0xffffffff)
			return 0;
		bytes = (__u32)n;
	} else {
		if (ret <= 0)
			return 0;
		bytes = (__u32)ret;
	}
	return emit_obj(id >> 32, act.obj, act.buf, bytes, act.dir, act.kind);
}

static __always_inline int go_enter(struct pt_regs *ctx, __u8 dir)
{
	__u64 id;
	__u64 bound;
	struct inflight_stash act = {};
	struct go_key key = {};
	struct go_stash stash = {};

	/* ABIInternal on amd64. g is R14, not the OS thread. */
	act.obj = ctx->ax;
	act.buf = ctx->bx;
	bound = ctx->cx;
	key.g = ctx->r14;
	if (!act.obj || !act.buf || !key.g || bound == 0)
		return 0;

	act.bound = bound > 0xffffffff ? 0xffffffff : (__u32)bound;
	act.dir = dir;
	act.kind = STREAM_KIND_GOTLS;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&inflight, &id, &act, BPF_ANY);

	key.tgid = id >> 32;
	stash.obj = act.obj;
	stash.buf = act.buf;
	stash.tid = id;
	stash.bound = act.bound;
	stash.dir = dir;
	bpf_map_update_elem(&go_active, &key, &stash, BPF_ANY);
	return 0;
}

static __always_inline int go_ret(struct pt_regs *ctx)
{
	__u64 id;
	struct go_key key = {};
	struct go_stash *found;
	struct go_stash stash;
	struct inflight_stash *left;
	__s64 n;
	__u32 bytes;

	id = bpf_get_current_pid_tgid();
	key.tgid = id >> 32;
	key.g = ctx->r14;
	if (!key.g)
		return 0;
	found = bpf_map_lookup_elem(&go_active, &key);
	if (!found)
		return 0;
	stash = *found;
	bpf_map_delete_elem(&go_active, &key);

	/* The goroutine may have resumed on another thread. Only drop the
	 * thread stash if it is still this call. */
	left = bpf_map_lookup_elem(&inflight, &stash.tid);
	if (left && left->obj == stash.obj)
		bpf_map_delete_elem(&inflight, &stash.tid);

	n = (__s64)ctx->ax;
	if (n <= 0)
		return 0;
	bytes = n > 0xffffffff ? 0xffffffff : (__u32)n;
	if (bytes > stash.bound)
		bytes = stash.bound;
	return emit_obj(stash.tid >> 32, stash.obj, stash.buf, bytes, stash.dir, STREAM_KIND_GOTLS);
}

SEC("uprobe")
int mochi_tls_enter(struct pt_regs *ctx)
{
	__u64 cookie = bpf_get_attach_cookie(ctx);

	if (cookie == TLS_COOKIE_SSL_WRITE)
		return ssl_enter(ctx, STREAM_DIR_SEND, 0);
	if (cookie == TLS_COOKIE_SSL_READ)
		return ssl_enter(ctx, STREAM_DIR_RECV, 0);
	if (cookie == TLS_COOKIE_SSL_WRITE_EX)
		return ssl_enter(ctx, STREAM_DIR_SEND, 1);
	if (cookie == TLS_COOKIE_SSL_READ_EX)
		return ssl_enter(ctx, STREAM_DIR_RECV, 1);
	if (cookie == TLS_COOKIE_GO_WRITE)
		return go_enter(ctx, STREAM_DIR_SEND);
	if (cookie == TLS_COOKIE_GO_READ)
		return go_enter(ctx, STREAM_DIR_RECV);
	return 0;
}

SEC("uprobe")
int mochi_tls_ret(struct pt_regs *ctx)
{
	__u64 cookie = bpf_get_attach_cookie(ctx);

	if (cookie == TLS_COOKIE_GO_RET)
		return go_ret(ctx);
	if (cookie == TLS_COOKIE_SSL_WRITE || cookie == TLS_COOKIE_SSL_READ ||
	    cookie == TLS_COOKIE_SSL_WRITE_EX || cookie == TLS_COOKIE_SSL_READ_EX)
		return ssl_ret(ctx);
	return 0;
}

SEC("fentry/tcp_sendmsg")
int BPF_PROG(mochi_tls_send_enter, struct sock *sk, struct msghdr *msg, size_t size)
{
	(void)msg;
	(void)size;
	return fill_tuple_map(sk);
}

SEC("fentry/tcp_recvmsg")
int BPF_PROG(mochi_tls_recv_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags)
{
	(void)msg;
	(void)len;
	(void)flags;
	return fill_tuple_map(sk);
}
