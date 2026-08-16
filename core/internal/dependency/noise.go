package dependency

import (
	"net"
	"strings"
)

func isLinkLocalOrMetadata(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// AWS/GCP/Azure metadata and similar link-local ranges.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}

func isDNSNoise(node NodeRef) bool {
	if !strings.EqualFold(node.Namespace, "kube-system") {
		return false
	}
	name := strings.ToLower(node.Name)
	return strings.Contains(name, "coredns") || strings.Contains(name, "kube-dns")
}

func sameNode(a, b NodeRef) bool {
	return a.Kind == b.Kind && a.Namespace == b.Namespace && a.Name == b.Name
}

func isValidIPAddress(ip string) bool {
	return ip != "" && net.ParseIP(ip) != nil
}
