// Task-related type definitions

export type TaskStatus = 
  | 'PROVISIONING'
  | 'PENDING'
  | 'ACTIVATING'
  | 'RUNNING'
  | 'DEACTIVATING'
  | 'STOPPING'
  | 'DEPROVISIONING'
  | 'STOPPED'
  | 'DELETED';

export type LaunchType = 'EC2' | 'FARGATE' | 'EXTERNAL';

export type HealthStatus = 'HEALTHY' | 'UNHEALTHY' | 'UNKNOWN';

export type Connectivity = 'CONNECTED' | 'DISCONNECTED';

export interface Task {
  taskArn: string;
  clusterArn: string;
  taskDefinitionArn: string;
  capacityProviderName?: string;
  connectivity?: Connectivity;
  connectivityAt?: string;
  containerInstanceArn?: string;
  containers?: Container[];
  cpu?: string;
  createdAt?: string;
  desiredStatus: TaskStatus;
  enableExecuteCommand?: boolean;
  ephemeralStorage?: EphemeralStorage;
  executionStoppedAt?: string;
  group?: string;
  healthStatus?: HealthStatus;
  inferenceAccelerators?: InferenceAccelerator[];
  lastStatus: TaskStatus;
  launchType?: LaunchType;
  memory?: string;
  overrides?: TaskOverride;
  platformFamily?: string;
  platformVersion?: string;
  pullStartedAt?: string;
  pullStoppedAt?: string;
  startedAt?: string;
  startedBy?: string;
  stopCode?: string;
  stoppedAt?: string;
  stoppedReason?: string;
  stoppingAt?: string;
  tags?: Tag[];
  version?: number;
  attributes?: Attribute[];
  availabilityZone?: string;
}

export interface Container {
  containerArn?: string;
  taskArn?: string;
  name: string;
  image?: string;
  imageDigest?: string;
  runtimeId?: string;
  lastStatus?: string;
  exitCode?: number;
  reason?: string;
  networkBindings?: NetworkBinding[];
  networkInterfaces?: NetworkInterface[];
  healthStatus?: HealthStatus;
  managedAgents?: ManagedAgent[];
  cpu?: string;
  memory?: string;
  memoryReservation?: string;
  gpuIds?: string[];
}

export interface NetworkBinding {
  bindIP?: string;
  containerPort?: number;
  hostPort?: number;
  protocol?: 'tcp' | 'udp';
  containerPortRange?: string;
  hostPortRange?: string;
}

export interface NetworkInterface {
  attachmentId?: string;
  privateIpv4Address?: string;
  ipv6Address?: string;
}

export interface ManagedAgent {
  lastStartedAt?: string;
  name?: string;
  reason?: string;
  lastStatus?: string;
}

export interface EphemeralStorage {
  sizeInGiB: number;
}

export interface InferenceAccelerator {
  deviceName: string;
  deviceType: string;
}

export interface TaskOverride {
  containerOverrides?: ContainerOverride[];
  cpu?: string;
  ephemeralStorage?: EphemeralStorage;
  executionRoleArn?: string;
  inferenceAcceleratorOverrides?: InferenceAcceleratorOverride[];
  memory?: string;
  taskRoleArn?: string;
}

export interface ContainerOverride {
  name?: string;
  command?: string[];
  environment?: KeyValuePair[];
  environmentFiles?: EnvironmentFile[];
  cpu?: number;
  memory?: number;
  memoryReservation?: number;
  resourceRequirements?: ResourceRequirement[];
}

export interface InferenceAcceleratorOverride {
  deviceName?: string;
  deviceType?: string;
}

export interface KeyValuePair {
  name?: string;
  value?: string;
}

export interface EnvironmentFile {
  value: string;
  type: 's3';
}

export interface ResourceRequirement {
  value: string;
  type: 'GPU' | 'InferenceAccelerator';
}

export interface Tag {
  key?: string;
  value?: string;
}

export interface Attribute {
  name: string;
  value?: string;
  targetType?: string;
  targetId?: string;
}

// Task Definition types
export interface TaskDefinition {
  taskDefinitionArn?: string;
  family: string;
  taskRoleArn?: string;
  executionRoleArn?: string;
  networkMode?: 'bridge' | 'host' | 'awsvpc' | 'none';
  revision?: number;
  volumes?: Volume[];
  status?: 'ACTIVE' | 'INACTIVE' | 'DELETE_IN_PROGRESS';
  requiresAttributes?: Attribute[];
  placementConstraints?: PlacementConstraint[];
  compatibilities?: string[];
  runtimePlatform?: RuntimePlatform;
  requiresCompatibilities?: string[];
  cpu?: string;
  memory?: string;
  pidMode?: 'host' | 'task';
  ipcMode?: 'host' | 'task' | 'none';
  proxyConfiguration?: ProxyConfiguration;
  registeredAt?: string;
  deregisteredAt?: string;
  registeredBy?: string;
  ephemeralStorage?: EphemeralStorage;
  containerDefinitions: ContainerDefinition[];
}

