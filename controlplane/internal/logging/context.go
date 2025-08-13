package logging

import (
	"context"
	"log/slog"
)

type contextKey int

const loggerKey contextKey = iota

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger from the context, or the global logger if not found
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return GetGlobalLogger()
}

// WithRequestID adds a request ID to the logger in the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx).With("request_id", requestID)
	return WithLogger(ctx, logger)
}

// WithComponent adds a component name to the logger in the context
func WithComponent(ctx context.Context, component string) context.Context {
	logger := FromContext(ctx).With("component", component)
	return WithLogger(ctx, logger)
}

// WithOperation adds an operation name to the logger in the context
func WithOperation(ctx context.Context, operation string) context.Context {
	logger := FromContext(ctx).With("operation", operation)
	return WithLogger(ctx, logger)
}

// WithError adds an error to the logger in the context
func WithError(ctx context.Context, err error) context.Context {
	logger := FromContext(ctx).With("error", err)
	return WithLogger(ctx, logger)
}
