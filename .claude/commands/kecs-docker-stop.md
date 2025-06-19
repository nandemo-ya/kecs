Stop and remove the running KECS container.

Execute:
```bash
if docker ps -q -f name=kecs | grep -q .; then
    echo "Stopping KECS container..."
    docker stop kecs
    
    if [ $? -eq 0 ]; then
        echo "Removing KECS container..."
        docker rm kecs
        echo "KECS container stopped and removed successfully"
    else
        echo "Failed to stop KECS container"
        exit 1
    fi
else
    echo "KECS container is not running"
    
    # Check if container exists but is stopped
    if docker ps -aq -f name=kecs | grep -q .; then
        echo "Removing stopped KECS container..."
        docker rm kecs
    fi
fi
```