package prometheus

import (
	"fmt"
)

func buildMochiNetMetricQuery(metric, namespace string) string {
	query := metric + `{`
	if namespace != "" {
		query += fmt.Sprintf(`src_namespace="%s"`, namespace)
	}
	query += `}`
	return query
}

func BuildMochiNetConnectsQuery(namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	base := buildMochiNetMetricQuery("mochi_net_connects_total", namespace)
	return fmt.Sprintf("increase(%s[%s])", base, rangeDuration), nil
}

func BuildMochiNetTxBytesQuery(namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	base := buildMochiNetMetricQuery("mochi_net_tx_bytes_total", namespace)
	return fmt.Sprintf("increase(%s[%s])", base, rangeDuration), nil
}

func BuildMochiNetRxBytesQuery(namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	base := buildMochiNetMetricQuery("mochi_net_rx_bytes_total", namespace)
	return fmt.Sprintf("increase(%s[%s])", base, rangeDuration), nil
}

func BuildMochiNetActiveConnectionsQuery(namespace string) (string, error) {
	return buildMochiNetMetricQuery("mochi_net_active_connections", namespace), nil
}
