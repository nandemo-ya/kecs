package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/sidecar"
)

func main() {
	var (
		localstackEndpoint string
		listenPort         int
		services           string
		debug              bool
		timeout            time.Duration
	)

	flag.StringVar(&localstackEndpoint, "localstack-endpoint", os.Getenv("LOCALSTACK_ENDPOINT"), "LocalStack endpoint URL")
	flag.IntVar(&listenPort, "port", 8080, "Port to listen on")
	flag.StringVar(&services, "services", os.Getenv("PROXY_SERVICES"), "Comma-separated list of services to proxy")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "HTTP client timeout")

	klog.InitFlags(nil)
	flag.Parse()

	// Set defaults from environment if not provided
	if localstackEndpoint == "" {
		localstackEndpoint = "http://localstack.localstack.svc.cluster.local:4566"
	}

	if services == "" {
		services = "s3,dynamodb,sqs,sns,ssm,secretsmanager,cloudwatch"
	}

	// Parse services
	serviceList := strings.Split(services, ",")
	for i := range serviceList {
		serviceList[i] = strings.TrimSpace(serviceList[i])
	}

	config := &sidecar.ProxyConfig{
		LocalStackEndpoint: localstackEndpoint,
		ListenPort:         listenPort,
		Services:           serviceList,
		Debug:              debug,
		Timeout:            timeout,
	}

	klog.Infof("Starting AWS SDK Proxy")
	klog.Infof("LocalStack endpoint: %s", config.LocalStackEndpoint)
	klog.Infof("Listen port: %d", config.ListenPort)
	klog.Infof("Services: %v", config.Services)

	proxy := sidecar.NewProxy(config)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		klog.Info("Received shutdown signal")
		cancel()
	}()

	// Start proxy
	if err := proxy.Start(ctx); err != nil {
		klog.Errorf("Failed to start proxy: %v", err)
		os.Exit(1)
	}

	klog.Info("AWS SDK Proxy stopped")
}