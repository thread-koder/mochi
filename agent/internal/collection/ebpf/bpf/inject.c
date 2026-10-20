// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE Traceparent inject: sockops + SOCKHASH sk_msg (HTTP/1 + HTTP/2/gRPC),
// iter/tcp seed, nested tcp_sendmsg_locked / tcp_recvmsg, Go Write and OpenSSL
// read uprobes. Recv and TLS read stash inbound TP into incoming_tp.
// Capture (tcp_stream, tls_plain) stays read-only.
// Regenerate: go generate ./internal/collection/ebpf

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_tracing.h>
#include "dest.h"
#include "iov.h"
#include "stream_hdr.h"
#include "traceparent.h"
#include "hpack.h"
#include "uprobe_abi.h"

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define MSG_PEEK           2
#define TCP_ESTABLISHED    1
#define MIN_HTTP_REQ_SIZE  12
#define SK_MSG_PULL_SIZE   256
/* Recv window for inbound TP. When the read is larger, copy the tail so an
 * appended HPACK Traceparent past preface/SETTINGS still lands in range. */
#define RECV_COPY_MAX      512
#define IP_LOCAL_PORT_MIN  32768
#define USER_MOVE_CHUNK    64
#define USER_MOVE_MAX      2048
#define USER_MOVE_ITERS    32
/* ASCII a-z vs A-Z differ by this bit. OR folds uppercase to lowercase. */
#define ASCII_CASE_BIT     0x20
#define H2_STREAM_ID_MASK  0x7f

#define INJECT_COOKIE_GO_WRITE 1
#define INJECT_COOKIE_GO_READ  2
#define INJECT_COOKIE_GO_RET   3
#define INJECT_COOKIE_SSL_READ 4
#define INJECT_COOKIE_SSL_READ_EX 5

struct inject_cfg {
	__u64 sample_threshold;
};

struct inject_scratch {
	__u8 buf[RECV_COPY_MAX];
	__u8 hdr[TP_HEADER_LEN];
	__u8 hpack[HPACK_TP_LIT_LEN];
	struct tp_ids tp;
	__u32 len;
	__u32 hpack_len;
	__u32 newline_off;
	__u32 stream;
	__u32 frame_off;
	__u32 insert_off;
	__u32 frame_len;
};

struct buf_scan_ctx {
	__u8 *buf;
	__u32 limit;
	__u32 newline_off;
	__u8 found_tp;
};

struct msg_copy_ctx {
	__u8 *data;
	__u8 *data_end;
	__u8 *dst;
	__u32 want;
	__u32 len;
};

/* sk_msg has no bpf_get_socket_cookie on this attach type. Use the send tuple. */
struct h2_seen_key {
	__u32 pid;
	__u16 sport;
	__u16 dport;
	__u32 stream;
};

struct h2_conn_key {
	__u32 pid;
	__u16 sport;
	__u16 dport;
	__u16 pad;
};

struct go_seen_key {
	__u32 tgid;
	__u32 stream;
	__u64 conn;
};

struct go_has_key {
	__u32 tgid;
	__u32 pad;
	__u64 conn;
};

struct conn_tuple {
	__u16 sport;
	__u16 dport;
	__u8 has_tuple;
	__u8 pad[3];
};

struct go_stash {
	__u64 conn;
	__u64 buf;
	__u64 tid;
	__u32 bound;
	__u32 insert_len;
	__u32 stream;
	__u8 did_inject;
	__u8 is_write;
	__u8 pad[2];
	struct tp_ids tp;
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

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct h2_seen_key);
	__type(value, __u8);
	__uint(max_entries, 16384);
} h2_hdr_seen SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct h2_conn_key);
	__type(value, __u8);
	__uint(max_entries, 8192);
} h2_has_tp SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct go_seen_key);
	__type(value, __u8);
	__uint(max_entries, 16384);
} go_hdr_seen SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct go_has_key);
	__type(value, __u8);
	__uint(max_entries, 8192);
} go_has_tp SEC(".maps");

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
} inject_tuples SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct inflight_stash);
	__uint(max_entries, 8192);
} inject_inflight SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__type(key, __u32);
	__type(value, __u8[USER_MOVE_CHUNK]);
	__uint(max_entries, 1);
} inject_move_scratch SEC(".maps");

static __always_inline __u64 sample_threshold(void)
{
	__u32 zero = 0;
	struct inject_cfg *cfg;

	cfg = bpf_map_lookup_elem(&inject_cfg, &zero);
	if (!cfg)
		return 0;
	return cfg->sample_threshold;
}

