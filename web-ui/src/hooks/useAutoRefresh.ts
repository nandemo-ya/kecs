import { useEffect, useRef, useState, useCallback } from 'react';

interface UseAutoRefreshOptions {
  enabled?: boolean;
  interval?: number; // in milliseconds
}

export function useAutoRefresh(
  refreshFunction: () => void,
  options: UseAutoRefreshOptions = {}
) {
  const { enabled = false, interval = 5000 } = options;
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const startRefresh = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    
    intervalRef.current = setInterval(() => {
      setIsRefreshing(true);
      refreshFunction();
      setTimeout(() => setIsRefreshing(false), 500); // Visual indicator duration
    }, interval);
  }, [refreshFunction, interval]);

  const stopRefresh = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const toggleRefresh = useCallback(() => {
    if (intervalRef.current) {
      stopRefresh();
    } else {
      startRefresh();
    }
  }, [startRefresh, stopRefresh]);

  useEffect(() => {
    if (enabled) {
      startRefresh();
    } else {
      stopRefresh();
    }

    return () => {
      stopRefresh();
    };
  }, [enabled, startRefresh, stopRefresh]);

  useEffect(() => {
    return () => {
      stopRefresh();
    };
  }, [stopRefresh]);

  return {
    isAutoRefreshEnabled: intervalRef.current !== null,
    isRefreshing,
    startRefresh,
    stopRefresh,
    toggleRefresh,
  };
}