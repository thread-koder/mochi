package prometheus

import (
	"fmt"
	"strings"
)

// HTTP identity labels matching agent mochi_http_* (without status_class).
var httpSumByLabels = []string{
	"src_pod_uid",
	"src_namespace",
	"src_pod",
	"dst_pod_uid",
	"dst_namespace",
	"dst_pod",
	"dst_ip",
	"dst_port",
	"actual_dst_ip",
	"actual_dst_port",
	"protocol",
	"dst_hostname",
	"method",
	"route",
}

var httpSumByClause = strings.Join(httpSumByLabels, ",")

func buildMochiHTTPMetricQuery(metric, namespace string, extraMatchers ...string) string {
	parts := make([]string, 0, 1+len(extraMatchers))
	if namespace != "" {
		parts = append(parts, fmt.Sprintf(`src_namespace="%s"`, namespace))
	}
	parts = append(parts, extraMatchers...)
	if len(parts) == 0 {
		return metric + `{}`
	}
	return metric + `{` + strings.Join(parts, ",") + `}`
}

func BuildMochiHTTPRequestsQuery(namespace, rangeDuration string) (string, error) {
	selector := buildMochiHTTPMetricQuery("mochi_http_requests_total", namespace)
	base, err := increaseOrNew(selector, rangeDuration)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sum by (%s) (%s)", httpSumByClause, base), nil
}

func BuildMochiHTTPErrorsQuery(namespace, rangeDuration string) (string, error) {
	selector := buildMochiHTTPMetricQuery("mochi_http_requests_total", namespace, `status_class="5xx"`)
	base, err := increaseOrNew(selector, rangeDuration)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sum by (%s) (%s)", httpSumByClause, base), nil
}

func BuildMochiHTTPDurationQuantileQuery(quantile float64, namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	selector := buildMochiHTTPMetricQuery("mochi_http_request_duration_seconds", namespace)
	return fmt.Sprintf(
		"histogram_quantile(%g, sum by (%s) (rate(%s[%s])))",
		quantile,
		httpSumByClause,
		selector,
		rangeDuration,
	), nil
}

func BuildMochiHTTPDurationP50Query(namespace, rangeDuration string) (string, error) {
	return BuildMochiHTTPDurationQuantileQuery(0.5, namespace, rangeDuration)
}

func BuildMochiHTTPDurationP95Query(namespace, rangeDuration string) (string, error) {
	return BuildMochiHTTPDurationQuantileQuery(0.95, namespace, rangeDuration)
}