static __always_inline int is_http_request_bytes(const __u8 *p)
{
	if (p[0] == 'G' && p[1] == 'E' && p[2] == 'T' && p[3] == ' ')
		return p[4] == '/' || (p[4] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'P' && p[1] == 'O' && p[2] == 'S' && p[3] == 'T' && p[4] == ' ')
		return p[5] == '/' || (p[5] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'P' && p[1] == 'U' && p[2] == 'T' && p[3] == ' ')
		return p[4] == '/' || (p[4] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'H' && p[1] == 'E' && p[2] == 'A' && p[3] == 'D' && p[4] == ' ')
		return p[5] == '/' || (p[5] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'D' && p[1] == 'E' && p[2] == 'L' && p[3] == 'E' &&
	    p[4] == 'T' && p[5] == 'E' && p[6] == ' ')
		return p[7] == '/' || (p[7] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'P' && p[1] == 'A' && p[2] == 'T' && p[3] == 'C' &&
	    p[4] == 'H' && p[5] == ' ')
		return p[6] == '/' || (p[6] | ASCII_CASE_BIT) == 'h';
	if (p[0] == 'O' && p[1] == 'P' && p[2] == 'T' && p[3] == 'I' &&
	    p[4] == 'O' && p[5] == 'N' && p[6] == 'S' && p[7] == ' ')
		return p[8] == '/' || (p[8] | ASCII_CASE_BIT) == 'h';
	return 0;
}

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
	__builtin_memcpy(tp->parent_id, parent->span_id, TP_SPAN_ID_SIZE);
	tp_fill_nonzero_span(tp->span_id);
	tp->flags = parent->flags & TP_SAMPLED_FLAG;
}

static __always_inline void mint_from_incoming(struct tp_ids *tp, __u32 tgid)
{
	__u64 key = tgid;
	struct tp_ids *parent;

	parent = bpf_map_lookup_elem(&incoming_tp, &key);
	if (parent && !tp_ids_zero_trace(parent->trace_id) &&
	    !tp_ids_zero_span(parent->span_id))
		mint_child(tp, parent);
	else
		mint_root(tp);
}

