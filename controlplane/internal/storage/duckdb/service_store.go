package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// serviceStore implements storage.ServiceStore using DuckDB
type serviceStore struct {
	db *sql.DB
}

// Create creates a new service
func (s *serviceStore) Create(ctx context.Context, service *storage.Service) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}

	now := time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = now
	}
	service.UpdatedAt = now

	query := `
	INSERT INTO services (
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	) VALUES (
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?
	)`

	_, err := s.db.ExecContext(ctx, query,
		service.ID, service.ARN, service.ServiceName, service.ClusterARN, service.TaskDefinitionARN,
		service.DesiredCount, service.RunningCount, service.PendingCount, service.LaunchType, service.PlatformVersion,
		service.Status, service.RoleARN, service.LoadBalancers, service.ServiceRegistries, service.NetworkConfiguration,
		service.DeploymentConfiguration, service.PlacementConstraints, service.PlacementStrategy,
		service.CapacityProviderStrategy, service.Tags, service.SchedulingStrategy, service.ServiceConnectConfiguration,
		service.EnableECSManagedTags, service.PropagateTags, service.EnableExecuteCommand,
		service.HealthCheckGracePeriodSeconds, service.Region, service.AccountID, service.DeploymentName, service.Namespace,
		service.CreatedAt, service.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// Get retrieves a service by cluster and service name
func (s *serviceStore) Get(ctx context.Context, cluster, serviceName string) (*storage.Service, error) {
	query := `
	SELECT 
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	FROM services 
	WHERE cluster_arn = ? AND service_name = ?`

	service := &storage.Service{}
	var platformVersion, roleARN, loadBalancers, serviceRegistries, networkConfiguration sql.NullString
	var deploymentConfiguration, placementConstraints, placementStrategy sql.NullString
	var capacityProviderStrategy, tags, serviceConnectConfiguration sql.NullString
	var propagateTags, deploymentName, namespace sql.NullString
	var healthCheckGracePeriodSeconds sql.NullInt32

	err := s.db.QueryRowContext(ctx, query, cluster, serviceName).Scan(
		&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
		&service.DesiredCount, &service.RunningCount, &service.PendingCount, &service.LaunchType, &platformVersion,
		&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
		&deploymentConfiguration, &placementConstraints, &placementStrategy,
		&capacityProviderStrategy, &tags, &service.SchedulingStrategy, &serviceConnectConfiguration,
		&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
		&healthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &deploymentName, &namespace,
		&service.CreatedAt, &service.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("service not found: cluster=%s, serviceName=%s", cluster, serviceName)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Handle nullable fields
	if platformVersion.Valid {
		service.PlatformVersion = platformVersion.String
	}
	if roleARN.Valid {
		service.RoleARN = roleARN.String
	}
	if loadBalancers.Valid {
		service.LoadBalancers = loadBalancers.String
	}
	if serviceRegistries.Valid {
		service.ServiceRegistries = serviceRegistries.String
	}
	if networkConfiguration.Valid {
		service.NetworkConfiguration = networkConfiguration.String
	}
	if deploymentConfiguration.Valid {
		service.DeploymentConfiguration = deploymentConfiguration.String
	}
	if placementConstraints.Valid {
		service.PlacementConstraints = placementConstraints.String
	}
	if placementStrategy.Valid {
		service.PlacementStrategy = placementStrategy.String
	}
	if capacityProviderStrategy.Valid {
		service.CapacityProviderStrategy = capacityProviderStrategy.String
	}
	if tags.Valid {
		service.Tags = tags.String
	}
	if serviceConnectConfiguration.Valid {
		service.ServiceConnectConfiguration = serviceConnectConfiguration.String
	}
	if propagateTags.Valid {
		service.PropagateTags = propagateTags.String
	}
	if healthCheckGracePeriodSeconds.Valid {
		service.HealthCheckGracePeriodSeconds = int(healthCheckGracePeriodSeconds.Int32)
	}
	if deploymentName.Valid {
		service.DeploymentName = deploymentName.String
	}
	if namespace.Valid {
		service.Namespace = namespace.String
	}

	return service, nil
}

// List retrieves services with filtering
func (s *serviceStore) List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*storage.Service, string, error) {
	var args []interface{}
	var conditions []string

	baseQuery := `
	SELECT 
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, created_at, updated_at
	FROM services`

	// Build WHERE conditions
	if cluster != "" {
		conditions = append(conditions, "cluster_arn = ?")
		args = append(args, cluster)
	}
	if serviceName != "" {
		conditions = append(conditions, "service_name = ?")
		args = append(args, serviceName)
	}
	if launchType != "" {
		conditions = append(conditions, "launch_type = ?")
		args = append(args, launchType)
	}

	// Add token-based pagination
	if nextToken != "" {
		conditions = append(conditions, "id > ?")
		args = append(args, nextToken)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering and limit
	baseQuery += " ORDER BY id"
	if limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, limit+1) // Get one extra to determine if there are more results
	}

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []*storage.Service
	for rows.Next() {
		service := &storage.Service{}
		var platformVersion, roleARN, loadBalancers, serviceRegistries, networkConfiguration sql.NullString
		var deploymentConfiguration, placementConstraints, placementStrategy sql.NullString
		var capacityProviderStrategy, tags, serviceConnectConfiguration sql.NullString
		var propagateTags sql.NullString
		var healthCheckGracePeriodSeconds sql.NullInt32

		err := rows.Scan(
			&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
			&service.DesiredCount, &service.RunningCount, &service.PendingCount, &service.LaunchType, &platformVersion,
			&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
			&deploymentConfiguration, &placementConstraints, &placementStrategy,
			&capacityProviderStrategy, &tags, &service.SchedulingStrategy, &serviceConnectConfiguration,
			&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
			&healthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &service.CreatedAt, &service.UpdatedAt,
		)

		if err != nil {
			return nil, "", fmt.Errorf("failed to scan service: %w", err)
		}

		// Handle nullable fields
		if platformVersion.Valid {
			service.PlatformVersion = platformVersion.String
		}
		if roleARN.Valid {
			service.RoleARN = roleARN.String
		}
		if loadBalancers.Valid {
			service.LoadBalancers = loadBalancers.String
		}
		if serviceRegistries.Valid {
			service.ServiceRegistries = serviceRegistries.String
		}
		if networkConfiguration.Valid {
			service.NetworkConfiguration = networkConfiguration.String
		}
		if deploymentConfiguration.Valid {
			service.DeploymentConfiguration = deploymentConfiguration.String
		}
		if placementConstraints.Valid {
			service.PlacementConstraints = placementConstraints.String
		}
		if placementStrategy.Valid {
			service.PlacementStrategy = placementStrategy.String
		}
		if capacityProviderStrategy.Valid {
			service.CapacityProviderStrategy = capacityProviderStrategy.String
		}
		if tags.Valid {
			service.Tags = tags.String
		}
		if serviceConnectConfiguration.Valid {
			service.ServiceConnectConfiguration = serviceConnectConfiguration.String
		}
		if propagateTags.Valid {
			service.PropagateTags = propagateTags.String
		}
		if healthCheckGracePeriodSeconds.Valid {
			service.HealthCheckGracePeriodSeconds = int(healthCheckGracePeriodSeconds.Int32)
		}

		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to iterate services: %w", err)
	}

	var newNextToken string
	if limit > 0 && len(services) > limit {
		// Remove the extra service and set nextToken
		services = services[:limit]
		newNextToken = services[limit-1].ID
	}

	return services, newNextToken, nil
}

