package prometheus

import (
	"fmt"
)

// Builds a PromQL query for pod CPU usage
func BuildPodCPUQuery(namespace, pod, container string, rangeDuration string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	return fmt.Sprintf("rate(%s)", BuildContainerMetricQuery("container_cpu_usage_seconds_total", namespace, pod, container, rangeDuration))
}

// Builds a PromQL query for pod memory usage
func BuildPodMemoryQuery(namespace, pod, container string) string {
	return BuildContainerMetricQuery("container_memory_working_set_bytes", namespace, pod, container, "")
}

// Builds a PromQL query for pod CPU throttling (CFS)
func BuildPodCPUThrottlingQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}

	throttledMetric := BuildContainerMetricQuery("container_cpu_cfs_throttled_periods_total", namespace, pod, container, "")
	totalMetric := BuildContainerMetricQuery("container_cpu_cfs_periods_total", namespace, pod, container, "")
	throttledRate := fmt.Sprintf("rate(%s[%s])", throttledMetric, rangeDuration)
	totalRate := fmt.Sprintf("rate(%s[%s])", totalMetric, rangeDuration)
	ratioQuery := fmt.Sprintf("clamp_min(%s / clamp_min(%s, 0.0001), 0)", throttledRate, totalRate)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", ratioQuery, timeRange, step)
}

// Builds a PromQL query for pod CPU pressure (stalled)
func BuildPodCPUPressureQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}

	baseMetric := BuildContainerMetricQuery("container_pressure_cpu_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", baseMetric, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step)
}

// Builds a PromQL query for pod memory fail count
func BuildPodMemoryFailCountQuery(namespace, pod, container string, rangeDuration string) string {
	return fmt.Sprintf("increase(%s)", BuildContainerMetricQuery("container_memory_failcnt", namespace, pod, container, rangeDuration))
}

// Builds a PromQL query for pod memory OOM events
func BuildPodMemoryOOMQuery(namespace, pod, container string, rangeDuration string) string {
	return fmt.Sprintf("increase(%s)", BuildContainerMetricQuery("container_oom_events_total", namespace, pod, container, rangeDuration))
}

// Builds a PromQL query for pod memory pressure (stalled)
func BuildPodMemoryPressureQuery(namespace, pod, container string, rangeDuration string, timeRange string, step string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}
	if step == "" {
		step = "5m"
	}

	baseMetric := BuildContainerMetricQuery("container_pressure_memory_stalled_seconds_total", namespace, pod, container, "")
	rateQuery := fmt.Sprintf("rate(%s[%s])", baseMetric, rangeDuration)
	return fmt.Sprintf("avg_over_time((%s)[%s:%s])", rateQuery, timeRange, step)
}

// Builds a PromQL query for container restarts
func BuildContainerRestartsQuery(namespace, pod, container string, rangeDuration string) string {
	query := fmt.Sprintf(`kube_pod_container_status_restarts_total{namespace="%s",pod="%s",container="%s"}`, namespace, pod, container)
	return fmt.Sprintf("increase(%s[%s])", query, rangeDuration)
}

// Builds a PromQL query for namespace CPU aggregation
func BuildNamespaceCPUQuery(namespace string, rangeDuration string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	query := `sum(rate(container_cpu_usage_seconds_total{container!="POD",container!=""`
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	query += fmt.Sprintf(`}[%s])) by (namespace)`, rangeDuration)

	return query
}

// Builds a PromQL query for namespace memory aggregation
func BuildNamespaceMemoryQuery(namespace string) string {
	query := `sum(container_memory_working_set_bytes{container!="POD",container!=""`
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	query += `}) by (namespace)`

	return query
}

// Helper to build container-scoped queries
func BuildContainerMetricQuery(metric, namespace, pod, container, rangeDuration string) string {
	query := fmt.Sprintf(`%s{container!="POD",container!=""`, metric)
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	if pod != "" {
		query += fmt.Sprintf(`,pod="%s"`, pod)
	}
	if container != "" {
		query += fmt.Sprintf(`,container="%s"`, container)
	}
	query += `}`

	if rangeDuration != "" {
		query += fmt.Sprintf(`[%s]`, rangeDuration)
	}
	return query
}
