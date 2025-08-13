package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/platforms"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	defaultNamespace = "kecs"
	labelPrefix      = "io.kecs."
)

// ContainerdRuntime implements Runtime interface for containerd
type ContainerdRuntime struct {
	client     *containerd.Client
	namespace  string
	socketPath string
}

// NewContainerdRuntime creates a new containerd runtime
func NewContainerdRuntime(socketPath string) (*ContainerdRuntime, error) {
	if socketPath == "" {
		// Try common socket paths
		socketPaths := []string{
			"/run/containerd/containerd.sock",
			"/var/run/containerd/containerd.sock",
			"/run/k3s/containerd/containerd.sock",
		}

		for _, path := range socketPaths {
			if _, err := os.Stat(path); err == nil {
				socketPath = path
				break
			}
		}

		if socketPath == "" {
			return nil, fmt.Errorf("containerd socket not found")
		}
	}

	client, err := containerd.New(socketPath)
	if err != nil {
		return nil, err
	}

	return &ContainerdRuntime{
		client:     client,
		namespace:  defaultNamespace,
		socketPath: socketPath,
	}, nil
}

// Name returns the runtime name
func (c *ContainerdRuntime) Name() string {
	return "containerd"
}

// IsAvailable checks if containerd is available
func (c *ContainerdRuntime) IsAvailable() bool {
	if c.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.Version(ctx)
	return err == nil
}

// CreateContainer creates a new container
func (c *ContainerdRuntime) CreateContainer(ctx context.Context, config *ContainerConfig) (*Container, error) {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	// Pull image if not exists
	image, err := c.ensureImage(ctx, config.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure image: %w", err)
	}

	// Create container with OCI options
	opts := []containerd.NewContainerOpts{
		containerd.WithImage(image),
		containerd.WithNewSnapshot(config.Name+"-snapshot", image),
		containerd.WithNewSpec(
			c.buildOCISpec(config)...,
		),
	}

	// Add labels
	labels := map[string]string{}
	for k, v := range config.Labels {
		labels[labelPrefix+k] = v
	}
	labels[labelPrefix+"name"] = config.Name
	opts = append(opts, containerd.WithContainerLabels(labels))

	// Create container
	container, err := c.client.NewContainer(ctx, config.Name, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return c.containerdToContainer(container)
}

// StartContainer starts a container
func (c *ContainerdRuntime) StartContainer(ctx context.Context, id string) error {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}

	// Create task
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Start task
	if err := task.Start(ctx); err != nil {
		task.Delete(ctx)
		return fmt.Errorf("failed to start task: %w", err)
	}

	return nil
}

// StopContainer stops a container
func (c *ContainerdRuntime) StopContainer(ctx context.Context, id string, timeout *int) error {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil // Already stopped
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Send SIGTERM
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill task: %w", err)
	}

	// Wait for task to exit or timeout
	exitCh, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait task: %w", err)
	}

	timeoutDuration := 30 * time.Second
	if timeout != nil {
		timeoutDuration = time.Duration(*timeout) * time.Second
	}

	select {
	case <-exitCh:
		// Task exited
	case <-time.After(timeoutDuration):
		// Force kill
		task.Kill(ctx, syscall.SIGKILL)
	}

	// Delete task
	_, err = task.Delete(ctx)
	return err
}

// RemoveContainer removes a container
func (c *ContainerdRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load container: %w", err)
	}

	// Stop task if running
	if force {
		if task, err := container.Task(ctx, nil); err == nil {
			task.Kill(ctx, syscall.SIGKILL)
			task.Delete(ctx)
		}
	}

	// Delete container
	return container.Delete(ctx, containerd.WithSnapshotCleanup)
}

// GetContainer gets container information
func (c *ContainerdRuntime) GetContainer(ctx context.Context, id string) (*Container, error) {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	return c.containerdToContainer(container)
}

