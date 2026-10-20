/* HPACK never-indexed Traceparent for HTTP/2 inject (RFC 7541 §6.2.3).
 * Does not touch the dynamic table. Inject-only. No full decoder. */

#ifndef MOCHI_BPF_HPACK_H
#define MOCHI_BPF_HPACK_H

#include "traceparent.h"

/* 0x10 | name_len 0x0b | "traceparent" | value_len 0x37 | 55-byte value */
#define HPACK_TP_LIT_LEN 69
#define HPACK_TP_NAME_LEN 11
#define HPACK_TP_VAL_LEN TP_VAL_LEN
#define HPACK_BUF_MAX 512
#define HPACK_SCAN_MAX HPACK_BUF_MAX

#define HPACK_NEVER_INDEX 0x10
#define HPACK_NO_INDEX    0x00
#define HPACK_INC_INDEX   0x40
#define HPACK_INDEXED_BIT 0x80
#define HPACK_INC_INDEX_MASK 0xc0
#define HPACK_INC_INDEX_VAL_MASK 0x3f
#define HPACK_LIT_PREFIX_MASK 0xf0
#define HPACK_LIT_INDEX_MASK  0x0f
#define HPACK_STRING_HUFF_BIT 0x80
#define HPACK_STRING_LEN_MASK 0x7f

#define H2_FRAME_HDR_LEN  9
#define H2_FRAME_HEADERS  0x1
#define H2_FLAG_END_HEADERS 0x4
#define H2_FLAG_PADDED    0x8
#define H2_FLAG_PRIORITY  0x20
#define H2_PRIORITY_LEN   5
#define H2_MAX_FRAME_SIZE 16384

/* Huffman-encoded "traceparent" (RFC 7541 Appendix B / grpc-go encoder). */
static const __u8 hpack_tp_name_huff[] = {
	0x4d, 0x83, 0x21, 0x6b, 0x1d, 0x85, 0xa9, 0x3f
};
#define HPACK_TP_NAME_HUFF_LEN 8

struct hpack_scan_ctx {
	const __u8 *base;
	__u32 start;
	__u32 end;
	struct tp_ids *out;
	__u8 found;
	__u8 parsed;
};

static __always_inline int hpack_name_raw(const __u8 *p)
{
	return p[0] == 't' && p[1] == 'r' && p[2] == 'a' && p[3] == 'c' &&
	       p[4] == 'e' && p[5] == 'p' && p[6] == 'a' && p[7] == 'r' &&
	       p[8] == 'e' && p[9] == 'n' && p[10] == 't';
}

static __always_inline int hpack_name_huff(const __u8 *p)
{
	return p[0] == hpack_tp_name_huff[0] &&
	       p[1] == hpack_tp_name_huff[1] &&
	       p[2] == hpack_tp_name_huff[2] &&
	       p[3] == hpack_tp_name_huff[3] &&
	       p[4] == hpack_tp_name_huff[4] &&
	       p[5] == hpack_tp_name_huff[5] &&
	       p[6] == hpack_tp_name_huff[6] &&
	       p[7] == hpack_tp_name_huff[7];
}

/* Write 69-octet never-indexed literal at dst. Caller bounds-checked. */
static __always_inline void hpack_write_tp(__u8 *dst, const struct tp_ids *tp)
{
	__u8 flags_lo;

	dst[0] = HPACK_NEVER_INDEX;
	dst[1] = HPACK_TP_NAME_LEN;
	dst[2] = 't';
	dst[3] = 'r';
	dst[4] = 'a';
	dst[5] = 'c';
	dst[6] = 'e';
	dst[7] = 'p';
	dst[8] = 'a';
	dst[9] = 'r';
	dst[10] = 'e';
	dst[11] = 'n';
	dst[12] = 't';
	dst[13] = HPACK_TP_VAL_LEN;
	dst[14] = '0';
	dst[15] = '0';
	dst[16] = '-';
	tp_encode_hex_trace(dst + 17, tp->trace_id);
	dst[49] = '-';
	tp_encode_hex_span(dst + 50, tp->span_id);
	dst[66] = '-';
	flags_lo = (tp->flags & TP_SAMPLED_FLAG) ? '1' : '0';
	dst[67] = '0';
	dst[68] = flags_lo;
}

