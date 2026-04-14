package prometheus

import (
	"fmt"
)

// BuildPodDiskReadBytesQuery builds the pod disk read byte rate query.
func BuildPodDiskReadBytesQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_reads_bytes_total", namespace, pod, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildPodDiskWriteBytesQuery builds the pod disk write byte rate query.
func BuildPodDiskWriteBytesQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_writes_bytes_total", namespace, pod, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildPodDiskReadOpsQuery builds the pod disk read operations rate query.
func BuildPodDiskReadOpsQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_reads_total", namespace, pod, container, rangeDuration)
	return fmt.Sprintf("sum(rate(%s))", base), nil
}

// BuildPodDiskWriteOpsQuery builds the pod disk write operations rate query.
func BuildPodDiskWriteOpsQuery(namespace, pod, container string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	base := BuildContainerMetricQuery("container_fs_writes_total", namespace, pod, container, rangeDuration)
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