// Update updates an existing service
func (s *serviceStore) Update(ctx context.Context, service *storage.Service) error {
	service.UpdatedAt = time.Now()

	log.Printf("DEBUG: Updating service ID: %s, ARN: %s, status: %s, desiredCount: %d", service.ID, service.ARN, service.Status, service.DesiredCount)

	// Small delay to avoid DuckDB concurrency issues
	time.Sleep(50 * time.Millisecond)

	// First, let's check if the record exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM services WHERE id = ?`
	err := s.db.QueryRowContext(ctx, checkQuery, service.ID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check service existence: %w", err)
	}
	log.Printf("DEBUG: Found %d records with ID %s", count, service.ID)

	// Update all relevant fields that might change
	query := `UPDATE services SET 
		task_definition_arn = ?,
		desired_count = ?,
		running_count = ?,
		pending_count = ?,
		platform_version = ?,
		status = ?,
		load_balancers = ?,
		service_registries = ?,
		network_configuration = ?,
		deployment_configuration = ?,
		placement_constraints = ?,
		placement_strategy = ?,
		capacity_provider_strategy = ?,
		tags = ?,
		service_connect_configuration = ?,
		enable_ecs_managed_tags = ?,
		propagate_tags = ?,
		enable_execute_command = ?,
		health_check_grace_period_seconds = ?,
		updated_at = ?
		WHERE arn = ? AND id = ?`

	result, err := s.db.ExecContext(ctx, query,
		service.TaskDefinitionARN,
		service.DesiredCount,
		service.RunningCount,
		service.PendingCount,
		nullString(service.PlatformVersion),
		service.Status,
		nullString(service.LoadBalancers),
		nullString(service.ServiceRegistries),
		nullString(service.NetworkConfiguration),
		nullString(service.DeploymentConfiguration),
		nullString(service.PlacementConstraints),
		nullString(service.PlacementStrategy),
		nullString(service.CapacityProviderStrategy),
		nullString(service.Tags),
		nullString(service.ServiceConnectConfiguration),
		service.EnableECSManagedTags,
		nullString(service.PropagateTags),
		service.EnableExecuteCommand,
		service.HealthCheckGracePeriodSeconds,
		service.UpdatedAt,
		service.ARN,
		service.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("service not found: %s", service.ID)
	}

	return nil
}

