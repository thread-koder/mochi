package prometheus

import (
	"fmt"
)

// BuildPodCPUQuery builds the pod CPU usage query.
func BuildPodCPUQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_cpu_usage_seconds_total", namespace, pod, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildPodMemoryQuery builds the pod memory working set query.
func BuildPodMemoryQuery(namespace, pod, container string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}

	base := BuildContainerMetricQuery("container_memory_working_set_bytes", namespace, pod, container, "")
	return fmt.Sprintf("sum(%s)", base), nil
}

// BuildPodCPUThrottlingQuery builds the pod CFS throttling ratio query.
func BuildPodCPUThrottlingQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Use a coarse default to avoid noisy short-window spikes.
		step = "5m"
	}

	throttledMetric := BuildContainerMetricQuery("container_cpu_cfs_throttled_periods_total", namespace, pod, container, "")
	totalMetric := BuildContainerMetricQuery("container_cpu_cfs_periods_total", namespace, pod, container, "")
	throttledRate := fmt.Sprintf("sum(rate(%s[%s]))", throttledMetric, rangeDuration)
	totalRate := fmt.Sprintf("sum(rate(%s[%s]))", totalMetric, rangeDuration)
	// Guard against zero denominator, so the ratio is bounded at 0+.
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
}

// BuildPodCPUPressureQuery builds the pod CPU pressure query.
func BuildPodCPUPressureQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Match range defaults so the averages are stable across callers.
		step = "5m"
	}

	base := BuildContainerMetricQuery("container_pressure_cpu_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", base, rangeDuration)

	// Container-level: sum stalled rate across all instances of this container
	if container != "" {
		aggRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Pod-level: average stalled rate across containers in the pod (keeps metric as a 0–1 fraction)
	aggRate := fmt.Sprintf("avg(%s)", rateQuery)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
}

// BuildPodMemoryFailCountQuery builds the pod memory failcnt increase query.
func BuildPodMemoryFailCountQuery(namespace, pod, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	base := BuildContainerMetricQuery("container_memory_failcnt", namespace, pod, container, timeRange)
	return fmt.Sprintf("sum(increase(%s))", base), nil
}

// BuildPodMemoryOOMQuery builds the pod OOM event increase query.
func BuildPodMemoryOOMQuery(namespace, pod, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	base := BuildContainerMetricQuery("container_oom_events_total", namespace, pod, container, timeRange)
	return fmt.Sprintf("sum(increase(%s))", base), nil
}

// BuildPodMemoryPressureQuery builds the pod memory pressure query.
func BuildPodMemoryPressureQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Match range defaults so the averages are stable across callers.
		step = "5m"
	}

	base := BuildContainerMetricQuery("container_pressure_memory_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", base, rangeDuration)

	// Container-level: sum stalled rate across all instances of this container
	if container != "" {
		aggRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Pod-level: average stalled rate across containers in the pod (keeps metric as a 0–1 fraction)
	aggRate := fmt.Sprintf("avg(%s)", rateQuery)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
}

// BuildPodRestartsQuery builds the pod/container restart increase query.
func BuildPodRestartsQuery(namespace, pod, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	// Container-level: restarts for a single container
	if container != "" {
		query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",pod="%s",container="%s"}`, namespace, pod, container)
		return fmt.Sprintf("sum(increase(%s[%s]))", query, timeRange), nil
	}

	// Pod-level: sum of restarts across all containers in the pod
	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",pod="%s"}`, namespace, pod)
	return fmt.Sprintf("sum(increase(%s[%s]))", query, timeRange), nil
}

// BuildNamespaceCPUQuery builds the namespace CPU usage query.
func BuildNamespaceCPUQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceMemoryQuery builds the namespace memory working set query.
func BuildNamespaceMemoryQuery(namespace string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	return fmt.Sprintf(`sum(container_memory_working_set_bytes{container!="POD",container!="",namespace="%s"})`, namespace), nil
}

// BuildNamespaceCPUThrottlingQuery builds the namespace CFS throttling ratio query.
func BuildNamespaceCPUThrottlingQuery(namespace string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Use a coarse default to avoid noisy short-window spikes.
		step = "5m"
	}

	base := fmt.Sprintf(`container_cpu_cfs_throttled_periods_total{container!="POD",container!="",namespace="%s"}`, namespace)
	throttledRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
	baseTotal := fmt.Sprintf(`container_cpu_cfs_periods_total{container!="POD",container!="",namespace="%s"}`, namespace)
	totalRate := fmt.Sprintf("sum(rate(%s[%s]))", baseTotal, rangeDuration)
	// Guard against zero denominator, so the ratio is bounded at 0+.
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
}

// BuildNamespaceCPUPressureQuery builds the namespace CPU pressure query.
func BuildNamespaceCPUPressureQuery(namespace string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}

	base := fmt.Sprintf(`container_pressure_cpu_stalled_seconds_total{container!="POD",container!="",namespace="%s"}`, namespace)
	// Average stalled rate across containers in the namespace (keeps metric as a 0–1 fraction)
	rateQuery := fmt.Sprintf("avg(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// BuildNamespaceMemoryFailCountQuery builds the namespace memory failcnt increase query.
func BuildNamespaceMemoryFailCountQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	base := fmt.Sprintf(`container_memory_failcnt{container!="POD",container!="",namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", base, timeRange), nil
}

// BuildNamespaceMemoryOOMQuery builds the namespace OOM event increase query.
func BuildNamespaceMemoryOOMQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	base := fmt.Sprintf(`container_oom_events_total{container!="POD",container!="",namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", base, timeRange), nil
}

// BuildNamespaceMemoryPressureQuery builds the namespace memory pressure query.
func BuildNamespaceMemoryPressureQuery(namespace string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}

	base := fmt.Sprintf(`container_pressure_memory_stalled_seconds_total{container!="POD",container!="",namespace="%s"}`, namespace)
	// Average stalled rate across containers in the namespace (keeps metric as a 0–1 fraction)
	rateQuery := fmt.Sprintf("avg(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// BuildNamespaceRestartsQuery builds the namespace container restart increase query.
func BuildNamespaceRestartsQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", query, timeRange), nil
}
