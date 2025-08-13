package runtime

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerRuntime implements Runtime interface for Docker
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerRuntime{
		client: cli,
	}, nil
}

// Name returns the runtime name
func (d *DockerRuntime) Name() string {
	return "docker"
}

// IsAvailable checks if Docker is available
func (d *DockerRuntime) IsAvailable() bool {
	if d.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.client.Ping(ctx)
	return err == nil
}

// CreateContainer creates a new container
func (d *DockerRuntime) CreateContainer(ctx context.Context, config *ContainerConfig) (*Container, error) {
	// Convert port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, port := range config.Ports {
		containerPort := nat.Port(fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
		exposedPorts[containerPort] = struct{}{}

		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   port.HostIP,
				HostPort: strconv.Itoa(int(port.HostPort)),
			},
		}
	}

	// Convert mounts
	mounts := []mount.Mount{}
	for _, m := range config.Mounts {
		mountType := mount.TypeBind
		switch m.Type {
		case "volume":
			mountType = mount.TypeVolume
		case "tmpfs":
			mountType = mount.TypeTmpfs
		}

		mounts = append(mounts, mount.Mount{
			Type:     mountType,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          config.Env,
		Cmd:          config.Cmd,
		Labels:       config.Labels,
		ExposedPorts: exposedPorts,
		User:         config.User,
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		RestartPolicy: container.RestartPolicy{
			Name:              container.RestartPolicyMode(config.RestartPolicy.Name),
			MaximumRetryCount: config.RestartPolicy.MaximumRetryCount,
		},
		GroupAdd: config.GroupAdd,
	}

	// Set resource limits if specified
	if config.Resources != nil {
		hostConfig.Resources = container.Resources{
			CPUShares:  config.Resources.CPUShares,
			Memory:     config.Resources.Memory,
			MemorySwap: config.Resources.MemorySwap,
			CPUQuota:   config.Resources.CPUQuota,
			CPUPeriod:  config.Resources.CPUPeriod,
		}
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{}
	if len(config.Networks) > 0 {
		networkConfig.EndpointsConfig = make(map[string]*network.EndpointSettings)
		for _, net := range config.Networks {
			networkConfig.EndpointsConfig[net] = &network.EndpointSettings{}
		}
	}

	// Create container
	resp, err := d.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		nil,
		config.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Get container info
	return d.GetContainer(ctx, resp.ID)
}

// StartContainer starts a container
func (d *DockerRuntime) StartContainer(ctx context.Context, id string) error {
	return d.client.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container
func (d *DockerRuntime) StopContainer(ctx context.Context, id string, timeout *int) error {
	return d.client.ContainerStop(ctx, id, container.StopOptions{
		Timeout: timeout,
	})
}

// RemoveContainer removes a container
func (d *DockerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.client.ContainerRemove(ctx, id, container.RemoveOptions{
		Force: force,
	})
}

// GetContainer gets container information
func (d *DockerRuntime) GetContainer(ctx context.Context, id string) (*Container, error) {
	inspect, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	return d.dockerContainerToContainer(inspect), nil
}

// ListContainers lists containers
func (d *DockerRuntime) ListContainers(ctx context.Context, opts ListContainersOptions) ([]*Container, error) {
	filterArgs := filters.NewArgs()

	// Add label filters
	for k, v := range opts.Labels {
		filterArgs.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add name filters
	for _, name := range opts.Names {
		filterArgs.Add("name", name)
	}

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All:     opts.All,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Container, 0, len(containers))
	for _, c := range containers {
		result = append(result, d.dockerSummaryToContainer(c))
	}

	return result, nil
}

// ContainerLogs gets container logs
func (d *DockerRuntime) ContainerLogs(ctx context.Context, id string, opts LogsOptions) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: opts.Stdout,
		ShowStderr: opts.Stderr,
		Follow:     opts.Follow,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	})
}

// PullImage pulls an image
func (d *DockerRuntime) PullImage(ctx context.Context, imageName string, opts PullImageOptions) (io.ReadCloser, error) {
	pullOpts := image.PullOptions{}

	if opts.Auth != nil {
		// TODO: Implement auth encoding
	}

	return d.client.ImagePull(ctx, imageName, pullOpts)
}

// Helper functions

func (d *DockerRuntime) dockerContainerToContainer(inspect types.ContainerJSON) *Container {
	c := &Container{
		ID:       inspect.ID,
		Name:     strings.TrimPrefix(inspect.Name, "/"),
		Image:    inspect.Config.Image,
		State:    inspect.State.Status,
		Status:   inspect.State.Status,
		Created:  time.Now(), // Docker API doesn't provide creation time in ContainerJSON
		Labels:   inspect.Config.Labels,
		Networks: make([]string, 0),
	}

	// Extract port bindings
	for port, bindings := range inspect.HostConfig.PortBindings {
		for _, binding := range bindings {
			hostPort, _ := strconv.Atoi(binding.HostPort)
			c.Ports = append(c.Ports, PortBinding{
				ContainerPort: uint16(port.Int()),
				HostPort:      uint16(hostPort),
				Protocol:      port.Proto(),
				HostIP:        binding.HostIP,
			})
		}
	}

	// Extract networks
	for name := range inspect.NetworkSettings.Networks {
		c.Networks = append(c.Networks, name)
	}

	return c
}

func (d *DockerRuntime) dockerSummaryToContainer(summary types.Container) *Container {
	c := &Container{
		ID:       summary.ID,
		Image:    summary.Image,
		State:    summary.State,
		Status:   summary.Status,
		Created:  time.Unix(summary.Created, 0),
		Labels:   summary.Labels,
		Networks: make([]string, 0),
	}

	// Use first name without leading slash
	if len(summary.Names) > 0 {
		c.Name = strings.TrimPrefix(summary.Names[0], "/")
	}

	// Extract networks
	for name := range summary.NetworkSettings.Networks {
		c.Networks = append(c.Networks, name)
	}

	return c
}

// Close closes the Docker client
func (d *DockerRuntime) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}
