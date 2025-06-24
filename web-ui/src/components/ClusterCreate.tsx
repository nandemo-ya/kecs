import React, { useState } from 'react';
import { Modal } from './Modal';
import { apiClient } from '../services/api';
import { CreateClusterRequest } from '../types/api';
import { useOperationNotification } from '../hooks/useOperationNotification';

interface ClusterCreateProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function ClusterCreate({ isOpen, onClose, onSuccess }: ClusterCreateProps) {
  const [clusterName, setClusterName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { notifySuccess, notifyError } = useOperationNotification();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!clusterName.trim()) {
      setError('Cluster name is required');
      return;
    }

    // Validate cluster name format
    if (!/^[a-zA-Z0-9]([a-zA-Z0-9\-_])*$/.test(clusterName)) {
      setError('Cluster name must start with a letter or number and can only contain letters, numbers, hyphens, and underscores');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const request: CreateClusterRequest = {
        clusterName: clusterName.trim(),
      };

      await apiClient.createCluster(request);
      
      notifySuccess(`Cluster "${clusterName}" created successfully`);
      setClusterName('');
      onSuccess();
      onClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create cluster';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      setClusterName('');
      setError(null);
      onClose();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create Cluster">
      <form onSubmit={handleSubmit}>
        <div className="modal-body">
          {error && (
            <div className="modal-error">
              {error}
            </div>
          )}
          
          <div className="form-group">
            <label htmlFor="clusterName">Cluster Name</label>
            <input
              id="clusterName"
              type="text"
              value={clusterName}
              onChange={(e) => setClusterName(e.target.value)}
              placeholder="my-cluster"
              disabled={loading}
              autoFocus
            />
            <div className="help-text">
              The name must be unique, start with a letter or number, and can only contain letters, numbers, hyphens, and underscores.
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
            disabled={loading || !clusterName.trim()}
          >
            {loading ? 'Creating...' : 'Create Cluster'}
          </button>
        </div>
      </form>
    </Modal>
  );
}