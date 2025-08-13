package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// elbv2Store implements storage.ELBv2Store using DuckDB
type elbv2Store struct {
	db *sql.DB
}

// NewELBv2Store creates a new ELBv2 store
func NewELBv2Store(db *sql.DB) storage.ELBv2Store {
	return &elbv2Store{db: db}
}

// CreateLoadBalancer creates a new load balancer
func (s *elbv2Store) CreateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	query := `
		INSERT INTO elbv2_load_balancers (
			arn, name, dns_name, canonical_hosted_zone_id, state, type, scheme,
			vpc_id, subnets, availability_zones, security_groups, ip_address_type,
			tags, region, account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	subnetsJSON, _ := json.Marshal(lb.Subnets)
	azsJSON, _ := json.Marshal(lb.AvailabilityZones)
	securityGroupsJSON, _ := json.Marshal(lb.SecurityGroups)
	tagsJSON, _ := json.Marshal(lb.Tags)

	_, err := s.db.ExecContext(ctx, query,
		lb.ARN, lb.Name, lb.DNSName, lb.CanonicalHostedZoneID, lb.State, lb.Type, lb.Scheme,
		lb.VpcID, string(subnetsJSON), string(azsJSON), string(securityGroupsJSON), lb.IpAddressType,
		string(tagsJSON), lb.Region, lb.AccountID, lb.CreatedAt, lb.UpdatedAt,
	)
	return err
}

// GetLoadBalancer retrieves a load balancer by ARN
func (s *elbv2Store) GetLoadBalancer(ctx context.Context, arn string) (*storage.ELBv2LoadBalancer, error) {
	query := `
		SELECT arn, name, dns_name, canonical_hosted_zone_id, state, type, scheme,
			vpc_id, subnets, availability_zones, security_groups, ip_address_type,
			tags, region, account_id, created_at, updated_at
		FROM elbv2_load_balancers
		WHERE arn = ?
	`

	var lb storage.ELBv2LoadBalancer
	var subnetsJSON, azsJSON, securityGroupsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID, &lb.State, &lb.Type, &lb.Scheme,
		&lb.VpcID, &subnetsJSON, &azsJSON, &securityGroupsJSON, &lb.IpAddressType,
		&tagsJSON, &lb.Region, &lb.AccountID, &lb.CreatedAt, &lb.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("load balancer not found: %s", arn)
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
	json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
	json.Unmarshal([]byte(securityGroupsJSON), &lb.SecurityGroups)
	json.Unmarshal([]byte(tagsJSON), &lb.Tags)

	return &lb, nil
}

// GetLoadBalancerByName retrieves a load balancer by name
func (s *elbv2Store) GetLoadBalancerByName(ctx context.Context, name string) (*storage.ELBv2LoadBalancer, error) {
	query := `
		SELECT arn, name, dns_name, canonical_hosted_zone_id, state, type, scheme,
			vpc_id, subnets, availability_zones, security_groups, ip_address_type,
			tags, region, account_id, created_at, updated_at
		FROM elbv2_load_balancers
		WHERE name = ?
	`

	var lb storage.ELBv2LoadBalancer
	var subnetsJSON, azsJSON, securityGroupsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID, &lb.State, &lb.Type, &lb.Scheme,
		&lb.VpcID, &subnetsJSON, &azsJSON, &securityGroupsJSON, &lb.IpAddressType,
		&tagsJSON, &lb.Region, &lb.AccountID, &lb.CreatedAt, &lb.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
	json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
	json.Unmarshal([]byte(securityGroupsJSON), &lb.SecurityGroups)
	json.Unmarshal([]byte(tagsJSON), &lb.Tags)

	return &lb, nil
}

// ListLoadBalancers lists all load balancers in a region
func (s *elbv2Store) ListLoadBalancers(ctx context.Context, region string) ([]*storage.ELBv2LoadBalancer, error) {
	query := `
		SELECT arn, name, dns_name, canonical_hosted_zone_id, state, type, scheme,
			vpc_id, subnets, availability_zones, security_groups, ip_address_type,
			tags, region, account_id, created_at, updated_at
		FROM elbv2_load_balancers
		WHERE region = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loadBalancers []*storage.ELBv2LoadBalancer
	for rows.Next() {
		var lb storage.ELBv2LoadBalancer
		var subnetsJSON, azsJSON, securityGroupsJSON, tagsJSON string

		err := rows.Scan(
			&lb.ARN, &lb.Name, &lb.DNSName, &lb.CanonicalHostedZoneID, &lb.State, &lb.Type, &lb.Scheme,
			&lb.VpcID, &subnetsJSON, &azsJSON, &securityGroupsJSON, &lb.IpAddressType,
			&tagsJSON, &lb.Region, &lb.AccountID, &lb.CreatedAt, &lb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(subnetsJSON), &lb.Subnets)
		json.Unmarshal([]byte(azsJSON), &lb.AvailabilityZones)
		json.Unmarshal([]byte(securityGroupsJSON), &lb.SecurityGroups)
		json.Unmarshal([]byte(tagsJSON), &lb.Tags)

		loadBalancers = append(loadBalancers, &lb)
	}

	return loadBalancers, nil
}

