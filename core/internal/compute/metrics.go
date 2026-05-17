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

// ResourceMetrics holds CPU and memory series from Prometheus range queries plus stability-related
// signals. Range queries populate CPU and Memory. Throttling, PSI, OOM, fail counts, and restarts come
// from instant queries and are stored as single-point slices so AnalyzeStability can treat
// every field uniformly.
type ResourceMetrics struct {
	CPU            []timeseries.DataPoint `json:"cpu"`
	Memory         []timeseries.DataPoint `json:"memory"`
	CPUThrottling  []timeseries.DataPoint `json:"cpu_throttling,omitempty"`
	CPUPressure    []timeseries.DataPoint `json:"cpu_pressure,omitempty"`
	MemoryFailCnt  []timeseries.DataPoint `json:"memory_fail_cnt,omitempty"`
	MemoryOOM      []timeseries.DataPoint `json:"memory_oom,omitempty"`
	MemoryPressure []timeseries.DataPoint `json:"memory_pressure,omitempty"`
	Restarts       []timeseries.DataPoint `json:"restarts,omitempty"`
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
		Pod:           container.PodName,
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
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodRestarts(gctx, opts.TimeRange, queryOpts)
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
		CPUThrottling:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuThrottling}},
		CPUPressure:    []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuPressure}},
		MemoryFailCnt:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: memFailCnt}},
		MemoryOOM:      []timeseries.DataPoint{{Timestamp: time.Now(), Value: memOOM}},
		MemoryPressure: []timeseries.DataPoint{{Timestamp: time.Now(), Value: memPressure}},
		Restarts:       []timeseries.DataPoint{{Timestamp: time.Now(), Value: restarts}},
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
		Pod:           pod.Name,
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
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryPodRestarts(gctx, opts.TimeRange, queryOpts)
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
		CPUThrottling:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuThrottling}},
		CPUPressure:    []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuPressure}},
		MemoryFailCnt:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: memFailCnt}},
		MemoryOOM:      []timeseries.DataPoint{{Timestamp: time.Now(), Value: memOOM}},
		MemoryPressure: []timeseries.DataPoint{{Timestamp: time.Now(), Value: memPressure}},
		Restarts:       []timeseries.DataPoint{{Timestamp: time.Now(), Value: restarts}},
	}, nil
}

// fetchWorkloadMetrics queries each pod in parallel, merges CPU and memory series by timestamp, then
// combines scalar stability signals: throttling and PSI percentages are averaged across pods,
// OOMs, fail counts, and restarts are summed.
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

	type podMetrics struct {
		CPU            []timeseries.DataPoint
		Memory         []timeseries.DataPoint
		CPUThrottling  float64
		CPUPressure    float64
		MemoryFailCnt  float64
		MemoryOOM      float64
		MemoryPressure float64
		Restarts       float64
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
				cpuMatrix     model.Matrix
				memoryMatrix  model.Matrix
				cpuThrottling float64
				cpuPressure   float64
				memFailCnt    float64
				memOOM        float64
				memPressure   float64
				restarts      float64
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
				value, _, err := prometheus.QueryPodCPUThrottling(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
				}
				cpuThrottling = value
				return nil
			})

			podG.Go(func() error {
				value, _, err := prometheus.QueryPodCPUPressure(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
				}
				cpuPressure = value
				return nil
			})

			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryFailCount(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory fail count metrics: %w", err)
				}
				memFailCnt = value
				return nil
			})

			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryOOM(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory OOM metrics: %w", err)
				}
				memOOM = value
				return nil
			})

			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryPressure(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory pressure metrics: %w", err)
				}
				memPressure = value
				return nil
			})

			podG.Go(func() error {
				value, _, err := prometheus.QueryPodRestarts(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query restarts metrics: %w", err)
				}
				restarts = value
				return nil
			})

			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				CPU:            timeseries.MatrixToDataPoints(cpuMatrix),
				Memory:         timeseries.MatrixToDataPoints(memoryMatrix),
				CPUThrottling:  cpuThrottling,
				CPUPressure:    cpuPressure,
				MemoryFailCnt:  memFailCnt,
				MemoryOOM:      memOOM,
				MemoryPressure: memPressure,
				Restarts:       restarts,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	var cpu, memory []timeseries.DataPoint
	var sumThrottling, sumPressure, sumMemPressure float64
	var sumFailCnt, sumOOM, sumRestarts float64
	for _, p := range results {
		cpu = timeseries.MergeDataPointsByTime(cpu, p.CPU)
		memory = timeseries.MergeDataPointsByTime(memory, p.Memory)
		sumThrottling += p.CPUThrottling
		sumPressure += p.CPUPressure
		sumMemPressure += p.MemoryPressure
		sumFailCnt += p.MemoryFailCnt
		sumOOM += p.MemoryOOM
		sumRestarts += p.Restarts
	}

	podsNum := float64(len(results))
	return ResourceMetrics{
		CPU:            cpu,
		Memory:         memory,
		CPUThrottling:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumThrottling / podsNum}},
		CPUPressure:    []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumPressure / podsNum}},
		MemoryPressure: []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumMemPressure / podsNum}},
		MemoryFailCnt:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumFailCnt}},
		MemoryOOM:      []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumOOM}},
		Restarts:       []timeseries.DataPoint{{Timestamp: time.Now(), Value: sumRestarts}},
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
		value, _, err := prometheus.QueryNamespaceCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
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
		value, _, err := prometheus.QueryNamespaceMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
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
		CPUThrottling:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuThrottling}},
		CPUPressure:    []timeseries.DataPoint{{Timestamp: time.Now(), Value: cpuPressure}},
		MemoryFailCnt:  []timeseries.DataPoint{{Timestamp: time.Now(), Value: memFailCnt}},
		MemoryOOM:      []timeseries.DataPoint{{Timestamp: time.Now(), Value: memOOM}},
		MemoryPressure: []timeseries.DataPoint{{Timestamp: time.Now(), Value: memPressure}},
		Restarts:       []timeseries.DataPoint{{Timestamp: time.Now(), Value: restarts}},
	}, nil
}
