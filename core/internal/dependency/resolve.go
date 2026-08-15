package dependency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

const SourceMochiEBPF = "mochi-ebpf"

type NodeRef struct {
	Kind      string
	Namespace string
	Name      string
}

type ResolvedEdge struct {
	From                NodeRef
	To                  NodeRef
	Protocol            string
	Port                int
	ViaServiceNamespace *string
	ViaServiceName      *string
	Source              string
	Connects            float64
	TxBytes             float64
	RxBytes             float64
	ActiveConnections   float64
	Evidence            json.RawMessage
}

type ResolveOptions struct {
	IncludeDNS bool
	cache      resolveCache
}

type resolveCache struct {
	serviceByIP  map[string]cachedService
	workloadByIP map[string]cachedWorkload
}

type cachedService struct {
	svc   *database.Service
	found bool
}

type cachedWorkload struct {
	ref NodeRef
	ok  bool
}

func DefaultResolveOptions() ResolveOptions {
	return ResolveOptions{
		IncludeDNS: false,
		cache: resolveCache{
			serviceByIP:  make(map[string]cachedService),
			workloadByIP: make(map[string]cachedWorkload),
		},
	}
}

// Resolve turns one connection series into a workload edge, or drops it as noise/unresolvable src.
func Resolve(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (ResolvedEdge, bool, error) {
	if series.Connects <= 0 && series.ActiveConnections <= 0 {
		return ResolvedEdge{}, false, nil
	}
	if isLinkLocalOrMetadata(series.ActualDstIP) {
		return ResolvedEdge{}, false, nil
	}

	srcPod, err := database.GetPodByUID(ctx, series.SrcPodUID)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			return ResolvedEdge{}, false, nil
		}
		return ResolvedEdge{}, false, fmt.Errorf("lookup src pod %s: %w", series.SrcPodUID, err)
	}

	from, ok, err := nodeRefFromPod(ctx, srcPod)
	if err != nil {
		return ResolvedEdge{}, false, err
	}
	if !ok {
		return ResolvedEdge{}, false, nil
	}

	to, err := resolveDestination(ctx, series, opts)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	if sameNode(from, to) {
		return ResolvedEdge{}, false, nil
	}
	if !opts.IncludeDNS && isDNSNoise(to) {
		return ResolvedEdge{}, false, nil
	}

	viaNS, viaName, err := viaService(ctx, series, opts)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	evidence, err := json.Marshal(map[string]string{
		"src_pod_uid":   series.SrcPodUID,
		"dst_ip":        series.DstIP,
		"actual_dst_ip": series.ActualDstIP,
	})
	if err != nil {
		return ResolvedEdge{}, false, fmt.Errorf("marshal evidence: %w", err)
	}

	edge := ResolvedEdge{
		From:                from,
		To:                  to,
		Protocol:            series.Protocol,
		Port:                series.ActualDstPort,
		ViaServiceNamespace: viaNS,
		ViaServiceName:      viaName,
		Source:              SourceMochiEBPF,
		Connects:            series.Connects,
		TxBytes:             series.TxBytes,
		RxBytes:             series.RxBytes,
		ActiveConnections:   series.ActiveConnections,
		Evidence:            evidence,
	}
	return edge, true, nil
}

func resolveDestination(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (NodeRef, error) {
	pods, err := database.GetPodsByIP(ctx, series.ActualDstIP)
	if err != nil {
		return NodeRef{}, fmt.Errorf("lookup pods by IP %s: %w", series.ActualDstIP, err)
	}
	if len(pods) > 0 {
		ref, ok, err := nodeRefFromPod(ctx, pods[0])
		if err != nil {
			return NodeRef{}, err
		}
		if ok {
			return ref, nil
		}
	}

	// NAT miss: actual_dst_ip is often still the ClusterIP. Map via Service endpoints.
	ref, ok, err := resolveViaServiceEndpoints(ctx, opts, series.ActualDstIP)
	if err != nil {
		return NodeRef{}, err
	}
	if ok {
		return ref, nil
	}
	if series.DstIP != "" && series.DstIP != series.ActualDstIP {
		ref, ok, err := resolveViaServiceEndpoints(ctx, opts, series.DstIP)
		if err != nil {
			return NodeRef{}, err
		}
		if ok {
			return ref, nil
		}
	}

	return NodeRef{
		Kind:      KindExternal,
		Namespace: "",
		Name:      series.ActualDstIP,
	}, nil
}

func lookupService(ctx context.Context, opts ResolveOptions, clusterIP string) (*database.Service, bool, error) {
	if cached, hit := opts.cache.serviceByIP[clusterIP]; hit {
		return cached.svc, cached.found, nil
	}

	svc, err := database.GetServiceByClusterIP(ctx, clusterIP)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			opts.cache.serviceByIP[clusterIP] = cachedService{}
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lookup service by cluster IP %s: %w", clusterIP, err)
	}

	opts.cache.serviceByIP[clusterIP] = cachedService{
		svc:   svc,
		found: true,
	}
	return svc, true, nil
}

func resolveViaServiceEndpoints(ctx context.Context, opts ResolveOptions, clusterIP string) (NodeRef, bool, error) {
	if cached, hit := opts.cache.workloadByIP[clusterIP]; hit {
		return cached.ref, cached.ok, nil
	}

	svc, found, err := lookupService(ctx, opts, clusterIP)
	if err != nil {
		return NodeRef{}, false, err
	}
	if !found {
		opts.cache.workloadByIP[clusterIP] = cachedWorkload{}
		return NodeRef{}, false, nil
	}

	slices, err := database.GetEndpointSlicesByService(ctx, svc.Namespace, svc.Name)
	if err != nil {
		return NodeRef{}, false, err
	}

	for _, es := range slices {
		addrs, err := endpointAddresses(es.Endpoints)
		if err != nil {
			return NodeRef{}, false, fmt.Errorf("parse endpoints for %s/%s: %w", es.Namespace, es.Name, err)
		}
		for _, addr := range addrs {
			pods, err := database.GetPodsByIP(ctx, addr)
			if err != nil {
				return NodeRef{}, false, fmt.Errorf("lookup pods by endpoint IP %s: %w", addr, err)
			}
			if len(pods) == 0 {
				continue
			}
			ref, ok, err := nodeRefFromPod(ctx, pods[0])
			if err != nil {
				return NodeRef{}, false, err
			}
			if ok {
				opts.cache.workloadByIP[clusterIP] = cachedWorkload{
					ref: ref,
					ok:  true,
				}
				return ref, true, nil
			}
		}
	}

	opts.cache.workloadByIP[clusterIP] = cachedWorkload{}
	return NodeRef{}, false, nil
}

func endpointAddresses(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var endpoints []struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(raw, &endpoints); err != nil {
		return nil, err
	}
	var results []string
	for _, ep := range endpoints {
		results = append(results, ep.Addresses...)
	}
	return results, nil
}

func viaService(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (*string, *string, error) {
	// NAT hit (dst=ClusterIP, actual=pod) or NAT miss (both ClusterIP): attribute the Service.
	for _, ip := range []string{series.DstIP, series.ActualDstIP} {
		if ip == "" {
			continue
		}
		svc, found, err := lookupService(ctx, opts, ip)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			continue
		}
		return new(svc.Namespace), new(svc.Name), nil
	}
	return nil, nil, nil
}
