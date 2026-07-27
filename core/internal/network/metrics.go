package network

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
		Pods:          []string{pod.Name},
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
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive bytes metrics: %w", err)
		}
		receiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
		}
		transmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive errors metrics: %w", err)
		}
		receiveErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit errors metrics: %w", err)
		}
		transmitErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveDropped(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive dropped metrics: %w", err)
		}
		receiveDropMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitDropped(gctx, r, queryOpts)
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
		receiveMatrix      model.Matrix
		transmitMatrix     model.Matrix
		receiveErrMatrix   model.Matrix
		transmitErrMatrix  model.Matrix
		receiveDropMatrix  model.Matrix
		transmitDropMatrix model.Matrix
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive bytes metrics: %w", err)
		}
		receiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
		}
		transmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive errors metrics: %w", err)
		}
		receiveErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit errors metrics: %w", err)
		}
		transmitErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkReceiveDropped(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive dropped metrics: %w", err)
		}
		receiveDropMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryWorkloadNetworkTransmitDropped(gctx, r, queryOpts)
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
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive bytes metrics: %w", err)
		}
		receiveMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitBytes(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit bytes metrics: %w", err)
		}
		transmitMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive errors metrics: %w", err)
		}
		receiveErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitErrors(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query transmit errors metrics: %w", err)
		}
		transmitErrMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkReceiveDropped(gctx, r, queryOpts)
		if err != nil {
			return fmt.Errorf("failed to query receive dropped metrics: %w", err)
		}
		receiveDropMatrix = matrix
		return nil
	})

	g.Go(func() error {
		matrix, _, err := prometheus.QueryNamespaceNetworkTransmitDropped(gctx, r, queryOpts)
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
