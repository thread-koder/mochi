package prometheus

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildContainerMetricQuery builds a container metric selector used by cAdvisor queries.
// It always excludes the synthetic POD container and empty container labels.
func BuildContainerMetricQuery(metric, namespace string, pods []string, container, rangeDuration string) string {
	query := fmt.Sprintf(`%s{container!="POD",container!=""`, metric)
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	if len(pods) > 0 {
		query += "," + podLabelMatcher(pods)
	}
	if container != "" {
		query += fmt.Sprintf(`,container="%s"`, container)
	}
	query += `}`

	if rangeDuration != "" {
		query += fmt.Sprintf(`[%s]`, rangeDuration)
	}
	return query
}

// podLabelMatcher returns a pod or pod=~ label matcher for PromQL selectors.
func podLabelMatcher(pods []string) string {
	if len(pods) == 1 {
		return fmt.Sprintf(`pod="%s"`, pods[0])
	}
	escaped := make([]string, len(pods))
	for i, p := range pods {
		escaped[i] = regexp.QuoteMeta(p)
	}
	return fmt.Sprintf(`pod=~"%s"`, strings.Join(escaped, "|"))
}

// increaseOrNew is window growth, or the current counter when the series did not exist one window ago.
// increase() misses the first increment of a new counter (first scrape is already N).
func increaseOrNew(selector, rangeDuration string) (string, error) {
	if rangeDuration == "" {
		return "", fmt.Errorf("rangeDuration is required")
	}
	inc := fmt.Sprintf("increase(%s[%s])", selector, rangeDuration)
	return fmt.Sprintf("(%s > 0) or (%s unless %s offset %s)", inc, selector, selector, rangeDuration), nil
}
