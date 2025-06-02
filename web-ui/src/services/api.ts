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
}

// Export singleton instance
export const apiClient = new KECSApiClient();
export default KECSApiClient;