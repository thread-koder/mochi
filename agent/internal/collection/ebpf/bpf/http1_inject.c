// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE: plaintext HTTP/1 Traceparent inject via sk_msg + sockops/SOCKHASH.
// Recv path stashes inbound TP into incoming_tp for multi-hop parents.
// Capture (tcp_stream) stays read-only. Regenerate: go generate ./internal/collection/ebpf

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>
#include "iov.h"
#include "traceparent.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define MSG_PEEK           2
#define TCP_ESTABLISHED    1
#define MIN_HTTP_REQ_SIZE  12
#define SK_MSG_PULL_SIZE   256
#define RECV_COPY_MAX      256
/* Default ip_local_port_range lower bound. Seed filter only. */
#define IP_LOCAL_PORT_MIN  32768

struct inject_cfg {
	__u64 sample_threshold;
};

struct inject_scratch {
	__u8 buf[SK_MSG_PULL_SIZE];
	__u8 hdr[TP_HEADER_LEN];
	struct tp_ids tp;
	__u32 len;
	__u32 newline_off;
};

struct buf_scan_ctx {
	__u8 *buf;
	__u32 limit;
	struct tp_ids *out;
	__u32 newline_off;
	__u8 found_tp;
	__u8 parsed_tp;
};

struct msg_copy_ctx {
	__u8 *data;
	__u8 *data_end;
	__u8 *dst;
	__u32 want;
	__u32 len;
};

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__type(key, __u32);
	__type(value, struct inject_cfg);
	__uint(max_entries, 1);
} inject_cfg SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_SOCKHASH);
	__type(key, __u64);
	__type(value, __u64);
	__uint(max_entries, 65535);
} inject_socks SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct tp_ids);
	__uint(max_entries, 16384);
} incoming_tp SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct egress_key);
	__type(value, struct tp_ids);
	__uint(max_entries, 16384);
} outgoing_tp SEC(".maps");

struct recv_stash {
	struct iov_seg segs[IOV_STASH_SEGS];
	__u8 nsegs;
	__u8 pad[7];
};

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct recv_stash);
	__uint(max_entries, 8192);
} inject_recv_pending SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__type(key, __u32);
	__type(value, struct inject_scratch);
	__uint(max_entries, 1);
} inject_scratch SEC(".maps");

static __always_inline __u64 sample_threshold(void)
{
	__u32 zero = 0;
	struct inject_cfg *cfg;

	cfg = bpf_map_lookup_elem(&inject_cfg, &zero);
	if (!cfg)
		return 0;
	return cfg->sample_threshold;
}

/* Caller must have proven data + MIN_HTTP_REQ_SIZE <= data_end. */
static __always_inline int is_http_request_bytes(const __u8 *p)
{
	if (p[0] == 'G' && p[1] == 'E' && p[2] == 'T' && p[3] == ' ')
		return p[4] == '/' || (p[4] | 0x20) == 'h';
	if (p[0] == 'P' && p[1] == 'O' && p[2] == 'S' && p[3] == 'T' && p[4] == ' ')
		return p[5] == '/' || (p[5] | 0x20) == 'h';
	if (p[0] == 'P' && p[1] == 'U' && p[2] == 'T' && p[3] == ' ')
		return p[4] == '/' || (p[4] | 0x20) == 'h';
	if (p[0] == 'H' && p[1] == 'E' && p[2] == 'A' && p[3] == 'D' && p[4] == ' ')
		return p[5] == '/' || (p[5] | 0x20) == 'h';
	if (p[0] == 'D' && p[1] == 'E' && p[2] == 'L' && p[3] == 'E' &&
	    p[4] == 'T' && p[5] == 'E' && p[6] == ' ')
		return p[7] == '/' || (p[7] | 0x20) == 'h';
	if (p[0] == 'P' && p[1] == 'A' && p[2] == 'T' && p[3] == 'C' &&
	    p[4] == 'H' && p[5] == ' ')
		return p[6] == '/' || (p[6] | 0x20) == 'h';
	if (p[0] == 'O' && p[1] == 'P' && p[2] == 'T' && p[3] == 'I' &&
	    p[4] == 'O' && p[5] == 'N' && p[6] == 'S' && p[7] == ' ')
		return p[8] == '/' || (p[8] | 0x20) == 'h';
	return 0;
}

