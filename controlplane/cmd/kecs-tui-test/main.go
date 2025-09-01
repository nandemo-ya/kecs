package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/spf13/cobra"
)

var (
	instanceName string
	apiPort      int
	adminPort    int
	debug        bool
)

var rootCmd = &cobra.Command{
	Use:   "kecs-tui-test",
	Short: "TUI Test Tool for KECS",
	Long:  `A command-line tool to test and debug KECS TUI functionality without the interactive interface.`,
}

var logsCmd = &cobra.Command{
	Use:   "logs [taskId] [container]",
	Short: "Fetch and display logs for a task container",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		taskId := args[0]
		containerName := args[1]

		// Create API client
		baseURL := fmt.Sprintf("http://localhost:%d", adminPort)
		client := tui.NewLogAPIClient(baseURL)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fmt.Printf("Fetching logs for task=%s container=%s from %s\n", taskId, containerName, baseURL)

		// Fetch logs
		logs, err := client.GetLogs(ctx, taskId, containerName, false)
		if err != nil {
			log.Fatalf("Failed to fetch logs: %v", err)
		}

		if len(logs) == 0 {
			fmt.Println("No logs found")
			return
		}

		// Display logs
		fmt.Printf("Found %d log entries:\n", len(logs))
		fmt.Println("----------------------------------------")
		for _, logEntry := range logs {
			timestamp := logEntry.Timestamp.Format("15:04:05.000")
			fmt.Printf("[%s] [%-5s] %s\n", timestamp, logEntry.LogLevel, logEntry.LogLine)
		}
	},
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List all tasks",
	Run: func(cmd *cobra.Command, args []string) {
		// Create API client
		baseURL := fmt.Sprintf("http://localhost:%d", apiPort)
		client := api.NewHTTPClient(baseURL)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Fetching tasks from %s\n", baseURL)

		// List clusters first
		clusterArns, err := client.ListClusters(ctx, instanceName)
		if err != nil {
			log.Fatalf("Failed to list clusters: %v", err)
		}

		if len(clusterArns) == 0 {
			fmt.Println("No clusters found")
			return
		}

		// For each cluster, list tasks
		for _, clusterArn := range clusterArns {
			// Extract cluster name from ARN
			parts := strings.Split(clusterArn, "/")
			clusterName := parts[len(parts)-1]
			fmt.Printf("\nCluster: %s\n", clusterName)

			tasks, err := client.ListTasks(ctx, instanceName, clusterArn, "")
			if err != nil {
				fmt.Printf("  Error listing tasks: %v\n", err)
				continue
			}

			if len(tasks) == 0 {
				fmt.Println("  No tasks")
				continue
			}

			// Describe each task to get details
			for _, taskArn := range tasks {
				describedTasks, err := client.DescribeTasks(ctx, instanceName, clusterArn, []string{taskArn})
				if err != nil {
					fmt.Printf("  Error describing task %s: %v\n", taskArn, err)
					continue
				}

				for _, task := range describedTasks {
					fmt.Printf("  Task: %s\n", task.TaskArn)
					fmt.Printf("    Status: %s\n", task.LastStatus)
					fmt.Printf("    Desired: %s\n", task.DesiredStatus)
					if task.TaskDefinitionArn != "" {
						fmt.Printf("    Definition: %s\n", task.TaskDefinitionArn)
					}

					// List containers
					for _, container := range task.Containers {
						fmt.Printf("    Container: %s\n", container.Name)
						fmt.Printf("      Status: %s\n", container.LastStatus)
						if container.Reason != "" {
							fmt.Printf("      Reason: %s\n", container.Reason)
						}
					}
				}
			}
		}
	},
}

var apiTestCmd = &cobra.Command{
	Use:   "api-test",
	Short: "Test all API endpoints",
	Run: func(cmd *cobra.Command, args []string) {
		baseURL := fmt.Sprintf("http://localhost:%d", apiPort)
		adminURL := fmt.Sprintf("http://localhost:%d", adminPort)

		fmt.Println("Testing KECS API endpoints...")
		fmt.Printf("API URL: %s\n", baseURL)
		fmt.Printf("Admin URL: %s\n", adminURL)
		fmt.Println("----------------------------------------")

		// Test API health
		testEndpoint("API Health", baseURL+"/health")

		// Test Admin health
		testEndpoint("Admin Health", adminURL+"/health")

		// Create API client for further tests
		client := api.NewHTTPClient(baseURL)
		ctx := context.Background()

		// Test list clusters
		fmt.Print("Testing ListClusters... ")
		clusterArns, err := client.ListClusters(ctx, instanceName)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("OK (found %d clusters)\n", len(clusterArns))
			if debug && len(clusterArns) > 0 {
				fmt.Printf("  Sample cluster: %s\n", clusterArns[0])
			}
		}

		// Test list task definitions
		fmt.Print("Testing ListTaskDefinitions... ")
		taskDefs, err := client.ListTaskDefinitions(ctx, instanceName)
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("OK (found %d definitions)\n", len(taskDefs))
		}

		// Test list services (if clusters exist)
		if len(clusterArns) > 0 {
			fmt.Print("Testing ListServices... ")
			services, err := client.ListServices(ctx, instanceName, clusterArns[0])
			if err != nil {
				fmt.Printf("FAILED: %v\n", err)
			} else {
				fmt.Printf("OK (found %d services)\n", len(services))
			}

			// Test list tasks
			fmt.Print("Testing ListTasks... ")
			tasks, err := client.ListTasks(ctx, instanceName, clusterArns[0], "")
			if err != nil {
				fmt.Printf("FAILED: %v\n", err)
			} else {
				fmt.Printf("OK (found %d tasks)\n", len(tasks))
			}
		}
	},
}

func testEndpoint(name, url string) {
	fmt.Printf("Testing %s... ", name)

	// Use standard HTTP client for simple endpoint tests
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("OK")
	} else {
		fmt.Printf("FAILED (status %d)\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		if debug && len(body) > 0 {
			fmt.Printf("  Response: %s\n", string(body))
		}
	}
}

func init() {
	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&instanceName, "instance", "i", "sad-hamilton", "KECS instance name")
	rootCmd.PersistentFlags().IntVarP(&apiPort, "api-port", "p", 5373, "API port")
	rootCmd.PersistentFlags().IntVarP(&adminPort, "admin-port", "a", 5374, "Admin port")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")

	// Add commands
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(apiTestCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
