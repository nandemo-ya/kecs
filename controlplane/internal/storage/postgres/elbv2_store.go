package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type elbv2Store struct {
	db *sql.DB
}

// Load Balancer operations

// CreateLoadBalancer creates a new load balancer
func (s *elbv2Store) CreateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	if lb.CreatedAt.IsZero() {
		lb.CreatedAt = time.Now()
	}
	lb.UpdatedAt = time.Now()

	// Convert arrays and maps to JSON
	subnetsJSON, _ := json.Marshal(lb.Subnets)
	azsJSON, _ := json.Marshal(lb.AvailabilityZones)
	sgJSON, _ := json.Marshal(lb.SecurityGroups)
	tagsJSON, _ := json.Marshal(lb.Tags)

	query := `
	INSERT INTO elbv2_load_balancers (
		arn, name, dns_name, canonical_hosted_zone_id,
		state, type, scheme, vpc_id, subnets,
		availability_zones, security_groups, ip_address_type,
		tags, region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17
	)`

	_, err := s.db.ExecContext(ctx, query,
		lb.ARN, lb.Name, lb.DNSName, lb.CanonicalHostedZoneID,
		lb.State, lb.Type, lb.Scheme, lb.VpcID,
		string(subnetsJSON), string(azsJSON), string(sgJSON),
		lb.IpAddressType, string(tagsJSON),
		lb.Region, lb.AccountID, lb.CreatedAt, lb.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	return nil
}

// GetLoadBalancer retrieves a load balancer by ARN
func (s *elbv2Store) GetLoadBalancer(ctx context.Context, arn string) (*storage.ELBv2LoadBalancer, error) {
	query := `
	SELECT arn, name, dns_name, canonical_hosted_zone_id,
		state, type, scheme, vpc_id, subnets,
		availability_zones, security_groups, ip_address_type,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_load_balancers
	WHERE arn = $1`

	var lb storage.ELBv2LoadBalancer
	var subnetsJSON, azsJSON, sgJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID,
		&lb.State, &lb.Type, &lb.Scheme, &lb.VpcID,
		&subnetsJSON, &azsJSON, &sgJSON, &lb.IpAddressType,
		&tagsJSON, &lb.Region, &lb.AccountID,
		&lb.CreatedAt, &lb.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get load balancer: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
	json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
	json.Unmarshal([]byte(sgJSON), &lb.SecurityGroups)
	json.Unmarshal([]byte(tagsJSON), &lb.Tags)

	return &lb, nil
}

// GetLoadBalancerByName retrieves a load balancer by name
func (s *elbv2Store) GetLoadBalancerByName(ctx context.Context, name string) (*storage.ELBv2LoadBalancer, error) {
	query := `
	SELECT arn, name, dns_name, canonical_hosted_zone_id,
		state, type, scheme, vpc_id, subnets,
		availability_zones, security_groups, ip_address_type,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_load_balancers
	WHERE name = $1`

	var lb storage.ELBv2LoadBalancer
	var subnetsJSON, azsJSON, sgJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID,
		&lb.State, &lb.Type, &lb.Scheme, &lb.VpcID,
		&subnetsJSON, &azsJSON, &sgJSON, &lb.IpAddressType,
		&tagsJSON, &lb.Region, &lb.AccountID,
		&lb.CreatedAt, &lb.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get load balancer by name: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
	json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
	json.Unmarshal([]byte(sgJSON), &lb.SecurityGroups)
	json.Unmarshal([]byte(tagsJSON), &lb.Tags)

	return &lb, nil
}

