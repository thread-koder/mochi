package prometheus

import (
	"fmt"
)

// BuildNetworkMetricQuery builds a network metric selector for pod/interface scope.
func BuildNetworkMetricQuery(metric, namespace string, pods []string, iface, rangeDuration string) string {
	query := fmt.Sprintf(`%s{`, metric)
	needsComma := false
	if namespace != "" {
		query += fmt.Sprintf(`namespace="%s"`, namespace)
		needsComma = true
	}
	if len(pods) > 0 {
		if needsComma {
			query += ","
		}
		query += podLabelMatcher(pods)
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

// BuildWorkloadNetworkReceiveBytesQuery builds the workload network receive rate query.
func BuildWorkloadNetworkReceiveBytesQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_bytes_total", namespace, pods, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadNetworkTransmitBytesQuery builds the workload network transmit rate query.
func BuildWorkloadNetworkTransmitBytesQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_bytes_total", namespace, pods, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadNetworkReceiveErrorsQuery builds the workload network receive error rate query.
func BuildWorkloadNetworkReceiveErrorsQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_errors_total", namespace, pods, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadNetworkTransmitErrorsQuery builds the workload network transmit error rate query.
func BuildWorkloadNetworkTransmitErrorsQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_errors_total", namespace, pods, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadNetworkReceiveDroppedQuery builds the workload dropped receive packet rate query.
func BuildWorkloadNetworkReceiveDroppedQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_receive_packets_dropped_total", namespace, pods, "", rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadNetworkTransmitDroppedQuery builds the workload dropped transmit packet rate query.
func BuildWorkloadNetworkTransmitDroppedQuery(namespace string, pods []string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildNetworkMetricQuery("container_network_transmit_packets_dropped_total", namespace, pods, "", rangeDuration)
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