static __always_inline void scratch_reset_meta(struct inject_scratch *scratch)
{
	scratch->len = 0;
	scratch->hpack_len = 0;
	scratch->newline_off = 0;
	scratch->stream = 0;
	scratch->frame_off = 0;
	scratch->insert_off = 0;
	scratch->frame_len = 0;
	__builtin_memset(&scratch->tp, 0, sizeof(scratch->tp));
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

static __always_inline int write_hpack_tp(struct sk_msg_md *msg, struct inject_scratch *scratch, __u32 offset)
{
	__u8 *data;
	__u8 *data_end;
	__u8 *ptr;
	__u8 *len_ptr;
	__u32 pull_end;
	__u32 new_len;
	__u32 frame_off;
	__u32 insert_off;
	long err;

	if (scratch->frame_len + HPACK_TP_LIT_LEN > H2_MAX_FRAME_SIZE)
		return -1;
	frame_off = scratch->frame_off & (SK_MSG_PULL_SIZE - 1);
	insert_off = offset & (SK_MSG_PULL_SIZE - 1);
	if (frame_off + 3 > SK_MSG_PULL_SIZE)
		return -1;
	if (insert_off + HPACK_TP_LIT_LEN > SK_MSG_PULL_SIZE + HPACK_TP_LIT_LEN)
		return -1;
	hpack_write_tp(scratch->hpack, &scratch->tp);

	err = bpf_msg_push_data(msg, insert_off, HPACK_TP_LIT_LEN, 0);
	if (err != 0)
		return -1;
	pull_end = frame_off + H2_FRAME_HDR_LEN + HPACK_TP_LIT_LEN;
	if (insert_off + HPACK_TP_LIT_LEN > pull_end)
		pull_end = insert_off + HPACK_TP_LIT_LEN;
	if (pull_end > SK_MSG_PULL_SIZE + HPACK_TP_LIT_LEN)
		return -1;
	if (bpf_msg_pull_data(msg, 0, pull_end, 0) < 0)
		return -1;
	data = msg->data;
	data_end = msg->data_end;
	if (!data)
		return -1;
	len_ptr = data + frame_off;
	if (len_ptr + 3 > data_end)
		return -1;
	new_len = scratch->frame_len + HPACK_TP_LIT_LEN;
	len_ptr[0] = (__u8)((new_len >> 16) & 0xff);
	len_ptr[1] = (__u8)((new_len >> 8) & 0xff);
	len_ptr[2] = (__u8)(new_len & 0xff);
	ptr = data + insert_off;
	if (ptr + HPACK_TP_LIT_LEN > data_end)
		return -1;
	__builtin_memcpy(ptr, scratch->hpack, HPACK_TP_LIT_LEN);
	return 0;
}

struct h2_find_ctx {
	__u8 *buf;
	__u32 size;
	struct inject_scratch *scratch;
	__u32 off;
	__u8 found;
};

static long find_h2_headers_cb(__u32 idx, void *data)
{
	struct h2_find_ctx *ctx = data;
	__u32 off = ctx->off & (SK_MSG_PULL_SIZE - 1);
	__u32 frame_len;
	__u8 type;
	__u8 flags;
	__u32 stream;
	__u32 payload_off;
	__u32 hpack_off;
	__u32 hpack_len;
	__u8 pad_len;
	__u32 prefix;
	__u32 next;

	(void)idx;
	if (ctx->found)
		return 1;
	if (off + H2_FRAME_HDR_LEN > ctx->size ||
	    off + H2_FRAME_HDR_LEN > SK_MSG_PULL_SIZE)
		return 1;
	frame_len = ((__u32)ctx->buf[off] << 16) |
		    ((__u32)ctx->buf[off + 1] << 8) |
		    ctx->buf[off + 2];
	type = ctx->buf[off + 3];
	flags = ctx->buf[off + 4];
	stream = ((__u32)(ctx->buf[off + 5] & H2_STREAM_ID_MASK) << 24) |
		 ((__u32)ctx->buf[off + 6] << 16) |
		 ((__u32)ctx->buf[off + 7] << 8) |
		 ctx->buf[off + 8];
	if (frame_len > H2_MAX_FRAME_SIZE)
		return 1;
	next = off + H2_FRAME_HDR_LEN + frame_len;
	if (next > ctx->size || next > SK_MSG_PULL_SIZE)
		return 1;
	if (type == H2_FRAME_HEADERS && (stream & 1) &&
	    (flags & H2_FLAG_END_HEADERS)) {
		payload_off = off + H2_FRAME_HDR_LEN;
		payload_off &= (SK_MSG_PULL_SIZE - 1);
		prefix = 0;
		pad_len = 0;
		if (flags & H2_FLAG_PADDED) {
			if (frame_len < 1)
				return 1;
			pad_len = ctx->buf[payload_off];
			prefix += 1;
		}
		if (flags & H2_FLAG_PRIORITY)
			prefix += H2_PRIORITY_LEN;
		if (prefix + pad_len > frame_len)
			return 1;
		hpack_off = payload_off + prefix;
		hpack_off &= (SK_MSG_PULL_SIZE - 1);
		hpack_len = frame_len - prefix - pad_len;
		ctx->scratch->frame_off = off;
		ctx->scratch->frame_len = frame_len;
		ctx->scratch->insert_off = hpack_off;
		ctx->scratch->stream = stream;
		ctx->scratch->hpack_len = hpack_len;
		ctx->found = 1;
		return 1;
	}
	ctx->off = next;
	return 0;
}

static __always_inline int find_h2_headers(__u8 *buf, __u32 size, struct inject_scratch *scratch)
{
	struct h2_find_ctx ctx = {};

	ctx.buf = buf;
	ctx.size = size;
	ctx.scratch = scratch;
	bpf_loop(8, find_h2_headers_cb, &ctx, 0);
	return ctx.found ? 0 : -1;
}

static __always_inline int inject_h2_sk_msg(struct sk_msg_md *msg, struct inject_scratch *scratch, __u32 pid)
{
	struct h2_seen_key seen = {};
	struct h2_conn_key ckey = {};
	__u8 one = 1;
	__u8 *has;
	struct egress_key ekey = {};
	__u32 hpack_len;
	__u32 insert_off;
	__u16 sport;
	__u16 dport;

	if (find_h2_headers(scratch->buf, scratch->len, scratch) < 0)
		return -1;

	sport = (__u16)msg->local_port;
	dport = (__u16)bpf_ntohl(msg->remote_port);
	ckey.pid = pid;
	ckey.sport = sport;
	ckey.dport = dport;
	has = bpf_map_lookup_elem(&h2_has_tp, &ckey);
	if (has)
		return -1;

	seen.pid = pid;
	seen.sport = sport;
	seen.dport = dport;
	seen.stream = scratch->stream;
	if (bpf_map_update_elem(&h2_hdr_seen, &seen, &one, BPF_NOEXIST) != 0)
		return -1;

	insert_off = scratch->insert_off;
	if (insert_off >= SK_MSG_PULL_SIZE)
		return -1;
	hpack_len = scratch->hpack_len;
	if (hpack_len > SK_MSG_PULL_SIZE ||
	    insert_off + hpack_len > SK_MSG_PULL_SIZE)
		return -1;
	if (hpack_scan_range(scratch->buf, insert_off, hpack_len, NULL)) {
		bpf_map_update_elem(&h2_has_tp, &ckey, &one, BPF_ANY);
		return -1;
	}
	/*
	 * Append after the HPACK block (before padding). Mid block insert
	 * breaks once pseudos use dynamic table indices after the first request.
	 */
	insert_off = insert_off + hpack_len;
	if (insert_off >= SK_MSG_PULL_SIZE)
		return -1;

	mint_from_incoming(&scratch->tp, pid);
	if (write_hpack_tp(msg, scratch, insert_off) < 0)
		return -1;

	ekey.pid = pid;
	ekey.sport = sport;
	ekey.dport = dport;
	ekey.stream = scratch->stream;
	bpf_map_update_elem(&outgoing_tp, &ekey, &scratch->tp, BPF_ANY);
	return 0;
}

static __always_inline int inject_h1_sk_msg(struct sk_msg_md *msg, struct inject_scratch *scratch, __u32 pid)
{
	struct egress_key ekey = {};

	if (scan_request_hdr(scratch->buf, scratch->len, &scratch->newline_off))
		return -1;
	if (scratch->newline_off == 0)
		return -1;

	mint_from_incoming(&scratch->tp, pid);
	if (write_tp_header(msg, scratch, scratch->newline_off) < 0)
		return -1;

	ekey.pid = pid;
	ekey.sport = (__u16)msg->local_port;
	ekey.dport = (__u16)bpf_ntohl(msg->remote_port);
	ekey.stream = 0;
	bpf_map_update_elem(&outgoing_tp, &ekey, &scratch->tp, BPF_ANY);
	return 0;
}

SEC("sk_msg")
int mochi_inject_sk_msg(struct sk_msg_md *msg)
{
	__u32 pid;
	__u32 size;
	__u32 want;
	__u32 zero = 0;
	struct inject_scratch *scratch;

	pid = (__u32)(bpf_get_current_pid_tgid() >> 32);

	size = msg->size;
	if (size < 9)
		return SK_PASS;
	want = size;
	if (want > SK_MSG_PULL_SIZE)
		want = SK_MSG_PULL_SIZE;
	if (bpf_msg_pull_data(msg, 0, want, 0) < 0)
		return SK_PASS;

	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return SK_PASS;
	scratch_reset_meta(scratch);

	if (copy_msg_to_scratch(msg, scratch, want) < 9)
		return SK_PASS;

	if (is_http_request_bytes(scratch->buf)) {
		if (scratch->len < MIN_HTTP_REQ_SIZE)
			return SK_PASS;
		inject_h1_sk_msg(msg, scratch, pid);
		return SK_PASS;
	}

	inject_h2_sk_msg(msg, scratch, pid);
	return SK_PASS;
}

SEC("sockops")
int mochi_inject_sockops(struct bpf_sock_ops *skops)
{
	__u32 op = skops->op;
	__u64 cookie;

	/* Outbound clients only. Accepted sockets in SOCKHASH break sk_msg.
	 * bpf_sock_hash_update uses the sockops context. Do not require skops sk. */
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

static __always_inline int recv_is_tls_record(struct recv_stash *stash)
{
	__u8 hdr[3] = {};
	int i;

#pragma unroll
	for (i = 0; i < IOV_STASH_SEGS; i++) {
		if (i >= stash->nsegs)
			break;
		if (stash->segs[i].len == 0 || stash->segs[i].base == 0)
			continue;
		if (stash->segs[i].len < 3)
			return 0;
		if (bpf_probe_read_user(hdr, 3, (void *)stash->segs[i].base) < 0)
			return 0;
		return tls_record_bytes(hdr);
	}
	return 0;
}

SEC("fentry/tcp_recvmsg")
int BPF_PROG(mochi_inject_recv_enter, struct sock *sk, struct msghdr *msg, size_t len, int flags)
{
	__u64 id;
	struct recv_stash st = {};
	int nsegs;

	(void)len;
	if (flags & MSG_PEEK)
		return 0;
	if (!remote_ok(sk))
		return 0;
	nsegs = iov_stash(msg, st.segs);
	if (nsegs <= 0)
		return 0;
	st.nsegs = (__u8)nsegs;
	if (recv_is_tls_record(&st))
		return 0;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&inject_recv_pending, &id, &st, BPF_ANY);
	return 0;
}

/*
 * Copy up to cap bytes of the received payload. When got > cap, take the tail
 * so an appended HPACK Traceparent past preface/SETTINGS still lands in range.
 * Walks stashed segs to the tail start, then probes that one segment into
 * dst[0]. A tail that spans later segs is a miss. UBUF fexit may still land.
 */
static __always_inline __u32 copy_recv_tail(struct recv_stash *stash, __u8 *dst, __u32 cap, __u32 got)
{
	__u32 need;
	__u32 skip;
	__u32 take;
	__u64 base;
	int i;

	if (stash->nsegs == 0 || got == 0)
		return 0;
	if (cap > RECV_COPY_MAX)
		cap = RECV_COPY_MAX;
	need = got;
	if (need > cap)
		need = cap;
	skip = got > need ? got - need : 0;

#pragma unroll
	for (i = 0; i < IOV_STASH_SEGS; i++) {
		__u32 seg_len;

		if (i >= stash->nsegs)
			break;
		seg_len = stash->segs[i].len;
		base = stash->segs[i].base;
		if (seg_len == 0 || base == 0)
			continue;
		if (skip >= seg_len) {
			skip -= seg_len;
			continue;
		}
		take = seg_len - skip;
		if (take > need)
			take = need;
		/* Do not mask with RECV_COPY_MAX minus one. A full size copy becomes 0. */
		if (take == 0 || take > RECV_COPY_MAX)
			return 0;
		if (bpf_probe_read_user(dst, take, (const void *)(base + skip)) < 0)
			return 0;
		return take;
	}
	return 0;
}

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
	if (cap > RECV_COPY_MAX)
		cap = RECV_COPY_MAX;
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
	if (!ubuf || off < got)
		return 0;
	start = (__u64)ubuf + off - chunk;
	if (bpf_probe_read_user(dst, chunk, (const void *)start) < 0)
		return 0;
	return chunk;
}