// ListLoadBalancers lists all load balancers in a region
func (s *elbv2Store) ListLoadBalancers(ctx context.Context, region string) ([]*storage.ELBv2LoadBalancer, error) {
	query := `
	SELECT arn, name, dns_name, canonical_hosted_zone_id,
		state, type, scheme, vpc_id, subnets,
		availability_zones, security_groups, ip_address_type,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_load_balancers
	WHERE region = $1
	ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to list load balancers: %w", err)
	}
	defer rows.Close()

	var lbs []*storage.ELBv2LoadBalancer
	for rows.Next() {
		var lb storage.ELBv2LoadBalancer
		var subnetsJSON, azsJSON, sgJSON, tagsJSON string

		err := rows.Scan(
			&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID,
			&lb.State, &lb.Type, &lb.Scheme, &lb.VpcID,
			&subnetsJSON, &azsJSON, &sgJSON, &lb.IpAddressType,
			&tagsJSON, &lb.Region, &lb.AccountID,
			&lb.CreatedAt, &lb.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan load balancer: %w", err)
		}

		// Parse JSON fields
		json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
		json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
		json.Unmarshal([]byte(sgJSON), &lb.SecurityGroups)
		json.Unmarshal([]byte(tagsJSON), &lb.Tags)

		lbs = append(lbs, &lb)
	}

	return lbs, nil
}

// UpdateLoadBalancer updates a load balancer
func (s *elbv2Store) UpdateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	lb.UpdatedAt = time.Now()

	// Convert arrays and maps to JSON
	subnetsJSON, _ := json.Marshal(lb.Subnets)
	azsJSON, _ := json.Marshal(lb.AvailabilityZones)
	sgJSON, _ := json.Marshal(lb.SecurityGroups)
	tagsJSON, _ := json.Marshal(lb.Tags)

	query := `
	UPDATE elbv2_load_balancers SET
		state = $1, subnets = $2, availability_zones = $3,
		security_groups = $4, tags = $5, updated_at = $6
	WHERE arn = $7`

	result, err := s.db.ExecContext(ctx, query,
		lb.State, string(subnetsJSON), string(azsJSON),
		string(sgJSON), string(tagsJSON), lb.UpdatedAt, lb.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update load balancer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

// DeleteLoadBalancer deletes a load balancer
func (s *elbv2Store) DeleteLoadBalancer(ctx context.Context, arn string) error {
	query := `DELETE FROM elbv2_load_balancers WHERE arn = $1`

	result, err := s.db.ExecContext(ctx, query, arn)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

// Target Group operations

// CreateTargetGroup creates a new target group
func (s *elbv2Store) CreateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	if tg.CreatedAt.IsZero() {
		tg.CreatedAt = time.Now()
	}
	tg.UpdatedAt = time.Now()

	// Convert arrays and maps to JSON
	lbArnsJSON, _ := json.Marshal(tg.LoadBalancerArns)
	tagsJSON, _ := json.Marshal(tg.Tags)

	query := `
	INSERT INTO elbv2_target_groups (
		arn, name, protocol, port, vpc_id, target_type,
		health_check_enabled, health_check_protocol, health_check_port,
		health_check_path, health_check_interval_seconds,
		health_check_timeout_seconds, healthy_threshold_count,
		unhealthy_threshold_count, matcher, load_balancer_arns,
		tags, region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
	)`

	_, err := s.db.ExecContext(ctx, query,
		tg.ARN, tg.Name, tg.Protocol, tg.Port, tg.VpcID, tg.TargetType,
		tg.HealthCheckEnabled, tg.HealthCheckProtocol, tg.HealthCheckPort,
		tg.HealthCheckPath, tg.HealthCheckIntervalSeconds,
		tg.HealthCheckTimeoutSeconds, tg.HealthyThresholdCount,
		tg.UnhealthyThresholdCount, tg.Matcher, string(lbArnsJSON),
		string(tagsJSON), tg.Region, tg.AccountID,
		tg.CreatedAt, tg.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to create target group: %w", err)
	}

	return nil
}

// GetTargetGroup retrieves a target group by ARN
func (s *elbv2Store) GetTargetGroup(ctx context.Context, arn string) (*storage.ELBv2TargetGroup, error) {
	query := `
	SELECT arn, name, protocol, port, vpc_id, target_type,
		health_check_enabled, health_check_protocol, health_check_port,
		health_check_path, health_check_interval_seconds,
		health_check_timeout_seconds, healthy_threshold_count,
		unhealthy_threshold_count, matcher, load_balancer_arns,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_target_groups
	WHERE arn = $1`

	var tg storage.ELBv2TargetGroup
	var lbArnsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
		&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
		&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds,
		&tg.HealthCheckTimeoutSeconds, &tg.HealthyThresholdCount,
		&tg.UnhealthyThresholdCount, &tg.Matcher, &lbArnsJSON,
		&tagsJSON, &tg.Region, &tg.AccountID,
		&tg.CreatedAt, &tg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get target group: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
	json.Unmarshal([]byte(tagsJSON), &tg.Tags)

	return &tg, nil
}

// GetTargetGroupByName retrieves a target group by name
func (s *elbv2Store) GetTargetGroupByName(ctx context.Context, name string) (*storage.ELBv2TargetGroup, error) {
	query := `
	SELECT arn, name, protocol, port, vpc_id, target_type,
		health_check_enabled, health_check_protocol, health_check_port,
		health_check_path, health_check_interval_seconds,
		health_check_timeout_seconds, healthy_threshold_count,
		unhealthy_threshold_count, matcher, load_balancer_arns,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_target_groups
	WHERE name = $1`

	var tg storage.ELBv2TargetGroup
	var lbArnsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
		&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
		&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds,
		&tg.HealthCheckTimeoutSeconds, &tg.HealthyThresholdCount,
		&tg.UnhealthyThresholdCount, &tg.Matcher, &lbArnsJSON,
		&tagsJSON, &tg.Region, &tg.AccountID,
		&tg.CreatedAt, &tg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get target group by name: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
	json.Unmarshal([]byte(tagsJSON), &tg.Tags)

	return &tg, nil
}

// ListTargetGroups lists all target groups in a region
func (s *elbv2Store) ListTargetGroups(ctx context.Context, region string) ([]*storage.ELBv2TargetGroup, error) {
	query := `
	SELECT arn, name, protocol, port, vpc_id, target_type,
		health_check_enabled, health_check_protocol, health_check_port,
		health_check_path, health_check_interval_seconds,
		health_check_timeout_seconds, healthy_threshold_count,
		unhealthy_threshold_count, matcher, load_balancer_arns,
		tags, region, account_id, created_at, updated_at
	FROM elbv2_target_groups
	WHERE region = $1
	ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to list target groups: %w", err)
	}
	defer rows.Close()

	var tgs []*storage.ELBv2TargetGroup
	for rows.Next() {
		var tg storage.ELBv2TargetGroup
		var lbArnsJSON, tagsJSON string

		err := rows.Scan(
			&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
			&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
			&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds,
			&tg.HealthCheckTimeoutSeconds, &tg.HealthyThresholdCount,
			&tg.UnhealthyThresholdCount, &tg.Matcher, &lbArnsJSON,
			&tagsJSON, &tg.Region, &tg.AccountID,
			&tg.CreatedAt, &tg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target group: %w", err)
		}

		// Parse JSON fields
		json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
		json.Unmarshal([]byte(tagsJSON), &tg.Tags)

		tgs = append(tgs, &tg)
	}

	return tgs, nil
}

