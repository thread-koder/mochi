package prometheus

import (
	"fmt"
)

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

func BuildNamespaceDiskReadBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceDiskWriteBytesQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_bytes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceDiskReadOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_reads_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}

func BuildNamespaceDiskWriteOpsQuery(namespace string, rangeDuration string) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if rangeDuration == "" {
		rangeDuration = "5m"
	}

	return fmt.Sprintf(`sum(rate(container_fs_writes_total{container!="POD",container!="",namespace="%s"}[%s]))`, namespace, rangeDuration), nil
}
