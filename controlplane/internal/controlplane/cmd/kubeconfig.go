package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfigOutputPath string
	kubeconfigRaw        bool
)

// kubeconfigCmd represents the kubeconfig command
func kubeconfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Manage kubeconfig for KECS k3d clusters",
		Long: `Get properly configured kubeconfig files for k3d clusters created by KECS.

This command automatically fixes common issues with k3d kubeconfig:
- Replaces host.docker.internal with 127.0.0.1
- Extracts and sets the correct port number
- Provides a working kubeconfig for kubectl access`,
	}

	cmd.AddCommand(getKubeconfigCmd())
	cmd.AddCommand(listKubeconfigCmd())

	return cmd
}

// getKubeconfigCmd returns the get subcommand
func getKubeconfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [cluster-name]",
		Short: "Get kubeconfig for a KECS cluster",
		Long: `Get a properly configured kubeconfig for the specified KECS cluster.

Examples:
  # Get kubeconfig and print to stdout
  kecs kubeconfig get test-cluster

  # Save kubeconfig to a file
  kecs kubeconfig get test-cluster -o ~/.kube/kecs-test-cluster

  # Get raw k3d kubeconfig without fixes
  kecs kubeconfig get test-cluster --raw`,
		Args: cobra.ExactArgs(1),
		RunE: runGetKubeconfig,
	}

	cmd.Flags().StringVarP(&kubeconfigOutputPath, "output", "o", "", "Write kubeconfig to file instead of stdout")
	cmd.Flags().BoolVar(&kubeconfigRaw, "raw", false, "Get raw k3d kubeconfig without applying fixes")

	return cmd
}

// listKubeconfigCmd returns the list subcommand
func listKubeconfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available KECS clusters",
		Long:  `List all KECS clusters that have corresponding k3d clusters.`,
		Args:  cobra.NoArgs,
		RunE:  runListKubeconfig,
	}
}

