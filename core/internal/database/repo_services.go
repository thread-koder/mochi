package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/thread_koder/mochi/core/internal/apperrors"
)

// ServiceNodePort is a Service spec port's allocated NodePort.
type ServiceNodePort struct {
	Protocol string
	Port     int
}

// ServiceUpsert is a Service row plus address indexes rebuilt on sync.
type ServiceUpsert struct {
	Service   *Service
	VIPs      []string
	NodePorts []ServiceNodePort
}

const serviceSelectColumns = `
	s.id, s.name, s.namespace, s.uid, s.type, s.cluster_ip, s.ports, s.selector,
	s.labels, s.annotations, s.created_at, s.updated_at, s.synced_at
`

func UpsertServicesBatch(ctx context.Context, services []*ServiceUpsert) error {
	if len(services) == 0 {
		return nil
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	upsertQuery := `
		INSERT INTO services (
			name, namespace, uid, type, cluster_ip, ports, selector,
			labels, annotations, created_at, synced_at
		) VALUES (
			@name, @namespace, @uid, @type, @cluster_ip, @ports, @selector,
			@labels, @annotations, @created_at, @synced_at
		)
		ON CONFLICT (namespace, name) DO UPDATE SET
			uid = EXCLUDED.uid,
			type = EXCLUDED.type,
			cluster_ip = EXCLUDED.cluster_ip,
			ports = EXCLUDED.ports,
			selector = EXCLUDED.selector,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			synced_at = EXCLUDED.synced_at
	`

	batch := &pgx.Batch{}
	for _, upsert := range services {
		service := upsert.Service
		batch.Queue(upsertQuery, pgx.StrictNamedArgs{
			"name":        service.Name,
			"namespace":   service.Namespace,
			"uid":         service.UID,
			"type":        service.Type,
			"cluster_ip":  service.ClusterIP,
			"ports":       service.Ports,
			"selector":    service.Selector,
			"labels":      service.Labels,
			"annotations": service.Annotations,
			"created_at":  service.CreatedAt,
			"synced_at":   service.SyncedAt,
		})
		queueServiceAddressRebuild(batch, service.Namespace, service.Name, upsert.VIPs, upsert.NodePorts)
	}

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("failed to execute batch upsert for services: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func queueServiceAddressRebuild(batch *pgx.Batch, namespace, name string, vips []string, nodePorts []ServiceNodePort) {
	batch.Queue(
		`DELETE FROM service_ips WHERE namespace = @namespace AND name = @name`,
		pgx.StrictNamedArgs{"namespace": namespace, "name": name},
	)
	batch.Queue(
		`DELETE FROM service_node_ports WHERE namespace = @namespace AND name = @name`,
		pgx.StrictNamedArgs{"namespace": namespace, "name": name},
	)
	if len(vips) > 0 {
		batch.Queue(
			`INSERT INTO service_ips (ip, namespace, name)
			 SELECT unnest(@ips::text[]), @namespace, @name`,
			pgx.StrictNamedArgs{"ips": vips, "namespace": namespace, "name": name},
		)
	}
	if len(nodePorts) > 0 {
		protocols := make([]string, len(nodePorts))
		ports := make([]int32, len(nodePorts))
		for i, np := range nodePorts {
			protocols[i] = np.Protocol
			ports[i] = int32(np.Port)
		}
		batch.Queue(
			`INSERT INTO service_node_ports (protocol, port, namespace, name)
			 SELECT p.proto, p.port, @namespace, @name
			 FROM unnest(@protocols::text[], @ports::int[]) AS p(proto, port)`,
			pgx.StrictNamedArgs{
				"protocols": protocols,
				"ports":     ports,
				"namespace": namespace,
				"name":      name,
			},
		)
	}
}

func collectServices(rows pgx.Rows) ([]*Service, error) {
	var services []*Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Namespace, &s.UID, &s.Type, &s.ClusterIP, &s.Ports, &s.Selector,
			&s.Labels, &s.Annotations, &s.CreatedAt, &s.UpdatedAt, &s.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate services: %w", err)
	}
	return services, nil
}

func GetServiceByIP(ctx context.Context, ip string) (*Service, error) {
	if ip == "" {
		return nil, apperrors.NewNotFound("service", ip)
	}

	query := `
		SELECT ` + serviceSelectColumns + `
		FROM services s
		JOIN service_ips si ON si.namespace = s.namespace AND si.name = s.name
		WHERE si.ip = @ip
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{"ip": ip})
	if err != nil {
		return nil, fmt.Errorf("failed to query service by IP: %w", err)
	}
	defer rows.Close()

	services, err := collectServices(rows)
	if err != nil {
		return nil, err
	}
	if len(services) != 1 {
		return nil, apperrors.NewNotFound("service", ip)
	}
	return services[0], nil
}

func GetServiceByNodePort(ctx context.Context, protocol string, port int) (*Service, error) {
	protocol = strings.ToLower(protocol)
	if protocol == "" || port <= 0 {
		return nil, apperrors.NewNotFound("service", fmt.Sprintf("%s:%d", protocol, port))
	}

	query := `
		SELECT ` + serviceSelectColumns + `
		FROM services s
		JOIN service_node_ports snp ON snp.namespace = s.namespace AND snp.name = s.name
		WHERE snp.protocol = @protocol AND snp.port = @port
	`

	rows, err := Pool.Query(ctx, query, pgx.StrictNamedArgs{
		"protocol": protocol,
		"port":     port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query service by node port: %w", err)
	}
	defer rows.Close()

	services, err := collectServices(rows)
	if err != nil {
		return nil, err
	}
	if len(services) != 1 {
		return nil, apperrors.NewNotFound("service", fmt.Sprintf("%s:%d", protocol, port))
	}
	return services[0], nil
}

// PruneServices deletes services not listed in uids.
// Empty uids deletes every service in the namespace.
func PruneServices(ctx context.Context, namespace string, uids []string) error {
	var err error
	if len(uids) == 0 {
		_, err = Pool.Exec(ctx,
			`DELETE FROM services WHERE namespace = @namespace`,
			pgx.StrictNamedArgs{"namespace": namespace},
		)
	} else {
		_, err = Pool.Exec(ctx,
			`DELETE FROM services WHERE namespace = @namespace AND NOT (uid = ANY(@uids::text[]))`,
			pgx.StrictNamedArgs{
				"namespace": namespace,
				"uids":      uids,
			},
		)
	}
	if err != nil {
		return fmt.Errorf("failed to prune services: %w", err)
	}
	return nil
}
