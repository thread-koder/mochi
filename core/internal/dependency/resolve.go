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
	Confidence          float32
	Connects            float64
	TxBytes             float64
	RxBytes             float64
	ActiveConnections   float64
	Evidence            json.RawMessage
}

type ResolveOptions struct {
	IncludeDNS bool
	VolumeNorm float64
}

func DefaultResolveOptions() ResolveOptions {
	return ResolveOptions{
		IncludeDNS: false,
		VolumeNorm: defaultVolumeNorm,
	}
}

// Resolve turns one connection series into a workload edge, or drops it as noise/unresolvable src.
func Resolve(ctx context.Context, series ConnectionSeries, opts ResolveOptions) (ResolvedEdge, bool, error) {
	if series.Connects <= 0 {
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

	to, err := resolveDestination(ctx, series)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	if sameNode(from, to) {
		return ResolvedEdge{}, false, nil
	}
	if !opts.IncludeDNS && isDNSNoise(to) {
		return ResolvedEdge{}, false, nil
	}

	viaNS, viaName, err := viaService(ctx, series)
	if err != nil {
		return ResolvedEdge{}, false, err
	}

	protocol := series.Protocol
	if protocol == "" {
		protocol = "tcp"
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
		Protocol:            protocol,
		Port:                series.ActualDstPort,
		ViaServiceNamespace: viaNS,
		ViaServiceName:      viaName,
		Source:              SourceMochiEBPF,
		Confidence:          Confidence(series.Connects, 1.0, opts.VolumeNorm),
		Connects:            series.Connects,
		TxBytes:             series.TxBytes,
		RxBytes:             series.RxBytes,
		ActiveConnections:   series.ActiveConnections,
		Evidence:            evidence,
	}
	return edge, true, nil
}

func resolveDestination(ctx context.Context, series ConnectionSeries) (NodeRef, error) {
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

	return NodeRef{
		Kind:      KindExternal,
		Namespace: "",
		Name:      series.ActualDstIP,
	}, nil
}

func viaService(ctx context.Context, series ConnectionSeries) (*string, *string, error) {
	if series.DstIP == "" || series.DstIP == series.ActualDstIP {
		return nil, nil, nil
	}

	svc, err := database.GetServiceByClusterIP(ctx, series.DstIP)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("lookup service by cluster IP %s: %w", series.DstIP, err)
	}

	return new(svc.Namespace), new(svc.Name), nil
}