// runGetKubeconfig handles the get subcommand
func runGetKubeconfig(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	k3dClusterName := fmt.Sprintf("kecs-%s", clusterName)

	// Check if k3d cluster exists
	if !k3dClusterExists(k3dClusterName) {
		return fmt.Errorf("k3d cluster '%s' does not exist", k3dClusterName)
	}

	// Get kubeconfig from k3d
	kubeconfig, err := getK3dKubeconfig(k3dClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Apply fixes unless raw mode is requested
	if !kubeconfigRaw {
		kubeconfig, err = fixKubeconfig(kubeconfig, k3dClusterName)
		if err != nil {
			return fmt.Errorf("failed to fix kubeconfig: %w", err)
		}
	}

	// Output kubeconfig
	if kubeconfigOutputPath != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(kubeconfigOutputPath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			kubeconfigOutputPath = filepath.Join(home, kubeconfigOutputPath[2:])
		}

		// Create directory if needed
		dir := filepath.Dir(kubeconfigOutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write to file
		if err := os.WriteFile(kubeconfigOutputPath, []byte(kubeconfig), 0600); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Kubeconfig written to: %s\n", kubeconfigOutputPath)
	} else {
		// Print to stdout
		fmt.Fprint(cmd.OutOrStdout(), kubeconfig)
	}

	return nil
}

// runListKubeconfig handles the list subcommand
func runListKubeconfig(cmd *cobra.Command, args []string) error {
	// Get all k3d clusters
	clusters, err := listK3dClusters()
	if err != nil {
		return fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	// Filter for KECS clusters
	kecsClusters := []string{}
	for _, cluster := range clusters {
		if strings.HasPrefix(cluster, "kecs-") {
			// Extract KECS cluster name
			kecsName := strings.TrimPrefix(cluster, "kecs-")
			kecsClusters = append(kecsClusters, kecsName)
		}
	}

	if len(kecsClusters) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No KECS clusters found")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Available KECS clusters:")
	for _, cluster := range kecsClusters {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", cluster)
	}

	return nil
}

// k3dClusterExists checks if a k3d cluster exists
func k3dClusterExists(clusterName string) bool {
	cmd := exec.Command("k3d", "cluster", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Simple check for cluster name in output
	return strings.Contains(string(output), fmt.Sprintf(`"name":"%s"`, clusterName))
}

// getK3dKubeconfig gets the kubeconfig for a k3d cluster
func getK3dKubeconfig(clusterName string) (string, error) {
	cmd := exec.Command("k3d", "kubeconfig", "get", clusterName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get k3d kubeconfig: %w", err)
	}
	return string(output), nil
}

// fixKubeconfig applies necessary fixes to the kubeconfig
func fixKubeconfig(kubeconfigContent string, k3dClusterName string) (string, error) {
	// Load kubeconfig
	config, err := clientcmd.Load([]byte(kubeconfigContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Get the server URL from the first cluster
	var serverURL string
	for _, cluster := range config.Clusters {
		serverURL = cluster.Server
		break
	}

	if serverURL == "" {
		return "", fmt.Errorf("no server URL found in kubeconfig")
	}

	// Replace host.docker.internal with 127.0.0.1
	fixedContent := strings.ReplaceAll(kubeconfigContent, "host.docker.internal", "127.0.0.1")
	
	// Also replace 0.0.0.0 with 127.0.0.1
	fixedContent = strings.ReplaceAll(fixedContent, "0.0.0.0", "127.0.0.1")

	// Get the actual port from docker
	port, err := getK3dAPIPort(k3dClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to get API port: %w", err)
	}

	// Debug: log the port we found
	fmt.Fprintf(os.Stderr, "DEBUG: Found port %s for cluster %s\n", port, k3dClusterName)

	// Fix the port in the server URL
	// Handle various cases:
	// 1. Empty port (https://127.0.0.1:)
	// 2. Port with number (https://127.0.0.1:1234)
	// 3. No port at all (https://127.0.0.1)
	// Match patterns and replace with correct port
	re := regexp.MustCompile(`(https://127\.0\.0\.1)(:\d+)?(:)?`)
	fixedContent = re.ReplaceAllStringFunc(fixedContent, func(match string) string {
		// Debug: log what we're replacing
		fmt.Fprintf(os.Stderr, "DEBUG: Replacing '%s' with 'https://127.0.0.1:%s'\n", match, port)
		// Always replace with the correct format including port
		return fmt.Sprintf("https://127.0.0.1:%s", port)
	})

	return fixedContent, nil
}

// getK3dAPIPort gets the exposed API port for a k3d cluster
func getK3dAPIPort(k3dClusterName string) (string, error) {
	// Run docker ps to find the loadbalancer container
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}:{{.Ports}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list docker containers: %w", err)
	}

	// Look for the loadbalancer container
	lines := strings.Split(string(output), "\n")
	lbName := fmt.Sprintf("k3d-%s-serverlb", k3dClusterName)
	
	// Debug: log docker ps output
	fmt.Fprintf(os.Stderr, "DEBUG: Looking for container %s\n", lbName)
	
	for _, line := range lines {
		if strings.HasPrefix(line, lbName) {
			// Debug: log the line we found
			fmt.Fprintf(os.Stderr, "DEBUG: Found container line: %s\n", line)
			// Extract port from format: "0.0.0.0:50715->6443/tcp"
			re := regexp.MustCompile(`0\.0\.0\.0:(\d+)->6443/tcp`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				fmt.Fprintf(os.Stderr, "DEBUG: Extracted port: %s\n", matches[1])
				return matches[1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find API port for cluster %s", k3dClusterName)
}

// listK3dClusters lists all k3d clusters
func listK3dClusters() ([]string, error) {
	cmd := exec.Command("k3d", "cluster", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	// Parse JSON output to extract cluster names
	// Simple regex-based extraction for now
	re := regexp.MustCompile(`"name":"([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	clusters := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			clusters = append(clusters, match[1])
		}
	}

	return clusters, nil
}

func init() {
	RootCmd.AddCommand(kubeconfigCmd())
}