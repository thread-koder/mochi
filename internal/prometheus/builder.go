package prometheus

import (
	"fmt"
)

// BuildContainerMetricQuery builds a container metric selector used by cAdvisor queries.
// It always excludes the synthetic POD container and empty container labels.
func BuildContainerMetricQuery(metric, namespace, pod, container, rangeDuration string) string {
	query := fmt.Sprintf(`%s{container!="POD",container!=""`, metric)
	if namespace != "" {
		query += fmt.Sprintf(`,namespace="%s"`, namespace)
	}
	if pod != "" {
		query += fmt.Sprintf(`,pod="%s"`, pod)
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
