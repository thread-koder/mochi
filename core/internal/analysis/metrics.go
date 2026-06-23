package analysis

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/core/internal/database"
	"github.com/thread_koder/mochi/core/internal/prometheus"
	"github.com/thread_koder/mochi/core/internal/timeseries"
	"golang.org/x/sync/errgroup"
)

// correlationMetrics stores the metric series for the correlation analysis.
type correlationMetrics struct {
	CPU             []timeseries.DataPoint
	Memory          []timeseries.DataPoint
	NetworkReceive  []timeseries.DataPoint
	NetworkTransmit []timeseries.DataPoint
	DiskRead        []timeseries.DataPoint
	DiskWrite       []timeseries.DataPoint
}

func fetchWorkloadCorrelationMetrics(ctx context.Context, pods []*database.Pod, opts CorrelationOptions) (correlationMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	podNames := make([]string, len(pods))
	for i, pod := range pods {
		podNames[i] = pod.Name
	}
	queryOpts := prometheus.QueryOptions{
		Namespace: pods[0].Namespace,
		Pods:      podNames,
		// Keep per-sample rate windows fixed so timeseries stay comparable even
		// when callers widen TimeRange.
		RangeDuration: "5m",
	}

	var (
		cpuMatrix             model.Matrix
		memoryMatrix          model.Matrix
		networkReceiveMatrix  model.Matrix
		networkTransmitMatrix model.Matrix
		diskReadMatrix        model.Matrix
		diskWriteMatrix       model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query network receive metrics: %w", err)
		}
		networkReceiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query network transmit metrics: %w", err)
		}
		networkTransmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query disk read metrics: %w", err)
		}
		diskReadMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query disk write metrics: %w", err)
		}
		diskWriteMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return correlationMetrics{}, err
	}

	return correlationMetrics{
		CPU:             timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:          timeseries.MatrixToDataPoints(memoryMatrix),
		NetworkReceive:  timeseries.MatrixToDataPoints(networkReceiveMatrix),
		NetworkTransmit: timeseries.MatrixToDataPoints(networkTransmitMatrix),
		DiskRead:        timeseries.MatrixToDataPoints(diskReadMatrix),
		DiskWrite:       timeseries.MatrixToDataPoints(diskWriteMatrix),
	}, nil
}
