import { useState, useCallback, useEffect } from 'react';
import { useWebSocket, useWebSocketSubscription } from './useWebSocket';

export interface MetricPoint {
  timestamp: Date;
  value: number;
  metadata?: Record<string, any>;
}

export interface MetricSeries {
  name: string;
  points: MetricPoint[];
  unit?: string;
  aggregation?: 'avg' | 'sum' | 'min' | 'max' | 'count';
}

export interface ResourceMetrics {
  taskId?: string;
  serviceName?: string;
  containerId?: string;
  cpu: MetricSeries;
  memory: MetricSeries;
  network?: {
    rx: MetricSeries;
    tx: MetricSeries;
  };
  disk?: {
    read: MetricSeries;
    write: MetricSeries;
  };
  custom?: Record<string, MetricSeries>;
}

interface UseWebSocketMetricsOptions {
  taskIds?: string[];
  serviceNames?: string[];
  containerIds?: string[];
  metrics?: string[];
  interval?: number; // Update interval in milliseconds
  retention?: number; // Data retention in minutes
  aggregation?: 'avg' | 'sum' | 'min' | 'max' | 'count';
}

interface UseWebSocketMetricsResult {
  metrics: ResourceMetrics[];
  isConnected: boolean;
  isReconnecting: boolean;
  error: Error | null;
  connect: () => void;
  disconnect: () => void;
  subscribe: (resourceId: string) => void;
  unsubscribe: (resourceId: string) => void;
  setOptions: (options: UseWebSocketMetricsOptions) => void;
  getHistory: (resourceId: string, duration: number) => void;
}

export function useWebSocketMetrics(options: UseWebSocketMetricsOptions = {}): UseWebSocketMetricsResult {
  const {
    taskIds = [],
    serviceNames = [],
    containerIds = [],
    metrics = ['cpu', 'memory'],
    interval = 5000,
    retention = 60,
    aggregation = 'avg',
  } = options;

  const [metricsData, setMetricsData] = useState<Map<string, ResourceMetrics>>(new Map());
  const [subscriptions, setSubscriptions] = useState<Set<string>>(new Set());

  // Build WebSocket parameters
  const wsParams: Record<string, string> = {
    interval: interval.toString(),
    retention: retention.toString(),
    aggregation,
  };

  // Initialize WebSocket connection
  const ws = useWebSocket({
    path: '/ws/metrics',
    params: wsParams,
    reconnect: true,
    reconnectInterval: 5000,
    maxReconnectAttempts: 10,
    onConnected: () => {
      console.log('Metrics WebSocket connected');
      // Re-subscribe to all resources
      subscriptions.forEach(resourceId => {
        ws.send({
          type: 'subscribe',
          payload: { resourceId, metrics },
        });
      });
    },
    onDisconnected: () => {
      console.log('Metrics WebSocket disconnected');
    },
  });

  // Handle metric updates
  const handleMetricUpdate = useCallback((data: {
    resourceId: string;
    metrics: ResourceMetrics;
  }) => {
    setMetricsData(prev => {
      const updated = new Map(prev);
      updated.set(data.resourceId, data.metrics);
      return updated;
    });
  }, []);

  // Handle metric batch updates
  const handleMetricBatch = useCallback((data: Array<{
    resourceId: string;
    metrics: ResourceMetrics;
  }>) => {
    setMetricsData(prev => {
      const updated = new Map(prev);
      data.forEach(({ resourceId, metrics }) => {
        updated.set(resourceId, metrics);
      });
      return updated;
    });
  }, []);

  // Handle historical data
  const handleHistoricalData = useCallback((data: {
    resourceId: string;
    metrics: ResourceMetrics;
  }) => {
    setMetricsData(prev => {
      const updated = new Map(prev);
      const existing = updated.get(data.resourceId);
      
      if (existing) {
        // Merge historical data with existing
        const merged: ResourceMetrics = {
          ...existing,
          cpu: mergeMetricSeries(data.metrics.cpu, existing.cpu),
          memory: mergeMetricSeries(data.metrics.memory, existing.memory),
        };
        
        if (data.metrics.network && existing.network) {
          merged.network = {
            rx: mergeMetricSeries(data.metrics.network.rx, existing.network.rx),
            tx: mergeMetricSeries(data.metrics.network.tx, existing.network.tx),
          };
        }
        
        updated.set(data.resourceId, merged);
      } else {
        updated.set(data.resourceId, data.metrics);
      }
      
      return updated;
    });
  }, []);

  // Subscribe to WebSocket messages
  useWebSocketSubscription(ws, 'metric_update', handleMetricUpdate, [handleMetricUpdate]);
  useWebSocketSubscription(ws, 'metric_batch', handleMetricBatch, [handleMetricBatch]);
  useWebSocketSubscription(ws, 'historical_data', handleHistoricalData, [handleHistoricalData]);

  // Subscribe to a resource
  const subscribe = useCallback((resourceId: string) => {
    if (subscriptions.has(resourceId)) return;

    setSubscriptions(prev => {
      const updated = new Set(prev);
      updated.add(resourceId);
      return updated;
    });

    ws.send({
      type: 'subscribe',
      payload: { resourceId, metrics },
    });
  }, [ws, metrics, subscriptions]);

  // Unsubscribe from a resource
  const unsubscribe = useCallback((resourceId: string) => {
    if (!subscriptions.has(resourceId)) return;

    setSubscriptions(prev => {
      const updated = new Set(prev);
      updated.delete(resourceId);
      return updated;
    });

    ws.send({
      type: 'unsubscribe',
      payload: { resourceId },
    });

    // Remove metrics data
    setMetricsData(prev => {
      const updated = new Map(prev);
      updated.delete(resourceId);
      return updated;
    });
  }, [ws, subscriptions]);

  // Set options
  const setOptions = useCallback((newOptions: UseWebSocketMetricsOptions) => {
    ws.send({
      type: 'update_options',
      payload: newOptions,
    });
  }, [ws]);

  // Get historical data
  const getHistory = useCallback((resourceId: string, duration: number) => {
    ws.send({
      type: 'get_history',
      payload: { resourceId, duration },
    });
  }, [ws]);

  // Auto-subscribe to initial resources
  useEffect(() => {
    if (!ws.isConnected) return;

    const resourceIds = [
      ...taskIds.map(id => `task:${id}`),
      ...serviceNames.map(name => `service:${name}`),
      ...containerIds.map(id => `container:${id}`),
    ];

    resourceIds.forEach(subscribe);

    return () => {
      resourceIds.forEach(unsubscribe);
    };
  }, [ws.isConnected, taskIds, serviceNames, containerIds, subscribe, unsubscribe]);

  return {
    metrics: Array.from(metricsData.values()),
    isConnected: ws.isConnected,
    isReconnecting: ws.isReconnecting,
    error: ws.error,
    connect: ws.connect,
    disconnect: ws.disconnect,
    subscribe,
    unsubscribe,
    setOptions,
    getHistory,
  };
}

// Helper function to merge metric series
function mergeMetricSeries(historical: MetricSeries, current: MetricSeries): MetricSeries {
  const allPoints = [...historical.points, ...current.points];
  
  // Remove duplicates and sort by timestamp
  const uniquePoints = Array.from(
    new Map(allPoints.map(p => [p.timestamp.getTime(), p])).values()
  ).sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());

  return {
    ...current,
    points: uniquePoints,
  };
}