/* One pass: first newline and existing Traceparent name. */
static long scan_hdr_cb(__u32 index, void *data)
{
	struct buf_scan_ctx *ctx = data;
	__u32 idx = index & (SK_MSG_PULL_SIZE - 1);
	__u32 vo;

	if (idx >= ctx->limit)
		return 1;
	if (ctx->newline_off == 0 && ctx->buf[idx] == '\n')
		ctx->newline_off = idx + 1;
	if (!ctx->found_tp &&
	    idx + TP_PREFIX_LEN + TP_VAL_LEN <= ctx->limit &&
	    tp_is_name(&ctx->buf[idx])) {
		vo = tp_value_off(&ctx->buf[idx], ctx->limit - idx);
		if (vo)
			ctx->found_tp = 1;
	}
	if (ctx->found_tp && ctx->newline_off != 0)
		return 1;
	return 0;
}

static long scan_tp_cb(__u32 index, void *data)
{
	struct buf_scan_ctx *ctx = data;
	__u32 idx = index & (SK_MSG_PULL_SIZE - 1);
	__u32 vo;

	if (idx + TP_PREFIX_LEN + TP_VAL_LEN > ctx->limit)
		return 1;
	if (!tp_is_name(&ctx->buf[idx]))
		return 0;
	vo = tp_value_off(&ctx->buf[idx], ctx->limit - idx);
	if (!vo)
		return 0;
	ctx->found_tp = 1;
	if (ctx->out && idx + vo + TP_VAL_LEN <= ctx->limit &&
	    tp_parse_value(&ctx->buf[idx + vo], ctx->out) == 0)
		ctx->parsed_tp = 1;
	return 1;
}

static long copy_msg_cb(__u32 index, void *data)
{
	struct msg_copy_ctx *ctx = data;
	__u32 idx = index & (SK_MSG_PULL_SIZE - 1);
	__u8 *p;

	if (idx >= ctx->want)
		return 1;
	p = ctx->data + idx;
	if (p + 1 > ctx->data_end)
		return 1;
	ctx->dst[idx] = *p;
	ctx->len = idx + 1;
	return 0;
}

/* Returns 1 when Traceparent is already present. Sets newline_off to first \n+1. */
static __always_inline int scan_request_hdr(__u8 *buf, __u32 size, __u32 *newline_off)
{
	struct buf_scan_ctx ctx = {.buf = buf};
	__u32 n = size;

	if (n > SK_MSG_PULL_SIZE)
		n = SK_MSG_PULL_SIZE;
	ctx.limit = n;
	bpf_loop(n, scan_hdr_cb, &ctx, 0);
	*newline_off = ctx.newline_off;
	return ctx.found_tp;
}

static __always_inline int parse_tp_in_buf(__u8 *buf, __u32 size, struct tp_ids *out)
{
	struct buf_scan_ctx ctx = {.buf = buf, .out = out};
	__u32 n;

	ctx.limit = size;
	if (ctx.limit > SK_MSG_PULL_SIZE)
		ctx.limit = SK_MSG_PULL_SIZE;
	if (ctx.limit < TP_PREFIX_LEN + TP_VAL_LEN)
		return -1;
	n = ctx.limit - (TP_PREFIX_LEN + TP_VAL_LEN) + 1;
	bpf_loop(n, scan_tp_cb, &ctx, 0);
	return ctx.parsed_tp ? 0 : -1;
}

static __always_inline __u32 copy_msg_to_scratch(struct sk_msg_md *msg, struct inject_scratch *scratch, __u32 want)
{
	struct msg_copy_ctx ctx = {};

	if (want > SK_MSG_PULL_SIZE)
		want = SK_MSG_PULL_SIZE;
	ctx.data = msg->data;
	ctx.data_end = msg->data_end;
	ctx.dst = scratch->buf;
	ctx.want = want;
	bpf_loop(want, copy_msg_cb, &ctx, 0);
	scratch->len = ctx.len;
	return ctx.len;
}

static __always_inline void mint_root(struct tp_ids *tp)
{
	__u64 thresh = sample_threshold();

	__builtin_memset(tp, 0, sizeof(*tp));
	tp_fill_nonzero_trace(tp->trace_id);
	tp_fill_nonzero_span(tp->span_id);
	if (tp_sampled_ratio(tp->trace_id, thresh))
		tp->flags = TP_SAMPLED_FLAG;
}

static __always_inline void mint_child(struct tp_ids *tp, const struct tp_ids *parent)
{
	__builtin_memset(tp, 0, sizeof(*tp));
	__builtin_memcpy(tp->trace_id, parent->trace_id, TP_TRACE_ID_SIZE);
	/* incoming_tp.span_id holds the inbound wire parent-id (caller span). */
	__builtin_memcpy(tp->parent_id, parent->span_id, TP_SPAN_ID_SIZE);
	tp_fill_nonzero_span(tp->span_id);
	tp->flags = parent->flags & TP_SAMPLED_FLAG;
}

