package database

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID                      uuid.UUID       `db:"id"`
	Name                    string          `db:"name"`
	UID                     string          `db:"uid"`
	InternalIP              *string         `db:"internal_ip"`
	ExternalIP              *string         `db:"external_ip"`
	OSImage                 string          `db:"os_image"`
	KernelVersion           string          `db:"kernel_version"`
	ContainerRuntimeVersion string          `db:"container_runtime_version"`
	KubeletVersion          string          `db:"kubelet_version"`
	CPUCapacity             string          `db:"cpu_capacity"`
	MemoryCapacity          string          `db:"memory_capacity"`
	CPUAllocatable          string          `db:"cpu_allocatable"`
	MemoryAllocatable       string          `db:"memory_allocatable"`
	Labels                  json.RawMessage `db:"labels"`
	Annotations             json.RawMessage `db:"annotations"`
	Conditions              json.RawMessage `db:"conditions"`
	CreatedAt               time.Time       `db:"created_at"`
	UpdatedAt               time.Time       `db:"updated_at"`
	SyncedAt                time.Time       `db:"synced_at"`
}

type Namespace struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	UID         string          `db:"uid"`
	Phase       string          `db:"phase"`
	Labels      json.RawMessage `db:"labels"`
	Annotations json.RawMessage `db:"annotations"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	SyncedAt    time.Time       `db:"synced_at"`
}

type Deployment struct {
	ID                uuid.UUID       `db:"id"`
	Name              string          `db:"name"`
	Namespace         string          `db:"namespace"`
	UID               string          `db:"uid"`
	Replicas          int             `db:"replicas"`
	ReadyReplicas     int             `db:"ready_replicas"`
	AvailableReplicas int             `db:"available_replicas"`
	Labels            json.RawMessage `db:"labels"`
	Annotations       json.RawMessage `db:"annotations"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
	SyncedAt          time.Time       `db:"synced_at"`
}

type StatefulSet struct {
	ID            uuid.UUID       `db:"id"`
	Name          string          `db:"name"`
	Namespace     string          `db:"namespace"`
	UID           string          `db:"uid"`
	Replicas      int             `db:"replicas"`
	ReadyReplicas int             `db:"ready_replicas"`
	Labels        json.RawMessage `db:"labels"`
	Annotations   json.RawMessage `db:"annotations"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
	SyncedAt      time.Time       `db:"synced_at"`
}

type DaemonSet struct {
	ID                     uuid.UUID       `db:"id"`
	Name                   string          `db:"name"`
	Namespace              string          `db:"namespace"`
	UID                    string          `db:"uid"`
	DesiredNumberScheduled int             `db:"desired_number_scheduled"`
	NumberReady            int             `db:"number_ready"`
	NumberAvailable        int             `db:"number_available"`
	Labels                 json.RawMessage `db:"labels"`
	Annotations            json.RawMessage `db:"annotations"`
	CreatedAt              time.Time       `db:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at"`
	SyncedAt               time.Time       `db:"synced_at"`
}

type Job struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	Namespace   string          `db:"namespace"`
	UID         string          `db:"uid"`
	Active      int             `db:"active"`
	Succeeded   int             `db:"succeeded"`
	Failed      int             `db:"failed"`
	Labels      json.RawMessage `db:"labels"`
	Annotations json.RawMessage `db:"annotations"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	SyncedAt    time.Time       `db:"synced_at"`
}

type CronJob struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	Namespace   string          `db:"namespace"`
	UID         string          `db:"uid"`
	Schedule    string          `db:"schedule"`
	Suspend     bool            `db:"suspend"`
	Labels      json.RawMessage `db:"labels"`
	Annotations json.RawMessage `db:"annotations"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	SyncedAt    time.Time       `db:"synced_at"`
}

type Service struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	Namespace   string          `db:"namespace"`
	UID         string          `db:"uid"`
	Type        string          `db:"type"`
	ClusterIP   *string         `db:"cluster_ip"`
	Ports       json.RawMessage `db:"ports"`
	Selector    json.RawMessage `db:"selector"`
	Labels      json.RawMessage `db:"labels"`
	Annotations json.RawMessage `db:"annotations"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	SyncedAt    time.Time       `db:"synced_at"`
}

type EndpointSlice struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`
	Namespace   string          `db:"namespace"`
	UID         string          `db:"uid"`
	AddressType string          `db:"address_type"`
	OwnerKind   *string         `db:"owner_kind"`
	OwnerName   *string         `db:"owner_name"`
	Endpoints   json.RawMessage `db:"endpoints"`
	Ports       json.RawMessage `db:"ports"`
	Labels      json.RawMessage `db:"labels"`
	Annotations json.RawMessage `db:"annotations"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	SyncedAt    time.Time       `db:"synced_at"`
}

type Pod struct {
	ID            uuid.UUID       `db:"id"`
	Name          string          `db:"name"`
	Namespace     string          `db:"namespace"`
	UID           string          `db:"uid"`
	Node          *string         `db:"node"`
	PodIP         *string         `db:"pod_ip"`
	Phase         string          `db:"phase"`
	RestartPolicy string          `db:"restart_policy"`
	Labels        json.RawMessage `db:"labels"`
	Annotations   json.RawMessage `db:"annotations"`
	WorkloadKind  *string         `db:"workload_kind"`
	WorkloadName  *string         `db:"workload_name"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
	SyncedAt      time.Time       `db:"synced_at"`
}

