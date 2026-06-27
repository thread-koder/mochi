package disk

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
		Pods:          []string{pod.Name},
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
		matrix, _, err := prometheus.QueryWorkloadDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteOpsRange(gctx, r, queryOpts)
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

	podNames := make([]string, len(pods))
	for i, pod := range pods {
		podNames[i] = pod.Name
	}
	queryOpts := prometheus.QueryOptions{
		Namespace:     pods[0].Namespace,
		Pods:          podNames,
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
		matrix, _, err := prometheus.QueryWorkloadDiskReadBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read bytes metrics: %w", err)
		}
		readBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query write bytes metrics: %w", err)
		}
		writeBytesMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskReadOpsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query read ops metrics: %w", err)
		}
		readOpsMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadDiskWriteOpsRange(gctx, r, queryOpts)
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
