package prometheus

import (
	"fmt"
)

// Builds a PromQL query for pod CPU usage
func BuildPodCPUQuery(namespace, pod, container string, rangeDuration string) string {
	baseQuery := `rate(container_cpu_usage_seconds_total{container!="POD",container!=""`

	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	query := baseQuery
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	if pod != "" {
		query += fmt.Sprintf(`,pod="%s"`, pod)
	}
	if container != "" {
		query += fmt.Sprintf(`,container="%s"`, container)
	}
	query += fmt.Sprintf(`}[%s])`, rangeDuration)

	return query
}

// Builds a PromQL query for pod memory usage
func BuildPodMemoryQuery(namespace, pod, container string) string {
	baseQuery := `container_memory_working_set_bytes{container!="POD",container!=""`

	query := baseQuery
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

	return query
}

// Builds a PromQL query for node CPU usage
func BuildNodeCPUQuery(node string, rangeDuration string) string {
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	query := `1 - avg(rate(node_cpu_seconds_total{mode="idle"`
	if node != "" {
		query += fmt.Sprintf(`,instance=~".*%s.*"`, node)
	}
	query += fmt.Sprintf(`}[%s]))`, rangeDuration)

	return query
}

// Builds a PromQL query for node memory usage
func BuildNodeMemoryQuery(node string) string {
	query := `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`
	if node != "" {
		query = fmt.Sprintf(`(1 - (node_memory_MemAvailable_bytes{instance=~".*%s.*"} / node_memory_MemTotal_bytes{instance=~".*%s.*"})) * 100`, node, node)
	}
	return query
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
