/* iov_iter CO-RE flavors. Callers include vmlinux.h and bpf_core_read.h first.
 * A datagram fits in the first buffer. A byte stream must stash segments at
 * fentry because the iterator advances before fexit and ret spans segments. */

#ifndef MOCHI_BPF_IOV_H
#define MOCHI_BPF_IOV_H

#define IOV_STASH_SEGS 4

struct iov_seg {
	__u64 base;
	__u32 len;
};

/* Kernel 6.4+ renamed iov_iter.iov to __iov. */
struct iov_iter___old {
	const struct iovec *iov;
	u8 iter_type;
	size_t iov_offset;
	unsigned long nr_segs;
} __attribute__((preserve_access_index));

struct iov_iter___new {
	const struct iovec *__iov;
	u8 iter_type;
	size_t iov_offset;
	unsigned long nr_segs;
} __attribute__((preserve_access_index));

/* Kernel 6.0+ ITER_UBUF is one buffer, not an iovec array.
 * count overlays the remaining length, the same slot as __ubuf_iovec.iov_len. */
struct iov_iter___ubuf {
	void *ubuf;
	u8 iter_type;
	size_t iov_offset;
	size_t count;
} __attribute__((preserve_access_index));

static __always_inline void *msg_user_buf(struct msghdr *msg)
{
	struct iov_iter *iter = &msg->msg_iter;
	struct iov_iter___ubuf *ubuf_iter = (void *)iter;
	struct iov_iter___new *new_iter = (void *)iter;
	struct iov_iter___old *old_iter = (void *)iter;
	const struct iovec *iov = NULL;

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
	return BPF_CORE_READ(iov, iov_base);
}

static __always_inline int read_iov_seg(const struct iovec *iov, struct iov_seg *seg)
{
	struct iovec tmp;

	if (!iov)
		return -1;
	if (bpf_probe_read_kernel(&tmp, sizeof(tmp), iov) < 0)
		return -1;
	if (!tmp.iov_base || tmp.iov_len == 0)
		return -1;
	seg->base = (__u64)tmp.iov_base;
	seg->len = tmp.iov_len > 0xffffffffUL ? 0xffffffffU : (__u32)tmp.iov_len;
	return 0;
}

/* Snapshot the iterator start. ITER_UBUF is one buffer and count is remaining.
 * ITER_IOVEC keeps the current segment plus the next few. iov_offset applies
 * only to the first. Returns how many segments were stashed. */
static __always_inline int iov_stash(struct msghdr *msg, struct iov_seg segs[IOV_STASH_SEGS])
{
	struct iov_iter *iter = &msg->msg_iter;
	struct iov_iter___ubuf *ubuf_iter = (void *)iter;
	struct iov_iter___new *new_iter = (void *)iter;
	struct iov_iter___old *old_iter = (void *)iter;
	const struct iovec *iov = NULL;
	size_t off = 0;
	unsigned long nr = 1;
	int n = 0;

	__builtin_memset(segs, 0, sizeof(struct iov_seg) * IOV_STASH_SEGS);

	if (bpf_core_field_exists(ubuf_iter->iter_type) &&
	    bpf_core_enum_value_exists(enum iter_type, ITER_UBUF)) {
		u8 type = BPF_CORE_READ(ubuf_iter, iter_type);
		if (type == bpf_core_enum_value(enum iter_type, ITER_UBUF)) {
			void *ubuf = BPF_CORE_READ(ubuf_iter, ubuf);
			size_t count = BPF_CORE_READ(ubuf_iter, count);
			off = BPF_CORE_READ(ubuf_iter, iov_offset);
			if (!ubuf || count == 0)
				return 0;
			segs[0].base = (__u64)ubuf + off;
			segs[0].len = count > 0xffffffffUL ? 0xffffffffU : (__u32)count;
			return 1;
		}
	}

	if (bpf_core_field_exists(new_iter->__iov)) {
		iov = BPF_CORE_READ(new_iter, __iov);
		off = BPF_CORE_READ(new_iter, iov_offset);
		nr = BPF_CORE_READ(new_iter, nr_segs);
	} else if (bpf_core_field_exists(old_iter->iov)) {
		iov = BPF_CORE_READ(old_iter, iov);
		off = BPF_CORE_READ(old_iter, iov_offset);
		nr = BPF_CORE_READ(old_iter, nr_segs);
	}
	if (!iov || nr == 0)
		return 0;

	if (read_iov_seg(iov, &segs[0]) == 0)
		n = 1;
	if (n == 1 && off != 0) {
		if (off >= segs[0].len)
			segs[0].len = 0;
		else {
			segs[0].base += off;
			segs[0].len -= (__u32)off;
		}
	}
	if (nr > 1 && read_iov_seg(iov + 1, &segs[1]) == 0)
		n = 2;
	if (nr > 2 && read_iov_seg(iov + 2, &segs[2]) == 0)
		n = 3;
	if (nr > 3 && read_iov_seg(iov + 3, &segs[3]) == 0)
		n = 4;
	/* Offset can consume the only segment. An empty stash must not occupy pending. */
	if (segs[0].len == 0 && segs[1].len == 0 && segs[2].len == 0 && segs[3].len == 0)
		return 0;
	return n;
}

#endif