static __always_inline void stash_incoming(__u32 tgid, struct tp_ids *tp)
{
	__u64 key = tgid;

	__builtin_memcpy(tp->span_id, tp->parent_id, TP_SPAN_ID_SIZE);
	__builtin_memset(tp->parent_id, 0, TP_SPAN_ID_SIZE);
	bpf_map_update_elem(&incoming_tp, &key, tp, BPF_ANY);
}

struct incoming_scan_ctx {
	const __u8 *buf;
	__u32 len;
	struct tp_ids *out;
	__u8 found;
};

/* One pass over ASCII Traceparent (tp_is_name) or HPACK 0x0b "traceparent" literal. */
static long incoming_scan_cb(__u32 index, void *data)
{
	struct incoming_scan_ctx *ctx = data;
	const __u8 *base = ctx->buf;
	__u32 i = index & (RECV_COPY_MAX - 1);
	__u32 prev;
	__u32 val_off;
	__u32 name_end;
	__u8 before = 0;
	__u8 hpack_lit = 0;

	if (i >= ctx->len)
		return 1;

	if (i + TP_PREFIX_LEN + TP_VAL_LEN <= ctx->len && tp_is_name(base + i)) {
		val_off = tp_value_off(base + i, ctx->len - i);
		if (val_off &&
		    i + val_off + TP_VAL_LEN <= ctx->len &&
		    tp_parse_value(base + i + val_off, ctx->out) == 0) {
			ctx->found = 1;
			return 1;
		}
	}

	if (i + HPACK_TP_NAME_LEN + 1 + HPACK_TP_VAL_LEN > ctx->len)
		return 1;
	if (i >= 1) {
		prev = (i - 1) & (RECV_COPY_MAX - 1);
		before = base[prev];
		if (before == HPACK_TP_NAME_LEN)
			hpack_lit = 1;
	}
	if (!hpack_lit || !hpack_name_raw(base + i))
		return 0;
	name_end = (i + HPACK_TP_NAME_LEN) & (RECV_COPY_MAX - 1);
	val_off = name_end;
	if (val_off >= ctx->len || val_off >= RECV_COPY_MAX)
		return 0;
	if ((base[val_off] & HPACK_STRING_LEN_MASK) != HPACK_TP_VAL_LEN)
		return 0;
	if (base[val_off] & HPACK_STRING_HUFF_BIT)
		return 0;
	val_off = (val_off + 1) & (RECV_COPY_MAX - 1);
	if (val_off + HPACK_TP_VAL_LEN > ctx->len ||
	    val_off + HPACK_TP_VAL_LEN > RECV_COPY_MAX)
		return 0;
	if (tp_parse_value(base + val_off, ctx->out) == 0) {
		ctx->found = 1;
		return 1;
	}
	return 0;
}

