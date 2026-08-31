package kubernetes

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/thread_koder/mochi/core/internal/database"
)

func serviceVIPs(svc *corev1.Service) []string {
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		return nil
	}

	seen := make(map[string]struct{})
	var ips []string
	add := func(ip string) {
		if ip == "" || ip == "None" {
			return
		}
		if _, ok := seen[ip]; ok {
			return
		}
		seen[ip] = struct{}{}
		ips = append(ips, ip)
	}

	for _, ip := range svc.Spec.ClusterIPs {
		add(ip)
	}
	if len(svc.Spec.ClusterIPs) == 0 {
		add(svc.Spec.ClusterIP)
	}
	for _, ip := range svc.Spec.ExternalIPs {
		add(ip)
	}
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		add(ing.IP)
	}
	return ips
}

func serviceNodePorts(svc *corev1.Service) []database.ServiceNodePort {
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		return nil
	}

	var ports []database.ServiceNodePort
	for _, p := range svc.Spec.Ports {
		if p.NodePort == 0 {
			continue
		}
		proto := string(p.Protocol)
		if proto == "" {
			proto = string(corev1.ProtocolTCP)
		}
		ports = append(ports, database.ServiceNodePort{
			Protocol: strings.ToLower(proto),
			Port:     int(p.NodePort),
		})
	}
	return ports
}

func nodeDialIPs(addresses []corev1.NodeAddress) (internalIP, externalIP *string, allIPs []string) {
	seen := make(map[string]struct{})
	var internal, external []string
	add := func(ip string, bucket *[]string) {
		if ip == "" {
			return
		}
		if _, ok := seen[ip]; ok {
			return
		}
		seen[ip] = struct{}{}
		*bucket = append(*bucket, ip)
	}

	for _, addr := range addresses {
		switch addr.Type {
		case corev1.NodeInternalIP:
			add(addr.Address, &internal)
		case corev1.NodeExternalIP:
			add(addr.Address, &external)
		}
	}

	allIPs = append(internal, external...)
	if len(internal) > 0 {
		internalIP = new(internal[0])
	}
	if len(external) > 0 {
		externalIP = new(external[0])
	}
	return internalIP, externalIP, allIPs
}
