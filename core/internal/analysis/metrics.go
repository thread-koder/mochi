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

type CorrelationMetrics struct {
	CPU             []timeseries.DataPoint
	Memory          []timeseries.DataPoint
	NetworkReceive  []timeseries.DataPoint
	NetworkTransmit []timeseries.DataPoint
	DiskRead        []timeseries.DataPoint
	DiskWrite       []timeseries.DataPoint
}

func fetchWorkloadCorrelationMetrics(ctx context.Context, pods []*database.Pod, opts CorrelationOptions) (CorrelationMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	podNames := database.UniquePodNames(pods)
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
		matrix, _, err := prometheus.QueryWorkloadCPUUsage(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadMemoryUsage(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query network receive metrics: %w", err)
		}
		networkReceiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query network transmit metrics: %w", err)
		}
		networkTransmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskReadBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query disk read metrics: %w", err)
		}
		diskReadMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query disk write metrics: %w", err)
		}
		diskWriteMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return CorrelationMetrics{}, err
	}

	return CorrelationMetrics{
		CPU:             timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:          timeseries.MatrixToDataPoints(memoryMatrix),
		NetworkReceive:  timeseries.MatrixToDataPoints(networkReceiveMatrix),
		NetworkTransmit: timeseries.MatrixToDataPoints(networkTransmitMatrix),
		DiskRead:        timeseries.MatrixToDataPoints(diskReadMatrix),
		DiskWrite:       timeseries.MatrixToDataPoints(diskWriteMatrix),
	}, nil
}