static __always_inline void parse_incoming_buf(__u8 *buf, __u32 copied, __u32 tgid, struct inject_scratch *scratch)
{
	struct incoming_scan_ctx ctx = {};
	__u32 n;

	if (copied < HPACK_TP_NAME_LEN + 1 + HPACK_TP_VAL_LEN)
		return;
	n = copied;
	if (n > RECV_COPY_MAX)
		n = RECV_COPY_MAX;
	ctx.buf = buf;
	ctx.len = n;
	ctx.out = &scratch->tp;
	bpf_loop(n, incoming_scan_cb, &ctx, 0);
	if (ctx.found &&
	    !tp_ids_zero_trace(scratch->tp.trace_id) &&
	    !tp_ids_zero_span(scratch->tp.parent_id))
		stash_incoming(tgid, &scratch->tp);
}

SEC("fexit/tcp_recvmsg")
int BPF_PROG(mochi_inject_recv_exit, struct sock *sk, struct msghdr *msg, size_t len, int flags, int ret)
{
	__u64 id;
	__u32 zero = 0;
	struct recv_stash *stash;
	struct inject_scratch *scratch;
	__u32 copied;

	(void)sk;
	(void)len;
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
	scratch_reset_meta(scratch);
	copied = copy_recv_tail(stash, scratch->buf, RECV_COPY_MAX, (__u32)ret);
	if (copied == 0)
		copied = copy_recv_from_msg(msg, scratch->buf, RECV_COPY_MAX, (__u32)ret);
	if (copied == 0)
		return 0;
	parse_incoming_buf(scratch->buf, copied, (__u32)(id >> 32), scratch);
	return 0;
}

