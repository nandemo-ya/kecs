import { useState, useCallback, useEffect } from 'react';
import { useWebSocket, useWebSocketSubscription } from './useWebSocket';
import { Task, TaskStatus } from '../types/task';

export interface TaskUpdate {
  taskId: string;
  updateType: 'created' | 'updated' | 'deleted' | 'status_changed';
  task?: Task;
  changes?: Partial<Task>;
  previousStatus?: TaskStatus;
  timestamp: Date;
}

interface UseWebSocketTaskUpdatesOptions {
  taskIds?: string[];
  serviceNames?: string[];
  clusterArn?: string;
  includeDeleted?: boolean;
  autoSubscribe?: boolean;
}

interface UseWebSocketTaskUpdatesResult {
  updates: TaskUpdate[];
  tasks: Map<string, Task>;
  isConnected: boolean;
  error: Error | null;
  subscribe: (taskId: string) => void;
  unsubscribe: (taskId: string) => void;
  subscribeToService: (serviceName: string) => void;
  unsubscribeFromService: (serviceName: string) => void;
  clearUpdates: () => void;
}

export function useWebSocketTaskUpdates(
  options: UseWebSocketTaskUpdatesOptions = {}
): UseWebSocketTaskUpdatesResult {
  const {
    taskIds = [],
    serviceNames = [],
    clusterArn,
    includeDeleted = false,
    autoSubscribe = true,
  } = options;

  const [updates, setUpdates] = useState<TaskUpdate[]>([]);
  const [tasks, setTasks] = useState<Map<string, Task>>(new Map());
  const [subscriptions, setSubscriptions] = useState<{
    tasks: Set<string>;
    services: Set<string>;
  }>({
    tasks: new Set(taskIds),
    services: new Set(serviceNames),
  });

  // Build WebSocket parameters
  const wsParams: Record<string, string> = {
    includeDeleted: includeDeleted.toString(),
  };
  
  if (clusterArn) {
    wsParams.clusterArn = clusterArn;
  }

  // Initialize WebSocket connection
  const ws = useWebSocket({
    path: '/ws/tasks',
    params: wsParams,
    reconnect: true,
    reconnectInterval: 5000,
    onConnected: () => {
      console.log('Task updates WebSocket connected');
      
      // Re-subscribe to all tasks and services
      if (autoSubscribe) {
        subscriptions.tasks.forEach(taskId => {
          ws.send({
            type: 'subscribe_task',
            payload: { taskId },
          });
        });
        
        subscriptions.services.forEach(serviceName => {
          ws.send({
            type: 'subscribe_service',
            payload: { serviceName },
          });
        });
      }
    },
  });

  // Handle task update
  const handleTaskUpdate = useCallback((update: TaskUpdate) => {
    // Add to updates list
    setUpdates(prev => [update, ...prev].slice(0, 100)); // Keep last 100 updates

    // Update tasks map
    if (update.task) {
      setTasks(prev => {
        const updated = new Map(prev);
        
        if (update.updateType === 'deleted') {
          updated.delete(update.taskId);
        } else {
          updated.set(update.taskId, update.task!);
        }
        
        return updated;
      });
    } else if (update.changes && update.updateType === 'updated') {
      setTasks(prev => {
        const updated = new Map(prev);
        const existingTask = updated.get(update.taskId);
        
        if (existingTask) {
          updated.set(update.taskId, { ...existingTask, ...update.changes });
        }
        
        return updated;
      });
    }
  }, []);

  // Handle batch updates
  const handleBatchUpdate = useCallback((updates: TaskUpdate[]) => {
    // Add to updates list
    setUpdates(prev => [...updates, ...prev].slice(0, 100));

    // Update tasks map
    setTasks(prev => {
      const updated = new Map(prev);
      
      updates.forEach(update => {
        if (update.task) {
          if (update.updateType === 'deleted') {
            updated.delete(update.taskId);
          } else {
            updated.set(update.taskId, update.task);
          }
        } else if (update.changes && update.updateType === 'updated') {
          const existingTask = updated.get(update.taskId);
          if (existingTask) {
            updated.set(update.taskId, { ...existingTask, ...update.changes });
          }
        }
      });
      
      return updated;
    });
  }, []);

  // Handle initial tasks
  const handleInitialTasks = useCallback((initialTasks: Task[]) => {
    setTasks(new Map(initialTasks.map(task => [task.taskArn, task])));
  }, []);

  // Subscribe to WebSocket messages
  useWebSocketSubscription(ws, 'task_update', handleTaskUpdate, [handleTaskUpdate]);
  useWebSocketSubscription(ws, 'task_batch_update', handleBatchUpdate, [handleBatchUpdate]);
  useWebSocketSubscription(ws, 'initial_tasks', handleInitialTasks, [handleInitialTasks]);

  // Subscribe to a task
  const subscribe = useCallback((taskId: string) => {
    if (subscriptions.tasks.has(taskId)) return;

    setSubscriptions(prev => {
      const newTasks = new Set(prev.tasks);
      newTasks.add(taskId);
      return {
        ...prev,
        tasks: newTasks,
      };
    });

    ws.send({
      type: 'subscribe_task',
      payload: { taskId },
    });
  }, [ws, subscriptions.tasks]);

  // Unsubscribe from a task
  const unsubscribe = useCallback((taskId: string) => {
    if (!subscriptions.tasks.has(taskId)) return;

    setSubscriptions(prev => {
      const newTasks = new Set(prev.tasks);
      newTasks.delete(taskId);
      return {
        ...prev,
        tasks: newTasks,
      };
    });

    ws.send({
      type: 'unsubscribe_task',
      payload: { taskId },
    });
  }, [ws, subscriptions.tasks]);

  // Subscribe to a service
  const subscribeToService = useCallback((serviceName: string) => {
    if (subscriptions.services.has(serviceName)) return;

    setSubscriptions(prev => {
      const newServices = new Set(prev.services);
      newServices.add(serviceName);
      return {
        ...prev,
        services: newServices,
      };
    });

    ws.send({
      type: 'subscribe_service',
      payload: { serviceName },
    });
  }, [ws, subscriptions.services]);

  // Unsubscribe from a service
  const unsubscribeFromService = useCallback((serviceName: string) => {
    if (!subscriptions.services.has(serviceName)) return;

    setSubscriptions(prev => {
      const newServices = new Set(prev.services);
      newServices.delete(serviceName);
      return {
        ...prev,
        services: newServices,
      };
    });

    ws.send({
      type: 'unsubscribe_service',
      payload: { serviceName },
    });
  }, [ws, subscriptions.services]);

  // Clear updates
  const clearUpdates = useCallback(() => {
    setUpdates([]);
  }, []);

  // Auto-subscribe on mount
  useEffect(() => {
    if (!ws.isConnected || !autoSubscribe) return;

    taskIds.forEach(subscribe);
    serviceNames.forEach(subscribeToService);

    return () => {
      taskIds.forEach(unsubscribe);
      serviceNames.forEach(unsubscribeFromService);
    };
  }, [ws.isConnected, autoSubscribe, taskIds, serviceNames, subscribe, unsubscribe, subscribeToService, unsubscribeFromService]);

  return {
    updates,
    tasks,
    isConnected: ws.isConnected,
    error: ws.error,
    subscribe,
    unsubscribe,
    subscribeToService,
    unsubscribeFromService,
    clearUpdates,
  };
}