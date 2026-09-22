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
	selector := buildMochiNetMetricQuery("mochi_net_connects_total", namespace)
	return increaseOrNew(selector, rangeDuration)
}

func BuildMochiNetTxBytesQuery(namespace, rangeDuration string) (string, error) {
	selector := buildMochiNetMetricQuery("mochi_net_tx_bytes_total", namespace)
	return increaseOrNew(selector, rangeDuration)
}

func BuildMochiNetRxBytesQuery(namespace, rangeDuration string) (string, error) {
	selector := buildMochiNetMetricQuery("mochi_net_rx_bytes_total", namespace)
	return increaseOrNew(selector, rangeDuration)
}

func BuildMochiNetActiveConnectionsQuery(namespace string) (string, error) {
	return buildMochiNetMetricQuery("mochi_net_active_connections", namespace), nil
}