struct user_move_ctx {
	__u64 base;
	__u32 off;
	__u32 remain;
	__u32 insert_len;
	__u8 *chunk;
	__u8 err;
};

static long user_move_cb(__u32 idx, void *data)
{
	struct user_move_ctx *ctx = data;
	__u32 n;
	__u64 src;
	__u64 dst;

	(void)idx;
	if (ctx->remain == 0 || ctx->err)
		return 1;
	n = ctx->remain;
	if (n > USER_MOVE_CHUNK)
		n = USER_MOVE_CHUNK;
	src = ctx->base + ctx->off + ctx->remain - n;
	dst = src + ctx->insert_len;
	if (bpf_probe_read_user(ctx->chunk, n, (const void *)src) < 0) {
		ctx->err = 1;
		return 1;
	}
	if (bpf_probe_write_user((void *)dst, ctx->chunk, n) < 0) {
		ctx->err = 1;
		return 1;
	}
	ctx->remain -= n;
	return ctx->remain == 0 ? 1 : 0;
}

static __always_inline int user_memmove_right(__u64 base, __u32 off, __u32 tail, __u32 insert_len)
{
	__u32 zero = 0;
	struct user_move_ctx ctx = {};
	__u32 iters;

	if (insert_len == 0 || tail == 0)
		return 0;
	if (tail > USER_MOVE_MAX)
		return -1;
	ctx.chunk = bpf_map_lookup_elem(&inject_move_scratch, &zero);
	if (!ctx.chunk)
		return -1;
	ctx.base = base;
	ctx.off = off;
	ctx.remain = tail;
	ctx.insert_len = insert_len;
	iters = (tail + USER_MOVE_CHUNK - 1) / USER_MOVE_CHUNK;
	if (iters > USER_MOVE_ITERS)
		iters = USER_MOVE_ITERS;
	bpf_loop(iters, user_move_cb, &ctx, 0);
	if (ctx.err || ctx.remain != 0)
		return -1;
	return 0;
}

static __always_inline void snapshot_inject_tuple(struct sock *sk, __u32 tgid, __u64 obj)
{
	struct obj_key key = {};
	struct conn_tuple tup = {};
	struct conn_tuple *cur;
	struct stream_hdr hdr = {};

	if (!obj || !remote_ok(sk))
		return;
	key.tgid = tgid;
	key.obj = obj;
	cur = bpf_map_lookup_elem(&inject_tuples, &key);
	if (cur && cur->has_tuple)
		return;
	fill_tuple(sk, &hdr);
	tup.sport = hdr.sport;
	tup.dport = hdr.dport;
	tup.has_tuple = 1;
	bpf_map_update_elem(&inject_tuples, &key, &tup, BPF_ANY);
}

/* Nested under Go Write. Locked covers non-sockmap and sockhashed TX. */
SEC("fentry/tcp_sendmsg_locked")
int BPF_PROG(mochi_inject_tx_enter, struct sock *sk, struct msghdr *msg, size_t size)
{
	__u64 id = bpf_get_current_pid_tgid();
	struct inflight_stash *inf;

	(void)msg;
	(void)size;
	inf = bpf_map_lookup_elem(&inject_inflight, &id);
	if (!inf || !inf->obj)
		return 0;
	snapshot_inject_tuple(sk, id >> 32, inf->obj);
	return 0;
}

