// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE program: client-outbound UDP flows via successful udp_sendmsg.
// First datagram of a 4-tuple emits an OPEN event. Later sends accumulate tx in-map.
// Flow end and lifetime bytes are handled in userspace idle GC.
// Regenerate Go bindings with: go generate ./internal/collection/ebpf (requires clang).

#include "vmlinux.h"
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define AF_INET 2
#define AF_INET6 10

struct flow_key {
	__u8 family;
	__u8 pad[1];
	__u16 sport;
	__u16 dport;
	__u8 saddr[16];
	__u8 daddr[16];
} __attribute__((packed));

struct flow_val {
	__u32 pid;
	__u32 pad;
	__u64 cgroup_id;
	__u64 tx_bytes;
	__u64 last_ns;
} __attribute__((packed));

struct open_event {
	__u32 pid;
	__u16 family;
	__u16 sport;
	__u16 dport;
	__u8 pad[2];
	__u8 saddr[16];
	__u8 daddr[16];
	__u64 cgroup_id;
} __attribute__((packed));

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 20);
} open_events SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, struct flow_key);
	__type(value, struct flow_val);
	__uint(max_entries, 65536);
} flows SEC(".maps");

static __always_inline int is_loopback_v4(__u8 *addr)
{
	return addr[0] == 127;
}

static __always_inline int is_unspecified_v4(__u8 *addr)
{
	return addr[0] == 0 && addr[1] == 0 && addr[2] == 0 && addr[3] == 0;
}

static __always_inline int is_unspecified_v6(__u8 *addr)
{
	for (int i = 0; i < 16; i++)
		if (addr[i] != 0)
			return 0;
	return 1;
}

static __always_inline int is_loopback_v6(__u8 *addr)
{
	if (addr[15] != 1)
		return 0;
	for (int i = 0; i < 15; i++)
		if (addr[i] != 0)
			return 0;
	return 1;
}

static __always_inline int is_v4mapped_v6(__u8 *addr)
{
	for (int i = 0; i < 10; i++)
		if (addr[i] != 0)
			return 0;
	return addr[10] == 0xff && addr[11] == 0xff;
}

static __always_inline int v6_dst_ok(__u8 *addr)
{
	if (is_unspecified_v6(addr) || is_loopback_v6(addr))
		return 0;
	if (is_v4mapped_v6(addr))
		return !is_unspecified_v4(addr + 12) && !is_loopback_v4(addr + 12);
	return 1;
}

static __always_inline void fill_src_v4(struct sock *sk, struct flow_key *key)
{
	__u32 saddr;

	saddr = BPF_CORE_READ(sk, __sk_common.skc_rcv_saddr);
	key->family = AF_INET;
	key->sport = BPF_CORE_READ(sk, __sk_common.skc_num);
	__builtin_memset(key->saddr, 0, sizeof(key->saddr));
	__builtin_memcpy(key->saddr, &saddr, 4);
}

static __always_inline void fill_src_v6(struct sock *sk, struct flow_key *key)
{
	key->family = AF_INET6;
	key->sport = BPF_CORE_READ(sk, __sk_common.skc_num);
	BPF_CORE_READ_INTO(&key->saddr, sk, __sk_common.skc_v6_rcv_saddr);
}

static __always_inline int fill_dst_v4_sock(struct sock *sk, struct flow_key *key)
{
	__u32 daddr;
	__u16 dport;

	daddr = BPF_CORE_READ(sk, __sk_common.skc_daddr);
	dport = BPF_CORE_READ(sk, __sk_common.skc_dport);
	if (daddr == 0 || dport == 0)
		return -1;
	key->dport = bpf_ntohs(dport);
	__builtin_memset(key->daddr, 0, sizeof(key->daddr));
	__builtin_memcpy(key->daddr, &daddr, 4);
	if (is_unspecified_v4(key->daddr) || is_loopback_v4(key->daddr))
		return -1;
	return 0;
}

static __always_inline int fill_dst_v6_sock(struct sock *sk, struct flow_key *key)
{
	__u16 dport;

	dport = BPF_CORE_READ(sk, __sk_common.skc_dport);
	if (dport == 0)
		return -1;
	key->dport = bpf_ntohs(dport);
	BPF_CORE_READ_INTO(&key->daddr, sk, __sk_common.skc_v6_daddr);
	if (!v6_dst_ok(key->daddr))
		return -1;
	return 0;
}

