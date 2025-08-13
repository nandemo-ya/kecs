#\!/bin/bash
echo "Monitoring containers..."
while true; do
    echo "=== $(date) ==="
    docker ps | grep -E "(kecs|kind)" | head -10
    kind get clusters 2>/dev/null | grep kecs | head -10
    echo ""
    sleep 2
done
