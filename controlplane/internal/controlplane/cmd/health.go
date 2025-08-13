package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	healthURL      string
	healthTimeout  time.Duration
	healthDetailed bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of KECS control plane",
	Long:  `Check the health status of the KECS control plane by querying the admin health endpoint.`,
	RunE:  runHealth,
}

func init() {
	RootCmd.AddCommand(healthCmd)

	healthCmd.Flags().StringVar(&healthURL, "url", "http://localhost:8081", "Admin server URL")
	healthCmd.Flags().DurationVar(&healthTimeout, "timeout", 5*time.Second, "Request timeout")
	healthCmd.Flags().BoolVar(&healthDetailed, "detailed", false, "Show detailed health information")
}

func runHealth(cmd *cobra.Command, args []string) error {
	client := &http.Client{
		Timeout: healthTimeout,
	}

	// Choose endpoint based on detailed flag
	endpoint := "/health"
	if healthDetailed {
		endpoint = "/health/detailed"
	}

	resp, err := client.Get(healthURL + endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to admin server: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	// Pretty print the result
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))

	// Exit with appropriate code
	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}

	return nil
}