// Delete deletes a service by cluster and service name
func (s *serviceStore) Delete(ctx context.Context, cluster, serviceName string) error {
	query := `DELETE FROM services WHERE cluster_arn = ? AND service_name = ?`

	result, err := s.db.ExecContext(ctx, query, cluster, serviceName)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("service not found: cluster=%s, serviceName=%s", cluster, serviceName)
	}

	return nil
}

// GetByARN retrieves a service by ARN
func (s *serviceStore) GetByARN(ctx context.Context, arn string) (*storage.Service, error) {
	query := `
	SELECT 
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	FROM services 
	WHERE arn = ?`

	service := &storage.Service{}
	var platformVersion, roleARN, loadBalancers, serviceRegistries, networkConfiguration sql.NullString
	var deploymentConfiguration, placementConstraints, placementStrategy sql.NullString
	var capacityProviderStrategy, tags, serviceConnectConfiguration sql.NullString
	var propagateTags, deploymentName, namespace sql.NullString
	var healthCheckGracePeriodSeconds sql.NullInt32

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
		&service.DesiredCount, &service.RunningCount, &service.PendingCount, &service.LaunchType, &platformVersion,
		&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
		&deploymentConfiguration, &placementConstraints, &placementStrategy,
		&capacityProviderStrategy, &tags, &service.SchedulingStrategy, &serviceConnectConfiguration,
		&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
		&healthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &deploymentName, &namespace,
		&service.CreatedAt, &service.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("service not found: arn=%s", arn)
		}
		return nil, fmt.Errorf("failed to get service by ARN: %w", err)
	}

	// Handle nullable fields
	if platformVersion.Valid {
		service.PlatformVersion = platformVersion.String
	}
	if roleARN.Valid {
		service.RoleARN = roleARN.String
	}
	if loadBalancers.Valid {
		service.LoadBalancers = loadBalancers.String
	}
	if serviceRegistries.Valid {
		service.ServiceRegistries = serviceRegistries.String
	}
	if networkConfiguration.Valid {
		service.NetworkConfiguration = networkConfiguration.String
	}
	if deploymentConfiguration.Valid {
		service.DeploymentConfiguration = deploymentConfiguration.String
	}
	if placementConstraints.Valid {
		service.PlacementConstraints = placementConstraints.String
	}
	if placementStrategy.Valid {
		service.PlacementStrategy = placementStrategy.String
	}
	if capacityProviderStrategy.Valid {
		service.CapacityProviderStrategy = capacityProviderStrategy.String
	}
	if tags.Valid {
		service.Tags = tags.String
	}
	if serviceConnectConfiguration.Valid {
		service.ServiceConnectConfiguration = serviceConnectConfiguration.String
	}
	if propagateTags.Valid {
		service.PropagateTags = propagateTags.String
	}
	if healthCheckGracePeriodSeconds.Valid {
		service.HealthCheckGracePeriodSeconds = int(healthCheckGracePeriodSeconds.Int32)
	}
	if deploymentName.Valid {
		service.DeploymentName = deploymentName.String
	}
	if namespace.Valid {
		service.Namespace = namespace.String
	}

	return service, nil
}
