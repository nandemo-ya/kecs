import React, { useState, useEffect } from 'react';
import { Tag } from '../types/api';
import { apiClient } from '../services/api';
import { useOperationNotification } from '../hooks/useOperationNotification';
import './TagEditor.css';

interface TagEditorProps {
  resourceArn: string;
  editable?: boolean;
  onTagsChange?: (tags: Tag[]) => void;
}

export function TagEditor({ resourceArn, editable = true, onTagsChange }: TagEditorProps) {
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newTag, setNewTag] = useState({ key: '', value: '' });
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editTag, setEditTag] = useState({ key: '', value: '' });
  const { notifySuccess, notifyError } = useOperationNotification();

  useEffect(() => {
    loadTags();
  }, [resourceArn]);

  const loadTags = async () => {
    if (!resourceArn) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await apiClient.listTagsForResource({ resourceArn });
      setTags(response.tags || []);
      onTagsChange?.(response.tags || []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load tags';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleAddTag = async () => {
    if (!newTag.key.trim()) {
      setError('Tag key is required');
      return;
    }

    // Check for duplicate keys
    if (tags.some(tag => tag.key === newTag.key.trim())) {
      setError('Tag key already exists');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      const tagToAdd: Tag = {
        key: newTag.key.trim(),
        value: newTag.value.trim()
      };

      await apiClient.tagResource({
        resourceArn,
        tags: [tagToAdd]
      });

      const updatedTags = [...tags, tagToAdd];
      setTags(updatedTags);
      onTagsChange?.(updatedTags);
      
      notifySuccess(`Tag "${tagToAdd.key}" added successfully`);
      setNewTag({ key: '', value: '' });
      setShowAddForm(false);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to add tag';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateTag = async () => {
    if (editingIndex === null) return;
    
    if (!editTag.key.trim()) {
      setError('Tag key is required');
      return;
    }

    // Check for duplicate keys (excluding the current tag)
    if (tags.some((tag, index) => index !== editingIndex && tag.key === editTag.key.trim())) {
      setError('Tag key already exists');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      const oldTag = tags[editingIndex];
      
      // If key changed, we need to remove old and add new
      if (oldTag.key !== editTag.key.trim()) {
        await apiClient.untagResource({
          resourceArn,
          tagKeys: [oldTag.key]
        });
      }

      await apiClient.tagResource({
        resourceArn,
        tags: [{
          key: editTag.key.trim(),
          value: editTag.value.trim()
        }]
      });

      const updatedTags = [...tags];
      updatedTags[editingIndex] = {
        key: editTag.key.trim(),
        value: editTag.value.trim()
      };
      setTags(updatedTags);
      onTagsChange?.(updatedTags);
      
      notifySuccess(`Tag "${editTag.key}" updated successfully`);
      setEditingIndex(null);
      setEditTag({ key: '', value: '' });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update tag';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteTag = async (tagKey: string) => {
    const confirmed = window.confirm(`Are you sure you want to delete the tag "${tagKey}"?`);
    if (!confirmed) return;

    setSaving(true);
    setError(null);

    try {
      await apiClient.untagResource({
        resourceArn,
        tagKeys: [tagKey]
      });

      const updatedTags = tags.filter(tag => tag.key !== tagKey);
      setTags(updatedTags);
      onTagsChange?.(updatedTags);
      
      notifySuccess(`Tag "${tagKey}" deleted successfully`);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete tag';
      setError(errorMessage);
      notifyError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const startEditTag = (index: number) => {
    setEditingIndex(index);
    setEditTag({ ...tags[index] });
    setShowAddForm(false);
  };

  const cancelEdit = () => {
    setEditingIndex(null);
    setEditTag({ key: '', value: '' });
    setError(null);
  };

  const cancelAdd = () => {
    setShowAddForm(false);
    setNewTag({ key: '', value: '' });
    setError(null);
  };

  if (loading) {
    return <div className="tag-editor-loading">Loading tags...</div>;
  }

  return (
    <div className="tag-editor">
      <div className="tag-editor-header">
        <h3>Tags</h3>
        {editable && !showAddForm && editingIndex === null && (
          <button
            className="button button-primary"
            onClick={() => setShowAddForm(true)}
            disabled={saving}
          >
            Add Tag
          </button>
        )}
      </div>

      {error && <div className="tag-editor-error">{error}</div>}

      {tags.length === 0 && !showAddForm ? (
        <div className="empty-tags">
          No tags defined for this resource.
          {editable && (
            <>
              <br />
              <button
                className="add-tag-button"
                onClick={() => setShowAddForm(true)}
              >
                Add your first tag
              </button>
            </>
          )}
        </div>
      ) : (
        <div className="tags-list">
          {tags.map((tag, index) => (
            <div
              key={tag.key}
              className={`tag-item ${editingIndex === index ? 'editing' : ''}`}
            >
              {editingIndex === index ? (
                <>
                  <input
                    type="text"
                    value={editTag.key}
                    onChange={(e) => setEditTag({ ...editTag, key: e.target.value })}
                    placeholder="Key"
                    disabled={saving}
                  />
                  <input
                    type="text"
                    value={editTag.value}
                    onChange={(e) => setEditTag({ ...editTag, value: e.target.value })}
                    placeholder="Value"
                    disabled={saving}
                  />
                  <div className="tag-actions">
                    <button
                      className="button button-primary"
                      onClick={handleUpdateTag}
                      disabled={saving}
                    >
                      Save
                    </button>
                    <button
                      className="button button-secondary"
                      onClick={cancelEdit}
                      disabled={saving}
                    >
                      Cancel
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <span className="tag-key">{tag.key}</span>
                  <span className="tag-value">{tag.value || '(empty)'}</span>
                  {editable && (
                    <div className="tag-actions">
                      <button
                        className="tag-button"
                        onClick={() => startEditTag(index)}
                        disabled={saving}
                        title="Edit tag"
                      >
                        Edit
                      </button>
                      <button
                        className="tag-button delete"
                        onClick={() => handleDeleteTag(tag.key)}
                        disabled={saving}
                        title="Delete tag"
                      >
                        Delete
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
          ))}
        </div>
      )}

      {showAddForm && (
        <div className="tag-form">
          <div className="tag-form-inputs">
            <input
              type="text"
              value={newTag.key}
              onChange={(e) => setNewTag({ ...newTag, key: e.target.value })}
              placeholder="Key"
              disabled={saving}
              autoFocus
            />
            <input
              type="text"
              value={newTag.value}
              onChange={(e) => setNewTag({ ...newTag, value: e.target.value })}
              placeholder="Value (optional)"
              disabled={saving}
            />
          </div>
          <div className="tag-form-actions">
            <button
              className="button button-primary"
              onClick={handleAddTag}
              disabled={saving || !newTag.key.trim()}
            >
              Add
            </button>
            <button
              className="button button-secondary"
              onClick={cancelAdd}
              disabled={saving}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {editable && (
        <div className="tag-help">
          Tags help you organize and categorize your resources. Each tag consists of a key and an optional value.
        </div>
      )}
    </div>
  );
}

// Component for displaying tags in a compact badge format (for list views)
export function TagBadges({ tags }: { tags?: Tag[] }) {
  if (!tags || tags.length === 0) return null;

  return (
    <div className="tag-badges">
      {tags.slice(0, 3).map((tag) => (
        <span key={tag.key} className="tag-badge" title={`${tag.key}: ${tag.value || '(empty)'}`}>
          <span className="tag-badge-key">{tag.key}</span>
          {tag.value && <span className="tag-badge-value">{tag.value}</span>}
        </span>
      ))}
      {tags.length > 3 && (
        <span className="tag-badge" title={`${tags.length - 3} more tags`}>
          +{tags.length - 3} more
        </span>
      )}
    </div>
  );
}