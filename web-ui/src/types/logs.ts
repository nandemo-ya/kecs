// Log viewer types and interfaces

// Log entry structure
export interface LogEntry {
  id: string;
  timestamp: Date;
  level: LogLevel;
  source: LogSource;
  message: string;
  metadata?: Record<string, any>;
  taskId?: string;
  containerId?: string;
  serviceName?: string;
  namespace?: string;
  pod?: string;
  container?: string;
  stackTrace?: string;
  correlationId?: string;
}

// Log levels
export type LogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';

// Log sources
export interface LogSource {
  type: 'container' | 'application' | 'system' | 'audit';
  name: string;
  identifier: string;
}

// Log filter configuration
export interface LogFilter {
  levels?: LogLevel[];
  sources?: string[];
  search?: string;
  regex?: string;
  timeRange?: TimeRange;
  taskIds?: string[];
  serviceNames?: string[];
  containerIds?: string[];
  includeMetadata?: boolean;
}

// Time range for log filtering
export interface TimeRange {
  start?: Date;
  end?: Date;
  preset?: 'last-5m' | 'last-15m' | 'last-30m' | 'last-1h' | 'last-6h' | 'last-24h' | 'last-7d';
}

// Log streaming configuration
export interface LogStreamConfig {
  enabled: boolean;
  maxBufferSize?: number;
  reconnectInterval?: number;
  heartbeatInterval?: number;
  compression?: boolean;
}

// Log export options
export interface LogExportOptions {
  format: 'json' | 'csv' | 'text';
  includeMetadata?: boolean;
  dateFormat?: string;
  delimiter?: string;
  filename?: string;
}

// Log statistics
export interface LogStats {
  totalEntries: number;
  entriesByLevel: Record<LogLevel, number>;
  entriesBySource: Record<string, number>;
  oldestEntry?: Date;
  newestEntry?: Date;
  errorRate: number;
  warningRate: number;
}

// Log viewer configuration
export interface LogViewerConfig {
  maxDisplayEntries?: number;
  enableVirtualization?: boolean;
  enableHighlighting?: boolean;
  enableLineNumbers?: boolean;
  enableWordWrap?: boolean;
  fontSize?: number;
  theme?: 'light' | 'dark' | 'auto';
  dateFormat?: string;
  showTimestamps?: boolean;
  showLevels?: boolean;
  showSources?: boolean;
  colorScheme?: LogColorScheme;
}

// Color scheme for log levels
export interface LogColorScheme {
  trace?: string;
  debug?: string;
  info?: string;
  warn?: string;
  error?: string;
  fatal?: string;
}

// Default color scheme
export const DEFAULT_LOG_COLOR_SCHEME: LogColorScheme = {
  trace: '#9ca3af',
  debug: '#6b7280',
  info: '#3b82f6',
  warn: '#f59e0b',
  error: '#ef4444',
  fatal: '#991b1b',
};

// Log context menu actions
export interface LogContextAction {
  id: string;
  label: string;
  icon?: string;
  handler: (entry: LogEntry) => void;
  condition?: (entry: LogEntry) => boolean;
}

// WebSocket message types for log streaming
export interface LogStreamMessage {
  type: 'log' | 'heartbeat' | 'error' | 'config';
  data?: any;
  timestamp: number;
}

// Log parser configuration
export interface LogParserConfig {
  format?: 'json' | 'plain' | 'structured';
  timestampFormat?: string;
  levelField?: string;
  messageField?: string;
  customPatterns?: Record<string, RegExp>;
}

// Utility functions
export function getLogLevelColor(level: LogLevel, scheme: LogColorScheme = DEFAULT_LOG_COLOR_SCHEME): string {
  return scheme[level] || '#374151';
}

export function getLogLevelWeight(level: LogLevel): number {
  const weights: Record<LogLevel, number> = {
    trace: 0,
    debug: 1,
    info: 2,
    warn: 3,
    error: 4,
    fatal: 5,
  };
  return weights[level] || 0;
}

export function filterLogs(logs: LogEntry[], filter: LogFilter): LogEntry[] {
  return logs.filter(log => {
    // Level filter
    if (filter.levels && filter.levels.length > 0 && !filter.levels.includes(log.level)) {
      return false;
    }

    // Source filter
    if (filter.sources && filter.sources.length > 0 && !filter.sources.includes(log.source.name)) {
      return false;
    }

    // Search filter
    if (filter.search) {
      const searchLower = filter.search.toLowerCase();
      const messageLower = log.message.toLowerCase();
      if (!messageLower.includes(searchLower)) {
        return false;
      }
    }

    // Regex filter
    if (filter.regex) {
      try {
        const regex = new RegExp(filter.regex);
        if (!regex.test(log.message)) {
          return false;
        }
      } catch (e) {
        // Invalid regex, skip filter
      }
    }

    // Time range filter
    if (filter.timeRange) {
      const logTime = log.timestamp.getTime();
      if (filter.timeRange.start && logTime < filter.timeRange.start.getTime()) {
        return false;
      }
      if (filter.timeRange.end && logTime > filter.timeRange.end.getTime()) {
        return false;
      }
    }

    // Task ID filter
    if (filter.taskIds && filter.taskIds.length > 0 && log.taskId && !filter.taskIds.includes(log.taskId)) {
      return false;
    }

    // Service name filter
    if (filter.serviceNames && filter.serviceNames.length > 0 && log.serviceName && !filter.serviceNames.includes(log.serviceName)) {
      return false;
    }

    // Container ID filter
    if (filter.containerIds && filter.containerIds.length > 0 && log.containerId && !filter.containerIds.includes(log.containerId)) {
      return false;
    }

    return true;
  });
}

export function formatLogTimestamp(date: Date, format: string = 'YYYY-MM-DD HH:mm:ss.SSS'): string {
  // Simple date formatting
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  const seconds = String(date.getSeconds()).padStart(2, '0');
  const milliseconds = String(date.getMilliseconds()).padStart(3, '0');

  return format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
    .replace('SSS', milliseconds);
}

export function parseLogLevel(levelStr: string): LogLevel {
  const normalized = levelStr.toLowerCase().trim();
  switch (normalized) {
    case 'trace':
    case 'trc':
      return 'trace';
    case 'debug':
    case 'dbg':
      return 'debug';
    case 'info':
    case 'inf':
      return 'info';
    case 'warn':
    case 'warning':
    case 'wrn':
      return 'warn';
    case 'error':
    case 'err':
      return 'error';
    case 'fatal':
    case 'ftl':
    case 'critical':
    case 'crit':
      return 'fatal';
    default:
      return 'info';
  }
}