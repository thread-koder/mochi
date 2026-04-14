package disk

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/thread_koder/mochi/internal/database"
	"github.com/thread_koder/mochi/internal/prometheus"
	"github.com/thread_koder/mochi/internal/timeseries"
	"golang.org/x/sync/errgroup"
)

// DiskMetrics holds time-aligned read/write byte rates and operation rates from Prometheus.
type DiskMetrics struct {
	ReadBytes  []timeseries.DataPoint `json:"read_bytes"`
	WriteBytes []timeseries.DataPoint `json:"write_bytes"`
	ReadOps    []timeseries.DataPoint `json:"read_ops"`
	WriteOps   []timeseries.DataPoint `json:"write_ops"`
}

func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	queryOpts := prometheus.QueryOptions{
		Namespace:     pod.Namespace,
		Pod:           pod.Name,
		RangeDuration: "5m",
	}

	var (
		readBytesMatrix  model.Matrix
		writeBytesMatrix model.Matrix
		readOpsMatrix    model.Matrix
		writeOpsMatrix   model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	return DiskMetrics{
		ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
		WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
		ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
		WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
	}, nil
}

// fetchWorkloadMetrics queries each pod in parallel, then merges per-pod series
// with MergeDataPointsByTime so each timestamp reflects summed rates across the workload.
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
	if len(pods) == 0 {
		return DiskMetrics{}, fmt.Errorf("no pods found for workload")
	}

	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	type podMetrics struct {
		ReadBytes  []timeseries.DataPoint
		WriteBytes []timeseries.DataPoint
		ReadOps    []timeseries.DataPoint
		WriteOps   []timeseries.DataPoint
	}
	results := make([]podMetrics, len(pods))

	g, gctx := errgroup.WithContext(ctx)

	for i, pod := range pods {
		queryOpts := prometheus.QueryOptions{
			Namespace:     pod.Namespace,
			Pod:           pod.Name,
			RangeDuration: "5m",
		}

		g.Go(func() error {
			var (
				readBytesMatrix  model.Matrix
				writeBytesMatrix model.Matrix
				readOpsMatrix    model.Matrix
				writeOpsMatrix   model.Matrix
			)

			podG, podCtx := errgroup.WithContext(gctx)

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read bytes metrics: %w", err)
				}
				readBytesMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write bytes metrics: %w", err)
				}
				writeBytesMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read ops metrics: %w", err)
				}
				readOpsMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write ops metrics: %w", err)
				}
				writeOpsMatrix = matrix
				return nil
			})

			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
				WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
				ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
				WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	var readBytes, writeBytes []timeseries.DataPoint
	var readOps, writeOps []timeseries.DataPoint
	for _, p := range results {
		readBytes = timeseries.MergeDataPointsByTime(readBytes, p.ReadBytes)
		writeBytes = timeseries.MergeDataPointsByTime(writeBytes, p.WriteBytes)
		readOps = timeseries.MergeDataPointsByTime(readOps, p.ReadOps)
		writeOps = timeseries.MergeDataPointsByTime(writeOps, p.WriteOps)
	}

	return DiskMetrics{
		ReadBytes:  readBytes,
		WriteBytes: writeBytes,
		ReadOps:    readOps,
		WriteOps:   writeOps,
	}, nil
}

func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (DiskMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	queryOpts := prometheus.QueryOptions{
		Namespace:     namespace,
		RangeDuration: "5m",
	}

	var (
		readBytesMatrix  model.Matrix
		writeBytesMatrix model.Matrix
		readOpsMatrix    model.Matrix
		writeOpsMatrix   model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	return DiskMetrics{
		ReadBytes:  timeseries.MatrixToDataPoints(readBytesMatrix),
		WriteBytes: timeseries.MatrixToDataPoints(writeBytesMatrix),
		ReadOps:    timeseries.MatrixToDataPoints(readOpsMatrix),
		WriteOps:   timeseries.MatrixToDataPoints(writeOpsMatrix),
	}, nil
}
