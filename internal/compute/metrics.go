package compute

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

// Represents resource metrics (raw data)
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

// Fetches container metrics
func fetchContainerMetrics(ctx context.Context, container *database.Container, opts AnalysisOptions) (ResourceMetrics, error) {
	// Set up time range
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

	// Execute all queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Query CPU throttling metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	// Query CPU pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	// Query memory fail count metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	// Query memory OOM metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	// Query memory pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	// Query restarts metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query container restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	// Wait for all queries to be completed and check for errors
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

// Aggregates metrics from all containers in a pod
func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
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
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
	)

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Query CPU throttling metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	// Query CPU pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	// Query memory fail count metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	// Query memory OOM metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	// Query memory pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	// Query restarts metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryPodRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	// Wait for all queries to be completed and check for errors
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

// Aggregates metrics from all pods in a workload
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (ResourceMetrics, error) {
	if len(pods) == 0 {
		return ResourceMetrics{}, fmt.Errorf("no pods found for workload")
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
				cpuMatrix     model.Matrix
				memoryMatrix  model.Matrix
				cpuThrottling float64
				cpuPressure   float64
				memFailCnt    float64
				memOOM        float64
				memPressure   float64
				restarts      float64
			)

			// Create a new error group for this pod
			podG, podCtx := errgroup.WithContext(gctx)

			// Query CPU metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodCPURange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU metrics: %w", err)
				}
				cpuMatrix = matrix
				return nil
			})

			// Query memory metrics
			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodMemoryRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory metrics: %w", err)
				}
				memoryMatrix = matrix
				return nil
			})

			// Query CPU throttling metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodCPUThrottling(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
				}
				cpuThrottling = value
				return nil
			})

			// Query CPU pressure metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodCPUPressure(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
				}
				cpuPressure = value
				return nil
			})

			// Query memory fail count metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryFailCount(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory fail count metrics: %w", err)
				}
				memFailCnt = value
				return nil
			})

			// Query memory OOM metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryOOM(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory OOM metrics: %w", err)
				}
				memOOM = value
				return nil
			})

			// Query memory pressure metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodMemoryPressure(podCtx, opts.TimeRange, opts.RangeStep, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query memory pressure metrics: %w", err)
				}
				memPressure = value
				return nil
			})

			// Query restarts metrics
			podG.Go(func() error {
				value, _, err := prometheus.QueryPodRestarts(podCtx, opts.TimeRange, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query restarts metrics: %w", err)
				}
				restarts = value
				return nil
			})

			// Wait for all queries to be completed and check for errors
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

	// Wait for all queries to be completed and check for errors
	if err := g.Wait(); err != nil {
		return ResourceMetrics{}, err
	}

	// Aggregate metrics across pods
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

// Fetches namespace metrics
func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (ResourceMetrics, error) {
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
		cpuMatrix     model.Matrix
		memoryMatrix  model.Matrix
		cpuThrottling float64
		cpuPressure   float64
		memFailCnt    float64
		memOOM        float64
		memPressure   float64
		restarts      float64
	)

	// Execute queries in parallel
	g, gctx := errgroup.WithContext(ctx)

	// Query namespace CPU metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceCPURange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU metrics: %w", err)
		}
		cpuMatrix = matrix
		return nil
	})

	// Query namespace memory metrics
	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceMemoryRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory metrics: %w", err)
		}
		memoryMatrix = matrix
		return nil
	})

	// Query CPU throttling metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceCPUThrottling(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU throttling metrics: %w", err)
		}
		cpuThrottling = value
		return nil
	})

	// Query CPU pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceCPUPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query CPU pressure metrics: %w", err)
		}
		cpuPressure = value
		return nil
	})

	// Query memory fail count metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryFailCount(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory fail count metrics: %w", err)
		}
		memFailCnt = value
		return nil
	})

	// Query memory OOM metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryOOM(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory OOM metrics: %w", err)
		}
		memOOM = value
		return nil
	})

	// Query memory pressure metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceMemoryPressure(gctx, opts.TimeRange, opts.RangeStep, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query memory pressure metrics: %w", err)
		}
		memPressure = value
		return nil
	})

	// Query restarts metrics
	g.Go(func() error {
		value, _, err := prometheus.QueryNamespaceRestarts(gctx, opts.TimeRange, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query restarts metrics: %w", err)
		}
		restarts = value
		return nil
	})

	// Wait for all queries to be completed and check for errors
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
