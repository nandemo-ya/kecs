// KECS API Types
// These types match the Go structs in the KECS Control Plane

export interface Cluster {
  clusterArn: string;
  clusterName: string;
  status: string;
  registeredContainerInstancesCount: number;
  runningTasksCount: number;
  pendingTasksCount: number;
  activeServicesCount: number;
  tags?: Tag[];
  settings?: string;
  configuration?: string;
}

export interface Service {
  serviceArn: string;
  serviceName: string;
  clusterArn: string;
  status: string;
  desiredCount: number;
  runningCount: number;
  pendingCount: number;
  launchType: string;
  taskDefinition: string;
  createdAt?: string;
  platformVersion?: string;
  schedulingStrategy?: string;
}

export interface Task {
  taskArn: string;
  clusterArn: string;
  taskDefinitionArn: string;
  lastStatus: string;
  desiredStatus: string;
  cpu?: string;
  memory?: string;
  startedAt?: string;
  startedBy?: string;
  launchType: string;
  group?: string;
  healthStatus?: string;
}

export interface TaskDefinition {
  taskDefinitionArn: string;
  family: string;
  revision: number;
  status: string;
  registeredAt: string;
  cpu?: string;
  memory?: string;
  networkMode?: string;
  requiresCompatibilities?: string[];
}

export interface Tag {
  key: string;
  value: string;
}

export interface Failure {
  arn: string;
  reason: string;
  detail?: string;
}

// API Request/Response Types
export interface ListClustersResponse {
  clusterArns: string[];
  nextToken?: string;
}

export interface DescribeClustersResponse {
  clusters: Cluster[];
  failures?: Failure[];
}

export interface ListServicesResponse {
  serviceArns: string[];
  nextToken?: string;
}

export interface DescribeServicesResponse {
  services: Service[];
  failures?: Failure[];
}

export interface ListTasksResponse {
  taskArns: string[];
  nextToken?: string;
}

export interface DescribeTasksResponse {
  tasks: Task[];
  failures?: Failure[];
}

export interface ListTaskDefinitionsResponse {
  taskDefinitionArns: string[];
  nextToken?: string;
}

export interface DescribeTaskDefinitionResponse {
  taskDefinition: TaskDefinition;
}

// Service Management Types
export interface CreateServiceRequest {
  cluster: string;
  serviceName: string;
  taskDefinition: string;
  desiredCount: number;
  launchType?: string;
  platformVersion?: string;
  schedulingStrategy?: string;
}

export interface CreateServiceResponse {
  service: Service;
}

export interface UpdateServiceRequest {
  cluster: string;
  service: string;
  desiredCount?: number;
  taskDefinition?: string;
  platformVersion?: string;
}

export interface UpdateServiceResponse {
  service: Service;
}

export interface DeleteServiceRequest {
  cluster: string;
  service: string;
  force?: boolean;
}

export interface DeleteServiceResponse {
  service: Service;
}

// Dashboard Summary Types
export interface DashboardStats {
  clusters: number;
  services: number;
  tasks: number;
  taskDefinitions: number;
}

export interface HealthStatus {
  status: 'connected' | 'connecting' | 'error';
  message: string;
  timestamp: string;
}