static __always_inline int write_tp_header(struct sk_msg_md *msg, struct inject_scratch *scratch, __u32 offset)
{
	__u8 *data;
	__u8 *data_end;
	__u8 *ptr;
	__u32 pull_end;
	long err;

	tp_write_header(scratch->hdr, &scratch->tp);

	err = bpf_msg_push_data(msg, offset, TP_HEADER_LEN, 0);
	if (err != 0)
		return -1;
	/* push grows msg->size. Pull through the inserted header. */
	pull_end = offset + TP_HEADER_LEN;
	if (pull_end > SK_MSG_PULL_SIZE + TP_HEADER_LEN)
		return -1;
	if (bpf_msg_pull_data(msg, 0, pull_end, 0) < 0)
		return -1;
	data = msg->data;
	data_end = msg->data_end;
	if (!data)
		return -1;
	if (offset > SK_MSG_PULL_SIZE)
		return -1;
	ptr = data + offset;
	if (ptr + TP_HEADER_LEN > data_end)
		return -1;
	__builtin_memcpy(ptr, scratch->hdr, TP_HEADER_LEN);
	return 0;
}

SEC("sk_msg")
int mochi_inject_sk_msg(struct sk_msg_md *msg)
{
	__u64 pid_tgid;
	__u32 pid;
	__u32 size;
	__u32 want;
	__u32 zero = 0;
	__u64 tgid;
	struct egress_key ekey = {};
	struct tp_ids *parent;
	struct inject_scratch *scratch;

	pid_tgid = bpf_get_current_pid_tgid();
	pid = (__u32)(pid_tgid >> 32);

	size = msg->size;
	if (size < MIN_HTTP_REQ_SIZE)
		return SK_PASS;
	want = size;
	if (want > SK_MSG_PULL_SIZE)
		want = SK_MSG_PULL_SIZE;
	if (bpf_msg_pull_data(msg, 0, want, 0) < 0)
		return SK_PASS;

	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return SK_PASS;
	__builtin_memset(scratch, 0, sizeof(*scratch));

	if (copy_msg_to_scratch(msg, scratch, want) < MIN_HTTP_REQ_SIZE)
		return SK_PASS;

	if (!is_http_request_bytes(scratch->buf))
		return SK_PASS;

	if (scan_request_hdr(scratch->buf, scratch->len, &scratch->newline_off))
		return SK_PASS;
	if (scratch->newline_off == 0)
		return SK_PASS;

	/* Key by tgid (pid), not pid_tgid. Go/http.Server and similar hand
	 * the request to another OS thread before the outbound client send. */
	tgid = pid;
	parent = bpf_map_lookup_elem(&incoming_tp, &tgid);
	if (parent && !tp_ids_zero_trace(parent->trace_id) &&
	    !tp_ids_zero_span(parent->span_id))
		mint_child(&scratch->tp, parent);
	else
		mint_root(&scratch->tp);

	if (write_tp_header(msg, scratch, scratch->newline_off) < 0)
		return SK_PASS;

	ekey.pid = pid;
	ekey.sport = (__u16)msg->local_port;
	/* remote_port is network-order u32. Low 16 after ntohl. */
	ekey.dport = (__u16)bpf_ntohl(msg->remote_port);
	bpf_map_update_elem(&outgoing_tp, &ekey, &scratch->tp, BPF_ANY);
	return SK_PASS;
}

SEC("sockops")
int mochi_inject_sockops(struct bpf_sock_ops *skops)
{
	__u32 op = skops->op;
	__u64 cookie;

	/* Outbound clients only. Accepted sockets in SOCKHASH break sk_msg.
	 * bpf_sock_hash_update uses the sockops context. Do not require skops->sk. */
	if (op != BPF_SOCK_OPS_ACTIVE_ESTABLISHED_CB)
		return 1;

	cookie = bpf_get_socket_cookie(skops);
	if (cookie == 0)
		return 1;
	bpf_sock_hash_update(skops, &inject_socks, &cookie, BPF_ANY);
	return 1;
}

SEC("iter/tcp")
int mochi_inject_sock_iter(struct bpf_iter__tcp *ctx)
{
	struct sock_common *skc = ctx->sk_common;
	__u8 state;
	__u16 sport;
	__u64 cookie;
	struct sock *sk;

	if (!skc)
		return 0;
	state = BPF_CORE_READ(skc, skc_state);
	if (state != TCP_ESTABLISHED)
		return 0;
	/* Default ip_local_port_range starts at IP_LOCAL_PORT_MIN.
	 * Skip likely listen ports. New clients still enter via sockops. */
	sport = BPF_CORE_READ(skc, skc_num);
	if (sport < IP_LOCAL_PORT_MIN)
		return 0;

	sk = (struct sock *)skc;
	cookie = bpf_get_socket_cookie(sk);
	if (cookie == 0)
		return 0;
	bpf_map_update_elem(&inject_socks, &cookie, sk, BPF_NOEXIST);
	return 0;
}

