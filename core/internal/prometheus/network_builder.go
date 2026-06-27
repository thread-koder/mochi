package prometheus

import (
	"fmt"
)

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

func BuildNamespaceNetworkReceiveBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceNetworkTransmitBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceNetworkReceiveErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceNetworkTransmitErrorsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_errors_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceNetworkReceiveDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_receive_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceNetworkTransmitDroppedQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_network_transmit_packets_dropped_total{namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
