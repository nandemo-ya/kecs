import React, { useState, useCallback, useMemo } from 'react';
import { LogFilter, LogLevel } from '../../types/logs';
import './Logs.css';

interface AdvancedSearchProps {
  filter: LogFilter;
  onChange: (filter: LogFilter) => void;
  onClose: () => void;
  availableTasks?: string[];
  availableServices?: string[];
  availableContainers?: string[];
}

interface SearchCondition {
  id: string;
  field: 'message' | 'taskId' | 'serviceName' | 'containerId' | 'level' | 'source' | 'metadata';
  operator: 'contains' | 'equals' | 'startsWith' | 'endsWith' | 'regex' | 'notContains' | 'in' | 'notIn';
  value: string;
  caseSensitive?: boolean;
}

interface SearchGroup {
  id: string;
  operator: 'AND' | 'OR';
  conditions: SearchCondition[];
}

export function AdvancedSearch({
  filter,
  onChange,
  onClose,
  availableTasks = [],
  availableServices = [],
  availableContainers = [],
}: AdvancedSearchProps) {
  const [searchGroups, setSearchGroups] = useState<SearchGroup[]>([
    {
      id: '1',
      operator: 'AND',
      conditions: [],
    },
  ]);
  const [savedSearches, setSavedSearches] = useState<{ name: string; filter: LogFilter }[]>([]);
  const [searchName, setSearchName] = useState('');

  // Add new condition
  const addCondition = useCallback((groupId: string) => {
    setSearchGroups(prev => prev.map(group => {
      if (group.id === groupId) {
        return {
          ...group,
          conditions: [
            ...group.conditions,
            {
              id: Date.now().toString(),
              field: 'message',
              operator: 'contains',
              value: '',
              caseSensitive: false,
            },
          ],
        };
      }
      return group;
    }));
  }, []);

  // Update condition
  const updateCondition = useCallback((groupId: string, conditionId: string, updates: Partial<SearchCondition>) => {
    setSearchGroups(prev => prev.map(group => {
      if (group.id === groupId) {
        return {
          ...group,
          conditions: group.conditions.map(condition => {
            if (condition.id === conditionId) {
              return { ...condition, ...updates };
            }
            return condition;
          }),
        };
      }
      return group;
    }));
  }, []);

  // Remove condition
  const removeCondition = useCallback((groupId: string, conditionId: string) => {
    setSearchGroups(prev => prev.map(group => {
      if (group.id === groupId) {
        return {
          ...group,
          conditions: group.conditions.filter(c => c.id !== conditionId),
        };
      }
      return group;
    }));
  }, []);

  // Add new group
  const addGroup = useCallback(() => {
    setSearchGroups(prev => [
      ...prev,
      {
        id: Date.now().toString(),
        operator: 'AND',
        conditions: [],
      },
    ]);
  }, []);

  // Remove group
  const removeGroup = useCallback((groupId: string) => {
    setSearchGroups(prev => prev.filter(g => g.id !== groupId));
  }, []);

  // Update group operator
  const updateGroupOperator = useCallback((groupId: string, operator: 'AND' | 'OR') => {
    setSearchGroups(prev => prev.map(group => {
      if (group.id === groupId) {
        return { ...group, operator };
      }
      return group;
    }));
  }, []);

  // Build search query from conditions
  const buildSearchQuery = useCallback(() => {
    // This is a simplified version - in a real implementation,
    // you would build a more complex query object or expression
    const queries: string[] = [];
    
    searchGroups.forEach(group => {
      const groupQueries: string[] = [];
      
      group.conditions.forEach(condition => {
        let query = '';
        
        switch (condition.operator) {
          case 'contains':
            query = condition.value;
            break;
          case 'equals':
            query = `^${condition.value}$`;
            break;
          case 'startsWith':
            query = `^${condition.value}`;
            break;
          case 'endsWith':
            query = `${condition.value}$`;
            break;
          case 'regex':
            query = condition.value;
            break;
          case 'notContains':
            query = `(?!.*${condition.value})`;
            break;
          default:
            query = condition.value;
        }
        
        if (query) {
          groupQueries.push(query);
        }
      });
      
      if (groupQueries.length > 0) {
        queries.push(`(${groupQueries.join(group.operator === 'AND' ? ' AND ' : ' OR ')})`);
      }
    });
    
    return queries.join(' AND ');
  }, [searchGroups]);

  // Apply search
  const applySearch = useCallback(() => {
    const searchQuery = buildSearchQuery();
    const newFilter: LogFilter = {
      ...filter,
      search: searchQuery,
      // Extract specific filters from conditions
      levels: searchGroups
        .flatMap(g => g.conditions)
        .filter(c => c.field === 'level')
        .map(c => c.value as LogLevel),
      taskIds: searchGroups
        .flatMap(g => g.conditions)
        .filter(c => c.field === 'taskId')
        .map(c => c.value),
      serviceNames: searchGroups
        .flatMap(g => g.conditions)
        .filter(c => c.field === 'serviceName')
        .map(c => c.value),
      containerIds: searchGroups
        .flatMap(g => g.conditions)
        .filter(c => c.field === 'containerId')
        .map(c => c.value),
    };
    
    onChange(newFilter);
    onClose();
  }, [searchGroups, filter, onChange, onClose, buildSearchQuery]);

  // Save current search
  const saveSearch = useCallback(() => {
    if (searchName.trim()) {
      setSavedSearches(prev => [
        ...prev,
        {
          name: searchName.trim(),
          filter: {
            ...filter,
            search: buildSearchQuery(),
          },
        },
      ]);
      setSearchName('');
    }
  }, [searchName, filter, buildSearchQuery]);

  // Load saved search
  const loadSearch = useCallback((savedFilter: LogFilter) => {
    onChange(savedFilter);
    onClose();
  }, [onChange, onClose]);

  // Get field options based on field type
  const getFieldValues = useCallback((field: string): string[] => {
    switch (field) {
      case 'level':
        return ['trace', 'debug', 'info', 'warn', 'error', 'fatal'];
      case 'taskId':
        return availableTasks;
      case 'serviceName':
        return availableServices;
      case 'containerId':
        return availableContainers;
      default:
        return [];
    }
  }, [availableTasks, availableServices, availableContainers]);

  return (
    <div className="advanced-search-modal">
      <div className="advanced-search-content">
        <div className="advanced-search-header">
          <h3>Advanced Search</h3>
          <button className="close-button" onClick={onClose}>✕</button>
        </div>

        <div className="search-groups">
          {searchGroups.map((group, groupIndex) => (
            <div key={group.id} className="search-group">
              {groupIndex > 0 && (
                <div className="group-operator">
                  <select
                    value={group.operator}
                    onChange={(e) => updateGroupOperator(group.id, e.target.value as 'AND' | 'OR')}
                  >
                    <option value="AND">AND</option>
                    <option value="OR">OR</option>
                  </select>
                </div>
              )}

              <div className="group-content">
                <div className="group-header">
                  <h4>Condition Group {groupIndex + 1}</h4>
                  {searchGroups.length > 1 && (
                    <button
                      className="remove-group-button"
                      onClick={() => removeGroup(group.id)}
                    >
                      Remove Group
                    </button>
                  )}
                </div>

                <div className="conditions">
                  {group.conditions.map((condition, conditionIndex) => (
                    <div key={condition.id} className="condition-row">
                      {conditionIndex > 0 && (
                        <div className="condition-operator">{group.operator}</div>
                      )}
                      
                      <select
                        className="field-select"
                        value={condition.field}
                        onChange={(e) => updateCondition(group.id, condition.id, { 
                          field: e.target.value as any,
                          value: '',
                        })}
                      >
                        <option value="message">Message</option>
                        <option value="level">Level</option>
                        <option value="source">Source</option>
                        <option value="taskId">Task ID</option>
                        <option value="serviceName">Service Name</option>
                        <option value="containerId">Container ID</option>
                        <option value="metadata">Metadata</option>
                      </select>

                      <select
                        className="operator-select"
                        value={condition.operator}
                        onChange={(e) => updateCondition(group.id, condition.id, { 
                          operator: e.target.value as any 
                        })}
                      >
                        <option value="contains">contains</option>
                        <option value="equals">equals</option>
                        <option value="startsWith">starts with</option>
                        <option value="endsWith">ends with</option>
                        <option value="notContains">does not contain</option>
                        <option value="regex">matches regex</option>
                        {['level', 'taskId', 'serviceName', 'containerId'].includes(condition.field) && (
                          <>
                            <option value="in">in</option>
                            <option value="notIn">not in</option>
                          </>
                        )}
                      </select>

                      {['in', 'notIn'].includes(condition.operator) && 
                       ['level', 'taskId', 'serviceName', 'containerId'].includes(condition.field) ? (
                        <select
                          className="value-select"
                          multiple
                          value={condition.value.split(',')}
                          onChange={(e) => {
                            const values = Array.from(e.target.selectedOptions, option => option.value);
                            updateCondition(group.id, condition.id, { 
                              value: values.join(',')
                            });
                          }}
                        >
                          {getFieldValues(condition.field).map(value => (
                            <option key={value} value={value}>
                              {value}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <input
                          type="text"
                          className="value-input"
                          value={condition.value}
                          onChange={(e) => updateCondition(group.id, condition.id, { 
                            value: e.target.value 
                          })}
                          placeholder={condition.operator === 'regex' ? 'Regular expression' : 'Value'}
                        />
                      )}

                      <label className="case-sensitive-label">
                        <input
                          type="checkbox"
                          checked={condition.caseSensitive || false}
                          onChange={(e) => updateCondition(group.id, condition.id, { 
                            caseSensitive: e.target.checked 
                          })}
                        />
                        Case sensitive
                      </label>

                      <button
                        className="remove-condition-button"
                        onClick={() => removeCondition(group.id, condition.id)}
                      >
                        ✕
                      </button>
                    </div>
                  ))}

                  <button
                    className="add-condition-button"
                    onClick={() => addCondition(group.id)}
                  >
                    + Add Condition
                  </button>
                </div>
              </div>
            </div>
          ))}

          <button className="add-group-button" onClick={addGroup}>
            + Add Condition Group
          </button>
        </div>

        <div className="saved-searches">
          <h4>Saved Searches</h4>
          <div className="save-search-row">
            <input
              type="text"
              placeholder="Search name"
              value={searchName}
              onChange={(e) => setSearchName(e.target.value)}
            />
            <button onClick={saveSearch} disabled={!searchName.trim()}>
              Save Current Search
            </button>
          </div>
          
          {savedSearches.length > 0 && (
            <div className="saved-search-list">
              {savedSearches.map((saved, index) => (
                <div key={index} className="saved-search-item">
                  <span>{saved.name}</span>
                  <button onClick={() => loadSearch(saved.filter)}>Load</button>
                  <button onClick={() => setSavedSearches(prev => prev.filter((_, i) => i !== index))}>
                    Delete
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="advanced-search-actions">
          <button className="cancel-button" onClick={onClose}>
            Cancel
          </button>
          <button className="apply-button" onClick={applySearch}>
            Apply Search
          </button>
        </div>
      </div>
    </div>
  );
}