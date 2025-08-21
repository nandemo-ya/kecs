package duckdb_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("TaskLogStore", func() {
	var (
		ctx      context.Context
		store    *duckdb.DuckDBStorage
		logStore storage.TaskLogStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).NotTo(HaveOccurred())

		err = store.Initialize(ctx)
		Expect(err).NotTo(HaveOccurred())

		logStore = store.TaskLogStore()
		Expect(logStore).NotTo(BeNil())
	})

	AfterEach(func() {
		if store != nil {
			store.Close()
		}
	})

	Describe("SaveLogs", func() {
		It("should save logs successfully", func() {
			logs := []storage.TaskLog{
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
					ContainerName: "app",
					Timestamp:     time.Now().Add(-1 * time.Minute),
					LogLine:       "Application started",
					LogLevel:      "INFO",
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
					ContainerName: "app",
					Timestamp:     time.Now(),
					LogLine:       "Processing request",
					LogLevel:      "DEBUG",
				},
			}

			err := logStore.SaveLogs(ctx, logs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle empty logs", func() {
			err := logStore.SaveLogs(ctx, []storage.TaskLog{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetLogs", func() {
		BeforeEach(func() {
			// Save some test logs
			testLogs := []storage.TaskLog{
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
					ContainerName: "app",
					Timestamp:     time.Now().Add(-5 * time.Minute),
					LogLine:       "Starting application",
					LogLevel:      "INFO",
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
					ContainerName: "app",
					Timestamp:     time.Now().Add(-4 * time.Minute),
					LogLine:       "Database connected",
					LogLevel:      "INFO",
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
					ContainerName: "app",
					Timestamp:     time.Now().Add(-3 * time.Minute),
					LogLine:       "Error connecting to service",
					LogLevel:      "ERROR",
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-456",
					ContainerName: "nginx",
					Timestamp:     time.Now().Add(-2 * time.Minute),
					LogLine:       "Request received",
					LogLevel:      "INFO",
				},
			}

			err := logStore.SaveLogs(ctx, testLogs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should retrieve logs by task ARN", func() {
			filter := storage.TaskLogFilter{
				TaskArn: "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
			}

			logs, err := logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(3))
		})

		It("should filter logs by container name", func() {
			filter := storage.TaskLogFilter{
				TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
				ContainerName: "app",
			}

			logs, err := logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(3))
			for _, log := range logs {
				Expect(log.ContainerName).To(Equal("app"))
			}
		})

		It("should filter logs by log level", func() {
			filter := storage.TaskLogFilter{
				LogLevel: "ERROR",
			}

			logs, err := logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(1))
			Expect(logs[0].LogLevel).To(Equal("ERROR"))
		})

		It("should search logs by text", func() {
			filter := storage.TaskLogFilter{
				SearchText: "connect",
			}

			logs, err := logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(2)) // "Database connected" and "Error connecting to service"
		})

		It("should support pagination", func() {
			filter := storage.TaskLogFilter{
				TaskArn: "arn:aws:ecs:us-east-1:123456789012:task/default/task-123",
				Limit:   2,
			}

			logs, err := logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(2))

			// Get next page
			filter.Offset = 2
			logs, err = logStore.GetLogs(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(HaveLen(1))
		})
	})

	Describe("DeleteOldLogs", func() {
		It("should delete logs older than specified time", func() {
			// Save logs with different timestamps
			oldTime := time.Now().Add(-48 * time.Hour)
			recentTime := time.Now().Add(-1 * time.Hour)

			logs := []storage.TaskLog{
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/old-task",
					ContainerName: "app",
					Timestamp:     oldTime,
					LogLine:       "Old log",
					CreatedAt:     oldTime,
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/recent-task",
					ContainerName: "app",
					Timestamp:     recentTime,
					LogLine:       "Recent log",
					CreatedAt:     recentTime,
				},
			}

			err := logStore.SaveLogs(ctx, logs)
			Expect(err).NotTo(HaveOccurred())

			// Delete logs older than 24 hours
			cutoffTime := time.Now().Add(-24 * time.Hour)
			deletedCount, err := logStore.DeleteOldLogs(ctx, cutoffTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(deletedCount).To(Equal(int64(1)))

			// Verify only recent log remains
			remainingLogs, err := logStore.GetLogs(ctx, storage.TaskLogFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(remainingLogs).To(HaveLen(1))
			Expect(remainingLogs[0].LogLine).To(Equal("Recent log"))
		})
	})

	Describe("DeleteTaskLogs", func() {
		It("should delete all logs for a specific task", func() {
			logs := []storage.TaskLog{
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-to-delete",
					ContainerName: "app",
					Timestamp:     time.Now(),
					LogLine:       "Log to delete",
				},
				{
					TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/default/task-to-keep",
					ContainerName: "app",
					Timestamp:     time.Now(),
					LogLine:       "Log to keep",
				},
			}

			err := logStore.SaveLogs(ctx, logs)
			Expect(err).NotTo(HaveOccurred())

			// Delete logs for specific task
			err = logStore.DeleteTaskLogs(ctx, "arn:aws:ecs:us-east-1:123456789012:task/default/task-to-delete")
			Expect(err).NotTo(HaveOccurred())

			// Verify only the other task's logs remain
			remainingLogs, err := logStore.GetLogs(ctx, storage.TaskLogFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(remainingLogs).To(HaveLen(1))
			Expect(remainingLogs[0].TaskArn).To(Equal("arn:aws:ecs:us-east-1:123456789012:task/default/task-to-keep"))
		})
	})
})
