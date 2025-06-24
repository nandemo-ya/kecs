import React, { useState, useEffect } from 'react';
import { apiClient } from '../services/api';
import { ServiceRevision, DescribeServiceRevisionsResponse } from '../types/api';
import './ServiceRevisions.css';

interface ServiceRevisionsProps {
  serviceArn: string;
  currentTaskDefinition?: string;
}

export function ServiceRevisions({ serviceArn, currentTaskDefinition }: ServiceRevisionsProps) {
  const [revisions, setRevisions] = useState<ServiceRevision[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedRevisions, setSelectedRevisions] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadRevisions();
  }, [serviceArn]);

  const loadRevisions = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // For now, we'll use the service ARN as the revision ARN
      // In a real implementation, you'd get the revision ARNs from another API
      const response: DescribeServiceRevisionsResponse = await apiClient.describeServiceRevisions({
        serviceRevisionArns: [serviceArn],
      });
      
      setRevisions(response.serviceRevisions || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load revisions');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectRevision = (revisionArn: string) => {
    const newSelected = new Set(selectedRevisions);
    if (newSelected.has(revisionArn)) {
      newSelected.delete(revisionArn);
    } else {
      newSelected.add(revisionArn);
    }
    setSelectedRevisions(newSelected);
  };

  const handleCompareRevisions = () => {
    // TODO: Implement revision comparison
    alert('Revision comparison feature coming soon!');
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleString();
  };

  const isCurrentRevision = (taskDef?: string) => {
    return taskDef === currentTaskDefinition;
  };

  if (loading) {
    return <div className="revisions-loading">Loading revisions...</div>;
  }

  if (error) {
    return <div className="revisions-error">Error: {error}</div>;
  }

  if (revisions.length === 0) {
    return <div className="revisions-empty">No revisions found for this service.</div>;
  }

  return (
    <div className="service-revisions">
      {selectedRevisions.size >= 2 && (
        <div className="revisions-actions">
          <button 
            className="btn btn-primary"
            onClick={handleCompareRevisions}
          >
            Compare Selected Revisions ({selectedRevisions.size})
          </button>
        </div>
      )}

      <div className="revisions-list">
        {revisions.map((revision, index) => (
          <div 
            key={revision.serviceRevisionArn || index} 
            className={`revision-card ${isCurrentRevision(revision.taskDefinition) ? 'current' : ''}`}
          >
            <div className="revision-header">
              <div className="revision-select">
                <input
                  type="checkbox"
                  checked={selectedRevisions.has(revision.serviceRevisionArn || '')}
                  onChange={() => handleSelectRevision(revision.serviceRevisionArn || '')}
                  disabled={!revision.serviceRevisionArn}
                />
              </div>
              <div className="revision-info">
                <div className="revision-arn">
                  {revision.serviceRevisionArn || 'Unknown ARN'}
                </div>
                {isCurrentRevision(revision.taskDefinition) && (
                  <span className="current-badge">Current</span>
                )}
              </div>
            </div>

            <div className="revision-details">
              <div className="detail-row">
                <span className="detail-label">Task Definition:</span>
                <span className="detail-value">{revision.taskDefinition || 'N/A'}</span>
              </div>

              <div className="detail-row">
                <span className="detail-label">Launch Type:</span>
                <span className="detail-value">{revision.launchType || 'N/A'}</span>
              </div>

              {revision.platformVersion && (
                <div className="detail-row">
                  <span className="detail-label">Platform Version:</span>
                  <span className="detail-value">{revision.platformVersion}</span>
                </div>
              )}

              {revision.platformFamily && (
                <div className="detail-row">
                  <span className="detail-label">Platform Family:</span>
                  <span className="detail-value">{revision.platformFamily}</span>
                </div>
              )}

              <div className="detail-row">
                <span className="detail-label">Created:</span>
                <span className="detail-value">{formatDate(revision.createdAt)}</span>
              </div>

              {revision.loadBalancers && revision.loadBalancers.length > 0 && (
                <div className="detail-section">
                  <h4>Load Balancers</h4>
                  {revision.loadBalancers.map((lb, lbIndex) => (
                    <div key={lbIndex} className="sub-detail">
                      {lb.targetGroupArn && (
                        <div>Target Group: {lb.targetGroupArn.split('/').pop()}</div>
                      )}
                      {lb.containerName && lb.containerPort && (
                        <div>Container: {lb.containerName}:{lb.containerPort}</div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {revision.serviceRegistries && revision.serviceRegistries.length > 0 && (
                <div className="detail-section">
                  <h4>Service Registries</h4>
                  {revision.serviceRegistries.map((sr, srIndex) => (
                    <div key={srIndex} className="sub-detail">
                      {sr.registryArn && (
                        <div>Registry: {sr.registryArn.split('/').pop()}</div>
                      )}
                      {sr.containerName && sr.containerPort && (
                        <div>Container: {sr.containerName}:{sr.containerPort}</div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {revision.containerImages && revision.containerImages.length > 0 && (
                <div className="detail-section">
                  <h4>Container Images</h4>
                  {revision.containerImages.map((img, imgIndex) => (
                    <div key={imgIndex} className="sub-detail">
                      <div className="container-name">{img.containerName}</div>
                      {img.image && <div className="image-uri">{img.image}</div>}
                      {img.imageDigest && (
                        <div className="image-digest">
                          Digest: {img.imageDigest.substring(0, 12)}...
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {isCurrentRevision(revision.taskDefinition) && (
              <div className="revision-actions">
                <button className="btn btn-secondary" disabled>
                  Current Revision
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}