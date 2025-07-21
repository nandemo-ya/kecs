# Dev Mode

KECS provides a development mode that enables local image building and testing without pushing to external registries.

## Prerequisites

1. Add the following entry to your `/etc/hosts` file:
   ```
   127.0.0.1 k3d-kecs-registry.localhost
   ```

2. Ensure Docker is running

## Usage

### 1. Start the k3d registry

```bash
kecs registry start
```

This will:
- Create a local k3d registry at `k3d-kecs-registry.localhost:5000`
- Start the registry container
- Display instructions for next steps

### 2. Build and push the control plane image

```bash
make docker-push-dev
```

This will:
- Build the control plane Docker image
- Tag it as `k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest`
- Push it to the local k3d registry

### 3. Start KECS in dev mode

```bash
kecs start --dev
```

This will:
- Connect to the existing k3d registry
- Configure the control plane to use `k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest`
- Deploy KECS using the locally built image

## How it works

1. **Registry Management**: The `kecs registry` command manages a standalone k3d registry container that listens on port 5000.

2. **Image References**: In dev mode, the control plane image is referenced as `k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest` instead of the default `ghcr.io/nandemo-ya/kecs:latest`.

3. **Registry Connection**: The k3d cluster is connected to the registry, allowing pods to pull images from it.

4. **Name Resolution**: The `/etc/hosts` entry ensures that both the host machine and the k3d cluster can resolve `k3d-kecs-registry.localhost` to the registry container.

## Registry Commands

### Check registry status

```bash
kecs registry status
```

Shows whether the registry is running and provides helpful information.

### Stop the registry

```bash
kecs registry stop
```

Stops the registry container (but doesn't delete it).

## Troubleshooting

### Registry not accessible

If you see "connection refused" errors:

1. Check if the registry is running:
   ```bash
   docker ps | grep k3d-kecs-registry
   ```

2. Start the registry manually if needed:
   ```bash
   docker start k3d-kecs-registry.localhost
   ```

### Image pull errors

If pods fail with `ErrImagePull`:

1. Verify the image was pushed:
   ```bash
   curl http://k3d-kecs-registry.localhost:5000/v2/_catalog
   ```

2. Check the image tags:
   ```bash
   curl http://k3d-kecs-registry.localhost:5000/v2/nandemo-ya/kecs-controlplane/tags/list
   ```

3. Ensure `/etc/hosts` has the registry entry

## Benefits

- **Fast iteration**: No need to push to external registries
- **Offline development**: Works without internet connection
- **Cost savings**: No registry storage costs
- **Security**: Images stay on local machine