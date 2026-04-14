package database

import (
	"encoding/json"
	"time"
)

// Node represents a Kubernetes Node metadata.
type Node struct {
	ID                      int64           `json:"id" db:"id"`
	Name                    string          `json:"name" db:"name"`
	UID                     string          `json:"uid" db:"uid"`
	InternalIP              *string         `json:"internal_ip,omitempty" db:"internal_ip"`
	ExternalIP              *string         `json:"external_ip,omitempty" db:"external_ip"`
	OSImage                 *string         `json:"os_image,omitempty" db:"os_image"`
	KernelVersion           *string         `json:"kernel_version,omitempty" db:"kernel_version"`
	ContainerRuntimeVersion *string         `json:"container_runtime_version,omitempty" db:"container_runtime_version"`
	KubeletVersion          *string         `json:"kubelet_version,omitempty" db:"kubelet_version"`
	CPUCapacity             *string         `json:"cpu_capacity,omitempty" db:"cpu_capacity"`
	MemoryCapacity          *string         `json:"memory_capacity,omitempty" db:"memory_capacity"`
	CPUAllocatable          *string         `json:"cpu_allocatable,omitempty" db:"cpu_allocatable"`
	MemoryAllocatable       *string         `json:"memory_allocatable,omitempty" db:"memory_allocatable"`
	Labels                  json.RawMessage `json:"labels" db:"labels"`
	Annotations             json.RawMessage `json:"annotations" db:"annotations"`
	Conditions              json.RawMessage `json:"conditions" db:"conditions"`
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt                time.Time       `json:"synced_at" db:"synced_at"`
}

// Namespace represents a Kubernetes Namespace metadata.
type Namespace struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	UID         string          `json:"uid" db:"uid"`
	Phase       string          `json:"phase" db:"phase"`
	Labels      json.RawMessage `json:"labels" db:"labels"`
	Annotations json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt    time.Time       `json:"synced_at" db:"synced_at"`
}

// Deployment represents a Kubernetes Deployment metadata.
type Deployment struct {
	ID                int64           `json:"id" db:"id"`
	Name              string          `json:"name" db:"name"`
	Namespace         string          `json:"namespace" db:"namespace"`
	UID               string          `json:"uid" db:"uid"`
	Replicas          int             `json:"replicas" db:"replicas"`
	ReadyReplicas     int             `json:"ready_replicas" db:"ready_replicas"`
	AvailableReplicas int             `json:"available_replicas" db:"available_replicas"`
	Labels            json.RawMessage `json:"labels" db:"labels"`
	Annotations       json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt          time.Time       `json:"synced_at" db:"synced_at"`
}

// ReplicaSet represents a Kubernetes ReplicaSet metadata.
type ReplicaSet struct {
	ID            int64           `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Namespace     string          `json:"namespace" db:"namespace"`
	UID           string          `json:"uid" db:"uid"`
	Replicas      int             `json:"replicas" db:"replicas"`
	ReadyReplicas int             `json:"ready_replicas" db:"ready_replicas"`
	OwnerKind     *string         `json:"owner_kind" db:"owner_kind"`
	OwnerName     *string         `json:"owner_name" db:"owner_name"`
	Labels        json.RawMessage `json:"labels" db:"labels"`
	Annotations   json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt      time.Time       `json:"synced_at" db:"synced_at"`
}

// StatefulSet represents a Kubernetes StatefulSet metadata.
type StatefulSet struct {
	ID            int64           `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Namespace     string          `json:"namespace" db:"namespace"`
	UID           string          `json:"uid" db:"uid"`
	Replicas      int             `json:"replicas" db:"replicas"`
	ReadyReplicas int             `json:"ready_replicas" db:"ready_replicas"`
	Labels        json.RawMessage `json:"labels" db:"labels"`
	Annotations   json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt      time.Time       `json:"synced_at" db:"synced_at"`
}

// DaemonSet represents a Kubernetes DaemonSet metadata.
type DaemonSet struct {
	ID                     int64           `json:"id" db:"id"`
	Name                   string          `json:"name" db:"name"`
	Namespace              string          `json:"namespace" db:"namespace"`
	UID                    string          `json:"uid" db:"uid"`
	DesiredNumberScheduled int             `json:"desired_number_scheduled" db:"desired_number_scheduled"`
	NumberReady            int             `json:"number_ready" db:"number_ready"`
	NumberAvailable        int             `json:"number_available" db:"number_available"`
	Labels                 json.RawMessage `json:"labels" db:"labels"`
	Annotations            json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt               time.Time       `json:"synced_at" db:"synced_at"`
}

