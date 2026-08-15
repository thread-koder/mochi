package dependency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

type AnalyzeOptions struct {
	TimeRange       time.Duration // Soft active window on last_seen_at (default: 7d).
	IncludeExternal bool          // Whether to include external nodes (default: true).
}

func DefaultAnalyzeOptions() AnalyzeOptions {
	return AnalyzeOptions{
		TimeRange:       7 * 24 * time.Hour,
		IncludeExternal: true,
	}
}

type NodeDTO struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type EdgeDTO struct {
	ID                  string    `json:"id"`
	FromNodeID          string    `json:"from_node_id"`
	ToNodeID            string    `json:"to_node_id"`
	Protocol            string    `json:"protocol"`
	Port                int       `json:"port"`
	ViaServiceNamespace *string   `json:"via_service_namespace"`
	ViaServiceName      *string   `json:"via_service_name"`
	Source              string    `json:"source"`
	Connects            float64   `json:"connects"`
	TxBytes             float64   `json:"tx_bytes"`
	RxBytes             float64   `json:"rx_bytes"`
	ActiveConnections   float64   `json:"active_connections"`
	FirstSeenAt         time.Time `json:"first_seen_at"`
	LastSeenAt          time.Time `json:"last_seen_at"`
}

type Graph struct {
	Nodes []NodeDTO `json:"nodes"`
	Edges []EdgeDTO `json:"edges"`
}

type WorkloadRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type WorkloadAnalysis struct {
	Workload   WorkloadRef `json:"workload"`
	Upstream   Graph       `json:"upstream"`
	Downstream Graph       `json:"downstream"`
}

type NamespaceAnalysis struct {
	Namespace string    `json:"namespace"`
	Nodes     []NodeDTO `json:"nodes"`
	Edges     []EdgeDTO `json:"edges"`
}

func AnalyzeWorkload(ctx context.Context, workloadType, name, namespace string, opts AnalyzeOptions) (*WorkloadAnalysis, error) {
	since := time.Now().UTC().Add(-opts.TimeRange)
	workload := WorkloadRef{Kind: workloadType, Namespace: namespace, Name: name}
	empty := Graph{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}

	center, err := database.GetDependencyNodeByKey(ctx, workloadType, namespace, name)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			return &WorkloadAnalysis{
				Workload:   workload,
				Upstream:   empty,
				Downstream: empty,
			}, nil
		}
		return nil, fmt.Errorf("get dependency node for workload %s/%s/%s: %w", workloadType, namespace, name, err)
	}

	downstreamEdges, err := database.GetDependencyEdgesByFromNode(ctx, center.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get downstream dependency edges: %w", err)
	}
	upstreamEdges, err := database.GetDependencyEdgesByToNode(ctx, center.ID, since)
	if err != nil {
		return nil, fmt.Errorf("get upstream dependency edges: %w", err)
	}

	downstreamGraph, err := buildGraph(ctx, downstreamEdges, opts.IncludeExternal, center)
	if err != nil {
		return nil, err
	}
	upstreamGraph, err := buildGraph(ctx, upstreamEdges, opts.IncludeExternal, center)
	if err != nil {
		return nil, err
	}

	return &WorkloadAnalysis{
		Workload:   workload,
		Upstream:   upstreamGraph,
		Downstream: downstreamGraph,
	}, nil
}

func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalyzeOptions) (*NamespaceAnalysis, error) {
	since := time.Now().UTC().Add(-opts.TimeRange)

	edges, err := database.GetDependencyEdgesForNamespace(ctx, namespace, since)
	if err != nil {
		return nil, fmt.Errorf("get dependency edges for namespace %s: %w", namespace, err)
	}

	graph, err := buildGraph(ctx, edges, opts.IncludeExternal, nil)
	if err != nil {
		return nil, err
	}

	return &NamespaceAnalysis{
		Namespace: namespace,
		Nodes:     graph.Nodes,
		Edges:     graph.Edges,
	}, nil
}

func buildGraph(ctx context.Context, edges []*database.DependencyEdge, includeExternal bool, center *database.DependencyNode) (Graph, error) {
	if len(edges) == 0 {
		return Graph{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
	}

	idSet := make(map[uuid.UUID]struct{}, len(edges)*2)
	for _, e := range edges {
		idSet[e.FromNodeID] = struct{}{}
		idSet[e.ToNodeID] = struct{}{}
	}

	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	nodes, err := database.GetDependencyNodesByIDs(ctx, ids)
	if err != nil {
		return Graph{}, fmt.Errorf("get dependency nodes by ids: %w", err)
	}

	nodesByID := make(map[uuid.UUID]*database.DependencyNode, len(nodes))
	for _, n := range nodes {
		if !includeExternal && n.Kind == KindExternal {
			continue
		}
		nodesByID[n.ID] = n
	}

	filteredEdges := make([]EdgeDTO, 0, len(edges))
	usedNodes := make(map[uuid.UUID]struct{})
	for _, e := range edges {
		if _, ok := nodesByID[e.FromNodeID]; !ok {
			continue
		}
		if _, ok := nodesByID[e.ToNodeID]; !ok {
			continue
		}
		filteredEdges = append(filteredEdges, toEdgeDTO(e))
		usedNodes[e.FromNodeID] = struct{}{}
		usedNodes[e.ToNodeID] = struct{}{}
	}

	if center != nil {
		if _, ok := nodesByID[center.ID]; ok {
			usedNodes[center.ID] = struct{}{}
		}
	}

	nodeDTOs := make([]NodeDTO, 0, len(usedNodes))
	for id := range usedNodes {
		n, ok := nodesByID[id]
		if !ok {
			continue
		}
		nodeDTOs = append(nodeDTOs, toNodeDTO(n))
	}

	return Graph{Nodes: nodeDTOs, Edges: filteredEdges}, nil
}

func toNodeDTO(n *database.DependencyNode) NodeDTO {
	return NodeDTO{
		ID:          n.ID.String(),
		Kind:        n.Kind,
		Namespace:   n.Namespace,
		Name:        n.Name,
		FirstSeenAt: n.FirstSeenAt,
		LastSeenAt:  n.LastSeenAt,
	}
}

func toEdgeDTO(e *database.DependencyEdge) EdgeDTO {
	return EdgeDTO{
		ID:                  e.ID.String(),
		FromNodeID:          e.FromNodeID.String(),
		ToNodeID:            e.ToNodeID.String(),
		Protocol:            e.Protocol,
		Port:                e.Port,
		ViaServiceNamespace: e.ViaServiceNamespace,
		ViaServiceName:      e.ViaServiceName,
		Source:              e.Source,
		Connects:            e.Connects,
		TxBytes:             e.TxBytes,
		RxBytes:             e.RxBytes,
		ActiveConnections:   e.ActiveConnections,
		FirstSeenAt:         e.FirstSeenAt,
		LastSeenAt:          e.LastSeenAt,
	}
}