// UpdateLoadBalancer updates a load balancer
func (s *elbv2Store) UpdateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	query := `
		UPDATE elbv2_load_balancers
		SET state = ?, security_groups = ?, tags = ?, updated_at = ?
		WHERE arn = ?
	`

	securityGroupsJSON, _ := json.Marshal(lb.SecurityGroups)
	tagsJSON, _ := json.Marshal(lb.Tags)
	lb.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		lb.State, string(securityGroupsJSON), string(tagsJSON), lb.UpdatedAt, lb.ARN,
	)
	return err
}

// DeleteLoadBalancer deletes a load balancer
func (s *elbv2Store) DeleteLoadBalancer(ctx context.Context, arn string) error {
	query := `DELETE FROM elbv2_load_balancers WHERE arn = ?`
	_, err := s.db.ExecContext(ctx, query, arn)
	return err
}

// CreateTargetGroup creates a new target group
func (s *elbv2Store) CreateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	query := `
		INSERT INTO elbv2_target_groups (
			arn, name, protocol, port, vpc_id, target_type,
			health_check_enabled, health_check_protocol, health_check_port,
			health_check_path, health_check_interval_seconds, health_check_timeout_seconds,
			healthy_threshold_count, unhealthy_threshold_count, matcher,
			load_balancer_arns, tags, region, account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	lbArnsJSON, _ := json.Marshal(tg.LoadBalancerArns)
	tagsJSON, _ := json.Marshal(tg.Tags)

	_, err := s.db.ExecContext(ctx, query,
		tg.ARN, tg.Name, tg.Protocol, tg.Port, tg.VpcID, tg.TargetType,
		tg.HealthCheckEnabled, tg.HealthCheckProtocol, tg.HealthCheckPort,
		tg.HealthCheckPath, tg.HealthCheckIntervalSeconds, tg.HealthCheckTimeoutSeconds,
		tg.HealthyThresholdCount, tg.UnhealthyThresholdCount, tg.Matcher,
		string(lbArnsJSON), string(tagsJSON), tg.Region, tg.AccountID, tg.CreatedAt, tg.UpdatedAt,
	)
	return err
}

// GetTargetGroup retrieves a target group by ARN
func (s *elbv2Store) GetTargetGroup(ctx context.Context, arn string) (*storage.ELBv2TargetGroup, error) {
	query := `
		SELECT arn, name, protocol, port, vpc_id, target_type,
			health_check_enabled, health_check_protocol, health_check_port,
			health_check_path, health_check_interval_seconds, health_check_timeout_seconds,
			healthy_threshold_count, unhealthy_threshold_count, matcher,
			load_balancer_arns, tags, region, account_id, created_at, updated_at
		FROM elbv2_target_groups
		WHERE arn = ?
	`

	var tg storage.ELBv2TargetGroup
	var lbArnsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
		&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
		&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds, &tg.HealthCheckTimeoutSeconds,
		&tg.HealthyThresholdCount, &tg.UnhealthyThresholdCount, &tg.Matcher,
		&lbArnsJSON, &tagsJSON, &tg.Region, &tg.AccountID, &tg.CreatedAt, &tg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("target group not found: %s", arn)
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
	json.Unmarshal([]byte(tagsJSON), &tg.Tags)

	return &tg, nil
}

// GetTargetGroupByName retrieves a target group by name
func (s *elbv2Store) GetTargetGroupByName(ctx context.Context, name string) (*storage.ELBv2TargetGroup, error) {
	query := `
		SELECT arn, name, protocol, port, vpc_id, target_type,
			health_check_enabled, health_check_protocol, health_check_port,
			health_check_path, health_check_interval_seconds, health_check_timeout_seconds,
			healthy_threshold_count, unhealthy_threshold_count, matcher,
			load_balancer_arns, tags, region, account_id, created_at, updated_at
		FROM elbv2_target_groups
		WHERE name = ?
	`

	var tg storage.ELBv2TargetGroup
	var lbArnsJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
		&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
		&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds, &tg.HealthCheckTimeoutSeconds,
		&tg.HealthyThresholdCount, &tg.UnhealthyThresholdCount, &tg.Matcher,
		&lbArnsJSON, &tagsJSON, &tg.Region, &tg.AccountID, &tg.CreatedAt, &tg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
	json.Unmarshal([]byte(tagsJSON), &tg.Tags)

	return &tg, nil
}

// ListTargetGroups lists all target groups in a region
func (s *elbv2Store) ListTargetGroups(ctx context.Context, region string) ([]*storage.ELBv2TargetGroup, error) {
	query := `
		SELECT arn, name, protocol, port, vpc_id, target_type,
			health_check_enabled, health_check_protocol, health_check_port,
			health_check_path, health_check_interval_seconds, health_check_timeout_seconds,
			healthy_threshold_count, unhealthy_threshold_count, matcher,
			load_balancer_arns, tags, region, account_id, created_at, updated_at
		FROM elbv2_target_groups
		WHERE region = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targetGroups []*storage.ELBv2TargetGroup
	for rows.Next() {
		var tg storage.ELBv2TargetGroup
		var lbArnsJSON, tagsJSON string

		err := rows.Scan(
			&tg.ARN, &tg.Name, &tg.Protocol, &tg.Port, &tg.VpcID, &tg.TargetType,
			&tg.HealthCheckEnabled, &tg.HealthCheckProtocol, &tg.HealthCheckPort,
			&tg.HealthCheckPath, &tg.HealthCheckIntervalSeconds, &tg.HealthCheckTimeoutSeconds,
			&tg.HealthyThresholdCount, &tg.UnhealthyThresholdCount, &tg.Matcher,
			&lbArnsJSON, &tagsJSON, &tg.Region, &tg.AccountID, &tg.CreatedAt, &tg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(lbArnsJSON), &tg.LoadBalancerArns)
		json.Unmarshal([]byte(tagsJSON), &tg.Tags)

		targetGroups = append(targetGroups, &tg)
	}

	return targetGroups, nil
}

