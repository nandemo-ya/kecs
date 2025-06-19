Stop the running KECS server gracefully.

Execute:
```bash
if [ -f kecs.pid ]; then
    PID=$(cat kecs.pid)
    if ps -p $PID > /dev/null 2>&1; then
        echo "Stopping KECS (PID: $PID)..."
        kill -TERM $PID
        
        # Wait for graceful shutdown (up to 10 seconds)
        for i in {1..10}; do
            if ! ps -p $PID > /dev/null 2>&1; then
                echo "KECS stopped successfully"
                rm -f kecs.pid
                exit 0
            fi
            sleep 1
        done
        
        # Force kill if still running
        echo "Force stopping KECS..."
        kill -9 $PID 2>/dev/null
        rm -f kecs.pid
        echo "KECS force stopped"
    else
        echo "KECS is not running (stale PID file)"
        rm -f kecs.pid
    fi
else
    echo "KECS is not running (no PID file found)"
fi
```