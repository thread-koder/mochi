/* Shared uprobe call-stash ABI for TLS capture and Traceparent inject.
 * Maps stay per-object. conn_tuple layouts differ and are not shared. */

#ifndef MOCHI_BPF_UPROBE_ABI_H
#define MOCHI_BPF_UPROBE_ABI_H

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

struct obj_key {
	__u32 tgid;
	__u32 pad;
	__u64 obj;
};

#endif
