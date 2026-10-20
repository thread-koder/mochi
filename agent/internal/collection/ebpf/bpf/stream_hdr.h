/* Wire header for stream, TLS plaintext, and inject (remote_ok, tls_record_bytes).
 * kind 0 is the socket path. TLS sets openssl or gotls.
 * Include after vmlinux.h, bpf_core_read.h, bpf_endian.h, and dest.h.
 */

#ifndef MOCHI_BPF_STREAM_HDR_H
#define MOCHI_BPF_STREAM_HDR_H

#define AF_INET 2
#define AF_INET6 10

#define STREAM_CAP 1024
#define STREAM_DIR_RECV 0
#define STREAM_DIR_SEND 1
#define STREAM_KIND_SOCKET 0
#define STREAM_KIND_OPENSSL 1
#define STREAM_KIND_GOTLS 2

struct stream_hdr {
	__u32 pid;
	__u32 len;
	__u64 cgroup_id;
	__u16 family;
	__u16 sport;
	__u16 dport;
	__u8 dir;
	__u8 kind;
	__u8 saddr[16];
	__u8 daddr[16];
} __attribute__((packed));

struct stream_event {
	struct stream_hdr hdr;
	__u8 data[STREAM_CAP];
} __attribute__((packed));

_Static_assert(sizeof(struct stream_hdr) == 56, "stream header drifted");
_Static_assert(sizeof(struct stream_event) == 56 + STREAM_CAP, "stream event drifted");

#define TLS_CONTENT_TYPE_MIN 0x14
#define TLS_CONTENT_TYPE_MAX 0x17
#define TLS_VERSION_MAJOR    0x03
#define TLS_VERSION_MINOR_MIN 0x01
#define TLS_VERSION_MINOR_MAX 0x04

static __always_inline int tls_record_bytes(const __u8 hdr[3])
{
	if (hdr[0] < TLS_CONTENT_TYPE_MIN || hdr[0] > TLS_CONTENT_TYPE_MAX)
		return 0;
	if (hdr[1] != TLS_VERSION_MAJOR)
		return 0;
	if (hdr[2] < TLS_VERSION_MINOR_MIN || hdr[2] > TLS_VERSION_MINOR_MAX)
		return 0;
	return 1;
}

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

#endif
