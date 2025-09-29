package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

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
		deployment_configuration, deployment_controller, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15,
		$16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25,
		$26, $27, $28, $29, $30,
		$31, $32, $33
	)`

	_, err := s.db.ExecContext(ctx, query,
		service.ID, service.ARN, service.ServiceName, service.ClusterARN, service.TaskDefinitionARN,
		service.DesiredCount, service.RunningCount, service.PendingCount,
		toNullString(service.LaunchType), toNullString(service.PlatformVersion),
		service.Status, toNullString(service.RoleARN),
		toNullString(service.LoadBalancers), toNullString(service.ServiceRegistries),
		toNullString(service.NetworkConfiguration),
		toNullString(service.DeploymentConfiguration), toNullString(service.DeploymentController),
		toNullString(service.PlacementConstraints), toNullString(service.PlacementStrategy),
		toNullString(service.CapacityProviderStrategy), toNullString(service.Tags),
		toNullString(service.SchedulingStrategy), toNullString(service.ServiceConnectConfiguration),
		service.EnableECSManagedTags, toNullString(service.PropagateTags), service.EnableExecuteCommand,
		service.HealthCheckGracePeriodSeconds, service.Region, service.AccountID,
		toNullString(service.DeploymentName), toNullString(service.Namespace),
		service.CreatedAt, service.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// Get retrieves a service by ARN or service name
func (s *serviceStore) Get(ctx context.Context, clusterARN, serviceNameOrARN string) (*storage.Service, error) {
	query := `
	SELECT
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, deployment_controller, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	FROM services
	WHERE cluster_arn = $1 AND (arn = $2 OR service_name = $2)`

	var service storage.Service
	var launchType, platformVersion, roleARN, loadBalancers, serviceRegistries sql.NullString
	var networkConfiguration, deploymentConfiguration, deploymentController sql.NullString
	var placementConstraints, placementStrategy, capacityProviderStrategy sql.NullString
	var tags, schedulingStrategy, serviceConnectConfiguration, propagateTags sql.NullString
	var deploymentName, namespace sql.NullString

	err := s.db.QueryRowContext(ctx, query, clusterARN, serviceNameOrARN).Scan(
		&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
		&service.DesiredCount, &service.RunningCount, &service.PendingCount, &launchType, &platformVersion,
		&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
		&deploymentConfiguration, &deploymentController, &placementConstraints, &placementStrategy,
		&capacityProviderStrategy, &tags, &schedulingStrategy, &serviceConnectConfiguration,
		&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
		&service.HealthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &deploymentName, &namespace,
		&service.CreatedAt, &service.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Convert null strings back to regular strings
	service.LaunchType = fromNullString(launchType)
	service.PlatformVersion = fromNullString(platformVersion)
	service.RoleARN = fromNullString(roleARN)
	service.LoadBalancers = fromNullString(loadBalancers)
	service.ServiceRegistries = fromNullString(serviceRegistries)
	service.NetworkConfiguration = fromNullString(networkConfiguration)
	service.DeploymentConfiguration = fromNullString(deploymentConfiguration)
	service.DeploymentController = fromNullString(deploymentController)
	service.PlacementConstraints = fromNullString(placementConstraints)
	service.PlacementStrategy = fromNullString(placementStrategy)
	service.CapacityProviderStrategy = fromNullString(capacityProviderStrategy)
	service.Tags = fromNullString(tags)
	service.SchedulingStrategy = fromNullString(schedulingStrategy)
	service.ServiceConnectConfiguration = fromNullString(serviceConnectConfiguration)
	service.PropagateTags = fromNullString(propagateTags)
	service.DeploymentName = fromNullString(deploymentName)
	service.Namespace = fromNullString(namespace)

	return &service, nil
}

// GetByARN retrieves a service by ARN only
func (s *serviceStore) GetByARN(ctx context.Context, serviceARN string) (*storage.Service, error) {
	query := `
	SELECT
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, deployment_controller, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	FROM services
	WHERE arn = $1`

	var service storage.Service
	var launchType, platformVersion, roleARN, loadBalancers, serviceRegistries sql.NullString
	var networkConfiguration, deploymentConfiguration, deploymentController sql.NullString
	var placementConstraints, placementStrategy, capacityProviderStrategy sql.NullString
	var tags, schedulingStrategy, serviceConnectConfiguration, propagateTags sql.NullString
	var deploymentName, namespace sql.NullString

	err := s.db.QueryRowContext(ctx, query, serviceARN).Scan(
		&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
		&service.DesiredCount, &service.RunningCount, &service.PendingCount, &launchType, &platformVersion,
		&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
		&deploymentConfiguration, &deploymentController, &placementConstraints, &placementStrategy,
		&capacityProviderStrategy, &tags, &schedulingStrategy, &serviceConnectConfiguration,
		&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
		&service.HealthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &deploymentName, &namespace,
		&service.CreatedAt, &service.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get service by ARN: %w", err)
	}

	// Convert null strings back to regular strings
	service.LaunchType = fromNullString(launchType)
	service.PlatformVersion = fromNullString(platformVersion)
	service.RoleARN = fromNullString(roleARN)
	service.LoadBalancers = fromNullString(loadBalancers)
	service.ServiceRegistries = fromNullString(serviceRegistries)
	service.NetworkConfiguration = fromNullString(networkConfiguration)
	service.DeploymentConfiguration = fromNullString(deploymentConfiguration)
	service.DeploymentController = fromNullString(deploymentController)
	service.PlacementConstraints = fromNullString(placementConstraints)
	service.PlacementStrategy = fromNullString(placementStrategy)
	service.CapacityProviderStrategy = fromNullString(capacityProviderStrategy)
	service.Tags = fromNullString(tags)
	service.SchedulingStrategy = fromNullString(schedulingStrategy)
	service.ServiceConnectConfiguration = fromNullString(serviceConnectConfiguration)
	service.PropagateTags = fromNullString(propagateTags)
	service.DeploymentName = fromNullString(deploymentName)
	service.Namespace = fromNullString(namespace)

	return &service, nil
}

// List retrieves services with filtering
func (s *serviceStore) List(ctx context.Context, clusterARN string, serviceName string, launchType string, limit int, nextToken string) ([]*storage.Service, string, error) {
	// Parse the next token to get offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	// Build query with filters
	query := `
	SELECT
		id, arn, service_name, cluster_arn, task_definition_arn,
		desired_count, running_count, pending_count, launch_type, platform_version,
		status, role_arn, load_balancers, service_registries, network_configuration,
		deployment_configuration, deployment_controller, placement_constraints, placement_strategy,
		capacity_provider_strategy, tags, scheduling_strategy, service_connect_configuration,
		enable_ecs_managed_tags, propagate_tags, enable_execute_command,
		health_check_grace_period_seconds, region, account_id, deployment_name, namespace,
		created_at, updated_at
	FROM services
	WHERE cluster_arn = $1`

	args := []interface{}{clusterARN}
	argNum := 2

	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name = $%d", argNum)
		args = append(args, serviceName)
		argNum++
	}

	if launchType != "" {
		query += fmt.Sprintf(" AND launch_type = $%d", argNum)
		args = append(args, launchType)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []*storage.Service
	for rows.Next() {
		var service storage.Service
		var launchType, platformVersion, roleARN, loadBalancers, serviceRegistries sql.NullString
		var networkConfiguration, deploymentConfiguration, deploymentController sql.NullString
		var placementConstraints, placementStrategy, capacityProviderStrategy sql.NullString
		var tags, schedulingStrategy, serviceConnectConfiguration, propagateTags sql.NullString
		var deploymentName, namespace sql.NullString

		err := rows.Scan(
			&service.ID, &service.ARN, &service.ServiceName, &service.ClusterARN, &service.TaskDefinitionARN,
			&service.DesiredCount, &service.RunningCount, &service.PendingCount, &launchType, &platformVersion,
			&service.Status, &roleARN, &loadBalancers, &serviceRegistries, &networkConfiguration,
			&deploymentConfiguration, &deploymentController, &placementConstraints, &placementStrategy,
			&capacityProviderStrategy, &tags, &schedulingStrategy, &serviceConnectConfiguration,
			&service.EnableECSManagedTags, &propagateTags, &service.EnableExecuteCommand,
			&service.HealthCheckGracePeriodSeconds, &service.Region, &service.AccountID, &deploymentName, &namespace,
			&service.CreatedAt, &service.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan service row: %w", err)
		}

		// Convert null strings back to regular strings
		service.LaunchType = fromNullString(launchType)
		service.PlatformVersion = fromNullString(platformVersion)
		service.RoleARN = fromNullString(roleARN)
		service.LoadBalancers = fromNullString(loadBalancers)
		service.ServiceRegistries = fromNullString(serviceRegistries)
		service.NetworkConfiguration = fromNullString(networkConfiguration)
		service.DeploymentConfiguration = fromNullString(deploymentConfiguration)
		service.DeploymentController = fromNullString(deploymentController)
		service.PlacementConstraints = fromNullString(placementConstraints)
		service.PlacementStrategy = fromNullString(placementStrategy)
		service.CapacityProviderStrategy = fromNullString(capacityProviderStrategy)
		service.Tags = fromNullString(tags)
		service.SchedulingStrategy = fromNullString(schedulingStrategy)
		service.ServiceConnectConfiguration = fromNullString(serviceConnectConfiguration)
		service.PropagateTags = fromNullString(propagateTags)
		service.DeploymentName = fromNullString(deploymentName)
		service.Namespace = fromNullString(namespace)

		services = append(services, &service)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if limit > 0 && len(services) == limit {
		newNextToken = fmt.Sprintf("%d", offset+limit)
	}

	return services, newNextToken, nil
}

// Update updates an existing service
func (s *serviceStore) Update(ctx context.Context, service *storage.Service) error {
	service.UpdatedAt = time.Now()

	query := `
	UPDATE services SET
		task_definition_arn = $1, desired_count = $2, running_count = $3,
		pending_count = $4, launch_type = $5, platform_version = $6,
		status = $7, role_arn = $8, load_balancers = $9, service_registries = $10,
		network_configuration = $11, deployment_configuration = $12, deployment_controller = $13,
		placement_constraints = $14, placement_strategy = $15, capacity_provider_strategy = $16,
		tags = $17, scheduling_strategy = $18, service_connect_configuration = $19,
		enable_ecs_managed_tags = $20, propagate_tags = $21, enable_execute_command = $22,
		health_check_grace_period_seconds = $23, deployment_name = $24, namespace = $25,
		updated_at = $26
	WHERE arn = $27`

	result, err := s.db.ExecContext(ctx, query,
		service.TaskDefinitionARN, service.DesiredCount, service.RunningCount,
		service.PendingCount, toNullString(service.LaunchType), toNullString(service.PlatformVersion),
		service.Status, toNullString(service.RoleARN),
		toNullString(service.LoadBalancers), toNullString(service.ServiceRegistries),
		toNullString(service.NetworkConfiguration), toNullString(service.DeploymentConfiguration),
		toNullString(service.DeploymentController), toNullString(service.PlacementConstraints),
		toNullString(service.PlacementStrategy), toNullString(service.CapacityProviderStrategy),
		toNullString(service.Tags), toNullString(service.SchedulingStrategy),
		toNullString(service.ServiceConnectConfiguration),
		service.EnableECSManagedTags, toNullString(service.PropagateTags), service.EnableExecuteCommand,
		service.HealthCheckGracePeriodSeconds, toNullString(service.DeploymentName),
		toNullString(service.Namespace), service.UpdatedAt, service.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
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

// Delete deletes a service
func (s *serviceStore) Delete(ctx context.Context, clusterARN, serviceNameOrARN string) error {
	query := `DELETE FROM services WHERE cluster_arn = $1 AND (arn = $2 OR service_name = $2)`

	result, err := s.db.ExecContext(ctx, query, clusterARN, serviceNameOrARN)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
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

// DeleteMarkedForDeletion deletes services marked for deletion
func (s *serviceStore) DeleteMarkedForDeletion(ctx context.Context, clusterARN string, before time.Time) (int, error) {
	query := `DELETE FROM services WHERE cluster_arn = $1 AND status = 'DELETE_IN_PROGRESS' AND updated_at < $2`

	result, err := s.db.ExecContext(ctx, query, clusterARN, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete marked services: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
