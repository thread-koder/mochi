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

// fetchWorkloadCorrelationMetrics queries per-pod metrics and merges them into workload-level
// series for cross-metric correlation.
func fetchWorkloadCorrelationMetrics(ctx context.Context, pods []*database.Pod, opts CorrelationOptions) (correlationMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	var metrics correlationMetrics
	g, gctx := errgroup.WithContext(ctx)

	type podMetrics struct {
		CPU             []timeseries.DataPoint
		Memory          []timeseries.DataPoint
		NetworkReceive  []timeseries.DataPoint
		NetworkTransmit []timeseries.DataPoint
		DiskRead        []timeseries.DataPoint
		DiskWrite       []timeseries.DataPoint
	}
	results := make([]podMetrics, len(pods))

	for i, pod := range pods {
		queryOpts := prometheus.QueryOptions{
			Namespace: pod.Namespace,
			Pod:       pod.Name,
			// Keep per-sample rate windows fixed so timeseries stay comparable even
			// when callers widen TimeRange.
			RangeDuration: "5m",
		}

		g.Go(func() error {
			var (
				cpuMatrix             model.Matrix
				memoryMatrix          model.Matrix
				networkReceiveMatrix  model.Matrix
				networkTransmitMatrix model.Matrix
				diskReadMatrix        model.Matrix
				diskWriteMatrix       model.Matrix
			)

			podG, podCtx := errgroup.WithContext(gctx)

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodCPURange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU metrics: %w", err)
				}
				cpuMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodMemoryRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory metrics: %w", err)
				}
				memoryMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkReceiveBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query network receive metrics: %w", err)
				}
				networkReceiveMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkTransmitBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query network transmit metrics: %w", err)
				}
				networkTransmitMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query disk read metrics: %w", err)
				}
				diskReadMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query disk write metrics: %w", err)
				}
				diskWriteMatrix = matrix
				return nil
			})

			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				CPU:             timeseries.MatrixToDataPoints(cpuMatrix),
				Memory:          timeseries.MatrixToDataPoints(memoryMatrix),
				NetworkReceive:  timeseries.MatrixToDataPoints(networkReceiveMatrix),
				NetworkTransmit: timeseries.MatrixToDataPoints(networkTransmitMatrix),
				DiskRead:        timeseries.MatrixToDataPoints(diskReadMatrix),
				DiskWrite:       timeseries.MatrixToDataPoints(diskWriteMatrix),
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return correlationMetrics{}, err
	}

	for _, pm := range results {
		metrics.CPU = timeseries.MergeDataPointsByTime(metrics.CPU, pm.CPU)
		metrics.Memory = timeseries.MergeDataPointsByTime(metrics.Memory, pm.Memory)
		metrics.NetworkReceive = timeseries.MergeDataPointsByTime(metrics.NetworkReceive, pm.NetworkReceive)
		metrics.NetworkTransmit = timeseries.MergeDataPointsByTime(metrics.NetworkTransmit, pm.NetworkTransmit)
		metrics.DiskRead = timeseries.MergeDataPointsByTime(metrics.DiskRead, pm.DiskRead)
		metrics.DiskWrite = timeseries.MergeDataPointsByTime(metrics.DiskWrite, pm.DiskWrite)
	}

	return metrics, nil
}
