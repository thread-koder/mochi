/* W3C Trace Context version-00 Traceparent header for sk_msg inject.
 * Full header: "Traceparent: 00-{32hex}-{16hex}-{2hex}\r\n" (70 octets). */

#ifndef MOCHI_BPF_TRACEPARENT_H
#define MOCHI_BPF_TRACEPARENT_H

#define TP_TRACE_ID_SIZE 16
#define TP_SPAN_ID_SIZE  8

#define TP_VAL_LEN       55
#define TP_HEADER_LEN    70
#define TP_NAME_LEN      11
#define TP_NAME_COLON    12
#define TP_PREFIX_LEN    13

#define TP_SAMPLED_FLAG  0x01

#define TP_HEX_TRACE_LEN 32
#define TP_HEX_SPAN_LEN  16
#define TP_HEX_FLAGS_LEN 2

struct tp_ids {
	__u8 trace_id[TP_TRACE_ID_SIZE];
	__u8 span_id[TP_SPAN_ID_SIZE];
	__u8 parent_id[TP_SPAN_ID_SIZE];
	__u8 flags;
	__u8 pad[3];
};

struct egress_key {
	__u32 pid;
	__u16 sport;
	__u16 dport;
	__u32 stream;
};

static __always_inline int tp_ids_zero_trace(const __u8 *id)
{
	__u64 a, b;

	__builtin_memcpy(&a, id, 8);
	__builtin_memcpy(&b, id + 8, 8);
	return a == 0 && b == 0;
}

static __always_inline int tp_ids_zero_span(const __u8 *id)
{
	__u64 a;

	__builtin_memcpy(&a, id, 8);
	return a == 0;
}

static __always_inline __u8 tp_hex_digit(__u8 nibble)
{
	return nibble < 10 ? ('0' + nibble) : ('a' + nibble - 10);
}

static __always_inline void tp_encode_byte(__u8 *dst, __u8 b)
{
	dst[0] = tp_hex_digit(b >> 4);
	dst[1] = tp_hex_digit(b & 0xf);
}

static __always_inline void tp_encode_hex_trace(__u8 *dst, const __u8 *src)
{
	__u32 i;

	for (i = 0; i < TP_TRACE_ID_SIZE; i++)
		tp_encode_byte(dst + i * 2, src[i]);
}

static __always_inline void tp_encode_hex_span(__u8 *dst, const __u8 *src)
{
	__u32 i;

	for (i = 0; i < TP_SPAN_ID_SIZE; i++)
		tp_encode_byte(dst + i * 2, src[i]);
}

static __always_inline int tp_decode_hex_nibble(__u8 c, __u8 *out)
{
	if (c >= '0' && c <= '9') {
		*out = c - '0';
		return 0;
	}
	if (c >= 'a' && c <= 'f') {
		*out = c - 'a' + 10;
		return 0;
	}
	if (c >= 'A' && c <= 'F') {
		*out = c - 'A' + 10;
		return 0;
	}
	return -1;
}

static __always_inline int tp_decode_byte(const __u8 *src, __u8 *dst)
{
	__u8 hi, lo;

	if (tp_decode_hex_nibble(src[0], &hi) < 0)
		return -1;
	if (tp_decode_hex_nibble(src[1], &lo) < 0)
		return -1;
	*dst = (__u8)((hi << 4) | lo);
	return 0;
}

static __always_inline int tp_decode_hex_trace(__u8 *dst, const __u8 *src)
{
	__u32 i;

	for (i = 0; i < TP_TRACE_ID_SIZE; i++) {
		if (tp_decode_byte(src + i * 2, &dst[i]) < 0)
			return -1;
	}
	return 0;
}

static __always_inline int tp_decode_hex_span(__u8 *dst, const __u8 *src)
{
	__u32 i;

	for (i = 0; i < TP_SPAN_ID_SIZE; i++) {
		if (tp_decode_byte(src + i * 2, &dst[i]) < 0)
			return -1;
	}
	return 0;
}

/* Write the 70-octet Traceparent header at buf. Caller bounds-checked. */
static __always_inline void tp_write_header(__u8 *buf, const struct tp_ids *tp)
{
	buf[0] = 'T';
	buf[1] = 'r';
	buf[2] = 'a';
	buf[3] = 'c';
	buf[4] = 'e';
	buf[5] = 'p';
	buf[6] = 'a';
	buf[7] = 'r';
	buf[8] = 'e';
	buf[9] = 'n';
	buf[10] = 't';
	buf[11] = ':';
	buf[12] = ' ';
	buf[13] = '0';
	buf[14] = '0';
	buf[15] = '-';
	tp_encode_hex_trace(buf + 16, tp->trace_id);
	buf[48] = '-';
	tp_encode_hex_span(buf + 49, tp->span_id);
	buf[65] = '-';
	buf[66] = '0';
	buf[67] = (tp->flags & TP_SAMPLED_FLAG) ? '1' : '0';
	buf[68] = '\r';
	buf[69] = '\n';
}

