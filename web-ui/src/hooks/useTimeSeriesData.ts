import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  TimeSeriesData,
  TimeSeriesDataPoint,
  TimeSeriesAggregation,
  TimeRange,
} from '../types/timeseries';

interface UseTimeSeriesDataOptions {
  autoRefresh?: boolean;
  refreshInterval?: number;
  maxDataPoints?: number;
  aggregation?: TimeSeriesAggregation;
}

interface UseTimeSeriesDataResult {
  data: TimeSeriesData[];
  loading: boolean;
  error: Error | null;
  refresh: () => void;
  addDataPoint: (seriesId: string, dataPoint: TimeSeriesDataPoint) => void;
  setTimeRange: (range: TimeRange) => void;
  aggregateData: (data: TimeSeriesDataPoint[], aggregation: TimeSeriesAggregation) => TimeSeriesDataPoint[];
}

export function useTimeSeriesData(
  dataSource: string | (() => Promise<TimeSeriesData[]>) | TimeSeriesData[],
  options: UseTimeSeriesDataOptions = {}
): UseTimeSeriesDataResult {
  const [data, setData] = useState<TimeSeriesData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange | null>(null);

  const {
    autoRefresh = false,
    refreshInterval = 30000,
    maxDataPoints = 1000,
    aggregation,
  } = options;

  // Data aggregation function
  const aggregateData = useCallback((
    dataPoints: TimeSeriesDataPoint[],
    agg: TimeSeriesAggregation
  ): TimeSeriesDataPoint[] => {
    if (agg.type === 'none') return dataPoints;

    const intervalMs = getIntervalMilliseconds(agg.interval);
    const groupedData = new Map<number, TimeSeriesDataPoint[]>();

    // Group data points by time intervals
    dataPoints.forEach(point => {
      const intervalKey = Math.floor(point.timestamp / intervalMs) * intervalMs;
      if (!groupedData.has(intervalKey)) {
        groupedData.set(intervalKey, []);
      }
      groupedData.get(intervalKey)!.push(point);
    });

    // Aggregate each group
    const aggregatedPoints: TimeSeriesDataPoint[] = [];
    groupedData.forEach((points, timestamp) => {
      const values = points.map(p => p.value);
      let aggregatedValue: number;

      switch (agg.type) {
        case 'average':
          aggregatedValue = values.reduce((a, b) => a + b, 0) / values.length;
          break;
        case 'sum':
          aggregatedValue = values.reduce((a, b) => a + b, 0);
          break;
        case 'min':
          aggregatedValue = Math.min(...values);
          break;
        case 'max':
          aggregatedValue = Math.max(...values);
          break;
        case 'count':
          aggregatedValue = values.length;
          break;
        default:
          aggregatedValue = values[0] || 0;
      }

      aggregatedPoints.push({
        timestamp,
        value: aggregatedValue,
        label: `${agg.type} of ${points.length} points`,
        metadata: {
          originalPointCount: points.length,
          aggregationType: agg.type,
        },
      });
    });

    // Apply smoothing if specified
    if (agg.smoothing && agg.smoothing !== 'none') {
      return applySmoothingFilter(aggregatedPoints, agg.smoothing, agg.windowSize || 3);
    }

    return aggregatedPoints.sort((a, b) => a.timestamp - b.timestamp);
  }, []);

  // Smoothing filters
  const applySmoothingFilter = useCallback((
    dataPoints: TimeSeriesDataPoint[],
    smoothingType: 'moving-average' | 'exponential',
    windowSize: number
  ): TimeSeriesDataPoint[] => {
    if (smoothingType === 'moving-average') {
      return dataPoints.map((point, index) => {
        const startIndex = Math.max(0, index - Math.floor(windowSize / 2));
        const endIndex = Math.min(dataPoints.length, index + Math.ceil(windowSize / 2));
        const window = dataPoints.slice(startIndex, endIndex);
        const smoothedValue = window.reduce((sum, p) => sum + p.value, 0) / window.length;
        
        return {
          ...point,
          value: smoothedValue,
          metadata: {
            ...point.metadata,
            smoothed: true,
            originalValue: point.value,
          },
        };
      });
    } else if (smoothingType === 'exponential') {
      const alpha = 2 / (windowSize + 1);
      let ema = dataPoints[0]?.value || 0;
      
      return dataPoints.map((point, index) => {
        if (index === 0) {
          ema = point.value;
        } else {
          ema = alpha * point.value + (1 - alpha) * ema;
        }
        
        return {
          ...point,
          value: ema,
          metadata: {
            ...point.metadata,
            smoothed: true,
            originalValue: point.value,
          },
        };
      });
    }
    
    return dataPoints;
  }, []);

  // Helper function to convert interval to milliseconds
  const getIntervalMilliseconds = (interval: string): number => {
    switch (interval) {
      case 'minute':
        return 60 * 1000;
      case 'hour':
        return 60 * 60 * 1000;
      case 'day':
        return 24 * 60 * 60 * 1000;
      case 'week':
        return 7 * 24 * 60 * 60 * 1000;
      default:
        return 60 * 1000;
    }
  };

  // Load data function
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      let newData: TimeSeriesData[];

      if (Array.isArray(dataSource)) {
        newData = dataSource;
      } else if (typeof dataSource === 'function') {
        newData = await dataSource();
      } else if (typeof dataSource === 'string') {
        // Fetch from API endpoint
        const response = await fetch(dataSource);
        if (!response.ok) {
          throw new Error(`Failed to fetch data: ${response.statusText}`);
        }
        newData = await response.json();
      } else {
        throw new Error('Invalid data source');
      }

      // Apply time range filter
      if (timeRange) {
        newData = newData.map(series => ({
          ...series,
          data: series.data.filter(
            point => point.timestamp >= timeRange.start && point.timestamp <= timeRange.end
          ),
        }));
      }

      // Apply aggregation
      if (aggregation) {
        newData = newData.map(series => ({
          ...series,
          data: aggregateData(series.data, aggregation),
        }));
      }

      // Limit data points to prevent performance issues
      if (maxDataPoints > 0) {
        newData = newData.map(series => ({
          ...series,
          data: series.data.slice(-maxDataPoints),
        }));
      }

      setData(newData);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error occurred'));
    } finally {
      setLoading(false);
    }
  }, [dataSource, timeRange, aggregation, maxDataPoints, aggregateData]);

  // Initial data load
  useEffect(() => {
    loadData();
  }, [loadData]);

  // Auto-refresh effect
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(loadData, refreshInterval);
    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, loadData]);

  // Add single data point (for real-time updates)
  const addDataPoint = useCallback((seriesId: string, dataPoint: TimeSeriesDataPoint) => {
    setData(prevData => 
      prevData.map(series => {
        if (series.id === seriesId) {
          const newData = [...series.data, dataPoint];
          // Keep only recent data points
          const trimmedData = maxDataPoints > 0 ? newData.slice(-maxDataPoints) : newData;
          
          return {
            ...series,
            data: trimmedData,
          };
        }
        return series;
      })
    );
  }, [maxDataPoints]);

  // Memoized processed data
  const processedData = useMemo(() => {
    return data.map(series => ({
      ...series,
      data: series.data.sort((a, b) => a.timestamp - b.timestamp),
    }));
  }, [data]);

  return {
    data: processedData,
    loading,
    error,
    refresh: loadData,
    addDataPoint,
    setTimeRange,
    aggregateData,
  };
}

