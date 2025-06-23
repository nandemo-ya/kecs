#!/bin/bash
# Script to clean up leftover k3d clusters from KECS tests

echo "🧹 Cleaning up KECS k3d clusters..."

# List all kecs clusters
clusters=$(k3d cluster list | grep -E '^kecs-' | awk '{print $1}')

if [ -z "$clusters" ]; then
    echo "✅ No KECS clusters to clean up"
    exit 0
fi

# Count clusters
count=$(echo "$clusters" | wc -l | tr -d ' ')
echo "Found $count KECS cluster(s) to delete:"
echo "$clusters"

# Confirm deletion
read -p "Do you want to delete all these clusters? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cancelled"
    exit 1
fi

# Delete clusters
echo "$clusters" | xargs -I {} k3d cluster delete {}

echo "✅ Cleanup complete!"