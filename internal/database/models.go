package database

import (
	"encoding/json"
	"time"
)

// Represents pod metadata
type Pod struct {
	ID            int64           `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Namespace     string          `json:"namespace" db:"namespace"`
	UID           string          `json:"uid" db:"uid"`
	NodeName      *string         `json:"node_name,omitempty" db:"node_name"`
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

// Represents container metadata
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

// Represents node metadata
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

// Represents namespace metadata
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

// Represents deployment metadata
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

// Represents statefulset metadata
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

// Represents daemonset metadata
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

// Represents service metadata
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

// Represents endpoint metadata
type Endpoint struct {
	ID          int64           `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Namespace   string          `json:"namespace" db:"namespace"`
	UID         string          `json:"uid" db:"uid"`
	Addresses   json.RawMessage `json:"addresses" db:"addresses"`
	Ports       json.RawMessage `json:"ports" db:"ports"`
	Labels      json.RawMessage `json:"labels" db:"labels"`
	Annotations json.RawMessage `json:"annotations" db:"annotations"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt    time.Time       `json:"synced_at" db:"synced_at"`
}

// Represents a resource recommendation for a container
type ContainerRecommendation struct {
	ID                       int64      `json:"id" db:"id"`
	ContainerID              int64      `json:"container_id" db:"container_id"`
	PodUID                   string     `json:"pod_uid" db:"pod_uid"`
	ContainerName            string     `json:"container_name" db:"container_name"`
	Namespace                string     `json:"namespace" db:"namespace"`
	CurrentCPURequest        *string    `json:"current_cpu_request,omitempty" db:"current_cpu_request"`
	CurrentCPULimit          *string    `json:"current_cpu_limit,omitempty" db:"current_cpu_limit"`
	CurrentMemoryRequest     *string    `json:"current_memory_request,omitempty" db:"current_memory_request"`
	CurrentMemoryLimit       *string    `json:"current_memory_limit,omitempty" db:"current_memory_limit"`
	RecommendedCPURequest    *string    `json:"recommended_cpu_request,omitempty" db:"recommended_cpu_request"`
	RecommendedCPULimit      *string    `json:"recommended_cpu_limit,omitempty" db:"recommended_cpu_limit"`
	RecommendedMemoryRequest *string    `json:"recommended_memory_request,omitempty" db:"recommended_memory_request"`
	RecommendedMemoryLimit   *string    `json:"recommended_memory_limit,omitempty" db:"recommended_memory_limit"`
	RecommendationMode       string     `json:"recommendation_mode" db:"recommendation_mode"` // "burstable" or "guaranteed"
	ConfidenceScore          float64    `json:"confidence_score" db:"confidence_score"`
	Status                   string     `json:"status" db:"status"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at" db:"updated_at"`
	AppliedAt                *time.Time `json:"applied_at,omitempty" db:"applied_at"`
}
