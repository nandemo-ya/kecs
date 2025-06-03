import React, { useMemo } from 'react';
import { LogEntry as LogEntryType, LogLevel, getLogLevelColor } from '../../types/logs';
import './Logs.css';

interface LogStatsProps {
  logs: LogEntryType[];
}

export function LogStats({ logs }: LogStatsProps) {
  const stats = useMemo(() => {
    const levelCounts: Record<LogLevel, number> = {
      trace: 0,
      debug: 0,
      info: 0,
      warn: 0,
      error: 0,
      fatal: 0,
    };

    const sourceCounts: Record<string, number> = {};
    const timeline: Record<string, number> = {};

    logs.forEach(log => {
      // Count by level
      levelCounts[log.level]++;

      // Count by source
      sourceCounts[log.source.name] = (sourceCounts[log.source.name] || 0) + 1;

      // Count by minute for timeline
      const minute = new Date(log.timestamp);
      minute.setSeconds(0);
      minute.setMilliseconds(0);
      const key = minute.toISOString();
      timeline[key] = (timeline[key] || 0) + 1;
    });

    // Calculate rates
    const total = logs.length;
    const errorCount = levelCounts.error + levelCounts.fatal;
    const warnCount = levelCounts.warn;

    return {
      total,
      levelCounts,
      sourceCounts,
      timeline,
      errorRate: total > 0 ? (errorCount / total) * 100 : 0,
      warnRate: total > 0 ? (warnCount / total) * 100 : 0,
      avgPerMinute: Object.keys(timeline).length > 0 
        ? total / Object.keys(timeline).length 
        : 0,
    };
  }, [logs]);

  const sortedSources = Object.entries(stats.sourceCounts)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 5);

  const maxSourceCount = sortedSources.length > 0 ? sortedSources[0][1] : 1;

  return (
    <div className="log-stats">
      <div className="stats-overview">
        <div className="stat-card">
          <h4>Total Logs</h4>
          <div className="stat-value">{stats.total}</div>
        </div>
        
        <div className="stat-card">
          <h4>Error Rate</h4>
          <div className="stat-value error">{stats.errorRate.toFixed(1)}%</div>
        </div>
        
        <div className="stat-card">
          <h4>Warning Rate</h4>
          <div className="stat-value warn">{stats.warnRate.toFixed(1)}%</div>
        </div>
        
        <div className="stat-card">
          <h4>Avg/Minute</h4>
          <div className="stat-value">{stats.avgPerMinute.toFixed(1)}</div>
        </div>
      </div>

      <div className="stats-details">
        <div className="stat-section">
          <h4>Log Levels</h4>
          <div className="level-bars">
            {(Object.entries(stats.levelCounts) as [LogLevel, number][]).map(([level, count]) => (
              <div key={level} className="level-bar">
                <div className="level-info">
                  <span 
                    className="level-name"
                    style={{ color: getLogLevelColor(level) }}
                  >
                    {level.toUpperCase()}
                  </span>
                  <span className="level-count">{count}</span>
                </div>
                <div className="level-progress">
                  <div
                    className="level-progress-bar"
                    style={{
                      width: `${stats.total > 0 ? (count / stats.total) * 100 : 0}%`,
                      backgroundColor: getLogLevelColor(level),
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="stat-section">
          <h4>Top Sources</h4>
          <div className="source-bars">
            {sortedSources.map(([source, count]) => (
              <div key={source} className="source-bar">
                <div className="source-info">
                  <span className="source-name">{source}</span>
                  <span className="source-count">{count}</span>
                </div>
                <div className="source-progress">
                  <div
                    className="source-progress-bar"
                    style={{
                      width: `${(count / maxSourceCount) * 100}%`,
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}