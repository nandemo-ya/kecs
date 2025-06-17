# Makefile for AWS Proxy

# Variables
BINARY_NAME := aws-proxy
DOCKER_IMAGE := kecs/aws-proxy
VERSION ?= dev
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags="-w -s -X main.Version=$(VERSION) -X main.CommitHash=$(COMMIT_HASH)"

.PHONY: all build clean test docker-build docker-push run

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v .

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		-f Dockerfile ..

docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Development targets
dev:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	./$(BINARY_NAME) -debug=true

deps:
	$(GOMOD) download
	$(GOMOD) verify

tidy:
	$(GOMOD) tidy