// PodAttribution is a bounded historical identity for a pod after live prune.
type PodAttribution struct {
	UID          string          `db:"uid"`
	Name         string          `db:"name"`
	Namespace    string          `db:"namespace"`
	WorkloadKind *string         `db:"workload_kind"`
	WorkloadName *string         `db:"workload_name"`
	Phase        string          `db:"phase"`
	Node         *string         `db:"node"`
	Containers   json.RawMessage `db:"containers"`
	FirstSeenAt  time.Time       `db:"first_seen_at"`
	LastSeenAt   time.Time       `db:"last_seen_at"`
	FinishedAt   *time.Time      `db:"finished_at"`
}

// AttributionContainerSpec is one container snapshot stored on a pod attribution.
type AttributionContainerSpec struct {
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	CPURequest    *string `json:"cpu_request,omitempty"`
	CPULimit      *string `json:"cpu_limit,omitempty"`
	MemoryRequest *string `json:"memory_request,omitempty"`
	MemoryLimit   *string `json:"memory_limit,omitempty"`
}

// PodsForAnalysis is the live inventory plus attributed identities for an analysis window.
type PodsForAnalysis struct {
	Live []*Pod
	All  []*Pod
}

type Container struct {
	ID              uuid.UUID       `db:"id"`
	Name            string          `db:"name"`
	PodUID          string          `db:"pod_uid"`
	PodName         string          `db:"pod_name"`
	Namespace       string          `db:"namespace"`
	Image           string          `db:"image"`
	ImagePullPolicy string          `db:"image_pull_policy"`
	Ports           json.RawMessage `db:"ports"`
	CPURequest      *string         `db:"cpu_request"`
	CPULimit        *string         `db:"cpu_limit"`
	MemoryRequest   *string         `db:"memory_request"`
	MemoryLimit     *string         `db:"memory_limit"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
	SyncedAt        time.Time       `db:"synced_at"`
}

// ComputeRecommendation represents a compute recommendation (CPU/memory) for a workload.
type ComputeRecommendation struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	WorkloadType       string          `json:"workload_type" db:"workload_type"`
	WorkloadName       string          `json:"workload_name" db:"workload_name"`
	Namespace          string          `json:"namespace" db:"namespace"`
	RecommendationMode string          `json:"recommendation_mode" db:"recommendation_mode"`
	Recommendations    json.RawMessage `json:"recommendations" db:"recommendations"`
	Status             string          `json:"status" db:"status"`
	AnalysisTimeRange  string          `json:"analysis_time_range" db:"analysis_time_range"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

type DependencyNode struct {
	ID          uuid.UUID       `db:"id"`
	Kind        string          `db:"kind"`
	Namespace   string          `db:"namespace"`
	Name        string          `db:"name"`
	Metadata    json.RawMessage `db:"metadata"`
	FirstSeenAt time.Time       `db:"first_seen_at"`
	LastSeenAt  time.Time       `db:"last_seen_at"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

type DependencyEdge struct {
	ID                  uuid.UUID       `db:"id"`
	FromNodeID          uuid.UUID       `db:"from_node_id"`
	ToNodeID            uuid.UUID       `db:"to_node_id"`
	Protocol            string          `db:"protocol"`
	Port                int             `db:"port"`
	ViaServiceNamespace *string         `db:"via_service_namespace"`
	ViaServiceName      *string         `db:"via_service_name"`
	ViaServicePort      *int            `db:"via_service_port"`
	Source              string          `db:"source"`
	Connects            float64         `db:"connects"`
	TxBytes             float64         `db:"tx_bytes"`
	RxBytes             float64         `db:"rx_bytes"`
	ActiveConnections   float64         `db:"active_connections"`
	FirstSeenAt         time.Time       `db:"first_seen_at"`
	LastSeenAt          time.Time       `db:"last_seen_at"`
	Evidence            json.RawMessage `db:"evidence"`
	Attrs               json.RawMessage `db:"attrs"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}