// Utility hook for real-time data streaming
export function useTimeSeriesStream(
  websocketUrl: string,
  seriesId: string,
  options: {
    onDataPoint?: (dataPoint: TimeSeriesDataPoint) => void;
    bufferSize?: number;
    reconnectAttempts?: number;
    reconnectDelay?: number;
  } = {}
) {
  const [connected, setConnected] = useState(false);
  const [buffer, setBuffer] = useState<TimeSeriesDataPoint[]>([]);
  const [error, setError] = useState<Error | null>(null);

  const {
    onDataPoint,
    bufferSize = 100,
    reconnectAttempts = 3,
    reconnectDelay = 1000,
  } = options;

  useEffect(() => {
    let ws: WebSocket;
    let reconnectCount = 0;
    let reconnectTimeout: NodeJS.Timeout;

    const connect = () => {
      try {
        ws = new WebSocket(websocketUrl);
        
        ws.onopen = () => {
          setConnected(true);
          setError(null);
          reconnectCount = 0;
        };
        
        ws.onmessage = (event) => {
          try {
            const dataPoint: TimeSeriesDataPoint = JSON.parse(event.data);
            
            setBuffer(prev => {
              const newBuffer = [...prev, dataPoint];
              return newBuffer.slice(-bufferSize);
            });
            
            onDataPoint?.(dataPoint);
          } catch (err) {
            console.error('Failed to parse WebSocket message:', err);
          }
        };
        
        ws.onclose = () => {
          setConnected(false);
          
          if (reconnectCount < reconnectAttempts) {
            reconnectCount++;
            reconnectTimeout = setTimeout(connect, reconnectDelay * reconnectCount);
          } else {
            setError(new Error('Failed to connect after multiple attempts'));
          }
        };
        
        ws.onerror = (event) => {
          setError(new Error('WebSocket connection error'));
        };
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to create WebSocket'));
      }
    };

    connect();

    return () => {
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      if (ws) {
        ws.close();
      }
    };
  }, [websocketUrl, bufferSize, reconnectAttempts, reconnectDelay, onDataPoint]);

  const clearBuffer = useCallback(() => {
    setBuffer([]);
  }, []);

  const getLatestData = useCallback(() => {
    return buffer.slice();
  }, [buffer]);

  return {
    connected,
    buffer,
    error,
    clearBuffer,
    getLatestData,
  };
}