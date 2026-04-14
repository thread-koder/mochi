package prometheus

import (
	"fmt"
)

// BuildNetworkMetricQuery builds a network metric selector for pod/interface scope.
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

// BuildPodNetworkReceiveBytesQuery builds the pod network receive rate query.
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

// BuildPodNetworkTransmitBytesQuery builds the pod network transmit rate query.
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

// BuildPodNetworkReceiveErrorsQuery builds the pod network receive error rate query.
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

// BuildPodNetworkTransmitErrorsQuery builds the pod network transmit error rate query.
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

// BuildPodNetworkReceiveDroppedQuery builds the pod dropped receive packet rate query.
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

// BuildPodNetworkTransmitDroppedQuery builds the pod dropped transmit packet rate query.
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

// BuildNamespaceNetworkReceiveBytesQuery builds the namespace network receive rate query.
func BuildNamespaceNetworkReceiveBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceNetworkTransmitBytesQuery builds the namespace network transmit rate query.
func BuildNamespaceNetworkTransmitBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceNetworkReceiveErrorsQuery builds the namespace receive error rate query.
func BuildNamespaceNetworkReceiveErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceNetworkTransmitErrorsQuery builds the namespace transmit error rate query.
func BuildNamespaceNetworkTransmitErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceNetworkReceiveDroppedQuery builds the namespace dropped receive packet rate query.
func BuildNamespaceNetworkReceiveDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceNetworkTransmitDroppedQuery builds the namespace dropped transmit packet rate query.
func BuildNamespaceNetworkTransmitDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
