package dependency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/thread_koder/mochi/core/internal/database"
)

type viaServiceMatch struct {
	namespace string
	name      string
	port      *int
}

func viaService(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (*viaServiceMatch, error) {
	// NAT hit (dst=Service VIP, actual=pod) or NAT miss (both VIP): attribute the Service.
	for _, pair := range []struct {
		ip   string
		port int
	}{
		{series.DstIP, series.DstPort},
		{series.ActualDstIP, series.ActualDstPort},
	} {
		if pair.ip == "" {
			continue
		}
		svc, found, err := lookupService(ctx, opts, pair.ip, pair.port, series.Protocol)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		port, err := mapServicePort(ctx, opts, svc, series, pair.port)
		if err != nil {
			return nil, err
		}
		return &viaServiceMatch{
			namespace: svc.Namespace,
			name:      svc.Name,
			port:      port,
		}, nil
	}
	// Headless clients dial pod IPs. Unique headless EndpointSlice membership only.
	for _, ip := range []string{series.DstIP, series.ActualDstIP} {
		if ip == "" {
			continue
		}
		svc, found, err := lookupHeadlessService(ctx, opts, ip)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		port, err := mapServicePort(ctx, opts, svc, series, series.DstPort)
		if err != nil {
			return nil, err
		}
		return &viaServiceMatch{
			namespace: svc.Namespace,
			name:      svc.Name,
			port:      port,
		}, nil
	}
	return nil, nil
}

func uniqueServiceSpecPortMatch(ports []corev1.ServicePort, protocol string, port int) bool {
	var matches int
	for _, p := range ports {
		if !protocolMatches(protocol, p.Protocol) {
			continue
		}
		if int(p.Port) != port {
			continue
		}
		matches++
		if matches > 1 {
			return false
		}
	}
	return matches == 1
}

func mapServicePort(ctx context.Context, opts ResolveOptions, svc *database.Service, series ConnectionSeries, candidatePort int) (*int, error) {
	ports, err := parsedServicePorts(opts, svc)
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, nil
	}

	if uniqueServiceSpecPortMatch(ports, series.Protocol, candidatePort) {
		return &candidatePort, nil
	}

	var nodePortSpecPort int
	var nodePortMatches int
	for _, p := range ports {
		if !protocolMatches(series.Protocol, p.Protocol) {
			continue
		}
		if p.NodePort == 0 || int(p.NodePort) != candidatePort {
			continue
		}
		nodePortMatches++
		nodePortSpecPort = int(p.Port)
		if nodePortMatches > 1 {
			break
		}
	}
	if nodePortMatches == 1 {
		return &nodePortSpecPort, nil
	}

	var reversePort int
	var reverseMatches int
	for _, p := range ports {
		if !protocolMatches(series.Protocol, p.Protocol) {
			continue
		}
		target, ok, err := resolvedTargetPort(ctx, opts, svc, p, series.ActualDstPort)
		if err != nil {
			return nil, err
		}
		if !ok || target != series.ActualDstPort {
			continue
		}
		reverseMatches++
		reversePort = int(p.Port)
		if reverseMatches > 1 {
			break
		}
	}
	if reverseMatches == 1 {
		return &reversePort, nil
	}
	return nil, nil
}

func parsedServicePorts(opts ResolveOptions, svc *database.Service) ([]corev1.ServicePort, error) {
	key := serviceCacheKey(svc.Namespace, svc.Name)
	if ports, hit := opts.cache.servicePorts[key]; hit {
		return ports, nil
	}

	var ports []corev1.ServicePort
	if len(svc.Ports) > 0 {
		if err := json.Unmarshal(svc.Ports, &ports); err != nil {
			return nil, fmt.Errorf("parse service ports for %s/%s: %w", svc.Namespace, svc.Name, err)
		}
	}
	opts.cache.servicePorts[key] = ports
	return ports, nil
}

func serviceCacheKey(namespace, name string) string {
	return strings.Join([]string{namespace, name}, "\x00")
}

func protocolMatches(seriesProtocol string, svcProtocol corev1.Protocol) bool {
	if svcProtocol == "" {
		svcProtocol = corev1.ProtocolTCP
	}
	return strings.EqualFold(string(svcProtocol), seriesProtocol)
}

func resolvedTargetPort(ctx context.Context, opts ResolveOptions, svc *database.Service, sp corev1.ServicePort, actualDstPort int) (int, bool, error) {
	switch sp.TargetPort.Type {
	case intstr.Int:
		if sp.TargetPort.IntVal == 0 {
			return int(sp.Port), true, nil
		}
		return int(sp.TargetPort.IntVal), true, nil
	case intstr.String:
		if sp.TargetPort.StrVal == "" {
			return int(sp.Port), true, nil
		}
		return endpointSlicePortByName(ctx, opts, svc.Namespace, svc.Name, sp.TargetPort.StrVal, actualDstPort)
	default:
		return int(sp.Port), true, nil
	}
}

func endpointSlicePortByName(ctx context.Context, opts ResolveOptions, namespace, serviceName, portName string, actualDstPort int) (int, bool, error) {
	slices, err := lookupEndpointSlices(ctx, opts, namespace, serviceName)
	if err != nil {
		return 0, false, err
	}

	var matches int
	for _, es := range slices {
		ports, err := endpointSlicePorts(es.Ports)
		if err != nil {
			return 0, false, err
		}
		for _, p := range ports {
			if p.Name == nil || *p.Name != portName {
				continue
			}
			if p.Port == nil || int(*p.Port) != actualDstPort {
				continue
			}
			matches++
		}
	}
	if matches != 1 {
		return 0, false, nil
	}
	return actualDstPort, true, nil
}

func endpointSlicePorts(raw json.RawMessage) ([]struct {
	Name *string `json:"name"`
	Port *int32  `json:"port"`
}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ports []struct {
		Name *string `json:"name"`
		Port *int32  `json:"port"`
	}
	if err := json.Unmarshal(raw, &ports); err != nil {
		return nil, fmt.Errorf("parse endpoint slice ports: %w", err)
	}
	return ports, nil
}
