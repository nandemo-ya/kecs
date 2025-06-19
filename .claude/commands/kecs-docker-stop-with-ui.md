Stop and remove the running KECS container with Web UI.

Execute:
```bash
if docker ps -q -f name=kecs-ui | grep -q .; then
    echo "Stopping KECS container with Web UI..."
    docker stop kecs-ui
    
    if [ $? -eq 0 ]; then
        echo "Removing KECS container..."
        docker rm kecs-ui
        echo "KECS container with Web UI stopped and removed successfully"
    else
        echo "Failed to stop KECS container"
        exit 1
    fi
else
    echo "KECS container with Web UI is not running"
    
    # Check if container exists but is stopped
    if docker ps -aq -f name=kecs-ui | grep -q .; then
        echo "Removing stopped KECS container..."
        docker rm kecs-ui
    fi
fi
```