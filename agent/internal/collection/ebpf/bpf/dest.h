/* Shared client-outbound dest filters. Not a series: loopback, unspecified,
 * or v4-mapped loopback. Source is not filtered (SYN_SENT often has no local IP yet). */

#ifndef MOCHI_BPF_DEST_H
#define MOCHI_BPF_DEST_H

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

static __always_inline int v4_dst_ok(__u8 *addr)
{
	return !is_unspecified_v4(addr) && !is_loopback_v4(addr);
}

#endif
