import {
  DashboardStats,
  HealthStatus,
  ListClustersResponse,
  DescribeClustersResponse,
  ListServicesResponse,
  DescribeServicesResponse,
  ListTasksResponse,
  DescribeTasksResponse,
  ListTaskDefinitionsResponse,
  DescribeTaskDefinitionResponse,
  CreateServiceRequest,
  CreateServiceResponse,
  UpdateServiceRequest,
  UpdateServiceResponse,
  DeleteServiceRequest,
  DeleteServiceResponse,
  RegisterTaskDefinitionRequest,
  RegisterTaskDefinitionResponse,
  DeregisterTaskDefinitionRequest,
  DeregisterTaskDefinitionResponse,
  CreateClusterRequest,
  CreateClusterResponse,
  DeleteClusterRequest,
  DeleteClusterResponse,
  UpdateClusterRequest,
  UpdateClusterResponse,
  RunTaskRequest,
  RunTaskResponse,
  StopTaskRequest,
  StopTaskResponse,
  StartTaskRequest,
  StartTaskResponse,
  DeleteTaskDefinitionsRequest,
  DeleteTaskDefinitionsResponse,
  TagResourceRequest,
  TagResourceResponse,
  UntagResourceRequest,
  UntagResourceResponse,
  ListTagsForResourceRequest,
  ListTagsForResourceResponse,
  DescribeServiceDeploymentsRequest,
  DescribeServiceDeploymentsResponse,
  DescribeServiceRevisionsRequest,
  DescribeServiceRevisionsResponse,
  ListServiceDeploymentsRequest,
  ListServiceDeploymentsResponse,
  CreateTaskSetRequest,
  CreateTaskSetResponse,
  ListContainerInstancesRequest,
  ListContainerInstancesResponse,
  CreateCapacityProviderRequest,
  CreateCapacityProviderResponse,
  PutAttributesRequest,
  PutAttributesResponse,
  ListAttributesRequest,
  ListAttributesResponse,
  DeleteAttributesRequest,
  DeleteAttributesResponse,
} from '../types/api';

// In development, use empty string to leverage proxy
// In production, use full URL
const API_BASE_URL = process.env.NODE_ENV === 'development' ? '' : 'http://localhost:8080';

class KECSApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  private async makeRequest<T>(endpoint: string, target: string, body: any = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const config: RequestInit = {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-amz-json-1.1',
        'X-Amz-Target': target,
      },
      body: JSON.stringify(body),
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      return await response.json();
    } catch (error) {
      console.error(`API request failed: ${endpoint}`, error);
      throw error;
    }
  }

  // Health Check
  async checkHealth(): Promise<HealthStatus> {
    try {
      const response = await fetch(`${this.baseUrl}/health`);
      const isHealthy = response.ok;
      
      return {
        status: isHealthy ? 'connected' : 'error',
        message: isHealthy ? 'Connected to KECS Control Plane' : 'Failed to connect',
        timestamp: new Date().toISOString(),
      };
    } catch (error) {
      return {
        status: 'error',
        message: 'Connection failed',
        timestamp: new Date().toISOString(),
      };
    }
  }

  // Cluster Operations
  async listClusters(): Promise<ListClustersResponse> {
    return this.makeRequest<ListClustersResponse>(
      '/v1/ListClusters',
      'AmazonEC2ContainerServiceV20141113.ListClusters',
      {}
    );
  }

  async describeClusters(clusterNames?: string[]): Promise<DescribeClustersResponse> {
    return this.makeRequest<DescribeClustersResponse>(
      '/v1/DescribeClusters',
      'AmazonEC2ContainerServiceV20141113.DescribeClusters',
      { clusters: clusterNames || [] }
    );
  }

  // Service Operations
  async listServices(cluster?: string): Promise<ListServicesResponse> {
    return this.makeRequest<ListServicesResponse>(
      '/v1/ListServices',
      'AmazonEC2ContainerServiceV20141113.ListServices',
      { cluster: cluster || 'default' }
    );
  }

  async describeServices(serviceNames: string[], cluster?: string): Promise<DescribeServicesResponse> {
    return this.makeRequest<DescribeServicesResponse>(
      '/v1/DescribeServices',
      'AmazonEC2ContainerServiceV20141113.DescribeServices',
      { cluster: cluster || 'default', services: serviceNames }
    );
  }

  // Task Operations
  async listTasks(cluster?: string): Promise<ListTasksResponse> {
    return this.makeRequest<ListTasksResponse>(
      '/v1/ListTasks',
      'AmazonEC2ContainerServiceV20141113.ListTasks',
      { cluster: cluster || 'default' }
    );
  }

  async describeTasks(taskArns: string[], cluster?: string): Promise<DescribeTasksResponse> {
    return this.makeRequest<DescribeTasksResponse>(
      '/v1/DescribeTasks',
      'AmazonEC2ContainerServiceV20141113.DescribeTasks',
      { cluster: cluster || 'default', tasks: taskArns }
    );
  }

  // Task Definition Operations
  async listTaskDefinitions(): Promise<ListTaskDefinitionsResponse> {
    return this.makeRequest<ListTaskDefinitionsResponse>(
      '/v1/ListTaskDefinitions',
      'AmazonEC2ContainerServiceV20141113.ListTaskDefinitions',
      {}
    );
  }

  async describeTaskDefinition(taskDefinition: string): Promise<DescribeTaskDefinitionResponse> {
    return this.makeRequest<DescribeTaskDefinitionResponse>(
      '/v1/DescribeTaskDefinition',
      'AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition',
      { taskDefinition }
    );
  }

  async registerTaskDefinition(request: RegisterTaskDefinitionRequest): Promise<RegisterTaskDefinitionResponse> {
    return this.makeRequest<RegisterTaskDefinitionResponse>(
      '/v1/RegisterTaskDefinition',
      'AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition',
      request
    );
  }

  async deregisterTaskDefinition(request: DeregisterTaskDefinitionRequest): Promise<DeregisterTaskDefinitionResponse> {
    return this.makeRequest<DeregisterTaskDefinitionResponse>(
      '/v1/DeregisterTaskDefinition',
      'AmazonEC2ContainerServiceV20141113.DeregisterTaskDefinition',
      request
    );
  }

  // Dashboard Statistics
  async getDashboardStats(): Promise<DashboardStats> {
    try {
      // Get cluster list first
      const clustersResponse = await this.listClusters();
      
      // Get task definitions (cluster-independent)
      const taskDefsResponse = await this.listTaskDefinitions();
      
      let totalServices = 0;
      let totalTasks = 0;
      
      // For each cluster, get services and tasks
      if (clustersResponse.clusterArns.length > 0) {
        const clusterNames = clustersResponse.clusterArns.map(arn => {
          const parts = arn.split('/');
          return parts[parts.length - 1];
        });
        
        // Get services and tasks for all clusters
        const servicePromises = clusterNames.map(clusterName => 
          this.listServices(clusterName).catch(() => ({ serviceArns: [] }))
        );
        const taskPromises = clusterNames.map(clusterName => 
          this.listTasks(clusterName).catch(() => ({ taskArns: [] }))
        );
        
        const [serviceResponses, taskResponses] = await Promise.all([
          Promise.all(servicePromises),
          Promise.all(taskPromises)
        ]);
        
        totalServices = serviceResponses.reduce((sum, response) => 
          sum + (response.serviceArns?.length || 0), 0);
        totalTasks = taskResponses.reduce((sum, response) => 
          sum + (response.taskArns?.length || 0), 0);
      }

      return {
        clusters: clustersResponse.clusterArns.length,
        services: totalServices,
        tasks: totalTasks,
        taskDefinitions: taskDefsResponse.taskDefinitionArns.length,
      };
    } catch (error) {
      console.error('Failed to get dashboard stats:', error);
      return {
        clusters: 0,
        services: 0,
        tasks: 0,
        taskDefinitions: 0,
      };
    }
  }

  // Service Management Operations
  async createService(request: CreateServiceRequest): Promise<CreateServiceResponse> {
    return this.makeRequest<CreateServiceResponse>(
      '/v1/CreateService',
      'AmazonEC2ContainerServiceV20141113.CreateService',
      request
    );
  }

  async updateService(request: UpdateServiceRequest): Promise<UpdateServiceResponse> {
    return this.makeRequest<UpdateServiceResponse>(
      '/v1/UpdateService',
      'AmazonEC2ContainerServiceV20141113.UpdateService',
      request
    );
  }

  async deleteService(request: DeleteServiceRequest): Promise<DeleteServiceResponse> {
    return this.makeRequest<DeleteServiceResponse>(
      '/v1/DeleteService',
      'AmazonEC2ContainerServiceV20141113.DeleteService',
      request
    );
  }

  // Cluster Management Operations
  async createCluster(request: CreateClusterRequest): Promise<CreateClusterResponse> {
    return this.makeRequest<CreateClusterResponse>(
      '/v1/CreateCluster',
      'AmazonEC2ContainerServiceV20141113.CreateCluster',
      request
    );
  }

  async deleteCluster(request: DeleteClusterRequest): Promise<DeleteClusterResponse> {
    return this.makeRequest<DeleteClusterResponse>(
      '/v1/DeleteCluster',
      'AmazonEC2ContainerServiceV20141113.DeleteCluster',
      request
    );
  }

  async updateCluster(request: UpdateClusterRequest): Promise<UpdateClusterResponse> {
    return this.makeRequest<UpdateClusterResponse>(
      '/v1/UpdateCluster',
      'AmazonEC2ContainerServiceV20141113.UpdateCluster',
      request
    );
  }

  // Task Management Operations
  async runTask(request: RunTaskRequest): Promise<RunTaskResponse> {
    return this.makeRequest<RunTaskResponse>(
      '/v1/RunTask',
      'AmazonEC2ContainerServiceV20141113.RunTask',
      request
    );
  }

  async stopTask(request: StopTaskRequest): Promise<StopTaskResponse> {
    return this.makeRequest<StopTaskResponse>(
      '/v1/StopTask',
      'AmazonEC2ContainerServiceV20141113.StopTask',
      request
    );
  }

  async startTask(request: StartTaskRequest): Promise<StartTaskResponse> {
    return this.makeRequest<StartTaskResponse>(
      '/v1/StartTask',
      'AmazonEC2ContainerServiceV20141113.StartTask',
      request
    );
  }

  // Task Definition Batch Operations
  async deleteTaskDefinitions(request: DeleteTaskDefinitionsRequest): Promise<DeleteTaskDefinitionsResponse> {
    return this.makeRequest<DeleteTaskDefinitionsResponse>(
      '/v1/DeleteTaskDefinitions',
      'AmazonEC2ContainerServiceV20141113.DeleteTaskDefinitions',
      request
    );
  }

  // Tagging Operations
  async tagResource(request: TagResourceRequest): Promise<TagResourceResponse> {
    return this.makeRequest<TagResourceResponse>(
      '/v1/TagResource',
      'AmazonEC2ContainerServiceV20141113.TagResource',
      request
    );
  }

  async untagResource(request: UntagResourceRequest): Promise<UntagResourceResponse> {
    return this.makeRequest<UntagResourceResponse>(
      '/v1/UntagResource',
      'AmazonEC2ContainerServiceV20141113.UntagResource',
      request
    );
  }

  async listTagsForResource(request: ListTagsForResourceRequest): Promise<ListTagsForResourceResponse> {
    return this.makeRequest<ListTagsForResourceResponse>(
      '/v1/ListTagsForResource',
      'AmazonEC2ContainerServiceV20141113.ListTagsForResource',
      request
    );
  }

  // Service Deployment Operations
  async describeServiceDeployments(request: DescribeServiceDeploymentsRequest): Promise<DescribeServiceDeploymentsResponse> {
    return this.makeRequest<DescribeServiceDeploymentsResponse>(
      '/v1/DescribeServiceDeployments',
      'AmazonEC2ContainerServiceV20141113.DescribeServiceDeployments',
      request
    );
  }

  async describeServiceRevisions(request: DescribeServiceRevisionsRequest): Promise<DescribeServiceRevisionsResponse> {
    return this.makeRequest<DescribeServiceRevisionsResponse>(
      '/v1/DescribeServiceRevisions',
      'AmazonEC2ContainerServiceV20141113.DescribeServiceRevisions',
      request
    );
  }

  async listServiceDeployments(request: ListServiceDeploymentsRequest): Promise<ListServiceDeploymentsResponse> {
    return this.makeRequest<ListServiceDeploymentsResponse>(
      '/v1/ListServiceDeployments',
      'AmazonEC2ContainerServiceV20141113.ListServiceDeployments',
      request
    );
  }

  // Task Set Operations
  async createTaskSet(request: CreateTaskSetRequest): Promise<CreateTaskSetResponse> {
    return this.makeRequest<CreateTaskSetResponse>(
      '/v1/CreateTaskSet',
      'AmazonEC2ContainerServiceV20141113.CreateTaskSet',
      request
    );
  }

  // Container Instance Operations
  async listContainerInstances(request: ListContainerInstancesRequest): Promise<ListContainerInstancesResponse> {
    return this.makeRequest<ListContainerInstancesResponse>(
      '/v1/ListContainerInstances',
      'AmazonEC2ContainerServiceV20141113.ListContainerInstances',
      request
    );
  }

  // Capacity Provider Operations
  async createCapacityProvider(request: CreateCapacityProviderRequest): Promise<CreateCapacityProviderResponse> {
    return this.makeRequest<CreateCapacityProviderResponse>(
      '/v1/CreateCapacityProvider',
      'AmazonEC2ContainerServiceV20141113.CreateCapacityProvider',
      request
    );
  }

  // Attribute Operations
  async putAttributes(request: PutAttributesRequest): Promise<PutAttributesResponse> {
    return this.makeRequest<PutAttributesResponse>(
      '/v1/PutAttributes',
      'AmazonEC2ContainerServiceV20141113.PutAttributes',
      request
    );
  }

  async listAttributes(request: ListAttributesRequest): Promise<ListAttributesResponse> {
    return this.makeRequest<ListAttributesResponse>(
      '/v1/ListAttributes',
      'AmazonEC2ContainerServiceV20141113.ListAttributes',
      request
    );
  }

  async deleteAttributes(request: DeleteAttributesRequest): Promise<DeleteAttributesResponse> {
    return this.makeRequest<DeleteAttributesResponse>(
      '/v1/DeleteAttributes',
      'AmazonEC2ContainerServiceV20141113.DeleteAttributes',
      request
    );
  }
}

// Export singleton instance
export const apiClient = new KECSApiClient();
export default KECSApiClient;