// Service represents a Kubernetes Service metadata.
type Service struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Namespace   string          `json:"namespace" db:"namespace"`
	UID         string          `json:"uid" db:"uid"`
	Type        string          `json:"type" db:"type"`
	ClusterIP   *string         `json:"cluster_ip,omitempty" db:"cluster_ip"`
	Ports       json.RawMessage `json:"ports" db:"ports"`
	Selector    json.RawMessage `json:"selector" db:"selector"`
	Labels      json.RawMessage `json:"labels" db:"labels"`
	Annotations json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt    time.Time       `json:"synced_at" db:"synced_at"`
}

// EndpointSlice represents a Kubernetes EndpointSlice metadata.
type EndpointSlice struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Namespace   string          `json:"namespace" db:"namespace"`
	UID         string          `json:"uid" db:"uid"`
	AddressType string          `json:"address_type" db:"address_type"`
	OwnerKind   *string         `json:"owner_kind" db:"owner_kind"`
	OwnerName   *string         `json:"owner_name" db:"owner_name"`
	Endpoints   json.RawMessage `json:"endpoints" db:"endpoints"`
	Ports       json.RawMessage `json:"ports" db:"ports"`
	Labels      json.RawMessage `json:"labels" db:"labels"`
	Annotations json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt    time.Time       `json:"synced_at" db:"synced_at"`
}

// Pod represents a Kubernetes Pod metadata.
type Pod struct {
	ID            int64           `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Namespace     string          `json:"namespace" db:"namespace"`
	UID           string          `json:"uid" db:"uid"`
	Node          *string         `json:"node,omitempty" db:"node"`
	Phase         string          `json:"phase" db:"phase"`
	RestartPolicy *string         `json:"restart_policy,omitempty" db:"restart_policy"`
	Labels        json.RawMessage `json:"labels" db:"labels"`
	Annotations   json.RawMessage `json:"annotations" db:"annotations"`
	OwnerKind     *string         `json:"owner_kind,omitempty" db:"owner_kind"`
	OwnerName     *string         `json:"owner_name,omitempty" db:"owner_name"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt      time.Time       `json:"synced_at" db:"synced_at"`
}

// Container represents a Kubernetes Container metadata.
type Container struct {
	ID              int64           `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	PodUID          string          `json:"pod_uid" db:"pod_uid"`
	PodName         string          `json:"pod_name" db:"pod_name"`
	Namespace       string          `json:"namespace" db:"namespace"`
	Image           string          `json:"image" db:"image"`
	ImagePullPolicy *string         `json:"image_pull_policy,omitempty" db:"image_pull_policy"`
	Ports           json.RawMessage `json:"ports" db:"ports"`
	CPURequest      *string         `json:"cpu_request,omitempty" db:"cpu_request"`
	CPULimit        *string         `json:"cpu_limit,omitempty" db:"cpu_limit"`
	MemoryRequest   *string         `json:"memory_request,omitempty" db:"memory_request"`
	MemoryLimit     *string         `json:"memory_limit,omitempty" db:"memory_limit"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt        time.Time       `json:"synced_at" db:"synced_at"`
}

// ComputeRecommendation represents a compute recommendation (CPU/memory) for a workload.
type ComputeRecommendation struct {
	ID                 int64           `json:"id" db:"id"`
	WorkloadType       string          `json:"workload_type" db:"workload_type"`
	WorkloadName       string          `json:"workload_name" db:"workload_name"`
	Namespace          string          `json:"namespace" db:"namespace"`
	RecommendationMode string          `json:"recommendation_mode" db:"recommendation_mode"`
	Recommendations    json.RawMessage `json:"recommendations" db:"recommendations"`
	Status             string          `json:"status" db:"status"`
	AnalysisTimeRange  string          `json:"analysis_time_range,omitempty" db:"analysis_time_range"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
	GeneratedAt        time.Time       `json:"generated_at" db:"generated_at"`
}
