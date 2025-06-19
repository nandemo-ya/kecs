# Web UI Guide

KECS includes a modern web-based user interface for managing your ECS resources. This guide covers how to use the Web UI effectively.

## Accessing the Web UI

### Default Access

The Web UI is available at:
```
http://localhost:8080/ui
```

### Custom Port Configuration

If running KECS on a different port:
```bash
# Start KECS with custom port
./bin/kecs server --api-port 9080

# Access UI at
http://localhost:9080/ui
```

### Authentication

By default, the Web UI doesn't require authentication in development mode. For production deployments, see [Security Configuration](/guides/security).

## Dashboard Overview

### Main Dashboard

The dashboard provides a real-time overview of your ECS environment:

- **Cluster Status**: Active clusters and their health
- **Service Metrics**: Running services across all clusters
- **Task Statistics**: Total tasks, running, pending, and stopped
- **Recent Events**: Latest deployment and scaling events
- **Resource Utilization**: CPU and memory usage graphs

### Navigation

The sidebar provides quick access to:
- **Clusters**: Manage ECS clusters
- **Services**: View and manage services
- **Tasks**: Monitor running tasks
- **Task Definitions**: Browse and create task definitions
- **Load Balancers**: Manage load balancer integrations
- **LocalStack**: LocalStack integration status

## Managing Clusters

### Viewing Clusters

1. Click **Clusters** in the sidebar
2. View the list of all clusters with:
   - Cluster name and ARN
   - Status (Active/Inactive)
   - Number of services
   - Running tasks count
   - Resource utilization

### Creating a Cluster

1. Click **Create Cluster** button
2. Enter cluster details:
   - **Cluster Name**: Unique name for your cluster
   - **Tags**: Optional key-value pairs
   - **Settings**: Container insights, logging options
3. Click **Create**

### Cluster Details

Click on a cluster name to view:
- **Overview**: Cluster statistics and configuration
- **Services**: Services running in this cluster
- **Tasks**: All tasks in the cluster
- **Container Instances**: EC2 instances (if applicable)
- **Metrics**: CPU, memory, network graphs
- **Events**: Cluster event history

## Managing Services

### Service List View

The services page shows:
- Service name and ARN
- Cluster assignment
- Task definition and revision
- Desired/Running/Pending counts
- Status and health
- Last deployment info

### Creating a Service

1. Click **Create Service**
2. Fill in the service configuration:

#### Step 1: Configure Service
- **Cluster**: Select target cluster
- **Service Name**: Unique service identifier
- **Task Definition**: Select family and revision
- **Service Type**: Replica or Daemon

#### Step 2: Configure Network
- **Launch Type**: Fargate or EC2
- **Number of Tasks**: Desired count
- **VPC and Subnets**: Network configuration
- **Security Groups**: Select or create
- **Public IP**: Enable/disable

#### Step 3: Load Balancing (Optional)
- **Load Balancer Type**: Application/Network
- **Target Group**: Select existing or create new
- **Container and Port**: Map to container

#### Step 4: Service Discovery (Optional)
- **Namespace**: Select DNS namespace
- **Service Discovery Name**: DNS name for service
- **DNS Record Type**: A or SRV
- **TTL**: DNS record TTL

#### Step 5: Auto Scaling (Optional)
- **Minimum Tasks**: Lower bound
- **Maximum Tasks**: Upper bound
- **Target Metric**: CPU or Memory
- **Target Value**: Threshold percentage

### Service Details

Click on a service to access:

#### Overview Tab
- Service configuration
- Current deployment status
- Task health summary
- Recent events

#### Tasks Tab
- List of all tasks
- Task status and health
- Start/stop individual tasks
- View task logs

#### Metrics Tab
- CPU utilization graph
- Memory utilization graph
- Network I/O metrics
- Request count (if load balanced)

#### Deployments Tab
- Active deployments
- Deployment history
- Rollback options
- Circuit breaker status

#### Events Tab
- Service event timeline
- Scaling events
- Deployment events
- Error events

### Updating Services

1. Click **Update** on service details page
2. Modify configuration:
   - Task definition revision
   - Desired count
   - Deployment options
   - Network configuration
3. Review changes
4. Click **Update Service**

The UI shows deployment progress in real-time.

## Managing Tasks

### Task List

View all tasks with:
- Task ID and ARN
- Task definition
- Cluster and service
- Status (Running/Pending/Stopped)
- Started/Stopped times
- Container instance (EC2)

### Task Filters