/* Parse "00-{tid}-{sid}-{flags}" at value (55 octets). Sets parent_id from
 * the wire parent-id field (caller span). */
static __always_inline int tp_parse_value(const __u8 *value, struct tp_ids *out)
{
	__u8 version;
	__u8 flags;

	if (value[2] != '-' || value[35] != '-' || value[52] != '-')
		return -1;
	if (tp_decode_byte(value, &version) < 0 || version == 0xff)
		return -1;
	if (tp_decode_hex_trace(out->trace_id, value + 3) < 0)
		return -1;
	if (tp_decode_hex_span(out->parent_id, value + 36) < 0)
		return -1;
	if (tp_decode_byte(value + 53, &flags) < 0)
		return -1;
	if (tp_ids_zero_trace(out->trace_id) || tp_ids_zero_span(out->parent_id))
		return -1;
	__builtin_memset(out->span_id, 0, TP_SPAN_ID_SIZE);
	out->flags = flags & TP_SAMPLED_FLAG;
	__builtin_memset(out->pad, 0, sizeof(out->pad));
	return 0;
}

static __always_inline int tp_is_name(const __u8 *p)
{
	__u8 c0 = p[0];

	if (!((c0 == 'T' || c0 == 't') &&
	      p[1] == 'r' && p[2] == 'a' && p[3] == 'c' && p[4] == 'e' &&
	      (p[5] == 'p' || p[5] == 'P') &&
	      p[6] == 'a' && p[7] == 'r' && p[8] == 'e' && p[9] == 'n' &&
	      p[10] == 't' && p[11] == ':'))
		return 0;
	return 1;
}

/* Skip optional spaces after "Traceparent:". Returns offset of value start
 * relative to name start, or 0 on failure. */
static __always_inline __u32 tp_value_off(const __u8 *p, __u32 remain)
{
	__u32 off = TP_NAME_COLON;

	if (remain < TP_PREFIX_LEN)
		return 0;
	if (p[off] == ' ')
		off++;
	if (remain < off + TP_VAL_LEN)
		return 0;
	return off;
}

static __always_inline void tp_store_u32(__u8 *dst, __u32 value)
{
	dst[0] = (__u8)value;
	dst[1] = (__u8)(value >> 8);
	dst[2] = (__u8)(value >> 16);
	dst[3] = (__u8)(value >> 24);
}

static __always_inline void tp_fill_rand_trace(__u8 *dst)
{
	tp_store_u32(dst, bpf_get_prandom_u32());
	tp_store_u32(dst + 4, bpf_get_prandom_u32());
	tp_store_u32(dst + 8, bpf_get_prandom_u32());
	tp_store_u32(dst + 12, bpf_get_prandom_u32());
}

static __always_inline void tp_fill_rand_span(__u8 *dst)
{
	tp_store_u32(dst, bpf_get_prandom_u32());
	tp_store_u32(dst + 4, bpf_get_prandom_u32());
}

static __always_inline void tp_fill_nonzero_trace(__u8 *dst)
{
	__u32 attempt;

	for (attempt = 0; attempt < 4; attempt++) {
		tp_fill_rand_trace(dst);
		if (!tp_ids_zero_trace(dst))
			return;
	}
	dst[TP_TRACE_ID_SIZE - 1] = 1;
}

static __always_inline void tp_fill_nonzero_span(__u8 *dst)
{
	__u32 attempt;

	for (attempt = 0; attempt < 4; attempt++) {
		tp_fill_rand_span(dst);
		if (!tp_ids_zero_span(dst))
			return;
	}
	dst[TP_SPAN_ID_SIZE - 1] = 1;
}

/* Big-endian lower 8 bytes of trace_id. TraceIDRatioBased (otel-go). */
static __always_inline int tp_sampled_ratio(const __u8 *trace_id, __u64 threshold)
{
	__u64 lower = ((__u64)trace_id[8] << 56) |
		     ((__u64)trace_id[9] << 48) |
		     ((__u64)trace_id[10] << 40) |
		     ((__u64)trace_id[11] << 32) |
		     ((__u64)trace_id[12] << 24) |
		     ((__u64)trace_id[13] << 16) |
		     ((__u64)trace_id[14] << 8) |
		     (__u64)trace_id[15];
	return (lower >> 1) < threshold;
}

#endif