// UpdateTargetGroup updates a target group
func (s *elbv2Store) UpdateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	tg.UpdatedAt = time.Now()

	// Convert arrays and maps to JSON
	lbArnsJSON, _ := json.Marshal(tg.LoadBalancerArns)
	tagsJSON, _ := json.Marshal(tg.Tags)

	query := `
	UPDATE elbv2_target_groups SET
		health_check_enabled = $1, health_check_protocol = $2,
		health_check_port = $3, health_check_path = $4,
		health_check_interval_seconds = $5, health_check_timeout_seconds = $6,
		healthy_threshold_count = $7, unhealthy_threshold_count = $8,
		matcher = $9, load_balancer_arns = $10, tags = $11, updated_at = $12
	WHERE arn = $13`

	result, err := s.db.ExecContext(ctx, query,
		tg.HealthCheckEnabled, tg.HealthCheckProtocol, tg.HealthCheckPort,
		tg.HealthCheckPath, tg.HealthCheckIntervalSeconds,
		tg.HealthCheckTimeoutSeconds, tg.HealthyThresholdCount,
		tg.UnhealthyThresholdCount, tg.Matcher,
		string(lbArnsJSON), string(tagsJSON), tg.UpdatedAt, tg.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update target group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

// DeleteTargetGroup deletes a target group
func (s *elbv2Store) DeleteTargetGroup(ctx context.Context, arn string) error {
	query := `DELETE FROM elbv2_target_groups WHERE arn = $1`

	result, err := s.db.ExecContext(ctx, query, arn)
	if err != nil {
		return fmt.Errorf("failed to delete target group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

// Placeholder implementations for remaining methods
// These will be fully implemented when the corresponding structs are defined

// CreateListener creates a new listener
func (s *elbv2Store) CreateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	return fmt.Errorf("not implemented")
}

// GetListener retrieves a listener by ARN
func (s *elbv2Store) GetListener(ctx context.Context, arn string) (*storage.ELBv2Listener, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListListeners lists all listeners for a load balancer
func (s *elbv2Store) ListListeners(ctx context.Context, loadBalancerArn string) ([]*storage.ELBv2Listener, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateListener updates a listener
func (s *elbv2Store) UpdateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	return fmt.Errorf("not implemented")
}

// DeleteListener deletes a listener
func (s *elbv2Store) DeleteListener(ctx context.Context, arn string) error {
	return fmt.Errorf("not implemented")
}

// RegisterTargets registers targets with a target group
func (s *elbv2Store) RegisterTargets(ctx context.Context, targetGroupArn string, targets []*storage.ELBv2Target) error {
	return fmt.Errorf("not implemented")
}

// DeregisterTargets deregisters targets from a target group
func (s *elbv2Store) DeregisterTargets(ctx context.Context, targetGroupArn string, targetIDs []string) error {
	return fmt.Errorf("not implemented")
}

// ListTargets lists all targets in a target group
func (s *elbv2Store) ListTargets(ctx context.Context, targetGroupArn string) ([]*storage.ELBv2Target, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateTargetHealth updates the health of a target
func (s *elbv2Store) UpdateTargetHealth(ctx context.Context, targetGroupArn, targetID string, health *storage.ELBv2TargetHealth) error {
	return fmt.Errorf("not implemented")
}

// CreateRule creates a new rule
func (s *elbv2Store) CreateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	return fmt.Errorf("not implemented")
}

// GetRule retrieves a rule by ARN
func (s *elbv2Store) GetRule(ctx context.Context, ruleArn string) (*storage.ELBv2Rule, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListRules lists all rules for a listener
func (s *elbv2Store) ListRules(ctx context.Context, listenerArn string) ([]*storage.ELBv2Rule, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateRule updates a rule
func (s *elbv2Store) UpdateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	return fmt.Errorf("not implemented")
}

// DeleteRule deletes a rule
func (s *elbv2Store) DeleteRule(ctx context.Context, ruleArn string) error {
	return fmt.Errorf("not implemented")
}
