import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient } from '../services/api';
import { 
  ResourceMetrics, 
  ServiceMetrics, 
  TaskDefinitionMetrics,
  DashboardMetrics,
  MetricsHistoryEntry,
  TimeRange,
  TimeSeriesData 
} from '../types/metrics';

interface UseMetricsOptions {
  autoCollect?: boolean;
  interval?: number; // in milliseconds
  maxHistoryEntries?: number;
}

export function useMetrics(options: UseMetricsOptions = {}) {
  const { 
    autoCollect = false, 
    interval = 60000, // 1 minute default
    maxHistoryEntries = 1000 
  } = options;

  const [metricsHistory, setMetricsHistory] = useState<MetricsHistoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const collectMetrics = useCallback(async (): Promise<MetricsHistoryEntry | null> => {
    try {
      setLoading(true);
      setError(null);

      const timestamp = Date.now();

      // Collect dashboard metrics
      const dashboardStats = await apiClient.getDashboardStats();
      
      // Collect cluster data
      const clustersResponse = await apiClient.listClusters();
      const clusterMetrics: ResourceMetrics[] = [];

      for (const clusterArn of clustersResponse.clusterArns) {
        const clusterName = clusterArn.split('/').pop() || 'unknown';
        
        try {
          // Get services for this cluster
          const servicesResponse = await apiClient.listServices(clusterName);
          const tasksResponse = await apiClient.listTasks(clusterName);

          clusterMetrics.push({
            clusterName,
            servicesCount: servicesResponse.serviceArns.length,
            tasksCount: tasksResponse.taskArns.length,
            runningTasks: Math.floor(tasksResponse.taskArns.length * 0.7), // Mock data
            pendingTasks: Math.floor(tasksResponse.taskArns.length * 0.2), // Mock data
            stoppedTasks: Math.floor(tasksResponse.taskArns.length * 0.1), // Mock data
            timestamp,
          });
        } catch (err) {
          console.warn(`Failed to collect metrics for cluster ${clusterName}:`, err);
        }
      }

      // Collect service metrics (sample from first few services)
      const serviceMetrics: ServiceMetrics[] = [];
      if (clusterMetrics.length > 0) {
        const firstCluster = clusterMetrics[0];
        try {
          const servicesResponse = await apiClient.listServices(firstCluster.clusterName);
          const sampleServices = servicesResponse.serviceArns.slice(0, 5); // Sample first 5 services

          for (const serviceArn of sampleServices) {
            const serviceName = serviceArn.split('/').pop() || 'unknown';
            try {
              const serviceDetails = await apiClient.describeServices([serviceName], firstCluster.clusterName);
              if (serviceDetails.services.length > 0) {
                const service = serviceDetails.services[0];
                serviceMetrics.push({
                  serviceName: service.serviceName,
                  clusterName: firstCluster.clusterName,
                  desiredCount: service.desiredCount,
                  runningCount: service.runningCount,
                  pendingCount: service.pendingCount,
                  cpuUtilization: Math.random() * 100, // Mock data
                  memoryUtilization: Math.random() * 100, // Mock data
                  timestamp,
                });
              }
            } catch (err) {
              console.warn(`Failed to collect metrics for service ${serviceName}:`, err);
            }
          }
        } catch (err) {
          console.warn('Failed to collect service metrics:', err);
        }
      }

      // Collect task definition metrics
      const taskDefinitionMetrics: TaskDefinitionMetrics[] = [];
      try {
        const taskDefsResponse = await apiClient.listTaskDefinitions();
        const families = new Map<string, number>();
        
        taskDefsResponse.taskDefinitionArns.forEach(arn => {
          const parts = arn.split('/').pop()?.split(':');
          if (parts && parts.length >= 1) {
            const family = parts[0];
            families.set(family, (families.get(family) || 0) + 1);
          }
        });

        families.forEach((count, family) => {
          taskDefinitionMetrics.push({
            family,
            activeRevisions: count,
            totalTasks: Math.floor(Math.random() * 20), // Mock data
            timestamp,
          });
        });
      } catch (err) {
        console.warn('Failed to collect task definition metrics:', err);
      }

      const dashboardMetrics: DashboardMetrics = {
        totalClusters: dashboardStats.clusters,
        totalServices: dashboardStats.services,
        totalTasks: dashboardStats.tasks,
        totalTaskDefinitions: dashboardStats.taskDefinitions,
        healthyServices: Math.floor(dashboardStats.services * 0.8), // Mock data
        unhealthyServices: Math.floor(dashboardStats.services * 0.2), // Mock data
        runningTasks: Math.floor(dashboardStats.tasks * 0.7), // Mock data
        pendingTasks: Math.floor(dashboardStats.tasks * 0.2), // Mock data
        stoppedTasks: Math.floor(dashboardStats.tasks * 0.1), // Mock data
        timestamp,
      };

      const entry: MetricsHistoryEntry = {
        timestamp,
        clusters: clusterMetrics,
        services: serviceMetrics,
        taskDefinitions: taskDefinitionMetrics,
        dashboard: dashboardMetrics,
      };

      return entry;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to collect metrics';
      setError(errorMessage);
      console.error('Metrics collection error:', err);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const addMetricsEntry = useCallback((entry: MetricsHistoryEntry) => {
    setMetricsHistory(prev => {
      const newHistory = [...prev, entry];
      // Keep only the most recent entries
      if (newHistory.length > maxHistoryEntries) {
        return newHistory.slice(-maxHistoryEntries);
      }
      return newHistory;
    });
  }, [maxHistoryEntries]);

  const collectAndStore = useCallback(async () => {
    const entry = await collectMetrics();
    if (entry) {
      addMetricsEntry(entry);
    }
  }, [collectMetrics, addMetricsEntry]);

  const startAutoCollection = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    
    intervalRef.current = setInterval(collectAndStore, interval);
    // Collect immediately
    collectAndStore();
  }, [collectAndStore, interval]);

  const stopAutoCollection = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const getTimeSeriesData = useCallback((
    metricKey: keyof DashboardMetrics, 
    timeRange: TimeRange = '24h'
  ): TimeSeriesData => {
    const now = Date.now();
    const ranges = {
      '1h': 60 * 60 * 1000,
      '6h': 6 * 60 * 60 * 1000,
      '24h': 24 * 60 * 60 * 1000,
      '7d': 7 * 24 * 60 * 60 * 1000,
      '30d': 30 * 24 * 60 * 60 * 1000,
    };
    
    const timeRangeMs = ranges[timeRange];
    const cutoffTime = now - timeRangeMs;
    
    const filteredHistory = metricsHistory.filter(entry => entry.timestamp >= cutoffTime);
    
    return {
      name: metricKey,
      data: filteredHistory.map(entry => ({
        timestamp: entry.timestamp,
        value: entry.dashboard[metricKey] as number,
      })),
    };
  }, [metricsHistory]);

  useEffect(() => {
    if (autoCollect) {
      startAutoCollection();
    }

    return () => {
      stopAutoCollection();
    };
  }, [autoCollect, startAutoCollection, stopAutoCollection]);

  return {
    metricsHistory,
    loading,
    error,
    collectMetrics: collectAndStore,
    startAutoCollection,
    stopAutoCollection,
    getTimeSeriesData,
    isCollecting: intervalRef.current !== null,
  };
}

// Hook for getting latest metrics
export function useLatestMetrics() {
  const { metricsHistory } = useMetrics();
  
  const latestEntry = metricsHistory.length > 0 
    ? metricsHistory[metricsHistory.length - 1] 
    : null;

  return {
    latest: latestEntry,
    dashboard: latestEntry?.dashboard || null,
    clusters: latestEntry?.clusters || [],
    services: latestEntry?.services || [],
    taskDefinitions: latestEntry?.taskDefinitions || [],
  };
}