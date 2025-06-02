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

  private async makeRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
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
    return this.makeRequest<ListClustersResponse>('/v1/listclusters', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async describeClusters(clusterNames?: string[]): Promise<DescribeClustersResponse> {
    return this.makeRequest<DescribeClustersResponse>('/v1/describeclusters', {
      method: 'POST',
      body: JSON.stringify({
        clusters: clusterNames || [],
      }),
    });
  }

  // Service Operations
  async listServices(cluster?: string): Promise<ListServicesResponse> {
    return this.makeRequest<ListServicesResponse>('/v1/listservices', {
      method: 'POST',
      body: JSON.stringify({
        cluster: cluster || 'default',
      }),
    });
  }

  async describeServices(serviceNames: string[], cluster?: string): Promise<DescribeServicesResponse> {
    return this.makeRequest<DescribeServicesResponse>('/v1/describeservices', {
      method: 'POST',
      body: JSON.stringify({
        cluster: cluster || 'default',
        services: serviceNames,
      }),
    });
  }

  // Task Operations
  async listTasks(cluster?: string): Promise<ListTasksResponse> {
    return this.makeRequest<ListTasksResponse>('/v1/listtasks', {
      method: 'POST',
      body: JSON.stringify({
        cluster: cluster || 'default',
      }),
    });
  }

  async describeTasks(taskArns: string[], cluster?: string): Promise<DescribeTasksResponse> {
    return this.makeRequest<DescribeTasksResponse>('/v1/describetasks', {
      method: 'POST',
      body: JSON.stringify({
        cluster: cluster || 'default',
        tasks: taskArns,
      }),
    });
  }

  // Task Definition Operations
  async listTaskDefinitions(): Promise<ListTaskDefinitionsResponse> {
    return this.makeRequest<ListTaskDefinitionsResponse>('/v1/listtaskdefinitions', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async describeTaskDefinition(taskDefinition: string): Promise<DescribeTaskDefinitionResponse> {
    return this.makeRequest<DescribeTaskDefinitionResponse>('/v1/describetaskdefinition', {
      method: 'POST',
      body: JSON.stringify({
        taskDefinition,
      }),
    });
  }

  // Dashboard Statistics
  async getDashboardStats(): Promise<DashboardStats> {
    try {
      // Get counts from various endpoints
      const [clustersResponse, servicesResponse, tasksResponse, taskDefsResponse] = await Promise.all([
        this.listClusters(),
        this.listServices(),
        this.listTasks(),
        this.listTaskDefinitions(),
      ]);

      return {
        clusters: clustersResponse.clusterArns.length,
        services: servicesResponse.serviceArns.length,
        tasks: tasksResponse.taskArns.length,
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