static __always_inline int fill_dst_v4_msg(void *msg_name, struct flow_key *key)
{
	struct sockaddr_in sin = {};

	if (!msg_name)
		return -1;
	if (bpf_probe_read_user(&sin, sizeof(sin), msg_name) < 0)
		return -1;
	if (sin.sin_port == 0)
		return -1;
	key->dport = bpf_ntohs(sin.sin_port);
	__builtin_memset(key->daddr, 0, sizeof(key->daddr));
	__builtin_memcpy(key->daddr, &sin.sin_addr.s_addr, 4);
	if (is_unspecified_v4(key->daddr) || is_loopback_v4(key->daddr))
		return -1;
	return 0;
}

static __always_inline int fill_dst_v6_msg(void *msg_name, struct flow_key *key)
{
	struct sockaddr_in6 sin6 = {};

	if (!msg_name)
		return -1;
	if (bpf_probe_read_user(&sin6, sizeof(sin6), msg_name) < 0)
		return -1;
	if (sin6.sin6_port == 0)
		return -1;
	key->dport = bpf_ntohs(sin6.sin6_port);
	__builtin_memset(key->daddr, 0, sizeof(key->daddr));
	__builtin_memcpy(key->daddr, &sin6.sin6_addr, 16);
	if (!v6_dst_ok(key->daddr))
		return -1;
	return 0;
}

static __always_inline int handle_udp_send(struct sock *sk, struct msghdr *msg, size_t len, int ret)
{
	struct flow_key key = {};
	struct flow_val *val;
	struct flow_val new_val = {};
	struct open_event *e;
	void *msg_name;
	__u64 now;
	int is_new = 0;

	if (ret < 0)
		return 0;

	if (BPF_CORE_READ(sk, __sk_common.skc_family) == AF_INET) {
		fill_src_v4(sk, &key);
		msg_name = BPF_CORE_READ(msg, msg_name);
		if (fill_dst_v4_msg(msg_name, &key) < 0) {
			if (fill_dst_v4_sock(sk, &key) < 0)
				return 0;
		}
	} else if (BPF_CORE_READ(sk, __sk_common.skc_family) == AF_INET6) {
		fill_src_v6(sk, &key);
		msg_name = BPF_CORE_READ(msg, msg_name);
		if (fill_dst_v6_msg(msg_name, &key) < 0) {
			if (fill_dst_v6_sock(sk, &key) < 0)
				return 0;
		}
	} else {
		return 0;
	}

	if (key.sport == 0 || key.dport == 0)
		return 0;

	now = bpf_ktime_get_ns();
	val = bpf_map_lookup_elem(&flows, &key);
	if (!val) {
		new_val.pid = (__u32)(bpf_get_current_pid_tgid() >> 32);
		new_val.cgroup_id = bpf_get_current_cgroup_id();
		new_val.tx_bytes = len;
		new_val.last_ns = now;
		if (bpf_map_update_elem(&flows, &key, &new_val, BPF_NOEXIST) != 0)
			return 0;
		is_new = 1;
	} else {
		__u64 tx = val->tx_bytes + len;
		val->tx_bytes = tx;
		val->last_ns = now;
	}

	if (!is_new)
		return 0;

	e = bpf_ringbuf_reserve(&open_events, sizeof(*e), 0);
	if (!e)
		return 0;

	e->pid = new_val.pid;
	e->cgroup_id = new_val.cgroup_id;
	e->family = key.family;
	e->sport = key.sport;
	e->dport = key.dport;
	__builtin_memcpy(e->saddr, key.saddr, sizeof(e->saddr));
	__builtin_memcpy(e->daddr, key.daddr, sizeof(e->daddr));

	bpf_ringbuf_submit(e, 0);
	return 0;
}

SEC("fexit/udp_sendmsg")
int BPF_PROG(mochi_udp_sendmsg, struct sock *sk, struct msghdr *msg, size_t len, int ret)
{
	return handle_udp_send(sk, msg, len, ret);
}

SEC("fexit/udpv6_sendmsg")
int BPF_PROG(mochi_udpv6_sendmsg, struct sock *sk, struct msghdr *msg, size_t len, int ret)
{
	return handle_udp_send(sk, msg, len, ret);
}