// ListContainers lists containers
func (c *ContainerdRuntime) ListContainers(ctx context.Context, opts ListContainersOptions) ([]*Container, error) {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	// Build filters
	var filters []string
	for k, v := range opts.Labels {
		filters = append(filters, fmt.Sprintf("labels.%s%s==%s", labelPrefix, k, v))
	}

	containers, err := c.client.Containers(ctx, filters...)
	if err != nil {
		return nil, err
	}

	result := make([]*Container, 0, len(containers))
	for _, container := range containers {
		// Check name filter
		if len(opts.Names) > 0 {
			info, err := container.Info(ctx)
			if err != nil {
				continue
			}

			found := false
			for _, name := range opts.Names {
				if info.ID == name || info.Labels[labelPrefix+"name"] == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		cont, err := c.containerdToContainer(container)
		if err == nil {
			result = append(result, cont)
		}
	}

	return result, nil
}

// ContainerLogs gets container logs
func (c *ContainerdRuntime) ContainerLogs(ctx context.Context, id string, opts LogsOptions) (io.ReadCloser, error) {
	// TODO: Implement log streaming for containerd
	// This is more complex as containerd doesn't have built-in log management
	// Logs are typically handled by the CRI or runtime
	return nil, fmt.Errorf("log streaming not yet implemented for containerd")
}

// PullImage pulls an image
func (c *ContainerdRuntime) PullImage(ctx context.Context, imageName string, opts PullImageOptions) (io.ReadCloser, error) {
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	// Use platform default if not specified
	platformStr := platforms.DefaultString()
	if opts.Platform != "" {
		platformStr = opts.Platform
	}

	// Pull image
	_, err := c.client.Pull(ctx, imageName,
		containerd.WithPlatform(platformStr),
		containerd.WithPullUnpack,
	)
	if err != nil {
		return nil, err
	}

	// Return a dummy reader since containerd doesn't provide progress
	return io.NopCloser(strings.NewReader("Image pulled successfully")), nil
}

// Helper functions

func (c *ContainerdRuntime) ensureImage(ctx context.Context, imageName string) (containerd.Image, error) {
	// Check if image exists
	image, err := c.client.GetImage(ctx, imageName)
	if err == nil {
		return image, nil
	}

	// Pull image
	image, err = c.client.Pull(ctx, imageName,
		containerd.WithPlatform(platforms.DefaultString()),
		containerd.WithPullUnpack,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	return image, nil
}

func (c *ContainerdRuntime) buildOCISpec(config *ContainerConfig) []oci.SpecOpts {
	opts := []oci.SpecOpts{
		oci.WithImageConfigArgs(nil, []string{}),
		oci.WithHostname(config.Name),
		oci.WithEnv(config.Env),
	}

	// Add command if specified
	if len(config.Cmd) > 0 {
		opts = append(opts, oci.WithProcessArgs(config.Cmd...))
	}

	// Add mounts
	for _, m := range config.Mounts {
		opts = append(opts, oci.WithMounts([]specs.Mount{
			{
				Source:      m.Source,
				Destination: m.Target,
				Type:        m.Type,
				Options:     c.getMountOptions(m),
			},
		}))
	}

	// Add resource limits using a custom function
	if config.Resources != nil {
		opts = append(opts, withResources(config.Resources))
	}

	return opts
}

func (c *ContainerdRuntime) getMountOptions(m Mount) []string {
	opts := []string{"rbind"}
	if m.ReadOnly {
		opts = append(opts, "ro")
	} else {
		opts = append(opts, "rw")
	}
	return opts
}

// withResources creates an OCI spec option for resource limits
func withResources(res *Resources) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, container *containers.Container, spec *oci.Spec) error {
		if spec.Linux == nil {
			spec.Linux = &specs.Linux{}
		}
		if spec.Linux.Resources == nil {
			spec.Linux.Resources = &specs.LinuxResources{}
		}

		// Set CPU resources
		if res.CPUShares > 0 || res.CPUQuota > 0 || res.CPUPeriod > 0 {
			if spec.Linux.Resources.CPU == nil {
				spec.Linux.Resources.CPU = &specs.LinuxCPU{}
			}
			if res.CPUShares > 0 {
				shares := uint64(res.CPUShares)
				spec.Linux.Resources.CPU.Shares = &shares
			}
			if res.CPUQuota > 0 {
				spec.Linux.Resources.CPU.Quota = &res.CPUQuota
			}
			if res.CPUPeriod > 0 {
				period := uint64(res.CPUPeriod)
				spec.Linux.Resources.CPU.Period = &period
			}
		}

		// Set memory resources
		if res.Memory > 0 || res.MemorySwap > 0 {
			if spec.Linux.Resources.Memory == nil {
				spec.Linux.Resources.Memory = &specs.LinuxMemory{}
			}
			if res.Memory > 0 {
				spec.Linux.Resources.Memory.Limit = &res.Memory
			}
			if res.MemorySwap > 0 {
				spec.Linux.Resources.Memory.Swap = &res.MemorySwap
			}
		}

		return nil
	}
}

func (c *ContainerdRuntime) containerdToContainer(container containerd.Container) (*Container, error) {
	ctx := namespaces.WithNamespace(context.Background(), c.namespace)

	info, err := container.Info(ctx)
	if err != nil {
		return nil, err
	}

	// Get task status
	state := "created"
	if task, err := container.Task(ctx, nil); err == nil {
		status, err := task.Status(ctx)
		if err == nil {
			state = string(status.Status)
		}
	}

	// Extract labels
	labels := make(map[string]string)
	for k, v := range info.Labels {
		if strings.HasPrefix(k, labelPrefix) {
			labels[strings.TrimPrefix(k, labelPrefix)] = v
		}
	}

	return &Container{
		ID:      info.ID,
		Name:    info.Labels[labelPrefix+"name"],
		Image:   info.Image,
		State:   state,
		Status:  state,
		Created: info.CreatedAt,
		Labels:  labels,
	}, nil
}

// Close closes the containerd client
func (c *ContainerdRuntime) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
