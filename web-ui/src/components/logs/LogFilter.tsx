import React, { useState, useCallback } from 'react';
import { LogFilter as LogFilterType, LogLevel, TimeRange } from '../../types/logs';
import './Logs.css';

interface LogFilterProps {
  filter: LogFilterType;
  onChange: (filter: LogFilterType) => void;
  availableSources?: string[];
}

const LOG_LEVELS: { value: LogLevel; label: string; color: string }[] = [
  { value: 'trace', label: 'Trace', color: '#9ca3af' },
  { value: 'debug', label: 'Debug', color: '#6b7280' },
  { value: 'info', label: 'Info', color: '#3b82f6' },
  { value: 'warn', label: 'Warn', color: '#f59e0b' },
  { value: 'error', label: 'Error', color: '#ef4444' },
  { value: 'fatal', label: 'Fatal', color: '#991b1b' },
];

const TIME_RANGES: { value: TimeRange['preset']; label: string }[] = [
  { value: 'last-5m', label: 'Last 5 minutes' },
  { value: 'last-15m', label: 'Last 15 minutes' },
  { value: 'last-30m', label: 'Last 30 minutes' },
  { value: 'last-1h', label: 'Last hour' },
  { value: 'last-6h', label: 'Last 6 hours' },
  { value: 'last-24h', label: 'Last 24 hours' },
  { value: 'last-7d', label: 'Last 7 days' },
];

export function LogFilter({ filter, onChange, availableSources = [] }: LogFilterProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [regexError, setRegexError] = useState<string | null>(null);

  // Handle level filter change
  const handleLevelChange = useCallback((level: LogLevel) => {
    const newLevels = filter.levels || [];
    const index = newLevels.indexOf(level);
    
    if (index >= 0) {
      onChange({
        ...filter,
        levels: newLevels.filter(l => l !== level),
      });
    } else {
      onChange({
        ...filter,
        levels: [...newLevels, level],
      });
    }
  }, [filter, onChange]);

  // Handle source filter change
  const handleSourceChange = useCallback((source: string) => {
    const newSources = filter.sources || [];
    const index = newSources.indexOf(source);
    
    if (index >= 0) {
      onChange({
        ...filter,
        sources: newSources.filter(s => s !== source),
      });
    } else {
      onChange({
        ...filter,
        sources: [...newSources, source],
      });
    }
  }, [filter, onChange]);

  // Handle search change
  const handleSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({
      ...filter,
      search: event.target.value,
    });
  }, [filter, onChange]);

  // Handle regex change
  const handleRegexChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // Validate regex
    if (value) {
      try {
        new RegExp(value);
        setRegexError(null);
      } catch (e) {
        setRegexError('Invalid regular expression');
      }
    } else {
      setRegexError(null);
    }
    
    onChange({
      ...filter,
      regex: value,
    });
  }, [filter, onChange]);

  // Handle time range change
  const handleTimeRangeChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const preset = event.target.value as TimeRange['preset'];
    
    let start: Date | undefined;
    const end = new Date();
    
    switch (preset) {
      case 'last-5m':
        start = new Date(end.getTime() - 5 * 60 * 1000);
        break;
      case 'last-15m':
        start = new Date(end.getTime() - 15 * 60 * 1000);
        break;
      case 'last-30m':
        start = new Date(end.getTime() - 30 * 60 * 1000);
        break;
      case 'last-1h':
        start = new Date(end.getTime() - 60 * 60 * 1000);
        break;
      case 'last-6h':
        start = new Date(end.getTime() - 6 * 60 * 60 * 1000);
        break;
      case 'last-24h':
        start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
        break;
      case 'last-7d':
        start = new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
    }
    
    onChange({
      ...filter,
      timeRange: preset ? { preset, start, end } : undefined,
    });
  }, [filter, onChange]);

  // Clear all filters
  const clearFilters = useCallback(() => {
    onChange({
      levels: [],
      sources: [],
      search: '',
      regex: '',
      timeRange: undefined,
      taskIds: filter.taskIds,
      serviceNames: filter.serviceNames,
      containerIds: filter.containerIds,
    });
  }, [filter, onChange]);

  const hasActiveFilters = 
    (filter.levels && filter.levels.length > 0) ||
    (filter.sources && filter.sources.length > 0) ||
    filter.search ||
    filter.regex ||
    filter.timeRange;

  return (
    <div className={`log-filter ${isExpanded ? 'expanded' : ''}`}>
      <div className="log-filter-header">
        <div className="log-filter-summary">
          <button
            className="filter-toggle"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            <span className="filter-icon">{isExpanded ? '▼' : '▶'}</span>
            <span className="filter-label">Filters</span>
            {hasActiveFilters && (
              <span className="filter-badge">
                {[
                  filter.levels?.length || 0,
                  filter.sources?.length || 0,
                  filter.search ? 1 : 0,
                  filter.regex ? 1 : 0,
                  filter.timeRange ? 1 : 0,
                ].reduce((a, b) => a + b, 0)}
              </span>
            )}
          </button>

          {!isExpanded && hasActiveFilters && (
            <div className="filter-chips">
              {filter.levels?.map(level => (
                <span
                  key={level}
                  className="filter-chip"
                  style={{ 
                    backgroundColor: LOG_LEVELS.find(l => l.value === level)?.color + '20',
                    borderColor: LOG_LEVELS.find(l => l.value === level)?.color,
                  }}
                >
                  {level}
                </span>
              ))}
              {filter.search && (
                <span className="filter-chip">
                  🔍 "{filter.search}"
                </span>
              )}
              {filter.regex && (
                <span className="filter-chip">
                  🔤 /{filter.regex}/
                </span>
              )}
            </div>
          )}
        </div>

        <div className="log-filter-search">
          <input
            type="text"
            placeholder="Search logs..."
            value={filter.search || ''}
            onChange={handleSearchChange}
            className="search-input"
          />
        </div>
      </div>

      {isExpanded && (
        <div className="log-filter-content">
          <div className="filter-section">
            <h4>Log Levels</h4>
            <div className="filter-options">
              {LOG_LEVELS.map(({ value, label, color }) => (
                <label key={value} className="filter-checkbox">
                  <input
                    type="checkbox"
                    checked={filter.levels?.includes(value) || false}
                    onChange={() => handleLevelChange(value)}
                  />
                  <span className="checkbox-label" style={{ color }}>
                    {label}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {availableSources.length > 0 && (
            <div className="filter-section">
              <h4>Sources</h4>
              <div className="filter-options">
                {availableSources.map(source => (
                  <label key={source} className="filter-checkbox">
                    <input
                      type="checkbox"
                      checked={filter.sources?.includes(source) || false}
                      onChange={() => handleSourceChange(source)}
                    />
                    <span className="checkbox-label">{source}</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="filter-section">
            <h4>Time Range</h4>
            <select
              value={filter.timeRange?.preset || ''}
              onChange={handleTimeRangeChange}
              className="time-range-select"
            >
              <option value="">All time</option>
              {TIME_RANGES.map(({ value, label }) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>

          <div className="filter-section">
            <h4>Regular Expression</h4>
            <input
              type="text"
              placeholder="Filter by regex pattern..."
              value={filter.regex || ''}
              onChange={handleRegexChange}
              className={`regex-input ${regexError ? 'error' : ''}`}
            />
            {regexError && (
              <span className="regex-error">{regexError}</span>
            )}
          </div>

          <div className="filter-actions">
            <button
              className="clear-filters-button"
              onClick={clearFilters}
              disabled={!hasActiveFilters}
            >
              Clear All Filters
            </button>
          </div>
        </div>
      )}
    </div>
  );
}