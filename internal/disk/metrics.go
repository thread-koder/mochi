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

// Represents raw disk metrics data
type DiskMetrics struct {
	ReadBytes  []timeseries.DataPoint `json:"read_bytes"`
	WriteBytes []timeseries.DataPoint `json:"write_bytes"`
	ReadOps    []timeseries.DataPoint `json:"read_ops"`
	WriteOps   []timeseries.DataPoint `json:"write_ops"`
}

// Fetches pod disk metrics
func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
	// Set up time range
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

	// Execute all queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query read bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	// Query write bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	// Query read ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	// Query write ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	// Wait for all queries to be completed and check for errors
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

// Aggregates metrics from all pods in a workload
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (DiskMetrics, error) {
	if len(pods) == 0 {
		return DiskMetrics{}, fmt.Errorf("no pods found for workload")
	}

	// Set up time range
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	// Per-pod results: each goroutine writes to its index
	type podMetrics struct {
		ReadBytes  []timeseries.DataPoint
		WriteBytes []timeseries.DataPoint
		ReadOps    []timeseries.DataPoint
		WriteOps   []timeseries.DataPoint
	}
	results := make([]podMetrics, len(pods))

	// Query all pods in parallel
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

			// Create a new error group for this pod
			podG, podCtx := errgroup.WithContext(gctx)

			// Query read bytes
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read bytes metrics: %w", err)
				}
				readBytesMatrix = matrix
				return nil
			})

			// Query write bytes
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write bytes metrics: %w", err)
				}
				writeBytesMatrix = matrix
				return nil
			})

			// Query read ops
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskReadOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query read ops metrics: %w", err)
				}
				readOpsMatrix = matrix
				return nil
			})

			// Query write ops
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodDiskWriteOpsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query write ops metrics: %w", err)
				}
				writeOpsMatrix = matrix
				return nil
			})

			// Wait for all queries to be completed and check for errors
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

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return DiskMetrics{}, err
	}

	// Aggregate metrics across pods
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

// Fetches namespace disk metrics
func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (DiskMetrics, error) {
	// Set up time range
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

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query namespace read bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	// Query namespace write bytes
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	// Query namespace read ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	// Query namespace write ops
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceDiskWriteOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write ops metrics: %w", err)
		}
		writeOpsMatrix = matrix
		return nil
	})

	// Wait for all queries to be completed and check for errors
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