static long hpack_scan_cb(__u32 index, void *data)
{
	struct hpack_scan_ctx *ctx = data;
	__u32 i = (ctx->start + index) & (HPACK_BUF_MAX - 1);
	__u8 prefix;
	__u32 name_off;
	__u8 name_len_byte;
	__u8 name_huff;
	__u32 name_len;
	__u32 val_off;
	__u8 val_len_byte;
	__u8 val_huff;
	__u32 val_len;

	if (index >= HPACK_SCAN_MAX)
		return 1;
	if (i >= ctx->end || i + 3 + HPACK_TP_NAME_LEN + 1 + HPACK_TP_VAL_LEN > ctx->end)
		return 1;
	if (i + 3 + HPACK_TP_NAME_LEN + 1 + HPACK_TP_VAL_LEN > HPACK_BUF_MAX)
		return 1;

	prefix = ctx->base[i];
	if (prefix & HPACK_INDEXED_BIT)
		return 0;
	if ((prefix & HPACK_INC_INDEX_MASK) == HPACK_INC_INDEX) {
		if ((prefix & HPACK_INC_INDEX_VAL_MASK) != 0)
			return 0;
		name_off = i + 1;
	} else if ((prefix & HPACK_LIT_PREFIX_MASK) == HPACK_NEVER_INDEX ||
		   (prefix & HPACK_LIT_PREFIX_MASK) == HPACK_NO_INDEX) {
		if ((prefix & HPACK_LIT_INDEX_MASK) != 0)
			return 0;
		name_off = i + 1;
	} else {
		return 0;
	}
	name_off &= (HPACK_BUF_MAX - 1);
	if (name_off >= ctx->end || name_off >= HPACK_BUF_MAX)
		return 0;
	name_len_byte = ctx->base[name_off];
	name_huff = name_len_byte & HPACK_STRING_HUFF_BIT;
	name_len = name_len_byte & HPACK_STRING_LEN_MASK;
	if (name_len == 0 || name_len > HPACK_STRING_LEN_MASK)
		return 0;
	if (name_off + 1 + name_len + 1 > ctx->end ||
	    name_off + 1 + name_len + 1 > HPACK_BUF_MAX)
		return 0;
	if (name_huff) {
		if (name_len != HPACK_TP_NAME_HUFF_LEN)
			return 0;
		if (!hpack_name_huff(ctx->base + name_off + 1))
			return 0;
	} else {
		if (name_len != HPACK_TP_NAME_LEN)
			return 0;
		if (!hpack_name_raw(ctx->base + name_off + 1))
			return 0;
	}
	val_off = name_off + 1 + name_len;
	val_off &= (HPACK_BUF_MAX - 1);
	if (val_off >= ctx->end || val_off >= HPACK_BUF_MAX)
		return 1;
	val_len_byte = ctx->base[val_off];
	val_huff = val_len_byte & HPACK_STRING_HUFF_BIT;
	val_len = val_len_byte & HPACK_STRING_LEN_MASK;
	/* Name matched. Wrong value length: keep scanning (not our field). */
	if (val_len != HPACK_TP_VAL_LEN)
		return 0;
	if (val_off + 1 + val_len > ctx->end ||
	    val_off + 1 + val_len > HPACK_BUF_MAX)
		return 0;
	ctx->found = 1;
	if (ctx->out && !val_huff &&
	    tp_parse_value(ctx->base + val_off + 1, ctx->out) == 0)
		ctx->parsed = 1;
	return 1;
}

/*
 * Scan HPACK bytes base[start, start+len) for a literal "traceparent" name.
 * base must be the start of a HPACK_BUF_MAX scratch window.
 */
static __always_inline int hpack_scan_range(const __u8 *base, __u32 start, __u32 len,
					    struct tp_ids *out)
{
	struct hpack_scan_ctx ctx = {};
	__u32 n;
	__u32 end;

	if (start >= HPACK_BUF_MAX)
		return 0;
	end = start + len;
	if (end > HPACK_BUF_MAX)
		end = HPACK_BUF_MAX;
	if (end < start + 3 + HPACK_TP_NAME_LEN + 1 + HPACK_TP_VAL_LEN)
		return 0;
	ctx.base = base;
	ctx.start = start;
	ctx.end = end;
	ctx.out = out;
	n = end - start;
	if (n > HPACK_SCAN_MAX)
		n = HPACK_SCAN_MAX;
	bpf_loop(n, hpack_scan_cb, &ctx, 0);
	if (out)
		return ctx.parsed;
	return ctx.found;
}

#endif
