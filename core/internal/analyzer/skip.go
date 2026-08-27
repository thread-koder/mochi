package analyzer

import (
	"context"
	"errors"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"golang.org/x/sync/errgroup"
)

// SkipNoMetrics runs fn for each item concurrently, skips NoMetricsError,
// and returns successful results compacted in input order.
func SkipNoMetrics[In, Out any](
	ctx context.Context,
	items []In,
	fn func(context.Context, In) (Out, error),
) ([]Out, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]Out, len(items))
	ok := make([]bool, len(items))

	g, gctx := errgroup.WithContext(ctx)
	for i, item := range items {
		g.Go(func() error {
			result, err := fn(gctx, item)
			if err != nil {
				if errors.Is(err, &apperrors.NoMetricsError{}) {
					return nil
				}
				return err
			}
			results[i] = result
			ok[i] = true
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	compact := make([]Out, 0, len(items))
	for i := range items {
		if ok[i] {
			compact = append(compact, results[i])
		}
	}

	return compact, nil
}
