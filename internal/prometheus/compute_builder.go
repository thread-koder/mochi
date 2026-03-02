package prometheus

import (
	"fmt"
)

// Builds a PromQL query for pod CPU usage
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
	return fmt.Sprintf("rate(%s)", BuildContainerMetricQuery("container_cpu_usage_seconds_total", namespace, pod, container, rangeDuration)), nil
}

// Builds a PromQL query for pod memory usage
func BuildPodMemoryQuery(namespace, pod, container string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	return BuildContainerMetricQuery("container_memory_working_set_bytes", namespace, pod, container, ""), nil
}

// Builds a PromQL query for pod CPU throttling (CFS)
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
		step = "5m"
	}

	throttledMetric := BuildContainerMetricQuery("container_cpu_cfs_throttled_periods_total", namespace, pod, container, "")
	totalMetric := BuildContainerMetricQuery("container_cpu_cfs_periods_total", namespace, pod, container, "")

	// Container-level: ratio for a single container
	if container != "" {
		throttledRate := fmt.Sprintf("rate(%s[%s])", throttledMetric, rangeDuration)
		totalRate := fmt.Sprintf("rate(%s[%s])", totalMetric, rangeDuration)
		ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
	}

	// Pod-level: ratio-of-sums across all containers in the pod
	throttledRate := fmt.Sprintf("sum(rate(%s[%s]))", throttledMetric, rangeDuration)
	totalRate := fmt.Sprintf("sum(rate(%s[%s]))", totalMetric, rangeDuration)
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
}

// Builds a PromQL query for pod CPU pressure (stalled)
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
		step = "5m"
	}

	baseMetric := BuildContainerMetricQuery("container_pressure_cpu_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", baseMetric, rangeDuration)

	// Container-level: single-container stalled fraction
	if container != "" {
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
	}

	// Pod-level: average stalled fraction across all containers in the pod
	aggRate := fmt.Sprintf("avg(%s)", rateQuery)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
}

// Builds a PromQL query for pod memory fail count
func BuildPodMemoryFailCountQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	metric := BuildContainerMetricQuery("container_memory_failcnt", namespace, pod, container, rangeDuration)

	// Container-level: increase for a single container
	if container != "" {
		return fmt.Sprintf("increase(%s)", metric), nil
	}

	// Pod-level: sum of increases across all containers in the pod
	query := fmt.Sprintf("sum(increase(%s))", metric)
	return query, nil
}

// Builds a PromQL query for pod memory OOM events
func BuildPodMemoryOOMQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	metric := BuildContainerMetricQuery("container_oom_events_total", namespace, pod, container, rangeDuration)

	// Container-level: increase for a single container
	if container != "" {
		return fmt.Sprintf("increase(%s)", metric), nil
	}

	// Pod-level: sum of increases across all containers in the pod
	query := fmt.Sprintf("sum(increase(%s))", metric)
	return query, nil
}

// Builds a PromQL query for pod memory pressure (stalled)
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
		step = "5m"
	}

	baseMetric := BuildContainerMetricQuery("container_pressure_memory_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", baseMetric, rangeDuration)

	// Container-level: single-container stalled fraction
	if container != "" {
		return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
	}

	// Pod-level: average stalled fraction across all containers in the pod
	aggRate := fmt.Sprintf("avg(%s)", rateQuery)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", aggRate, timeRange, step), nil
}

// Builds a PromQL query for pod restarts
func BuildPodRestartsQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if container != "" {
		// Container-level: increase for a single container
		query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",pod="%s",container="%s"}`, namespace, pod, container)
		return fmt.Sprintf("increase(%s[%s])", query, rangeDuration), nil
	}

	// Pod-level: sum of increases across all containers in the pod
	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",pod="%s"}`, namespace, pod)
	return fmt.Sprintf("sum(increase(%s[%s]))", query, rangeDuration), nil
}

// Builds a PromQL query for namespace CPU usage
func BuildNamespaceCPUQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	query := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration)
	return query, nil
}

// Builds a PromQL query for namespace memory usage
func BuildNamespaceMemoryQuery(namespace string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	query := fmt.Sprintf(`sum(container_memory_working_set_bytes{container!="POD",container!="",namespace="%s"})`, namespace)

	return query, nil
}

// Builds a PromQL query for namespace CPU throttling (CFS)
func BuildNamespaceCPUThrottlingQuery(namespace string, rangeDuration string, timeRange string, step string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}
	base := fmt.Sprintf(`container_cpu_cfs_throttled_periods_total{container!="POD",container!="",namespace="%s"}`, namespace)
	throttledRate := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
	baseTotal := fmt.Sprintf(`container_cpu_cfs_periods_total{container!="POD",container!="",namespace="%s"}`, namespace)
	totalRate := fmt.Sprintf("sum(rate(%s[%s]))", baseTotal, rangeDuration)
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step), nil
}

// Builds a PromQL query for namespace CPU pressure (stalled)
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
	rateQuery := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// Builds a PromQL query for namespace memory fail count
func BuildNamespaceMemoryFailCountQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	base := fmt.Sprintf(`container_memory_failcnt{container!="POD",container!="",namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", base, timeRange), nil
}

// Builds a PromQL query for namespace memory OOM events
func BuildNamespaceMemoryOOMQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	base := fmt.Sprintf(`container_oom_events_total{container!="POD",container!="",namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", base, timeRange), nil
}

// Builds a PromQL query for namespace memory pressure (stalled)
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
	rateQuery := fmt.Sprintf("sum(rate(%s[%s]))", base, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step), nil
}

// Builds a PromQL query for namespace container restarts
func BuildNamespaceRestartsQuery(namespace string, timeRange string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s"}`, namespace)
	return fmt.Sprintf("sum(increase(%s[%s]))", query, timeRange), nil
}
