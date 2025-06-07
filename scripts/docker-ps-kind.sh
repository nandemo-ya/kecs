#!/bin/bash
# Helper script to show containers running inside kind clusters
# Similar to "docker ps" but for containers inside kind

set -e

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS] [CLUSTER_NAME]"
    echo ""
    echo "Show containers running inside kind clusters (similar to docker ps)"
    echo ""
    echo "Options:"
    echo "  -a, --all       Show all containers (default shows only running)"
    echo "  -h, --help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Show containers in all kind clusters"
    echo "  $0 my-cluster         # Show containers in specific cluster"
    echo "  $0 -a my-cluster      # Show all containers including stopped ones"
}

# Parse arguments
SHOW_ALL=""
CLUSTER_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--all)
            SHOW_ALL="-a"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            CLUSTER_NAME="$1"
            shift
            ;;
    esac
done

# Function to format container info
format_container_info() {
    local cluster=$1
    echo "=== Cluster: $cluster ==="
    echo "CONTAINER ID    IMAGE                                               COMMAND                  CREATED         STATUS          NAMES"
    docker exec "$cluster-control-plane" crictl ps $SHOW_ALL --no-trunc | tail -n +2 | while IFS= read -r line; do
        # Extract fields from crictl output
        container_id=$(echo "$line" | awk '{print substr($1, 1, 12)}')
        image=$(echo "$line" | awk '{print $2}' | awk -F'/' '{print $NF}' | sed 's/:.*$//')
        created=$(echo "$line" | awk '{print $3, $4, $5}')
        state=$(echo "$line" | awk '{print $6}')
        name=$(echo "$line" | awk '{print $7}')
        
        # Format similar to docker ps
        printf "%-15s %-50s %-24s %-15s %-15s %s\n" \
            "$container_id" "$image" "" "$created" "$state" "$name"
    done
    echo ""
}

# Get list of kind clusters
if [ -n "$CLUSTER_NAME" ]; then
    # Check if specified cluster exists
    if ! kind get clusters | grep -q "^$CLUSTER_NAME$"; then
        echo "Error: Cluster '$CLUSTER_NAME' not found"
        echo "Available clusters:"
        kind get clusters
        exit 1
    fi
    clusters="$CLUSTER_NAME"
else
    # Get all clusters
    clusters=$(kind get clusters)
    if [ -z "$clusters" ]; then
        echo "No kind clusters found"
        exit 0
    fi
fi

# Show containers for each cluster
for cluster in $clusters; do
    format_container_info "$cluster"
done