// UpdateTargetGroup updates a target group
func (s *elbv2Store) UpdateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	query := `
		UPDATE elbv2_target_groups
		SET health_check_enabled = ?, health_check_protocol = ?, health_check_port = ?,
			health_check_path = ?, health_check_interval_seconds = ?, health_check_timeout_seconds = ?,
			healthy_threshold_count = ?, unhealthy_threshold_count = ?, matcher = ?,
			load_balancer_arns = ?, tags = ?, updated_at = ?
		WHERE arn = ?
	`

	lbArnsJSON, _ := json.Marshal(tg.LoadBalancerArns)
	tagsJSON, _ := json.Marshal(tg.Tags)
	tg.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		tg.HealthCheckEnabled, tg.HealthCheckProtocol, tg.HealthCheckPort,
		tg.HealthCheckPath, tg.HealthCheckIntervalSeconds, tg.HealthCheckTimeoutSeconds,
		tg.HealthyThresholdCount, tg.UnhealthyThresholdCount, tg.Matcher,
		string(lbArnsJSON), string(tagsJSON), tg.UpdatedAt, tg.ARN,
	)
	return err
}

// DeleteTargetGroup deletes a target group
func (s *elbv2Store) DeleteTargetGroup(ctx context.Context, arn string) error {
	// Delete associated targets first
	_, err := s.db.ExecContext(ctx, `DELETE FROM elbv2_targets WHERE target_group_arn = ?`, arn)
	if err != nil {
		return err
	}

	// Delete the target group
	_, err = s.db.ExecContext(ctx, `DELETE FROM elbv2_target_groups WHERE arn = ?`, arn)
	return err
}

