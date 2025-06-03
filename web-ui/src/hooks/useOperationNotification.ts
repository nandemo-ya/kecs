import { useCallback } from 'react';
import { useNotifications } from '../contexts/NotificationContext';

interface OperationOptions {
  successTitle: string;
  successMessage?: string;
  errorTitle?: string;
  inProgressTitle?: string;
  inProgressMessage?: string;
}

export function useOperationNotification() {
  const { addNotification } = useNotifications();

  const executeWithNotification = useCallback(async <T,>(
    operation: () => Promise<T>,
    options: OperationOptions
  ): Promise<T | null> => {
    // Show in-progress notification if provided
    if (options.inProgressTitle) {
      addNotification({
        type: 'info',
        title: options.inProgressTitle,
        message: options.inProgressMessage,
        duration: 2000,
      });
    }

    try {
      const result = await operation();
      
      // Show success notification
      addNotification({
        type: 'success',
        title: options.successTitle,
        message: options.successMessage,
      });
      
      return result;
    } catch (error) {
      // Show error notification
      const errorMessage = error instanceof Error ? error.message : 'An unknown error occurred';
      addNotification({
        type: 'error',
        title: options.errorTitle || 'Operation Failed',
        message: errorMessage,
        duration: 0, // Don't auto-dismiss errors
      });
      
      return null;
    }
  }, [addNotification]);

  return { executeWithNotification };
}