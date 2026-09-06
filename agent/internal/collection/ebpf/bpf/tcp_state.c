// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// CO-RE program: TCP state transitions for client-outbound dependency edges.
// Initiator TGID and cgroup id are stashed on CLOSE→SYN_SENT (process context)
// and looked up on later transitions. Establish often runs in softirq (pid 0).
// Regenerate Go bindings with: go generate ./internal/collection/ebpf (requires clang).

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

#define AF_INET 2
#define AF_INET6 10

#define TCP_ESTABLISHED 1
#define TCP_SYN_SENT 2
#define TCP_CLOSE 7

struct event {
	__u32 pid;
	__u16 family;
	__u16 sport;
	__u16 dport;
	__u8 oldstate;
	__u8 newstate;
	__u8 pad[2];
	__u8 saddr[16];
	__u8 daddr[16];
	__u64 tx_bytes;
	__u64 rx_bytes;
	__u64 cgroup_id;
} __attribute__((packed));

struct sk_info {
	__u32 pid;
	__u32 pad;
	__u64 cgroup_id;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 24);
} events SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__type(key, __u64);
	__type(value, struct sk_info);
	__uint(max_entries, 65536);
} sk_owners SEC(".maps");

static __always_inline void stash_sk(__u64 sk_key)
{
	struct sk_info info = {};

	info.pid = (__u32)(bpf_get_current_pid_tgid() >> 32);
	info.cgroup_id = bpf_get_current_cgroup_id();
	if (info.pid == 0 && info.cgroup_id == 0)
		return;
	bpf_map_update_elem(&sk_owners, &sk_key, &info, BPF_ANY);
}

SEC("tracepoint/sock/inet_sock_set_state")
int mochi_inet_sock_set_state(struct trace_event_raw_inet_sock_set_state *args)
{
	struct event *e;
	struct sk_info *info;
	__u64 sk_key;
	__u64 cgroup_id;
	__u32 pid;
	__u16 family;
	int oldstate;
	int newstate;
	int delete_after;

	oldstate = BPF_CORE_READ(args, oldstate);
	newstate = BPF_CORE_READ(args, newstate);
	sk_key = (__u64)BPF_CORE_READ(args, skaddr);

	// SYN_SENT is TCP-only. Skip sk_protocol: it is a bitfield and may be unset
	// on CLOSE→SYN_SENT when read via CO-RE.
	if (oldstate == TCP_CLOSE && newstate == TCP_SYN_SENT) {
		stash_sk(sk_key);
		return 0;
	}

	if (oldstate == TCP_SYN_SENT && newstate == TCP_CLOSE) {
		bpf_map_delete_elem(&sk_owners, &sk_key);
		return 0;
	}

	if (oldstate == TCP_SYN_SENT && newstate == TCP_ESTABLISHED) {
		delete_after = 0;
	} else if (oldstate == TCP_ESTABLISHED && newstate != TCP_ESTABLISHED) {
		delete_after = 1;
	} else {
		return 0;
	}

	info = bpf_map_lookup_elem(&sk_owners, &sk_key);
	if (!info)
		return 0;
	pid = info->pid;
	cgroup_id = info->cgroup_id;
	if (delete_after)
		bpf_map_delete_elem(&sk_owners, &sk_key);

	e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e)
		return 0;

	e->pid = pid;
	e->cgroup_id = cgroup_id;

	e->oldstate = (__u8)oldstate;
	e->newstate = (__u8)newstate;
	e->pad[0] = 0;
	e->pad[1] = 0;
	e->tx_bytes = 0;
	e->rx_bytes = 0;
	if (delete_after) {
		struct sock *sk = (struct sock *)sk_key;
		struct tcp_sock *tp = (struct tcp_sock *)sk;

		e->tx_bytes = BPF_CORE_READ(tp, bytes_acked);
		e->rx_bytes = BPF_CORE_READ(tp, bytes_received);
	}

	family = BPF_CORE_READ(args, family);
	e->family = family;
	e->sport = BPF_CORE_READ(args, sport);
	e->dport = BPF_CORE_READ(args, dport);

	__builtin_memset(e->saddr, 0, sizeof(e->saddr));
	__builtin_memset(e->daddr, 0, sizeof(e->daddr));

	if (family == AF_INET) {
		__u8 s4[4];
		__u8 d4[4];

		bpf_core_read(&s4, sizeof(s4), &args->saddr);
		bpf_core_read(&d4, sizeof(d4), &args->daddr);
		__builtin_memcpy(e->saddr, s4, 4);
		__builtin_memcpy(e->daddr, d4, 4);
	} else if (family == AF_INET6) {
		BPF_CORE_READ_INTO(&e->saddr, args, saddr_v6);
		BPF_CORE_READ_INTO(&e->daddr, args, daddr_v6);
	} else {
		bpf_ringbuf_discard(e, 0);
		return 0;
	}

	bpf_ringbuf_submit(e, 0);
	return 0;
}
