package compute

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

type ResourceMetrics struct {
	CPU            []timeseries.DataPoint `json:"cpu"`
	Memory         []timeseries.DataPoint `json:"memory"`
	CPUThrottling  float64                `json:"cpu_throttling,omitempty"`
	CPUPressure    float64                `json:"cpu_pressure,omitempty"`
	MemoryFailCnt  float64                `json:"memory_fail_cnt,omitempty"`
	MemoryOOM      float64                `json:"memory_oom,omitempty"`
	MemoryPressure float64                `json:"memory_pressure,omitempty"`
	Restarts       float64                `json:"restarts,omitempty"`
}

func fetchContainerMetrics(ctx context.Context, container *database.Container, opts AnalysisOptions) (ResourceMetrics, error) {
	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	queryOpts := prometheus.QueryOptions{
		Namespace:     container.Namespace,
		Pods:          []string{container.PodName},
		Container:     container.Name,
		RangeDuration: "5m",
	}

	var (
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
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
		value, _, err := prometheus.QueryWorkloadCPUThrottling(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadCPUPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query container restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
		CPUThrottling:  cpuThrottling,
		CPUPressure:    cpuPressure,
		MemoryFailCnt:  memFailCnt,
		MemoryOOM:      memOOM,
		MemoryPressure: memPressure,
		Restarts:       restarts,
	}, nil
}

func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
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
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
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
		value, _, err := prometheus.QueryWorkloadCPUThrottling(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadCPUPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
		CPUThrottling:  cpuThrottling,
		CPUPressure:    cpuPressure,
		MemoryFailCnt:  memFailCnt,
		MemoryOOM:      memOOM,
		MemoryPressure: memPressure,
		Restarts:       restarts,
	}, nil
}

func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
	if len(pods) == 0 {
		return ResourceMetrics{}, fmt.Errorf("no pods found for workload")
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
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
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
		value, _, err := prometheus.QueryWorkloadCPUThrottling(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadCPUPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadMemoryPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryWorkloadRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
		CPUThrottling:  cpuThrottling,
		CPUPressure:    cpuPressure,
		MemoryFailCnt:  memFailCnt,
		MemoryOOM:      memOOM,
		MemoryPressure: memPressure,
		Restarts:       restarts,
	}, nil
}

func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (ResourceMetrics, error) {
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
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceCPUThrottling(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceCPUPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryPressure(gctx, opts.TimeRange, opts.stabilitySubqueryStep(), queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	return ResourceMetrics{
		CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
		Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
		CPUThrottling:  cpuThrottling,
		CPUPressure:    cpuPressure,
		MemoryFailCnt:  memFailCnt,
		MemoryOOM:      memOOM,
		MemoryPressure: memPressure,
		Restarts:       restarts,
	}, nil
}
