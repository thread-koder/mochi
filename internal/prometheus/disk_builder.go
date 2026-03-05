package prometheus

import (
	"fmt"
)

// Builds a PromQL query for pod disk read bytes (bytes/sec)
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

// Builds a PromQL query for pod disk write bytes (bytes/sec)
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

// Builds a PromQL query for pod disk read operations (IOPS)
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

// Builds a PromQL query for pod disk write operations (IOPS)
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

// Builds a PromQL query for namespace disk read bytes (bytes/sec)
func BuildNamespaceDiskReadBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace disk write bytes (bytes/sec)
func BuildNamespaceDiskWriteBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace disk read operations (IOPS)
func BuildNamespaceDiskReadOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

// Builds a PromQL query for namespace disk write operations (IOPS)
func BuildNamespaceDiskWriteOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
