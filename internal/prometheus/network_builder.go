package prometheus

import (
	"fmt"
)

// Builds a network interface-scoped PromQL selector for a metric.
func BuildNetworkMetricQuery(metric, namespace, pod, iface, rangeDuration string) string {
	query := fmt.Sprintf(`%s{`, metric)
	needsComma := false
	if namespace != "" {
		query += fmt.Sprintf(`namespace="%s"`, namespace)
		needsComma = true
	}
	if pod != "" {
		if needsComma {
			query += ","
		}
		query += fmt.Sprintf(`pod="%s"`, pod)
		needsComma = true
	}
	if iface != "" {
		if needsComma {
			query += ","
		}
		query += fmt.Sprintf(`interface="%s"`, iface)
	}
	query += `}`

	if rangeDuration != "" {
		query += fmt.Sprintf(`[%s]`, rangeDuration)
	}
	return query
}

// Builds a PromQL query for pod network receive bytes (bytes/sec)
func BuildPodNetworkReceiveBytesQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_bytes_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for pod network transmit bytes (bytes/sec)
func BuildPodNetworkTransmitBytesQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_bytes_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for pod network receive errors (errors/sec)
func BuildPodNetworkReceiveErrorsQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_errors_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for pod network transmit errors (errors/sec)
func BuildPodNetworkTransmitErrorsQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_errors_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for pod network receive packets dropped (packets/sec)
func BuildPodNetworkReceiveDroppedQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_packets_dropped_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for pod network transmit packets dropped (packets/sec)
func BuildPodNetworkTransmitDroppedQuery(namespace, pod string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_packets_dropped_total", namespace, pod, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// Builds a PromQL query for namespace network receive bytes (bytes/sec)
func BuildNamespaceNetworkReceiveBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace network transmit bytes (bytes/sec)
func BuildNamespaceNetworkTransmitBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace network receive errors (errors/sec)
func BuildNamespaceNetworkReceiveErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace network transmit errors (errors/sec)
func BuildNamespaceNetworkTransmitErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace network receive packets dropped (packets/sec)
func BuildNamespaceNetworkReceiveDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace network transmit packets dropped (packets/sec)
func BuildNamespaceNetworkTransmitDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
