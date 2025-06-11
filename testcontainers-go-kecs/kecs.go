// Package kecs provides a testcontainers integration for KECS (Kubernetes-based ECS Compatible Service).
// It allows you to run KECS in Docker containers for integration testing with AWS ECS-compatible APIs.
package kecs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// DefaultImage is the default KECS Docker image
	DefaultImage = "ghcr.io/nandemo-ya/kecs:latest"
	
	// DefaultAPIPort is the default port for the ECS API
	DefaultAPIPort = "8080"
	
	// DefaultAdminPort is the default port for the admin API (health checks)
	DefaultAdminPort = "8081"
	
	// DefaultRegion is the default AWS region
	DefaultRegion = "us-east-1"
)

// Container represents a running KECS container
type Container struct {
	testcontainers.Container
	endpoint   string
	adminEndpoint string
	region     string
}

// Option is a functional option for configuring KECS container
type Option func(*containerOptions)

type containerOptions struct {
	image         string
	region        string
	apiPort       string
	adminPort     string
	env           map[string]string
	testMode      bool
	waitTimeout   time.Duration
	logConsumer   testcontainers.LogConsumer
}

// WithImage sets a custom Docker image
func WithImage(image string) Option {
	return func(o *containerOptions) {
		o.image = image
	}
}

// WithRegion sets the AWS region
func WithRegion(region string) Option {
	return func(o *containerOptions) {
		o.region = region
	}
}

// WithAPIPort sets a custom API port
func WithAPIPort(port string) Option {
	return func(o *containerOptions) {
		o.apiPort = port
	}
}

// WithAdminPort sets a custom admin port
func WithAdminPort(port string) Option {
	return func(o *containerOptions) {
		o.adminPort = port
	}
}

// WithEnv sets additional environment variables
func WithEnv(env map[string]string) Option {
	return func(o *containerOptions) {
		for k, v := range env {
			o.env[k] = v
		}
	}
}

// WithTestMode enables test mode (no Kubernetes required)
func WithTestMode() Option {
	return func(o *containerOptions) {
		o.testMode = true
		o.env["KECS_TEST_MODE"] = "true"
	}
}

// WithWaitTimeout sets the timeout for waiting for the container to be ready
func WithWaitTimeout(timeout time.Duration) Option {
	return func(o *containerOptions) {
		o.waitTimeout = timeout
	}
}

// WithLogConsumer sets a log consumer for container logs
func WithLogConsumer(consumer testcontainers.LogConsumer) Option {
	return func(o *containerOptions) {
		o.logConsumer = consumer
	}
}

// StartContainer starts a new KECS container with the given options
func StartContainer(ctx context.Context, opts ...Option) (*Container, error) {
	options := &containerOptions{
		image:       DefaultImage,
		region:      DefaultRegion,
		apiPort:     DefaultAPIPort,
		adminPort:   DefaultAdminPort,
		env:         make(map[string]string),
		testMode:    false,
		waitTimeout: 60 * time.Second,
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Set default environment variables
	env := map[string]string{
		"AWS_REGION":        options.region,
		"AWS_DEFAULT_REGION": options.region,
	}
	
	// Merge with user-provided environment variables
	for k, v := range options.env {
		env[k] = v
	}

	// Prepare container request
	req := testcontainers.ContainerRequest{
		Image:        options.image,
		ExposedPorts: []string{options.apiPort + "/tcp", options.adminPort + "/tcp"},
		Env:          env,
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/health").
				WithPort(nat.Port(options.adminPort+"/tcp")).
				WithStartupTimeout(options.waitTimeout),
		),
	}

	// Add log consumer if provided
	if options.logConsumer != nil {
		req.LogConsumerCfg = &testcontainers.LogConsumerConfig{
			Consumers: []testcontainers.LogConsumer{options.logConsumer},
		}
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start KECS container: %w", err)
	}

	// Get container endpoints
	apiEndpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get API endpoint: %w", err)
	}

	adminHost, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get admin host: %w", err)
	}

	adminPort, err := container.MappedPort(ctx, nat.Port(options.adminPort))
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get admin port: %w", err)
	}

	adminEndpoint := fmt.Sprintf("http://%s:%s", adminHost, adminPort.Port())

	// Override API endpoint with correct port
	apiHost, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get API host: %w", err)
	}

	apiPort, err := container.MappedPort(ctx, nat.Port(options.apiPort))
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get API port: %w", err)
	}

	apiEndpoint = fmt.Sprintf("http://%s:%s", apiHost, apiPort.Port())

	return &Container{
		Container:     container,
		endpoint:      apiEndpoint,
		adminEndpoint: adminEndpoint,
		region:        options.region,
	}, nil
}

// Endpoint returns the API endpoint URL
func (c *Container) Endpoint() string {
	return c.endpoint
}

// AdminEndpoint returns the admin endpoint URL (for health checks, etc.)
func (c *Container) AdminEndpoint() string {
	return c.adminEndpoint
}

// Region returns the configured AWS region
func (c *Container) Region() string {
	return c.region
}

// NewECSClient creates a new AWS ECS client configured to use this KECS container
func (c *Container) NewECSClient(ctx context.Context) (*ecs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == ecs.ServiceID {
					return aws.Endpoint{
						URL:               c.endpoint,
						HostnameImmutable: true,
						Source:            aws.EndpointSourceCustom,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return ecs.NewFromConfig(cfg), nil
}

// Cleanup terminates the container and cleans up resources
func (c *Container) Cleanup(ctx context.Context) error {
	if c.Container != nil {
		return c.Container.Terminate(ctx)
	}
	return nil
}