SEC("fentry/tcp_recvmsg")
int BPF_PROG(mochi_inject_recv_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags)
{
	__u64 id;
	struct recv_stash st = {};
	int nsegs;

	if (flags & MSG_PEEK)
		return 0;
	nsegs = iov_stash(msg, st.segs);
	if (nsegs <= 0)
		return 0;
	st.nsegs = (__u8)nsegs;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&inject_recv_pending, &id, &st, BPF_ANY);
	return 0;
}

/* First segment only. Enough for Traceparent in typical request headers. */
static __always_inline __u32 copy_recv_prefix(struct recv_stash *stash, __u8 *dst, __u32 cap, __u32 remain)
{
	__u32 chunk;

	if (stash->nsegs == 0 || stash->segs[0].base == 0 || stash->segs[0].len == 0)
		return 0;
	if (remain > cap)
		remain = cap;
	chunk = stash->segs[0].len;
	if (chunk > remain)
		chunk = remain;
	/* Do not mask with (RECV_COPY_MAX-1). A full 128/256-byte copy
	 * becomes 0 and drops Traceparent on typical buffered reads. */
	if (chunk == 0)
		return 0;
	if (bpf_probe_read_user(dst, chunk, (const void *)stash->segs[0].base) < 0)
		return 0;
	return chunk;
}

/* ITER_UBUF fallback. tcp_recvmsg advances the iterator by ret, so the bytes just
 * written sit immediately before the current ubuf cursor. */
static __always_inline __u32 copy_recv_from_msg(struct msghdr *msg, __u8 *dst, __u32 cap, __u32 got)
{
	struct iov_iter *iter = &msg->msg_iter;
	struct iov_iter___ubuf *ubuf_iter = (void *)iter;
	void *ubuf;
	size_t off;
	__u32 chunk;
	__u64 start;

	if (got == 0)
		return 0;
	chunk = got;
	if (chunk > cap)
		chunk = cap;
	if (chunk == 0 || chunk > RECV_COPY_MAX)
		return 0;

	if (!bpf_core_field_exists(ubuf_iter->ubuf) ||
	    !bpf_core_enum_value_exists(enum iter_type, ITER_UBUF))
		return 0;
	if (BPF_CORE_READ(ubuf_iter, iter_type) !=
	    bpf_core_enum_value(enum iter_type, ITER_UBUF))
		return 0;
	ubuf = BPF_CORE_READ(ubuf_iter, ubuf);
	off = BPF_CORE_READ(ubuf_iter, iov_offset);
	if (!ubuf || off < chunk)
		return 0;
	start = (__u64)ubuf + off - chunk;
	if (bpf_probe_read_user(dst, chunk, (const void *)start) < 0)
		return 0;
	return chunk;
}

SEC("fexit/tcp_recvmsg")
int BPF_PROG(mochi_inject_recv_exit, struct sock *sk, struct msghdr *msg, size_t len, int flags, int ret)
{
	__u64 id;
	__u32 zero = 0;
	__u64 tgid;
	struct recv_stash *stash;
	struct inject_scratch *scratch;
	__u32 copied;

	id = bpf_get_current_pid_tgid();
	stash = bpf_map_lookup_elem(&inject_recv_pending, &id);
	if (!stash)
		return 0;
	bpf_map_delete_elem(&inject_recv_pending, &id);

	if (ret <= 0 || (flags & MSG_PEEK))
		return 0;

	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return 0;
	__builtin_memset(scratch, 0, sizeof(*scratch));
	copied = copy_recv_prefix(stash, scratch->buf, RECV_COPY_MAX, (__u32)ret);
	if (copied == 0)
		copied = copy_recv_from_msg(msg, scratch->buf, RECV_COPY_MAX, (__u32)ret);
	/* sk_msg push can land Traceparent in a later TCP segment than the
	 * request-line. Accept TP in any recv chunk (not only HTTP-prefixed). */
	if (copied < TP_PREFIX_LEN + TP_VAL_LEN)
		return 0;
	if (parse_tp_in_buf(scratch->buf, copied, &scratch->tp) < 0)
		return 0;

	/* incoming_tp.span_id stores the inbound wire parent-id for the next child mint. */
	__builtin_memcpy(scratch->tp.span_id, scratch->tp.parent_id, TP_SPAN_ID_SIZE);
	__builtin_memset(scratch->tp.parent_id, 0, TP_SPAN_ID_SIZE);
	tgid = (__u32)(id >> 32);
	bpf_map_update_elem(&incoming_tp, &tgid, &scratch->tp, BPF_ANY);
	return 0;
}