// CreateListener creates a new listener
func (s *elbv2Store) CreateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	query := `
		INSERT INTO elbv2_listeners (
			arn, load_balancer_arn, port, protocol, default_actions,
			ssl_policy, certificates, alpn_policy, tags,
			region, account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	alpnPolicyJSON, _ := json.Marshal(listener.AlpnPolicy)
	tagsJSON, _ := json.Marshal(listener.Tags)

	_, err := s.db.ExecContext(ctx, query,
		listener.ARN, listener.LoadBalancerArn, listener.Port, listener.Protocol, listener.DefaultActions,
		listener.SslPolicy, listener.Certificates, string(alpnPolicyJSON), string(tagsJSON),
		listener.Region, listener.AccountID, listener.CreatedAt, listener.UpdatedAt,
	)
	return err
}

// GetListener retrieves a listener by ARN
func (s *elbv2Store) GetListener(ctx context.Context, arn string) (*storage.ELBv2Listener, error) {
	query := `
		SELECT arn, load_balancer_arn, port, protocol, default_actions,
			ssl_policy, certificates, alpn_policy, tags,
			region, account_id, created_at, updated_at
		FROM elbv2_listeners
		WHERE arn = ?
	`

	var listener storage.ELBv2Listener
	var alpnPolicyJSON, tagsJSON string

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&listener.ARN, &listener.LoadBalancerArn, &listener.Port, &listener.Protocol, &listener.DefaultActions,
		&listener.SslPolicy, &listener.Certificates, &alpnPolicyJSON, &tagsJSON,
		&listener.Region, &listener.AccountID, &listener.CreatedAt, &listener.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("listener not found: %s", arn)
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(alpnPolicyJSON), &listener.AlpnPolicy)
	json.Unmarshal([]byte(tagsJSON), &listener.Tags)

	return &listener, nil
}

// ListListeners lists all listeners for a load balancer
func (s *elbv2Store) ListListeners(ctx context.Context, loadBalancerArn string) ([]*storage.ELBv2Listener, error) {
	query := `
		SELECT arn, load_balancer_arn, port, protocol, default_actions,
			ssl_policy, certificates, alpn_policy, tags,
			region, account_id, created_at, updated_at
		FROM elbv2_listeners
		WHERE load_balancer_arn = ?
		ORDER BY port ASC
	`

	rows, err := s.db.QueryContext(ctx, query, loadBalancerArn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listeners []*storage.ELBv2Listener
	for rows.Next() {
		var listener storage.ELBv2Listener
		var alpnPolicyJSON, tagsJSON string

		err := rows.Scan(
			&listener.ARN, &listener.LoadBalancerArn, &listener.Port, &listener.Protocol, &listener.DefaultActions,
			&listener.SslPolicy, &listener.Certificates, &alpnPolicyJSON, &tagsJSON,
			&listener.Region, &listener.AccountID, &listener.CreatedAt, &listener.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(alpnPolicyJSON), &listener.AlpnPolicy)
		json.Unmarshal([]byte(tagsJSON), &listener.Tags)

		listeners = append(listeners, &listener)
	}

	return listeners, nil
}

// UpdateListener updates a listener
func (s *elbv2Store) UpdateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	query := `
		UPDATE elbv2_listeners
		SET default_actions = ?, ssl_policy = ?, certificates = ?,
			alpn_policy = ?, tags = ?, updated_at = ?
		WHERE arn = ?
	`

	alpnPolicyJSON, _ := json.Marshal(listener.AlpnPolicy)
	tagsJSON, _ := json.Marshal(listener.Tags)
	listener.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		listener.DefaultActions, listener.SslPolicy, listener.Certificates,
		string(alpnPolicyJSON), string(tagsJSON), listener.UpdatedAt, listener.ARN,
	)
	return err
}

// DeleteListener deletes a listener
func (s *elbv2Store) DeleteListener(ctx context.Context, arn string) error {
	query := `DELETE FROM elbv2_listeners WHERE arn = ?`
	_, err := s.db.ExecContext(ctx, query, arn)
	return err
}

// RegisterTargets registers targets with a target group
func (s *elbv2Store) RegisterTargets(ctx context.Context, targetGroupArn string, targets []*storage.ELBv2Target) error {
	if len(targets) == 0 {
		return nil
	}

	// Build the bulk insert query
	valueStrings := make([]string, 0, len(targets))
	valueArgs := make([]interface{}, 0, len(targets)*10)

	for _, target := range targets {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			targetGroupArn, target.ID, target.Port, target.AvailabilityZone,
			target.HealthState, target.HealthReason, target.HealthDescription,
			target.RegisteredAt, target.UpdatedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO elbv2_targets (
			target_group_arn, id, port, availability_zone,
			health_state, health_reason, health_description,
			registered_at, updated_at
		) VALUES %s
	`, strings.Join(valueStrings, ","))

	_, err := s.db.ExecContext(ctx, query, valueArgs...)
	return err
}

