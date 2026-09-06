package ebpf

// Regenerate CO-RE bindings (requires clang + bpftool-generated vmlinux.h):
//
//	bpftool btf dump file /sys/kernel/btf/vmlinux format c > internal/collection/ebpf/bpf/vmlinux.h
//	go generate ./internal/collection/ebpf
//
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel,bpfeb tcpstate bpf/tcp_state.c -- -I./bpf -I/usr/include -O2 -g -Wno-missing-declarations
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel,bpfeb udpflow bpf/udp_flow.c -- -I./bpf -I/usr/include -O2 -g -Wno-missing-declarations
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel,bpfeb dnsrecv bpf/dns_recv.c -- -I./bpf -I/usr/include -O2 -g -Wno-missing-declarations
