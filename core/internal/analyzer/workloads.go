package analyzer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
	"golang.org/x/sync/errgroup"
)

// AnalyzeFunc is the per-workload callback each domain analyzer passes to AnalyzeWorkloads.
type AnalyzeFunc[T any] func(ctx context.Context, kind, name, namespace string, pods []*database.Pod) (T, error)

func AnalyzeWorkloads[T any](
	ctx context.Context,
	namespace string,
	analyze AnalyzeFunc[T],
) ([]T, error) {
	results := make([]T, 0)
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)

	runAnalyze := func(kind, name string, pods []*database.Pod) {
		g.Go(func() error {
			result, err := analyze(gctx, kind, name, namespace, pods)
			if err != nil {
				if errors.Is(err, &apperrors.NoMetricsError{}) {
					return nil
				}
				return fmt.Errorf("failed to analyze workload %s/%s: %w", kind, name, err)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
			return nil
		})
	}

	analyzeEntry := func(kind, name string, resolvedPods []*database.Pod) {
		if resolvedPods != nil {
			runAnalyze(kind, name, resolvedPods)
			return
		}

		g.Go(func() error {
			pods, err := database.GetPodsByWorkload(gctx, kind, name, namespace)
			if err != nil {
				return fmt.Errorf("failed to fetch pods for workload %s/%s: %w", kind, name, err)
			}
			if len(pods) == 0 {
				return nil
			}
			runAnalyze(kind, name, pods)
			return nil
		})
	}

	g.Go(func() error {
		deployments, err := database.GetDeploymentsByNamespace(gctx, namespace)
		if err != nil {
			return err
		}
		for _, deployment := range deployments {
			analyzeEntry("Deployment", deployment.Name, nil)
		}
		return nil
	})

	g.Go(func() error {
		statefulSets, err := database.GetStatefulSetsByNamespace(gctx, namespace)
		if err != nil {
			return err
		}
		for _, statefulSet := range statefulSets {
			analyzeEntry("StatefulSet", statefulSet.Name, nil)
		}
		return nil
	})

	g.Go(func() error {
		daemonSets, err := database.GetDaemonSetsByNamespace(gctx, namespace)
		if err != nil {
			return err
		}
		for _, daemonSet := range daemonSets {
			analyzeEntry("DaemonSet", daemonSet.Name, nil)
		}
		return nil
	})

	g.Go(func() error {
		standalonePods, err := database.GetStandalonePodsByNamespace(gctx, namespace)
		if err != nil {
			return err
		}
		for _, pod := range standalonePods {
			analyzeEntry("Pod", pod.Name, []*database.Pod{pod})
		}
		return nil
	})

	g.Go(func() error {
		systemPods, err := database.GetPodsByOwnerKind(gctx, "Node", namespace)
		if err != nil {
			return err
		}
		for _, pod := range systemPods {
			analyzeEntry("Pod", pod.Name, []*database.Pod{pod})
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
