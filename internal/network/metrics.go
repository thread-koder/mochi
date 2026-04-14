package network

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

// NetworkMetrics holds time-aligned receive/transmit byte rates plus error and drop series from Prometheus.
type NetworkMetrics struct {
	ReceiveBytes    []timeseries.DataPoint `json:"receive_bytes"`
	TransmitBytes   []timeseries.DataPoint `json:"transmit_bytes"`
	ReceiveErrors   []timeseries.DataPoint `json:"receive_errors"`
	TransmitErrors  []timeseries.DataPoint `json:"transmit_errors"`
	ReceiveDropped  []timeseries.DataPoint `json:"receive_dropped"`
	TransmitDropped []timeseries.DataPoint `json:"transmit_dropped"`
}

func fetchPodMetrics(ctx context.Context, pod *database.Pod, opts AnalysisOptions) (NetworkMetrics, error) {
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
		receiveMatrix      model.Matrix
		transmitMatrix     model.Matrix
		receiveErrMatrix   model.Matrix
		transmitErrMatrix  model.Matrix
		receiveDropMatrix  model.Matrix
		transmitDropMatrix model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkReceiveBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive bytes metrics: %w", err)
		}
		receiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkTransmitBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
		}
		transmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkReceiveErrorsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive errors metrics: %w", err)
		}
		receiveErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkTransmitErrorsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit errors metrics: %w", err)
		}
		transmitErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkReceiveDroppedRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive dropped metrics: %w", err)
		}
		receiveDropMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryPodNetworkTransmitDroppedRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit dropped metrics: %w", err)
		}
		transmitDropMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return NetworkMetrics{}, err
	}

	return NetworkMetrics{
		ReceiveBytes:    timeseries.MatrixToDataPoints(receiveMatrix),
		TransmitBytes:   timeseries.MatrixToDataPoints(transmitMatrix),
		ReceiveErrors:   timeseries.MatrixToDataPoints(receiveErrMatrix),
		TransmitErrors:  timeseries.MatrixToDataPoints(transmitErrMatrix),
		ReceiveDropped:  timeseries.MatrixToDataPoints(receiveDropMatrix),
		TransmitDropped: timeseries.MatrixToDataPoints(transmitDropMatrix),
	}, nil
}

// fetchWorkloadMetrics queries each pod in parallel, then merges per-pod series with MergeDataPointsByTime
// so each timestamp reflects summed rates across the workload.
func fetchWorkloadMetrics(ctx context.Context, pods []*database.Pod, opts AnalysisOptions) (NetworkMetrics, error) {
	if len(pods) == 0 {
		return NetworkMetrics{}, fmt.Errorf("no pods found for workload")
	}

	end := time.Now()
	start := end.Add(-opts.TimeRange)
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  opts.RangeStep,
	}

	type podMetrics struct {
		ReceiveBytes    []timeseries.DataPoint
		TransmitBytes   []timeseries.DataPoint
		ReceiveErrors   []timeseries.DataPoint
		TransmitErrors  []timeseries.DataPoint
		ReceiveDropped  []timeseries.DataPoint
		TransmitDropped []timeseries.DataPoint
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
				receiveMatrix      model.Matrix
				transmitMatrix     model.Matrix
				receiveErrMatrix   model.Matrix
				transmitErrMatrix  model.Matrix
				receiveDropMatrix  model.Matrix
				transmitDropMatrix model.Matrix
			)

			podG, podCtx := errgroup.WithContext(gctx)

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkReceiveBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query receive bytes metrics: %w", err)
				}
				receiveMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkTransmitBytesRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
				}
				transmitMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkReceiveErrorsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query receive errors metrics: %w", err)
				}
				receiveErrMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkTransmitErrorsRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query transmit errors metrics: %w", err)
				}
				transmitErrMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkReceiveDroppedRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query receive dropped metrics: %w", err)
				}
				receiveDropMatrix = matrix
				return nil
			})

			podG.Go(func() error {
				matrix, _, err := prometheus.QueryPodNetworkTransmitDroppedRange(podCtx, r, queryOpts)
				if err != nil {
					return fmt.Errorf("failed to query transmit dropped metrics: %w", err)
				}
				transmitDropMatrix = matrix
				return nil
			})

			if err := podG.Wait(); err != nil {
				return err
			}

			results[i] = podMetrics{
				ReceiveBytes:    timeseries.MatrixToDataPoints(receiveMatrix),
				TransmitBytes:   timeseries.MatrixToDataPoints(transmitMatrix),
				ReceiveErrors:   timeseries.MatrixToDataPoints(receiveErrMatrix),
				TransmitErrors:  timeseries.MatrixToDataPoints(transmitErrMatrix),
				ReceiveDropped:  timeseries.MatrixToDataPoints(receiveDropMatrix),
				TransmitDropped: timeseries.MatrixToDataPoints(transmitDropMatrix),
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return NetworkMetrics{}, err
	}

	var receiveBytes, transmitBytes []timeseries.DataPoint
	var receiveErrors, transmitErrors []timeseries.DataPoint
	var receiveDropped, transmitDropped []timeseries.DataPoint
	for _, p := range results {
		receiveBytes = timeseries.MergeDataPointsByTime(receiveBytes, p.ReceiveBytes)
		transmitBytes = timeseries.MergeDataPointsByTime(transmitBytes, p.TransmitBytes)
		receiveErrors = timeseries.MergeDataPointsByTime(receiveErrors, p.ReceiveErrors)
		transmitErrors = timeseries.MergeDataPointsByTime(transmitErrors, p.TransmitErrors)
		receiveDropped = timeseries.MergeDataPointsByTime(receiveDropped, p.ReceiveDropped)
		transmitDropped = timeseries.MergeDataPointsByTime(transmitDropped, p.TransmitDropped)
	}

	return NetworkMetrics{
		ReceiveBytes:    receiveBytes,
		TransmitBytes:   transmitBytes,
		ReceiveErrors:   receiveErrors,
		TransmitErrors:  transmitErrors,
		ReceiveDropped:  receiveDropped,
		TransmitDropped: transmitDropped,
	}, nil
}

func fetchNamespaceMetrics(ctx context.Context, namespace string, opts AnalysisOptions) (NetworkMetrics, error) {
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
		receiveMatrix      model.Matrix
		transmitMatrix     model.Matrix
		receiveErrMatrix   model.Matrix
		transmitErrMatrix  model.Matrix
		receiveDropMatrix  model.Matrix
		transmitDropMatrix model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive bytes metrics: %w", err)
		}
		receiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitBytesRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
		}
		transmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveErrorsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive errors metrics: %w", err)
		}
		receiveErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitErrorsRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit errors metrics: %w", err)
		}
		transmitErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveDroppedRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive dropped metrics: %w", err)
		}
		receiveDropMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitDroppedRange(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit dropped metrics: %w", err)
		}
		transmitDropMatrix = matrix
		return nil
	})

	if err := g.Wait(); err != nil {
		return NetworkMetrics{}, err
	}

	return NetworkMetrics{
		ReceiveBytes:    timeseries.MatrixToDataPoints(receiveMatrix),
		TransmitBytes:   timeseries.MatrixToDataPoints(transmitMatrix),
		ReceiveErrors:   timeseries.MatrixToDataPoints(receiveErrMatrix),
		TransmitErrors:  timeseries.MatrixToDataPoints(transmitErrMatrix),
		ReceiveDropped:  timeseries.MatrixToDataPoints(receiveDropMatrix),
		TransmitDropped: timeseries.MatrixToDataPoints(transmitDropMatrix),
	}, nil
}
