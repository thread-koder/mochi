package database

import (
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
