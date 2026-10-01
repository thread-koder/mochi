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

func buildMochiHTTPMetricQuery(metric string, matchers ...string) string {
	if len(matchers) == 0 {
		return metric + `{}`
	}
	return metric + `{` + strings.Join(matchers, ",") + `}`
}

// buildMochiHTTPNamespaceScopedExpr builds a PromQL instant vector for one metric.
// A single selector cannot express src_namespace=X OR dst_namespace=X. Use binary or,
// then the caller sum-by so identical label sets (both ends in ns) dedupe.
func buildMochiHTTPNamespaceScopedExpr(
	metric, namespace string,
	extraMatchers []string,
	wrap func(selector string) (string, error),
) (string, error) {
	if namespace == "" {
		return wrap(buildMochiHTTPMetricQuery(metric, extraMatchers...))
	}

	srcMatchers := append([]string{fmt.Sprintf(`src_namespace="%s"`, namespace)}, extraMatchers...)
	dstMatchers := append([]string{fmt.Sprintf(`dst_namespace="%s"`, namespace)}, extraMatchers...)

	srcExpr, err := wrap(buildMochiHTTPMetricQuery(metric, srcMatchers...))
	if err != nil {
		return "", err
	}
	dstExpr, err := wrap(buildMochiHTTPMetricQuery(metric, dstMatchers...))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s) or (%s)", srcExpr, dstExpr), nil
}

func BuildMochiHTTPRequestsQuery(namespace, rangeDuration string) (string, error) {
	scoped, err := buildMochiHTTPNamespaceScopedExpr(
		"mochi_http_requests_total",
		namespace,
		nil,
		func(selector string) (string, error) {
			return increaseOrNew(selector, rangeDuration)
		},
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sum by (%s) (%s)", httpSumByClause, scoped), nil
}

func BuildMochiHTTPErrorsQuery(namespace, rangeDuration string) (string, error) {
	scoped, err := buildMochiHTTPNamespaceScopedExpr(
		"mochi_http_requests_total",
		namespace,
		[]string{`status_class="5xx"`},
		func(selector string) (string, error) {
			return increaseOrNew(selector, rangeDuration)
		},
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sum by (%s) (%s)", httpSumByClause, scoped), nil
}

func BuildMochiHTTPDurationQuantileQuery(quantile float64, namespace, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	scoped, err := buildMochiHTTPNamespaceScopedExpr(
		"mochi_http_request_duration_seconds",
		namespace,
		nil,
		func(selector string) (string, error) {
			return fmt.Sprintf("rate(%s[%s])", selector, rangeDuration), nil
		},
	)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"histogram_quantile(%g, sum by (%s) (%s))",
		quantile,
		httpSumByClause,
		scoped,
	), nil
}

func BuildMochiHTTPDurationP50Query(namespace, rangeDuration string) (string, error) {
	return BuildMochiHTTPDurationQuantileQuery(0.5, namespace, rangeDuration)
}

func BuildMochiHTTPDurationP95Query(namespace, rangeDuration string) (string, error) {
	return BuildMochiHTTPDurationQuantileQuery(0.95, namespace, rangeDuration)
}
