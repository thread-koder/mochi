package prometheus

import (
	"fmt"
)

// BuildWorkloadCPUQuery builds the workload CPU usage query.
func BuildWorkloadCPUQuery(namespace string, pods []string, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_cpu_usage_seconds_total", namespace, pods, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadMemoryQuery builds the workload memory working set query.
func BuildWorkloadMemoryQuery(namespace string, pods []string, container string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}

	base := BuildContainerMetricQuery("container_memory_working_set_bytes", namespace, pods, container, "")
	return fmt.Sprintf("sum(%s)", base), nil
}

// BuildWorkloadCPUThrottlingQuery builds the workload CFS throttling ratio query.
func BuildWorkloadCPUThrottlingQuery(namespace string, pods []string, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Use a coarse default to avoid noisy short-window spikes.
		step = "5m"
	}

	throttledMetric := BuildContainerMetricQuery("container_cpu_cfs_throttled_periods_total", namespace, pods, container, "")
	totalMetric := BuildContainerMetricQuery("container_cpu_cfs_periods_total", namespace, pods, container, "")
	throttledRate := fmt.Sprintf("sum(rate(%s[%s]))", throttledMetric, rangeDuration)
	totalRate := fmt.Sprintf("sum(rate(%s[%s]))", totalMetric, rangeDuration)
	// Guard against zero denominator, so the ratio is bounded at 0+.
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
}

// BuildWorkloadCPUPressureQuery builds the workload CPU pressure query.
func BuildWorkloadCPUPressureQuery(namespace string, pods []string, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Match range defaults so the averages are stable across callers.
		step = "5m"
	}

	base := BuildContainerMetricQuery("container_pressure_cpu_stalled_seconds_total", namespace, pods, container, "")

	// Container-level: sum stalled rate across all instances of this container
	if container != "" {
		aggRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Single pod: average stalled rate across containers in the pod (keeps metric as a 0–1 fraction)
	if len(pods) == 1 {
		rateQuery := fmt.Sprintf("rate(%s[%s])", base, rangeDuration)
		aggRate := fmt.Sprintf("avg(%s)", rateQuery)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Multi-pod: average stalled rate across containers in the workload (keeps metric as a 0–1 fraction)
	rateQuery := fmt.Sprintf("avg(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// BuildWorkloadMemoryFailCountQuery builds the workload memory failcnt increase query.
func BuildWorkloadMemoryFailCountQuery(namespace string, pods []string, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	base := BuildContainerMetricQuery("container_memory_failcnt", namespace, pods, container, timeRange)
	return fmt.Sprintf("sum(increase(%s))", base), nil
}

// BuildWorkloadMemoryOOMQuery builds the workload OOM event increase query.
func BuildWorkloadMemoryOOMQuery(namespace string, pods []string, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	base := BuildContainerMetricQuery("container_oom_events_total", namespace, pods, container, timeRange)
	return fmt.Sprintf("sum(increase(%s))", base), nil
}

// BuildWorkloadMemoryPressureQuery builds the workload memory pressure query.
func BuildWorkloadMemoryPressureQuery(namespace string, pods []string, container string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		// Match range defaults so the averages are stable across callers.
		step = "5m"
	}

	base := BuildContainerMetricQuery("container_pressure_memory_stalled_seconds_total", namespace, pods, container, "")

	// Container-level: sum stalled rate across all instances of this container
	if container != "" {
		aggRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Single pod: average stalled rate across containers in the pod (keeps metric as a 0–1 fraction)
	if len(pods) == 1 {
		rateQuery := fmt.Sprintf("rate(%s[%s])", base, rangeDuration)
		aggRate := fmt.Sprintf("avg(%s)", rateQuery)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
	}

	// Multi-pod: average stalled rate across containers in the workload (keeps metric as a 0–1 fraction)
	rateQuery := fmt.Sprintf("avg(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// BuildWorkloadRestartsQuery builds the workload/container restart increase query.
func BuildWorkloadRestartsQuery(namespace string, pods []string, container string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if timeRange == "" {
		timeRange = "5m"
	}

	podMatch := podLabelMatcher(pods)

	// Container-level: restarts for a single container
	if container != "" {
		query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",%s,container="%s"}`, namespace, podMatch, container)
		return fmt.Sprintf("sum(increase(%s[%s]))", query, timeRange), nil
	}

	// Workload-level: sum of restarts across all containers in the matched pods
	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",%s}`, namespace, podMatch)
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
