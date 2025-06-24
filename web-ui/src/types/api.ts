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
  tags?: Tag[];
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
  containerDefinitions?: ContainerDefinition[];
  tags?: Tag[];
}

export interface ContainerDefinition {
  name: string;
  image: string;
  cpu?: number;
  memory?: number;
  memoryReservation?: number;
  portMappings?: PortMapping[];
  essential?: boolean;
  environment?: EnvironmentVariable[];
  mountPoints?: MountPoint[];
  volumesFrom?: VolumeFrom[];
  command?: string[];
  entryPoint?: string[];
  workingDirectory?: string;
  user?: string;
  privileged?: boolean;
  dockerLabels?: { [key: string]: string };
  ulimits?: Ulimit[];
}

export interface PortMapping {
  containerPort: number;
  hostPort?: number;
  protocol?: string;
}

export interface EnvironmentVariable {
  name: string;
  value: string;
}

export interface MountPoint {
  sourceVolume: string;
  containerPath: string;
  readOnly?: boolean;
}

export interface VolumeFrom {
  sourceContainer: string;
  readOnly?: boolean;
}

export interface Ulimit {
  name: string;
  softLimit: number;
  hardLimit: number;
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

// Task Definition Management Types
export interface RegisterTaskDefinitionRequest {
  family: string;
  taskRoleArn?: string;
  executionRoleArn?: string;
  networkMode?: string;
  containerDefinitions: ContainerDefinition[];
  volumes?: Volume[];
  placementConstraints?: PlacementConstraint[];
  requiresCompatibilities?: string[];
  cpu?: string;
  memory?: string;
  tags?: Tag[];
}

export interface RegisterTaskDefinitionResponse {
  taskDefinition: TaskDefinition;
}

export interface DeregisterTaskDefinitionRequest {
  taskDefinition: string;
}

export interface DeregisterTaskDefinitionResponse {
  taskDefinition: TaskDefinition;
}

export interface Volume {
  name: string;
  host?: {
    sourcePath?: string;
  };
}

export interface PlacementConstraint {
  type?: string;
  expression?: string;
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

// Cluster Management Types
export interface CreateClusterRequest {
  clusterName: string;
  capacityProviders?: string[];
  defaultCapacityProviderStrategy?: CapacityProviderStrategyItem[];
  tags?: Tag[];
  settings?: ClusterSetting[];
  configuration?: ClusterConfiguration;
}

export interface CreateClusterResponse {
  cluster: Cluster;
}

export interface DeleteClusterRequest {
  cluster: string;
}

export interface DeleteClusterResponse {
  cluster: Cluster;
}

export interface UpdateClusterRequest {
  cluster: string;
  settings?: ClusterSetting[];
  configuration?: ClusterConfiguration;
}

export interface UpdateClusterResponse {
  cluster: Cluster;
}

// Task Management Types
export interface RunTaskRequest {
  cluster?: string;
  taskDefinition: string;
  count?: number;
  startedBy?: string;
  group?: string;
  overrides?: TaskOverride;
  networkConfiguration?: NetworkConfiguration;
  launchType?: string;
  platformVersion?: string;
  placementConstraints?: PlacementConstraint[];
  placementStrategy?: PlacementStrategy[];
  tags?: Tag[];
  enableECSManagedTags?: boolean;
  enableExecuteCommand?: boolean;
  propagateTags?: string;
  referenceId?: string;
}

export interface RunTaskResponse {
  tasks: Task[];
  failures?: Failure[];
}

export interface StopTaskRequest {
  cluster?: string;
  task: string;
  reason?: string;
}

export interface StopTaskResponse {
  task: Task;
}

export interface StartTaskRequest {
  cluster?: string;
  taskDefinition: string;
  containerInstances: string[];
  overrides?: TaskOverride;
  networkConfiguration?: NetworkConfiguration;
  startedBy?: string;
  tags?: Tag[];
  enableECSManagedTags?: boolean;
  enableExecuteCommand?: boolean;
  propagateTags?: string;
  referenceId?: string;
}

export interface StartTaskResponse {
  tasks: Task[];
  failures?: Failure[];
}

// Task Definition Batch Operations
export interface DeleteTaskDefinitionsRequest {
  taskDefinitions: string[];
}

export interface DeleteTaskDefinitionsResponse {
  taskDefinitions?: TaskDefinition[];
  failures?: Failure[];
}

// Service Deployment Types
export interface ServiceDeployment {
  id?: string;
  status?: string;
  taskDefinition?: string;
  desiredCount?: number;
  pendingCount?: number;
  runningCount?: number;
  failedTasks?: number;
  createdAt?: string;
  updatedAt?: string;
  launchType?: string;
  platformVersion?: string;
  platformFamily?: string;
  networkConfiguration?: NetworkConfiguration;
  rolloutState?: string;
  rolloutStateReason?: string;
  serviceConnectConfiguration?: ServiceConnectConfiguration;
  volumeConfigurations?: ServiceVolumeConfiguration[];
}

export interface ServiceRevision {
  serviceRevisionArn?: string;
  serviceArn?: string;
  clusterArn?: string;
  taskDefinition?: string;
  capacityProviderStrategy?: CapacityProviderStrategyItem[];
  launchType?: string;
  platformVersion?: string;
  platformFamily?: string;
  loadBalancers?: LoadBalancer[];
  serviceRegistries?: ServiceRegistry[];
  networkConfiguration?: NetworkConfiguration;
  containerImages?: ContainerImage[];
  createdAt?: string;
}

export interface DescribeServiceDeploymentsRequest {
  serviceDeploymentArns: string[];
}

export interface DescribeServiceDeploymentsResponse {
  serviceDeployments?: ServiceDeployment[];
  failures?: Failure[];
}

export interface DescribeServiceRevisionsRequest {
  serviceRevisionArns: string[];
}

export interface DescribeServiceRevisionsResponse {
  serviceRevisions?: ServiceRevision[];
  failures?: Failure[];
}

export interface ListServiceDeploymentsRequest {
  service: string;
  cluster?: string;
  status?: string[];
  maxResults?: number;
  nextToken?: string;
}

export interface ListServiceDeploymentsResponse {
  serviceDeployments?: ServiceDeployment[];
  nextToken?: string;
}

// Task Set Types (for blue/green deployments)
export interface TaskSet {
  id?: string;
  taskSetArn?: string;
  serviceArn?: string;
  clusterArn?: string;
  startedBy?: string;
  taskDefinition?: string;
  computedDesiredCount?: number;
  pendingCount?: number;
  runningCount?: number;
  createdAt?: string;
  updatedAt?: string;
  launchType?: string;
  platformVersion?: string;
  platformFamily?: string;
  networkConfiguration?: NetworkConfiguration;
  loadBalancers?: LoadBalancer[];
  serviceRegistries?: ServiceRegistry[];
  scale?: Scale;
  stabilityStatus?: string;
  stabilityStatusAt?: string;
  tags?: Tag[];
}

export interface CreateTaskSetRequest {
  service: string;
  cluster: string;
  taskDefinition: string;
  externalId?: string;
  networkConfiguration?: NetworkConfiguration;
  loadBalancers?: LoadBalancer[];
  serviceRegistries?: ServiceRegistry[];
  launchType?: string;
  capacityProviderStrategy?: CapacityProviderStrategyItem[];
  platformVersion?: string;
  scale?: Scale;
  clientToken?: string;
  tags?: Tag[];
}

export interface CreateTaskSetResponse {
  taskSet?: TaskSet;
}

// Container Instance Types
export interface ContainerInstance {
  containerInstanceArn?: string;
  ec2InstanceId?: string;
  capacityProviderName?: string;
  version?: number;
  versionInfo?: VersionInfo;
  remainingResources?: Resource[];
  registeredResources?: Resource[];
  status?: string;
  statusReason?: string;
  agentConnected?: boolean;
  runningTasksCount?: number;
  pendingTasksCount?: number;
  agentUpdateStatus?: string;
  attributes?: Attribute[];
  registeredAt?: string;
  attachments?: Attachment[];
  tags?: Tag[];
  healthStatus?: ContainerInstanceHealthStatus;
}

export interface ListContainerInstancesRequest {
  cluster?: string;
  filter?: string;
  maxResults?: number;
  nextToken?: string;
  status?: string;
}

export interface ListContainerInstancesResponse {
  containerInstanceArns?: string[];
  nextToken?: string;
}

// Capacity Provider Types
export interface CapacityProvider {
  capacityProviderArn?: string;
  name?: string;
  status?: string;
  autoScalingGroupProvider?: AutoScalingGroupProvider;
  updateStatus?: string;
  updateStatusReason?: string;
  tags?: Tag[];
}

export interface CreateCapacityProviderRequest {
  name: string;
  autoScalingGroupProvider: AutoScalingGroupProvider;
  tags?: Tag[];
}

export interface CreateCapacityProviderResponse {
  capacityProvider?: CapacityProvider;
}

// Tagging Types
export interface TagResourceRequest {
  resourceArn: string;
  tags: Tag[];
}

export interface TagResourceResponse {}

export interface UntagResourceRequest {
  resourceArn: string;
  tagKeys: string[];
}

export interface UntagResourceResponse {}

export interface ListTagsForResourceRequest {
  resourceArn: string;
}

export interface ListTagsForResourceResponse {
  tags?: Tag[];
}

// Additional Supporting Types
export interface TaskOverride {
  containerOverrides?: ContainerOverride[];
  cpu?: string;
  memory?: string;
  taskRoleArn?: string;
  executionRoleArn?: string;
  inferenceAcceleratorOverrides?: InferenceAcceleratorOverride[];
  ephemeralStorage?: EphemeralStorage;
}

export interface ContainerOverride {
  name?: string;
  command?: string[];
  environment?: EnvironmentVariable[];
  environmentFiles?: EnvironmentFile[];
  cpu?: number;
  memory?: number;
  memoryReservation?: number;
  resourceRequirements?: ResourceRequirement[];
}

export interface NetworkConfiguration {
  awsvpcConfiguration?: AwsVpcConfiguration;
}

export interface AwsVpcConfiguration {
  subnets: string[];
  securityGroups?: string[];
  assignPublicIp?: string;
}

export interface PlacementStrategy {
  type?: string;
  field?: string;
}

export interface LoadBalancer {
  targetGroupArn?: string;
  loadBalancerName?: string;
  containerName?: string;
  containerPort?: number;
}

export interface ServiceRegistry {
  registryArn?: string;
  port?: number;
  containerName?: string;
  containerPort?: number;
}

export interface ServiceConnectConfiguration {
  enabled: boolean;
  namespace?: string;
  services?: ServiceConnectService[];
  logConfiguration?: LogConfiguration;
}

export interface ServiceConnectService {
  portName: string;
  discoveryName?: string;
  clientAliases?: ServiceConnectClientAlias[];
  ingressPortOverride?: number;
  timeout?: TimeoutConfiguration;
  tls?: ServiceConnectTlsConfiguration;
}

export interface ServiceConnectClientAlias {
  port: number;
  dnsName?: string;
}

export interface TimeoutConfiguration {
  idleTimeoutSeconds?: number;
  perRequestTimeoutSeconds?: number;
}

export interface ServiceConnectTlsConfiguration {
  issuerCertificateAuthority: ServiceConnectTlsCertificateAuthority;
  kmsKey?: string;
  roleArn?: string;
}

export interface ServiceConnectTlsCertificateAuthority {
  awsPcaAuthorityArn?: string;
}

export interface ServiceVolumeConfiguration {
  name: string;
  managedEBSVolume?: ServiceManagedEBSVolumeConfiguration;
}

export interface ServiceManagedEBSVolumeConfiguration {
  encrypted?: boolean;
  kmsKeyId?: string;
  volumeType?: string;
  sizeInGiB?: number;
  snapshotId?: string;
  iops?: number;
  throughput?: number;
  tagSpecifications?: EBSTagSpecification[];
  roleArn: string;
  filesystemType?: string;
}

export interface EBSTagSpecification {
  resourceType: string;
  tags?: Tag[];
  propagateTags?: string;
}

export interface ContainerImage {
  containerName?: string;
  imageDigest?: string;
  image?: string;
}

export interface CapacityProviderStrategyItem {
  capacityProvider: string;
  weight?: number;
  base?: number;
}

export interface Scale {
  value?: number;
  unit?: string;
}

export interface Resource {
  name?: string;
  type?: string;
  doubleValue?: number;
  longValue?: number;
  integerValue?: number;
  stringSetValue?: string[];
}

export interface VersionInfo {
  agentVersion?: string;
  agentHash?: string;
  dockerVersion?: string;
}

export interface Attachment {
  id?: string;
  type?: string;
  status?: string;
  details?: KeyValuePair[];
}

export interface KeyValuePair {
  name?: string;
  value?: string;
}

export interface ContainerInstanceHealthStatus {
  overallStatus?: string;
  details?: InstanceHealthCheckResult[];
}

export interface InstanceHealthCheckResult {
  type?: string;
  status?: string;
  lastUpdated?: string;
  lastStatusChange?: string;
}

export interface AutoScalingGroupProvider {
  autoScalingGroupArn: string;
  managedTerminationProtection?: string;
  managedDraining?: string;
}

export interface InferenceAcceleratorOverride {
  deviceName?: string;
  deviceType?: string;
}

export interface EphemeralStorage {
  sizeInGiB: number;
}

export interface EnvironmentFile {
  value: string;
  type: string;
}

export interface ResourceRequirement {
  value: string;
  type: string;
}

export interface LogConfiguration {
  logDriver: string;
  options?: { [key: string]: string };
  secretOptions?: Secret[];
}

export interface Secret {
  name: string;
  valueFrom: string;
}

export interface ClusterSetting {
  name?: string;
  value?: string;
}

export interface ClusterConfiguration {
  executeCommandConfiguration?: ExecuteCommandConfiguration;
  managedStorageConfiguration?: ManagedStorageConfiguration;
}

export interface ExecuteCommandConfiguration {
  kmsKeyId?: string;
  logging?: string;
  logConfiguration?: ExecuteCommandLogConfiguration;
}

export interface ExecuteCommandLogConfiguration {
  cloudWatchLogGroupName?: string;
  cloudWatchEncryptionEnabled?: boolean;
  s3BucketName?: string;
  s3EncryptionEnabled?: boolean;
  s3KeyPrefix?: string;
}

export interface ManagedStorageConfiguration {
  kmsKeyId?: string;
  fargateEphemeralStorageKmsKeyId?: string;
}

// Attribute Types (for container instances and tasks)
export interface Attribute {
  name: string;
  value?: string;
  targetType?: string;
  targetId?: string;
}

export interface PutAttributesRequest {
  cluster?: string;
  attributes: Attribute[];
}

export interface PutAttributesResponse {
  attributes?: Attribute[];
}

export interface ListAttributesRequest {
  cluster?: string;
  targetType: string;
  attributeName?: string;
  attributeValue?: string;
  maxResults?: number;
  nextToken?: string;
}

export interface ListAttributesResponse {
  attributes?: Attribute[];
  nextToken?: string;
}

export interface DeleteAttributesRequest {
  cluster?: string;
  attributes: Attribute[];
}

export interface DeleteAttributesResponse {
  attributes?: Attribute[];
}