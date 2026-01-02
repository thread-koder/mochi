package database

import (
	"encoding/json"
	"time"
)

// Represents a resource optimization recommendation for a workload
type Recommendation struct {
	ID               int64      `json:"id" db:"id"`
	WorkloadType     string     `json:"workload_type" db:"workload_type"` // deployment, statefulset, daemonset, job, cronjob, pod
	WorkloadName     string     `json:"workload_name" db:"workload_name"`
	Namespace        string     `json:"namespace" db:"namespace"`
	ResourceType     string     `json:"resource_type" db:"resource_type"` // cpu, memory
	CurrentValue     string     `json:"current_value" db:"current_value"`
	RecommendedValue string     `json:"recommended_value" db:"recommended_value"`
	Confidence       float64    `json:"confidence" db:"confidence"` // 0.0 to 1.0
	Reason           string     `json:"reason" db:"reason"`
	Status           string     `json:"status" db:"status"` // pending, applied, rejected, expired
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	AppliedAt        *time.Time `json:"applied_at,omitempty" db:"applied_at"`
}

// Represents cost data for a namespace, team, or service
type Cost struct {
	ID          int64     `json:"id" db:"id"`
	Namespace   string    `json:"namespace" db:"namespace"`
	Team        string    `json:"team" db:"team"`
	Service     string    `json:"service" db:"service"`
	CostType    string    `json:"cost_type" db:"cost_type"` // compute, storage, network
	Amount      float64   `json:"amount" db:"amount"`
	Currency    string    `json:"currency" db:"currency"` // USD, EUR, etc.
	Period      string    `json:"period" db:"period"`     // daily, weekly, monthly
	PeriodStart time.Time `json:"period_start" db:"period_start"`
	PeriodEnd   time.Time `json:"period_end" db:"period_end"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Represents historical data for recommendations, costs, or resource changes
type History struct {
	ID         int64     `json:"id" db:"id"`
	EntityType string    `json:"entity_type" db:"entity_type"` // recommendation, cost, resource_change
	EntityID   int64     `json:"entity_id" db:"entity_id"`
	Action     string    `json:"action" db:"action"`         // created, updated, deleted, applied
	Details    string    `json:"details" db:"details"`       // JSON string with additional details
	CreatedBy  string    `json:"created_by" db:"created_by"` // user or system
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Represents pod metadata
type Pod struct {
	ID            int64           `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Namespace     string          `json:"namespace" db:"namespace"`
	UID           string          `json:"uid" db:"uid"`
	NodeName      *string         `json:"node_name,omitempty" db:"node_name"`
	Phase         string          `json:"phase" db:"phase"`
	RestartPolicy *string         `json:"restart_policy,omitempty" db:"restart_policy"`
	CPURequest    *string         `json:"cpu_request,omitempty" db:"cpu_request"`
	CPULimit      *string         `json:"cpu_limit,omitempty" db:"cpu_limit"`
	MemoryRequest *string         `json:"memory_request,omitempty" db:"memory_request"`
	MemoryLimit   *string         `json:"memory_limit,omitempty" db:"memory_limit"`
	Labels        json.RawMessage `json:"labels" db:"labels"`
	Annotations   json.RawMessage `json:"annotations" db:"annotations"`
	OwnerKind     *string         `json:"owner_kind,omitempty" db:"owner_kind"`
	OwnerName     *string         `json:"owner_name,omitempty" db:"owner_name"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	SyncedAt      time.Time       `json:"synced_at" db:"synced_at"`
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
