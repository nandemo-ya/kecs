package postgres_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("TaskLogStore", func() {
	var (
		store storage.Storage
		ctx   context.Context
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("SaveLogs", func() {
		Context("when saving new log entries", func() {
			It("should save logs successfully", func() {
				logs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-123",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Application started successfully",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-123",
						ContainerName: "app",
						LogLevel:      "DEBUG",
						LogLine:       "Connecting to database",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-123",
						ContainerName: "sidecar",
						LogLevel:      "WARN",
						LogLine:       "Connection timeout, retrying...",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
				}

				err := store.TaskLogStore().SaveLogs(ctx, logs)
				Expect(err).NotTo(HaveOccurred())

				// Verify logs were saved
				filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-123",
				}
				savedLogs, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(savedLogs).To(HaveLen(3))
			})
		})

		Context("when saving empty log list", func() {
			It("should not error", func() {
				err := store.TaskLogStore().SaveLogs(ctx, []storage.TaskLog{})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("GetLogs", func() {
		BeforeEach(func() {
			// Create test logs
			now := time.Now()
			testLogs := []storage.TaskLog{
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					ContainerName: "app",
					LogLevel:      "INFO",
					LogLine:       "Info message",
					Timestamp:     now.Add(-2 * time.Hour),
					CreatedAt:     now,
				},
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					ContainerName: "app",
					LogLevel:      "ERROR",
					LogLine:       "Error occurred",
					Timestamp:     now.Add(-1 * time.Hour),
					CreatedAt:     now,
				},
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-2",
					ContainerName: "nginx",
					LogLevel:      "INFO",
					LogLine:       "Nginx started",
					Timestamp:     now.Add(-30 * time.Minute),
					CreatedAt:     now,
				},
			}
			err := store.TaskLogStore().SaveLogs(ctx, testLogs)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when filtering by task ARN", func() {
			It("should return logs for specific task", func() {
				filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
				}
				logs, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(HaveLen(2))
				for _, log := range logs {
					Expect(log.TaskArn).To(Equal("arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1"))
				}
			})
		})

		Context("when filtering by container name", func() {
			It("should return logs for specific container", func() {
				filter := storage.TaskLogFilter{
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					ContainerName: "app",
				}
				logs, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(HaveLen(2))
				for _, log := range logs {
					Expect(log.ContainerName).To(Equal("app"))
				}
			})
		})

		Context("when filtering by log level", func() {
			It("should return logs of specific level", func() {
				filter := storage.TaskLogFilter{
					TaskArn:  "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					LogLevel: "ERROR",
				}
				logs, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(HaveLen(1))
				Expect(logs[0].LogLevel).To(Equal("ERROR"))
			})
		})

		Context("when filtering by time range", func() {
			It("should return logs within time range", func() {
				now := time.Now()
				from := now.Add(-90 * time.Minute)
				to := now
				filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					From:    &from,
					To:      &to,
				}
				logs, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(HaveLen(1)) // Only the error log from 1 hour ago
				Expect(logs[0].LogLevel).To(Equal("ERROR"))
			})
		})

		Context("when using limit and offset", func() {
			It("should return paginated results", func() {
				// Add more logs for pagination testing
				moreLogs := []storage.TaskLog{}
				for i := 0; i < 10; i++ {
					moreLogs = append(moreLogs, storage.TaskLog{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-page",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       fmt.Sprintf("Log message %d", i),
						Timestamp:     time.Now().Add(time.Duration(-i) * time.Minute),
						CreatedAt:     time.Now(),
					})
				}
				err := store.TaskLogStore().SaveLogs(ctx, moreLogs)
				Expect(err).NotTo(HaveOccurred())

				// First page
				filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-page",
					Limit:   5,
					Offset:  0,
				}
				logs1, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs1).To(HaveLen(5))

				// Second page
				filter.Offset = 5
				logs2, err := store.TaskLogStore().GetLogs(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs2).To(HaveLen(5))

				// Verify different logs
				Expect(logs1[0].ID).NotTo(Equal(logs2[0].ID))
			})
		})
	})

	Describe("GetLogCount", func() {
		BeforeEach(func() {
			// Create test logs
			testLogs := []storage.TaskLog{
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/count-task",
					ContainerName: "app",
					LogLevel:      "INFO",
					LogLine:       "Info 1",
					Timestamp:     time.Now(),
					CreatedAt:     time.Now(),
				},
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/count-task",
					ContainerName: "app",
					LogLevel:      "INFO",
					LogLine:       "Info 2",
					Timestamp:     time.Now(),
					CreatedAt:     time.Now(),
				},
				{
					ID:            uuid.New().String(),
					TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/count-task",
					ContainerName: "app",
					LogLevel:      "ERROR",
					LogLine:       "Error 1",
					Timestamp:     time.Now(),
					CreatedAt:     time.Now(),
				},
			}
			err := store.TaskLogStore().SaveLogs(ctx, testLogs)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when counting all logs for a task", func() {
			It("should return correct count", func() {
				filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/count-task",
				}
				count, err := store.TaskLogStore().GetLogCount(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(3)))
			})
		})

		Context("when counting with filter", func() {
			It("should return filtered count", func() {
				filter := storage.TaskLogFilter{
					TaskArn:  "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/count-task",
					LogLevel: "INFO",
				}
				count, err := store.TaskLogStore().GetLogCount(ctx, filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(2)))
			})
		})
	})

	Describe("DeleteOldLogs", func() {
		Context("when deleting old logs", func() {
			It("should delete logs older than retention period", func() {
				// Create logs with different ages
				now := time.Now()
				oldLogs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/old-task",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Old log",
						Timestamp:     now.Add(-48 * time.Hour),
						CreatedAt:     now.Add(-48 * time.Hour),
					},
				}
				recentLogs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/recent-task",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Recent log",
						Timestamp:     now.Add(-1 * time.Hour),
						CreatedAt:     now.Add(-1 * time.Hour),
					},
				}

				err := store.TaskLogStore().SaveLogs(ctx, oldLogs)
				Expect(err).NotTo(HaveOccurred())
				err = store.TaskLogStore().SaveLogs(ctx, recentLogs)
				Expect(err).NotTo(HaveOccurred())

				// Delete logs older than 24 hours
				cutoffTime := now.Add(-24 * time.Hour)
				deletedCount, err := store.TaskLogStore().DeleteOldLogs(ctx, cutoffTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedCount).To(Equal(int64(1)))

				// Verify old log is deleted
				oldFilter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/old-task",
				}
				oldLogsResult, err := store.TaskLogStore().GetLogs(ctx, oldFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(oldLogsResult).To(BeEmpty())

				// Verify recent log still exists
				recentFilter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/recent-task",
				}
				recentLogsResult, err := store.TaskLogStore().GetLogs(ctx, recentFilter)
				Expect(err).NotTo(HaveOccurred())
				Expect(recentLogsResult).To(HaveLen(1))
			})
		})

		Context("when no logs to delete", func() {
			It("should return zero count", func() {
				// All logs are recent, none should be deleted
				recentLogs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Recent",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				err := store.TaskLogStore().SaveLogs(ctx, recentLogs)
				Expect(err).NotTo(HaveOccurred())

				cutoffTime := time.Now().Add(-24 * time.Hour)
				deletedCount, err := store.TaskLogStore().DeleteOldLogs(ctx, cutoffTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedCount).To(Equal(int64(0)))
			})
		})
	})

	Describe("DeleteTaskLogs", func() {
		Context("when deleting logs for a specific task", func() {
			It("should delete all logs for the task", func() {
				// Create logs for multiple tasks
				task1Logs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-1",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Task 1 log 1",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-1",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Task 1 log 2",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
				}
				task2Logs := []storage.TaskLog{
					{
						ID:            uuid.New().String(),
						TaskArn:       "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-2",
						ContainerName: "app",
						LogLevel:      "INFO",
						LogLine:       "Task 2 log",
						Timestamp:     time.Now(),
						CreatedAt:     time.Now(),
					},
				}

				err := store.TaskLogStore().SaveLogs(ctx, task1Logs)
				Expect(err).NotTo(HaveOccurred())
				err = store.TaskLogStore().SaveLogs(ctx, task2Logs)
				Expect(err).NotTo(HaveOccurred())

				// Delete logs for task-1
				err = store.TaskLogStore().DeleteTaskLogs(ctx, "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-1")
				Expect(err).NotTo(HaveOccurred())

				// Verify task-1 logs are deleted
				task1Filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-1",
				}
				task1Result, err := store.TaskLogStore().GetLogs(ctx, task1Filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(task1Result).To(BeEmpty())

				// Verify task-2 logs still exist
				task2Filter := storage.TaskLogFilter{
					TaskArn: "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/delete-task-2",
				}
				task2Result, err := store.TaskLogStore().GetLogs(ctx, task2Filter)
				Expect(err).NotTo(HaveOccurred())
				Expect(task2Result).To(HaveLen(1))
			})
		})

		Context("when deleting logs for non-existent task", func() {
			It("should not error", func() {
				err := store.TaskLogStore().DeleteTaskLogs(ctx, "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/non-existent")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
