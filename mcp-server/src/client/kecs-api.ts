import axios, { AxiosInstance, AxiosError } from 'axios';
import { logger } from '../utils/logger.js';

export interface KecsApiConfig {
  baseUrl: string;
  timeout?: number;
  headers?: Record<string, string>;
}

export interface Cluster {
  clusterArn: string;
  clusterName: string;
  status: string;
  registeredContainerInstancesCount?: number;
  runningTasksCount?: number;
  pendingTasksCount?: number;
  activeServicesCount?: number;
  statistics?: Array<{ name: string; value: string }>;
  tags?: Array<{ key: string; value: string }>;
  settings?: Array<{ name: string; value: string }>;
}

export interface Service {
  serviceArn: string;
  serviceName: string;
  clusterArn: string;
  status: string;
  desiredCount: number;
  runningCount: number;
  pendingCount: number;
  taskDefinition: string;
  deployments?: Array<{
    id: string;
    status: string;
    taskDefinition: string;
    desiredCount: number;
    runningCount: number;
    pendingCount: number;
  }>;
}

export interface Task {
  taskArn: string;
  clusterArn: string;
  taskDefinitionArn: string;
  containerInstanceArn?: string;
  lastStatus: string;
  desiredStatus: string;
  cpu?: string;
  memory?: string;
  containers?: Array<{
    name: string;
    containerArn: string;
    lastStatus: string;
  }>;
}

export interface TaskDefinition {
  taskDefinitionArn: string;
  family: string;
  revision: number;
  status: string;
  cpu?: string;
  memory?: string;
  networkMode?: string;
  containerDefinitions: Array<{
    name: string;
    image: string;
    cpu?: number;
    memory?: number;
    memoryReservation?: number;
    portMappings?: Array<{
      containerPort: number;
      protocol?: string;
    }>;
    environment?: Array<{
      name: string;
      value: string;
    }>;
  }>;
}

export class KecsApiClient {
  private client: AxiosInstance;

  constructor(config: KecsApiConfig) {
    this.client = axios.create({
      baseURL: config.baseUrl,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/x-amz-json-1.1',
        ...config.headers,
      },
    });

    // Add request/response logging
    this.client.interceptors.request.use(
      (config) => {
        logger.debug(`API Request: ${config.method?.toUpperCase()} ${config.url}`, {
          headers: config.headers,
          data: config.data,
        });
        return config;
      },
      (error) => {
        logger.error('API Request Error:', error);
        return Promise.reject(error);
      }
    );