static __always_inline int classify_and_inject_user(__u64 buf, __u32 len, __u32 cap, struct go_stash *stash, __u32 tgid)
{
	__u32 zero = 0;
	__u32 want;
	__u32 insert_len;
	__u32 insert_off;
	__u32 tail;
	__u32 hpack_len;
	struct inject_scratch *scratch;
	struct go_seen_key seen = {};
	struct go_has_key has_key = {};
	__u8 one = 1;
	__u8 *has;
	__u8 fl[3];
	__u32 new_len;

	if (len < 9 || cap < len)
		return -1;
	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return -1;
	scratch_reset_meta(scratch);

	want = len;
	if (want > SK_MSG_PULL_SIZE)
		want = SK_MSG_PULL_SIZE;
	if (bpf_probe_read_user(scratch->buf, want, (const void *)buf) < 0)
		return -1;
	scratch->len = want;

	has_key.tgid = tgid;
	has_key.conn = stash->conn;
	has = bpf_map_lookup_elem(&go_has_tp, &has_key);
	if (has)
		return -1;

	if (is_http_request_bytes(scratch->buf)) {
		if (want < MIN_HTTP_REQ_SIZE)
			return -1;
		if (scan_request_hdr(scratch->buf, want, &scratch->newline_off))
			return -1;
		if (scratch->newline_off == 0)
			return -1;
		insert_off = scratch->newline_off;
		insert_len = TP_HEADER_LEN;
		stash->stream = 0;
		if (cap - len < insert_len)
			return -1;
		mint_from_incoming(&stash->tp, tgid);
		tail = len - insert_off;
		if (user_memmove_right(buf, insert_off, tail, insert_len) < 0)
			return -1;
		tp_write_header(scratch->hdr, &stash->tp);
		if (bpf_probe_write_user((void *)(buf + insert_off), scratch->hdr, TP_HEADER_LEN) < 0)
			return -1;
		stash->insert_len = insert_len;
		stash->did_inject = 1;
		return 0;
	}

	if (find_h2_headers(scratch->buf, scratch->len, scratch) < 0)
		return -1;

	seen.tgid = tgid;
	seen.stream = scratch->stream;
	seen.conn = stash->conn;
	if (bpf_map_update_elem(&go_hdr_seen, &seen, &one, BPF_NOEXIST) != 0)
		return -1;

	if (scratch->insert_off >= SK_MSG_PULL_SIZE)
		return -1;
	hpack_len = scratch->hpack_len;
	if (hpack_len > SK_MSG_PULL_SIZE ||
	    scratch->insert_off + hpack_len > SK_MSG_PULL_SIZE)
		return -1;
	if (hpack_scan_range(scratch->buf, scratch->insert_off, hpack_len, NULL)) {
		bpf_map_update_elem(&go_has_tp, &has_key, &one, BPF_ANY);
		return -1;
	}
	if (scratch->frame_len + HPACK_TP_LIT_LEN > H2_MAX_FRAME_SIZE)
		return -1;
	insert_off = scratch->insert_off + hpack_len;
	insert_len = HPACK_TP_LIT_LEN;
	if (insert_off >= SK_MSG_PULL_SIZE || cap - len < insert_len)
		return -1;

	mint_from_incoming(&stash->tp, tgid);
	tail = len - insert_off;
	if (user_memmove_right(buf, insert_off, tail, insert_len) < 0)
		return -1;
	hpack_write_tp(scratch->hpack, &stash->tp);
	if (bpf_probe_write_user((void *)(buf + insert_off), scratch->hpack, HPACK_TP_LIT_LEN) < 0)
		return -1;
	new_len = scratch->frame_len + HPACK_TP_LIT_LEN;
	fl[0] = (__u8)((new_len >> 16) & 0xff);
	fl[1] = (__u8)((new_len >> 8) & 0xff);
	fl[2] = (__u8)(new_len & 0xff);
	if (bpf_probe_write_user((void *)(buf + scratch->frame_off), fl, 3) < 0)
		return -1;
	stash->stream = scratch->stream;
	stash->insert_len = insert_len;
	stash->did_inject = 1;
	return 0;
}

SEC("uprobe")
int mochi_inject_go_enter(struct pt_regs *ctx)
{
	__u64 id;
	__u64 cookie = bpf_get_attach_cookie(ctx);
	struct go_key key = {};
	struct go_stash stash = {};
	struct inflight_stash inf = {};
	__u32 tgid;
	__u64 bound;
	__u64 cap;

	id = bpf_get_current_pid_tgid();
	tgid = id >> 32;

	if (cookie == INJECT_COOKIE_GO_WRITE) {
		stash.conn = ctx->ax;
		stash.buf = ctx->bx;
		stash.bound = ctx->cx > 0xffffffff ? 0xffffffff : (__u32)ctx->cx;
		cap = ctx->di;
		bound = stash.bound;
		key.g = ctx->r14;
		if (!stash.conn || !stash.buf || !key.g || bound == 0)
			return 0;
		stash.tid = id;
		stash.is_write = 1;
		if (classify_and_inject_user(stash.buf, stash.bound, (__u32)cap, &stash, tgid) == 0) {
			ctx->cx = stash.bound + stash.insert_len;
			/* Nested tcp_sendmsg(_locked) only snapshots the Conn* after a successful insert. */
			inf.obj = stash.conn;
			inf.buf = stash.buf;
			inf.bound = stash.bound + stash.insert_len;
			inf.dir = STREAM_DIR_SEND;
			inf.kind = STREAM_KIND_GOTLS;
			bpf_map_update_elem(&inject_inflight, &id, &inf, BPF_ANY);
		}
		key.tgid = tgid;
		bpf_map_update_elem(&go_active, &key, &stash, BPF_ANY);
		return 0;
	}

	if (cookie == INJECT_COOKIE_GO_READ) {
		stash.conn = ctx->ax;
		stash.buf = ctx->bx;
		bound = ctx->cx;
		key.g = ctx->r14;
		if (!stash.conn || !stash.buf || !key.g || bound == 0)
			return 0;
		stash.bound = bound > 0xffffffff ? 0xffffffff : (__u32)bound;
		stash.tid = id;
		stash.did_inject = 0;
		stash.is_write = 0;
		key.tgid = tgid;
		bpf_map_update_elem(&go_active, &key, &stash, BPF_ANY);
		return 0;
	}
	return 0;
}