Filter tasks by:
- **Status**: Running, Pending, Stopped
- **Service**: Tasks belonging to specific service
- **Task Definition**: Specific family or revision
- **Launch Type**: Fargate or EC2

### Task Details

Click on a task to view:

#### Overview
- Task configuration
- Network details (IP addresses)
- IAM roles
- Resource allocation

#### Containers
- Container status
- Exit codes
- Resource usage
- Environment variables

#### Logs
- Real-time log streaming
- Download logs
- Filter by container
- Search functionality

#### Metrics
- CPU usage over time
- Memory usage over time
- Network metrics
- Disk I/O

### Running Tasks Manually

1. Click **Run Task**
2. Select:
   - Cluster
   - Task definition
   - Launch type
   - Number of tasks
3. Configure overrides (optional):
   - Container overrides
   - Environment variables
   - Resource limits
4. Click **Run**

## Task Definitions

### Browsing Task Definitions

View all task definition families:
- Family name
- Latest revision
- Status
- Compatible launch types
- Created date

### Creating Task Definitions

1. Click **Create new Task Definition**
2. Choose compatibility:
   - Fargate
   - EC2
   - External

#### Container Configuration
- **Container Name**: Unique identifier
- **Image**: Docker image URI
- **Memory Limits**: Hard and soft limits
- **Port Mappings**: Container ports
- **Environment Variables**: Key-value pairs
- **Secrets**: From Secrets Manager or SSM

#### Advanced Configuration
- Health checks
- Logging configuration
- Volumes
- Docker labels
- System controls

### Task Definition Details

View comprehensive information:
- JSON definition
- Container definitions
- Network mode
- IAM roles
- Volumes
- Revision history

## Real-time Features

### WebSocket Connections

The UI maintains WebSocket connections for:
- Live task status updates
- Service deployment progress
- Cluster events
- Metrics streaming

### Auto-refresh

Data refreshes automatically:
- Dashboard: Every 5 seconds
- Service list: Every 10 seconds
- Task list: Every 10 seconds
- Metrics: Real-time streaming

### Notifications

Receive in-app notifications for:
- Deployment completions
- Task failures
- Scaling events
- Error conditions

## LocalStack Integration

### LocalStack Dashboard

When LocalStack is enabled:
1. Click **LocalStack** in sidebar
2. View integration status:
   - LocalStack health
   - Available services
   - Endpoint URLs
   - Configuration

### AWS Service Proxies

Access proxied AWS services:
- IAM policies and roles
- CloudWatch logs and metrics
- S3 buckets
- Secrets Manager
- SSM Parameter Store

## Keyboard Shortcuts

Improve productivity with shortcuts:

- `Ctrl/Cmd + K`: Quick search
- `G then C`: Go to Clusters
- `G then S`: Go to Services
- `G then T`: Go to Tasks
- `G then D`: Go to Task Definitions
- `R`: Refresh current view
- `?`: Show keyboard shortcuts

## Settings and Preferences

### UI Preferences

Access via gear icon:
- **Theme**: Light/Dark mode
- **Density**: Comfortable/Compact
- **Refresh Intervals**: Customize auto-refresh
- **Notifications**: Enable/disable types

### Data Export

Export data in multiple formats:
- CSV export for lists
- JSON export for configurations
- PDF reports for metrics

## Tips and Best Practices

### 1. Use Filters Effectively

- Save frequently used filters
- Combine multiple filters
- Use search for quick access

### 2. Monitor Deployments

- Keep deployment tab open during updates
- Watch event stream for issues
- Use circuit breaker for safety

### 3. Leverage Keyboard Navigation

- Learn shortcuts for efficiency
- Use quick search (`Ctrl+K`)
- Navigate with arrow keys

### 4. Customize Views

- Adjust column visibility
- Sort by relevant metrics
- Use compact mode for more data

### 5. Set Up Alerts

- Configure deployment notifications
- Enable failure alerts
- Monitor resource thresholds

## Troubleshooting UI Issues

### Connection Problems

If the UI can't connect to the API:
1. Verify KECS is running
2. Check browser console for errors
3. Ensure WebSocket connections are allowed
4. Try refreshing the page

### Performance Issues

For slow UI performance:
1. Reduce auto-refresh frequency
2. Limit number of items per page
3. Close unused browser tabs
4. Clear browser cache

### Display Problems

If UI elements are missing or broken:
1. Check browser compatibility (Chrome, Firefox, Safari supported)
2. Disable browser extensions
3. Clear local storage
4. Try incognito/private mode

For more help, see our [Troubleshooting Guide](/guides/troubleshooting).