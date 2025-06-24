import React, { useState, useEffect } from 'react';
import { Modal } from './Modal';
import { apiClient } from '../services/api';
import { RunTaskRequest, ListTaskDefinitionsResponse } from '../types/api';
import { useOperationNotification } from '../hooks/useOperationNotification';

interface RunTaskProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  clusterName?: string;
}

export function RunTask({ isOpen, onClose, onSuccess, clusterName }: RunTaskProps) {
  const [taskDefinition, setTaskDefinition] = useState('');
  const [taskDefinitions, setTaskDefinitions] = useState<string[]>([]);
  const [count, setCount] = useState(1);
  const [loading, setLoading] = useState(false);
  const [loadingTaskDefs, setLoadingTaskDefs] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { notifySuccess, notifyError } = useOperationNotification();

  useEffect(() => {
    if (isOpen) {
      loadTaskDefinitions();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  const loadTaskDefinitions = async () => {
    setLoadingTaskDefs(true);
    try {
      const response: ListTaskDefinitionsResponse = await apiClient.listTaskDefinitions();
      
      // Extract task definition names from ARNs
      const taskDefNames = response.taskDefinitionArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });
      
      setTaskDefinitions(taskDefNames);
      
      // Select the first task definition by default
      if (taskDefNames.length > 0 && !taskDefinition) {
        setTaskDefinition(taskDefNames[0]);
      }
    } catch (err) {
      console.error('Failed to load task definitions:', err);
      setError('Failed to load task definitions');
    } finally {
      setLoadingTaskDefs(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!taskDefinition) {
      setError('Task definition is required');
      return;
    }

    if (count < 1 || count > 10) {
      setError('Count must be between 1 and 10');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const request: RunTaskRequest = {
        cluster: clusterName,
        taskDefinition,
        count,
      };

      const response = await apiClient.runTask(request);
      
      const successCount = response.tasks.length;
      const failureCount = response.failures?.length || 0;
      
      if (successCount > 0) {
        notifySuccess(`Successfully started ${successCount} task(s)`);
      }
      
      if (failureCount > 0) {
        const failureReasons = response.failures?.map(f => f.reason).join(', ');
        notifyError(`Failed to start ${failureCount} task(s): ${failureReasons}`);
      }
      
      onSuccess();
      handleClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to run task';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      setTaskDefinition('');
      setCount(1);
      setError(null);
      onClose();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Run Task">
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          {error && (
            <div className="modal-error">
              {error}
            </div>
          )}
          
          <div className="form-group">
            <label htmlFor="cluster">Cluster</label>
            <input
              id="cluster"
              type="text"
              value={clusterName || 'default'}
              disabled
            />
          </div>
          
          <div className="form-group">
            <label htmlFor="taskDefinition">Task Definition</label>
            {loadingTaskDefs ? (
              <div className="loading">Loading task definitions...</div>
            ) : taskDefinitions.length === 0 ? (
              <div className="help-text">No task definitions found. Please register a task definition first.</div>
            ) : (
              <select
                id="taskDefinition"
                value={taskDefinition}
                onChange={(e) => setTaskDefinition(e.target.value)}
                disabled={loading}
              >
                <option value="">Select a task definition</option>
                {taskDefinitions.map(td => (
                  <option key={td} value={td}>{td}</option>
                ))}
              </select>
            )}
          </div>
          
          <div className="form-group">
            <label htmlFor="count">Number of Tasks</label>
            <input
              id="count"
              type="number"
              value={count}
              onChange={(e) => setCount(parseInt(e.target.value) || 1)}
              min="1"
              max="10"
              disabled={loading}
            />
            <div className="help-text">
              Number of tasks to run (1-10)
            </div>
          </div>
        </div>

        <div className="modal-footer">
          <button
            type="button"
            className="button button-secondary"
            onClick={handleClose}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="button button-primary"
            disabled={loading || !taskDefinition || taskDefinitions.length === 0}
          >
            {loading ? 'Running...' : 'Run Task'}
          </button>
        </div>
      </form>
    </Modal>
  );
}