package prometheus

import (
	"fmt"
)

// BuildWorkloadDiskReadBytesQuery builds the workload disk read byte rate query.
func BuildWorkloadDiskReadBytesQuery(namespace string, pods []string, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_reads_bytes_total", namespace, pods, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadDiskWriteBytesQuery builds the workload disk write byte rate query.
func BuildWorkloadDiskWriteBytesQuery(namespace string, pods []string, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_writes_bytes_total", namespace, pods, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadDiskReadOpsQuery builds the workload disk read operations rate query.
func BuildWorkloadDiskReadOpsQuery(namespace string, pods []string, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_reads_total", namespace, pods, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildWorkloadDiskWriteOpsQuery builds the workload disk write operations rate query.
func BuildWorkloadDiskWriteOpsQuery(namespace string, pods []string, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("pods are required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_writes_total", namespace, pods, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildNamespaceDiskReadBytesQuery builds the namespace disk read byte rate query.
func BuildNamespaceDiskReadBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceDiskWriteBytesQuery builds the namespace disk write byte rate query.
func BuildNamespaceDiskWriteBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceDiskReadOpsQuery builds the namespace disk read operations rate query.
func BuildNamespaceDiskReadOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// BuildNamespaceDiskWriteOpsQuery builds the namespace disk write operations rate query.
func BuildNamespaceDiskWriteOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