    this.client.interceptors.response.use(
      (response) => {
        logger.debug(`API Response: ${response.status} ${response.config.url}`, {
          data: response.data,
        });
        return response;
      },
      (error: AxiosError) => {
        logger.error(`API Response Error: ${error.response?.status} ${error.config?.url}`, {
          data: error.response?.data,
        });
        return Promise.reject(error);
      }
    );
  }

  private async ecsRequest<T>(action: string, params: any = {}): Promise<T> {
    const response = await this.client.post('/v1/' + action, params, {
      headers: {
        'X-Amz-Target': `AmazonEC2ContainerServiceV20141113.${action}`,
      },
    });
    return response.data;
  }

  // Cluster operations
  async listClusters(params?: { nextToken?: string; maxResults?: number }): Promise<{
    clusterArns: string[];
    nextToken?: string;
  }> {
    return this.ecsRequest('ListClusters', params);
  }

  async describeClusters(params: { clusters?: string[]; include?: string[] }): Promise<{
    clusters: Cluster[];
    failures?: Array<{ arn: string; reason: string }>;
  }> {
    return this.ecsRequest('DescribeClusters', params);
  }

  async createCluster(params: { clusterName: string; tags?: Array<{ key: string; value: string }> }): Promise<{
    cluster: Cluster;
  }> {
    return this.ecsRequest('CreateCluster', params);
  }

  async deleteCluster(params: { cluster: string }): Promise<{
    cluster: Cluster;
  }> {
    return this.ecsRequest('DeleteCluster', params);
  }

  // Service operations
  async listServices(params: {
    cluster?: string;
    nextToken?: string;
    maxResults?: number;
  }): Promise<{
    serviceArns: string[];
    nextToken?: string;
  }> {
    return this.ecsRequest('ListServices', params);
  }

  async describeServices(params: {
    cluster?: string;
    services: string[];
    include?: string[];
  }): Promise<{
    services: Service[];
    failures?: Array<{ arn: string; reason: string }>;
  }> {
    return this.ecsRequest('DescribeServices', params);
  }

  async createService(params: {
    cluster?: string;
    serviceName: string;
    taskDefinition: string;
    desiredCount?: number;
    launchType?: string;
    platformVersion?: string;
    role?: string;
    deploymentConfiguration?: any;
    placementConstraints?: any[];
    placementStrategy?: any[];
    networkConfiguration?: any;
    healthCheckGracePeriodSeconds?: number;
    schedulingStrategy?: string;
    tags?: Array<{ key: string; value: string }>;
  }): Promise<{
    service: Service;
  }> {
    return this.ecsRequest('CreateService', params);
  }

  async updateService(params: {
    cluster?: string;
    service: string;
    desiredCount?: number;
    taskDefinition?: string;
    deploymentConfiguration?: any;
    networkConfiguration?: any;
    platformVersion?: string;
    forceNewDeployment?: boolean;
  }): Promise<{
    service: Service;
  }> {
    return this.ecsRequest('UpdateService', params);
  }

  async deleteService(params: {
    cluster?: string;
    service: string;
    force?: boolean;
  }): Promise<{
    service: Service;
  }> {
    return this.ecsRequest('DeleteService', params);
  }

  // Task operations
  async listTasks(params: {
    cluster?: string;
    containerInstance?: string;
    family?: string;
    serviceName?: string;
    desiredStatus?: string;
    startedBy?: string;
    nextToken?: string;
    maxResults?: number;
  }): Promise<{
    taskArns: string[];
    nextToken?: string;
  }> {
    return this.ecsRequest('ListTasks', params);
  }

  async describeTasks(params: {
    cluster?: string;
    tasks: string[];
    include?: string[];
  }): Promise<{
    tasks: Task[];
    failures?: Array<{ arn: string; reason: string }>;
  }> {
    return this.ecsRequest('DescribeTasks', params);
  }

  async runTask(params: {
    cluster?: string;
    taskDefinition: string;
    count?: number;
    startedBy?: string;
    group?: string;
    placementConstraints?: any[];
    placementStrategy?: any[];
    launchType?: string;
    platformVersion?: string;
    networkConfiguration?: any;
    overrides?: any;
    tags?: Array<{ key: string; value: string }>;
  }): Promise<{
    tasks: Task[];
    failures?: Array<{ arn: string; reason: string }>;
  }> {
    return this.ecsRequest('RunTask', params);
  }

  async stopTask(params: {
    cluster?: string;
    task: string;
    reason?: string;
  }): Promise<{
    task: Task;
  }> {
    return this.ecsRequest('StopTask', params);
  }

  // Task Definition operations
  async listTaskDefinitions(params?: {
    familyPrefix?: string;
    status?: string;
    sort?: string;
    nextToken?: string;
    maxResults?: number;
  }): Promise<{
    taskDefinitionArns: string[];
    nextToken?: string;
  }> {
    return this.ecsRequest('ListTaskDefinitions', params);
  }

  async describeTaskDefinition(params: {
    taskDefinition: string;
    include?: string[];
  }): Promise<{
    taskDefinition: TaskDefinition;
    tags?: Array<{ key: string; value: string }>;
  }> {
    return this.ecsRequest('DescribeTaskDefinition', params);
  }

  async registerTaskDefinition(params: {
    family: string;
    taskRoleArn?: string;
    executionRoleArn?: string;
    networkMode?: string;
    containerDefinitions: any[];
    volumes?: any[];
    placementConstraints?: any[];
    requiresCompatibilities?: string[];
    cpu?: string;
    memory?: string;
    tags?: Array<{ key: string; value: string }>;
  }): Promise<{
    taskDefinition: TaskDefinition;
  }> {
    return this.ecsRequest('RegisterTaskDefinition', params);
  }

  async deregisterTaskDefinition(params: {
    taskDefinition: string;
  }): Promise<{
    taskDefinition: TaskDefinition;
  }> {
    return this.ecsRequest('DeregisterTaskDefinition', params);
  }
}