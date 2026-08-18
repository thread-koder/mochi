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

// increaseOrNew is window growth, or the current counter when the series did not exist one window ago.
// increase() misses the first increment of a new counter (first scrape is already N).
func increaseOrNew(metric, namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	base := buildMochiNetMetricQuery(metric, namespace)
	inc := fmt.Sprintf("increase(%s[%s])", base, rangeDuration)
	return fmt.Sprintf("(%s > 0) or (%s unless %s offset %s)", inc, base, base, rangeDuration), nil
}

func BuildMochiNetConnectsQuery(namespace, rangeDuration string) (string, error) {
	return increaseOrNew("mochi_net_connects_total", namespace, rangeDuration)
}

func BuildMochiNetTxBytesQuery(namespace, rangeDuration string) (string, error) {
	return increaseOrNew("mochi_net_tx_bytes_total", namespace, rangeDuration)
}

func BuildMochiNetRxBytesQuery(namespace, rangeDuration string) (string, error) {
	return increaseOrNew("mochi_net_rx_bytes_total", namespace, rangeDuration)
}

func BuildMochiNetActiveConnectionsQuery(namespace string) (string, error) {
	return buildMochiNetMetricQuery("mochi_net_active_connections", namespace), nil
}
