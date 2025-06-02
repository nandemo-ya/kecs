import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '../services/api';
import { DashboardStats, HealthStatus } from '../types/api';

// Custom hook for dashboard stats
export function useDashboardStats() {
  const [stats, setStats] = useState<DashboardStats>({
    clusters: 0,
    services: 0,
    tasks: 0,
    taskDefinitions: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const newStats = await apiClient.getDashboardStats();
      setStats(newStats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch dashboard stats');
      console.error('Dashboard stats error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
    
    // Refresh stats every 30 seconds
    const interval = setInterval(fetchStats, 30000);
    
    return () => clearInterval(interval);
  }, [fetchStats]);

  return { stats, loading, error, refresh: fetchStats };
}

// Custom hook for health status
export function useHealthStatus() {
  const [health, setHealth] = useState<HealthStatus>({
    status: 'connecting',
    message: 'Connecting to KECS Control Plane...',
    timestamp: new Date().toISOString(),
  });

  const checkHealth = useCallback(async () => {
    try {
      const healthStatus = await apiClient.checkHealth();
      setHealth(healthStatus);
    } catch (err) {
      setHealth({
        status: 'error',
        message: 'Failed to connect to KECS Control Plane',
        timestamp: new Date().toISOString(),
      });
    }
  }, []);

  useEffect(() => {
    checkHealth();
    
    // Check health every 10 seconds
    const interval = setInterval(checkHealth, 10000);
    
    return () => clearInterval(interval);
  }, [checkHealth]);

  return { health, refresh: checkHealth };
}

// Generic API hook for any endpoint
export function useApiData<T>(
  apiCall: () => Promise<T>,
  dependencies: any[] = [],
  refreshInterval?: number
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiCall();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'API call failed');
      console.error('API error:', err);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiCall, ...dependencies]);

  useEffect(() => {
    fetchData();
    
    if (refreshInterval) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { data, loading, error, refresh: fetchData };
}