// DeregisterTargets deregisters targets from a target group
func (s *elbv2Store) DeregisterTargets(ctx context.Context, targetGroupArn string, targetIDs []string) error {
	if len(targetIDs) == 0 {
		return nil
	}

	// Build the IN clause
	placeholders := make([]string, len(targetIDs))
	args := make([]interface{}, len(targetIDs)+1)
	args[0] = targetGroupArn
	for i, id := range targetIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		DELETE FROM elbv2_targets
		WHERE target_group_arn = ? AND id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListTargets lists all targets for a target group
func (s *elbv2Store) ListTargets(ctx context.Context, targetGroupArn string) ([]*storage.ELBv2Target, error) {
	query := `
		SELECT target_group_arn, id, port, availability_zone,
			health_state, health_reason, health_description,
			registered_at, updated_at
		FROM elbv2_targets
		WHERE target_group_arn = ?
		ORDER BY registered_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, targetGroupArn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*storage.ELBv2Target
	for rows.Next() {
		var target storage.ELBv2Target
		err := rows.Scan(
			&target.TargetGroupArn, &target.ID, &target.Port, &target.AvailabilityZone,
			&target.HealthState, &target.HealthReason, &target.HealthDescription,
			&target.RegisteredAt, &target.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		targets = append(targets, &target)
	}

	return targets, nil
}

// UpdateTargetHealth updates the health status of a target
func (s *elbv2Store) UpdateTargetHealth(ctx context.Context, targetGroupArn, targetID string, health *storage.ELBv2TargetHealth) error {
	query := `
		UPDATE elbv2_targets
		SET health_state = ?, health_reason = ?, health_description = ?, updated_at = ?
		WHERE target_group_arn = ? AND id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		health.State, health.Reason, health.Description, time.Now(),
		targetGroupArn, targetID,
	)
	if err != nil {
		logging.Error("Failed to update target health", "targetGroupArn", targetGroupArn, "targetID", targetID, "error", err)
	}
	return err
}

// Rule operations

// CreateRule creates a new rule
func (s *elbv2Store) CreateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	query := `
		INSERT INTO elbv2_rules (
			arn, listener_arn, priority, conditions, actions,
			is_default, tags, region, account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	tagsJSON, _ := json.Marshal(rule.Tags)

	_, err := s.db.ExecContext(ctx, query,
		rule.ARN, rule.ListenerArn, rule.Priority, rule.Conditions, rule.Actions,
		rule.IsDefault, string(tagsJSON), rule.Region, rule.AccountID, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

// GetRule retrieves a rule by ARN
func (s *elbv2Store) GetRule(ctx context.Context, ruleArn string) (*storage.ELBv2Rule, error) {
	query := `
		SELECT arn, listener_arn, priority, conditions, actions,
			is_default, tags, region, account_id, created_at, updated_at
		FROM elbv2_rules
		WHERE arn = ?
	`

	var rule storage.ELBv2Rule
	var tagsJSON string

	err := s.db.QueryRowContext(ctx, query, ruleArn).Scan(
		&rule.ARN, &rule.ListenerArn, &rule.Priority, &rule.Conditions, &rule.Actions,
		&rule.IsDefault, &tagsJSON, &rule.Region, &rule.AccountID, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: %s", ruleArn)
	} else if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(tagsJSON), &rule.Tags)

	return &rule, nil
}

// ListRules lists all rules for a listener
func (s *elbv2Store) ListRules(ctx context.Context, listenerArn string) ([]*storage.ELBv2Rule, error) {
	query := `
		SELECT arn, listener_arn, priority, conditions, actions,
			is_default, tags, region, account_id, created_at, updated_at
		FROM elbv2_rules
		WHERE listener_arn = ?
		ORDER BY priority ASC
	`

	rows, err := s.db.QueryContext(ctx, query, listenerArn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*storage.ELBv2Rule
	for rows.Next() {
		var rule storage.ELBv2Rule
		var tagsJSON string

		err := rows.Scan(
			&rule.ARN, &rule.ListenerArn, &rule.Priority, &rule.Conditions, &rule.Actions,
			&rule.IsDefault, &tagsJSON, &rule.Region, &rule.AccountID, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		json.Unmarshal([]byte(tagsJSON), &rule.Tags)

		rules = append(rules, &rule)
	}

	return rules, nil
}

// UpdateRule updates a rule
func (s *elbv2Store) UpdateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	query := `
		UPDATE elbv2_rules
		SET conditions = ?, actions = ?, tags = ?, updated_at = ?
		WHERE arn = ?
	`

	tagsJSON, _ := json.Marshal(rule.Tags)
	rule.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query,
		rule.Conditions, rule.Actions, string(tagsJSON), rule.UpdatedAt, rule.ARN,
	)
	return err
}

// DeleteRule deletes a rule
func (s *elbv2Store) DeleteRule(ctx context.Context, ruleArn string) error {
	query := `DELETE FROM elbv2_rules WHERE arn = ?`
	_, err := s.db.ExecContext(ctx, query, ruleArn)
	return err
}