export interface ContainerDefinition {
  name: string;
  image: string;
  repositoryCredentials?: RepositoryCredentials;
  cpu?: number;
  memory?: number;
  memoryReservation?: number;
  links?: string[];
  portMappings?: PortMapping[];
  essential?: boolean;
  entryPoint?: string[];
  command?: string[];
  environment?: KeyValuePair[];
  environmentFiles?: EnvironmentFile[];
  mountPoints?: MountPoint[];
  volumesFrom?: VolumeFrom[];
  linuxParameters?: LinuxParameters;
  secrets?: Secret[];
  dependsOn?: ContainerDependency[];
  startTimeout?: number;
  stopTimeout?: number;
  hostname?: string;
  user?: string;
  workingDirectory?: string;
  disableNetworking?: boolean;
  privileged?: boolean;
  readonlyRootFilesystem?: boolean;
  dnsServers?: string[];
  dnsSearchDomains?: string[];
  extraHosts?: HostEntry[];
  dockerSecurityOptions?: string[];
  interactive?: boolean;
  pseudoTerminal?: boolean;
  dockerLabels?: Record<string, string>;
  ulimits?: Ulimit[];
  logConfiguration?: LogConfiguration;
  healthCheck?: HealthCheck;
  systemControls?: SystemControl[];
  resourceRequirements?: ResourceRequirement[];
  firelensConfiguration?: FirelensConfiguration;
}

export interface RepositoryCredentials {
  credentialsParameter: string;
}

export interface PortMapping {
  containerPort?: number;
  hostPort?: number;
  protocol?: 'tcp' | 'udp';
  name?: string;
  appProtocol?: string;
  containerPortRange?: string;
}

export interface MountPoint {
  sourceVolume?: string;
  containerPath?: string;
  readOnly?: boolean;
}

export interface VolumeFrom {
  sourceContainer?: string;
  readOnly?: boolean;
}

export interface LinuxParameters {
  capabilities?: KernelCapabilities;
  devices?: Device[];
  initProcessEnabled?: boolean;
  sharedMemorySize?: number;
  tmpfs?: Tmpfs[];
  maxSwap?: number;
  swappiness?: number;
}

export interface KernelCapabilities {
  add?: string[];
  drop?: string[];
}

export interface Device {
  hostPath: string;
  containerPath?: string;
  permissions?: string[];
}

export interface Tmpfs {
  containerPath: string;
  size: number;
  mountOptions?: string[];
}

export interface Secret {
  name: string;
  valueFrom: string;
}

export interface ContainerDependency {
  containerName: string;
  condition: 'START' | 'COMPLETE' | 'SUCCESS' | 'HEALTHY';
}

export interface HostEntry {
  hostname: string;
  ipAddress: string;
}

export interface Ulimit {
  name: string;
  softLimit: number;
  hardLimit: number;
}

export interface LogConfiguration {
  logDriver: string;
  options?: Record<string, string>;
  secretOptions?: Secret[];
}

export interface HealthCheck {
  command: string[];
  interval?: number;
  timeout?: number;
  retries?: number;
  startPeriod?: number;
}

export interface SystemControl {
  namespace?: string;
  value?: string;
}

export interface FirelensConfiguration {
  type: 'fluentd' | 'fluentbit';
  options?: Record<string, string>;
}

export interface Volume {
  name?: string;
  host?: HostVolumeProperties;
  dockerVolumeConfiguration?: DockerVolumeConfiguration;
  efsVolumeConfiguration?: EFSVolumeConfiguration;
  fsxWindowsFileServerVolumeConfiguration?: FSxWindowsFileServerVolumeConfiguration;
}

export interface HostVolumeProperties {
  sourcePath?: string;
}

export interface DockerVolumeConfiguration {
  scope?: 'task' | 'shared';
  autoprovision?: boolean;
  driver?: string;
  driverOpts?: Record<string, string>;
  labels?: Record<string, string>;
}

export interface EFSVolumeConfiguration {
  fileSystemId: string;
  rootDirectory?: string;
  transitEncryption?: 'ENABLED' | 'DISABLED';
  transitEncryptionPort?: number;
  authorizationConfig?: EFSAuthorizationConfig;
}

export interface EFSAuthorizationConfig {
  accessPointId?: string;
  iam?: 'ENABLED' | 'DISABLED';
}

export interface FSxWindowsFileServerVolumeConfiguration {
  fileSystemId: string;
  rootDirectory: string;
  authorizationConfig: FSxWindowsFileServerAuthorizationConfig;
}

export interface FSxWindowsFileServerAuthorizationConfig {
  credentialsParameter: string;
  domain: string;
}

export interface PlacementConstraint {
  type?: 'memberOf';
  expression?: string;
}

export interface RuntimePlatform {
  cpuArchitecture?: 'X86_64' | 'ARM64';
  operatingSystemFamily?: 'WINDOWS_SERVER_2019_FULL' | 'WINDOWS_SERVER_2019_CORE' | 'WINDOWS_SERVER_2016_FULL' | 'WINDOWS_SERVER_2004_CORE' | 'WINDOWS_SERVER_2022_CORE' | 'WINDOWS_SERVER_2022_FULL' | 'WINDOWS_SERVER_20H2_CORE' | 'LINUX';
}

export interface ProxyConfiguration {
  type?: 'APPMESH';
  containerName: string;
  properties?: KeyValuePair[];
}