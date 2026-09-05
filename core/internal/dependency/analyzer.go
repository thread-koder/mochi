package dependency

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thread_koder/mochi/core/internal/apperrors"
	"github.com/thread_koder/mochi/core/internal/database"
)

type AnalysisOptions struct {
	TimeRange       time.Duration // Soft active window on last_seen_at (default: 7d).
	IncludeExternal bool          // Whether to include external nodes (default: true).
	IncludeDNS      bool          // Whether to include CoreDNS / kube-dns nodes (default: false).
	IncludeUnknown  bool          // Whether to include unresolved cluster leftovers (default: true).
}

func DefaultAnalysisOptions() AnalysisOptions {
	return AnalysisOptions{
		TimeRange:       7 * 24 * time.Hour,
		IncludeExternal: true,
		IncludeDNS:      false,
		IncludeUnknown:  true,
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
	ViaServicePort      *int      `json:"via_service_port"`
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

func AnalyzeWorkload(ctx context.Context, workloadType, name, namespace string, opts AnalysisOptions) (WorkloadAnalysis, error) {
	since := time.Now().UTC().Add(-opts.TimeRange)
	workload := WorkloadRef{Kind: workloadType, Namespace: namespace, Name: name}
	empty := Graph{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}

	center, err := database.GetDependencyNode(ctx, workloadType, namespace, name)
	if err != nil {
		if errors.Is(err, &apperrors.NotFoundError{}) {
			return WorkloadAnalysis{
				Workload:   workload,
				Upstream:   empty,
				Downstream: empty,
			}, nil
		}
		return WorkloadAnalysis{}, fmt.Errorf("get dependency node for workload %s/%s/%s: %w", workloadType, namespace, name, err)
	}

	downstreamEdges, err := database.GetDependencyEdgesByFromNode(ctx, center.ID, since)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("get downstream dependency edges: %w", err)
	}
	upstreamEdges, err := database.GetDependencyEdgesByToNode(ctx, center.ID, since)
	if err != nil {
		return WorkloadAnalysis{}, fmt.Errorf("get upstream dependency edges: %w", err)
	}

	nodesByID, err := loadGraphNodes(ctx, append(downstreamEdges, upstreamEdges...))
	if err != nil {
		return WorkloadAnalysis{}, err
	}

	return WorkloadAnalysis{
		Workload:   workload,
		Upstream:   assembleGraph(upstreamEdges, nodesByID, opts, center),
		Downstream: assembleGraph(downstreamEdges, nodesByID, opts, center),
	}, nil
}

func AnalyzeNamespace(ctx context.Context, namespace string, opts AnalysisOptions) (NamespaceAnalysis, error) {
	since := time.Now().UTC().Add(-opts.TimeRange)

	edges, err := database.GetDependencyEdgesForNamespace(ctx, namespace, since)
	if err != nil {
		return NamespaceAnalysis{}, fmt.Errorf("get dependency edges for namespace %s: %w", namespace, err)
	}

	nodesByID, err := loadGraphNodes(ctx, edges)
	if err != nil {
		return NamespaceAnalysis{}, err
	}
	graph := assembleGraph(edges, nodesByID, opts, nil)

	return NamespaceAnalysis{
		Namespace: namespace,
		Nodes:     graph.Nodes,
		Edges:     graph.Edges,
	}, nil
}

func loadGraphNodes(ctx context.Context, edges []*database.DependencyEdge) (map[uuid.UUID]*database.DependencyNode, error) {
	if len(edges) == 0 {
		return map[uuid.UUID]*database.DependencyNode{}, nil
	}

	idSet := make(map[uuid.UUID]struct{}, len(edges)*2)
	for _, edge := range edges {
		idSet[edge.FromNodeID] = struct{}{}
		idSet[edge.ToNodeID] = struct{}{}
	}

	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	nodes, err := database.GetDependencyNodesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get dependency nodes by ids: %w", err)
	}

	nodesByID := make(map[uuid.UUID]*database.DependencyNode, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}
	return nodesByID, nil
}

func assembleGraph(edges []*database.DependencyEdge, allNodes map[uuid.UUID]*database.DependencyNode, opts AnalysisOptions, center *database.DependencyNode) Graph {
	if len(edges) == 0 {
		nodes := []NodeDTO{}
		if center != nil {
			nodes = []NodeDTO{toNodeDTO(center)}
		}
		return Graph{Nodes: nodes, Edges: []EdgeDTO{}}
	}

	nodesByID := make(map[uuid.UUID]*database.DependencyNode, len(allNodes))
	for id, node := range allNodes {
		if !opts.IncludeExternal && node.Kind == KindExternal {
			continue
		}
		if !opts.IncludeUnknown && node.Kind == KindUnknown {
			continue
		}
		if !opts.IncludeDNS && isDNSNoise(node.Namespace, node.Name) {
			continue
		}
		nodesByID[id] = node
	}

	filteredEdges := make([]EdgeDTO, 0, len(edges))
	usedNodes := make(map[uuid.UUID]struct{})
	for _, edge := range edges {
		if _, ok := nodesByID[edge.FromNodeID]; !ok {
			continue
		}
		if _, ok := nodesByID[edge.ToNodeID]; !ok {
			continue
		}
		filteredEdges = append(filteredEdges, toEdgeDTO(edge))
		usedNodes[edge.FromNodeID] = struct{}{}
		usedNodes[edge.ToNodeID] = struct{}{}
	}

	if center != nil {
		if _, ok := nodesByID[center.ID]; ok {
			usedNodes[center.ID] = struct{}{}
		}
	}

	nodeDTOs := make([]NodeDTO, 0, len(usedNodes))
	for id := range usedNodes {
		node, ok := nodesByID[id]
		if !ok {
			continue
		}
		nodeDTOs = append(nodeDTOs, toNodeDTO(node))
	}

	return Graph{Nodes: nodeDTOs, Edges: filteredEdges}
}

func isDNSNoise(namespace, name string) bool {
	if !strings.EqualFold(namespace, "kube-system") {
		return false
	}
	name = strings.ToLower(name)
	return strings.Contains(name, "coredns") || strings.Contains(name, "kube-dns")
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
		ViaServicePort:      e.ViaServicePort,
		Source:              e.Source,
		Connects:            e.Connects,
		TxBytes:             e.TxBytes,
		RxBytes:             e.RxBytes,
		ActiveConnections:   e.ActiveConnections,
		FirstSeenAt:         e.FirstSeenAt,
		LastSeenAt:          e.LastSeenAt,
	}
}