SEC("uprobe")
int mochi_inject_go_ret(struct pt_regs *ctx)
{
	__u64 id;
	__u64 cookie = bpf_get_attach_cookie(ctx);
	struct go_key key = {};
	struct go_stash *found;
	struct go_stash stash;
	struct inflight_stash *inf;
	struct obj_key okey = {};
	struct conn_tuple *tup;
	struct egress_key ekey = {};
	__s64 n;
	__u32 zero = 0;
	struct inject_scratch *scratch;
	__u32 copy_len;
	__u32 skip;
	__u64 src;

	if (cookie != INJECT_COOKIE_GO_RET)
		return 0;

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

	inf = bpf_map_lookup_elem(&inject_inflight, &stash.tid);
	if (inf && inf->obj == stash.conn)
		bpf_map_delete_elem(&inject_inflight, &stash.tid);

	if (stash.is_write) {
		if (!stash.did_inject)
			return 0;
		/* Restore caller visible n to the original slice length. */
		ctx->ax = stash.bound;
		okey.tgid = key.tgid;
		okey.obj = stash.conn;
		tup = bpf_map_lookup_elem(&inject_tuples, &okey);
		if (tup && tup->has_tuple) {
			ekey.pid = key.tgid;
			ekey.sport = tup->sport;
			ekey.dport = tup->dport;
			ekey.stream = stash.stream;
			bpf_map_update_elem(&outgoing_tp, &ekey, &stash.tp, BPF_ANY);
		}
		return 0;
	}

	n = (__s64)ctx->ax;
	if (n <= 0)
		return 0;
	copy_len = n > 0xffffffff ? 0xffffffff : (__u32)n;
	if (copy_len > stash.bound)
		copy_len = stash.bound;
	skip = 0;
	if (copy_len > RECV_COPY_MAX) {
		skip = copy_len - RECV_COPY_MAX;
		copy_len = RECV_COPY_MAX;
	}
	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return 0;
	scratch_reset_meta(scratch);
	src = stash.buf + skip;
	if (bpf_probe_read_user(scratch->buf, copy_len, (const void *)src) < 0)
		return 0;
	parse_incoming_buf(scratch->buf, copy_len, key.tgid, scratch);
	return 0;
}

SEC("uprobe")
int mochi_inject_ssl_enter(struct pt_regs *ctx)
{
	__u64 id;
	__u64 cookie = bpf_get_attach_cookie(ctx);
	struct inflight_stash act = {};

	if (cookie != INJECT_COOKIE_SSL_READ && cookie != INJECT_COOKIE_SSL_READ_EX)
		return 0;
	act.obj = ctx->di;
	act.buf = ctx->si;
	if (!act.obj || !act.buf)
		return 0;
	act.dir = STREAM_DIR_RECV;
	act.kind = STREAM_KIND_OPENSSL;
	act.ex = cookie == INJECT_COOKIE_SSL_READ_EX;
	if (act.ex)
		act.aux = ctx->cx;
	id = bpf_get_current_pid_tgid();
	bpf_map_update_elem(&inject_inflight, &id, &act, BPF_ANY);
	return 0;
}

SEC("uretprobe")
int mochi_inject_ssl_ret(struct pt_regs *ctx)
{
	__u64 id;
	struct inflight_stash *found;
	struct inflight_stash act;
	int ret;
	__u32 bytes = 0;
	__u64 n = 0;
	__u32 zero = 0;
	struct inject_scratch *scratch;
	__u32 skip = 0;
	__u64 src;

	id = bpf_get_current_pid_tgid();
	found = bpf_map_lookup_elem(&inject_inflight, &id);
	if (!found)
		return 0;
	act = *found;
	bpf_map_delete_elem(&inject_inflight, &id);
	if (act.dir != STREAM_DIR_RECV || act.kind != STREAM_KIND_OPENSSL)
		return 0;

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
	if (bytes > RECV_COPY_MAX) {
		skip = bytes - RECV_COPY_MAX;
		bytes = RECV_COPY_MAX;
	}
	scratch = bpf_map_lookup_elem(&inject_scratch, &zero);
	if (!scratch)
		return 0;
	scratch_reset_meta(scratch);
	src = act.buf + skip;
	if (bpf_probe_read_user(scratch->buf, bytes, (const void *)src) < 0)
		return 0;
	parse_incoming_buf(scratch->buf, bytes, id >> 32, scratch);
	return 0;
}
