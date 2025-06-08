#!/bin/bash

# Direct database query test
echo "Checking database content for cluster..."

# Find the database file
DB_FILE=$(find ~/.kecs -name "kecs.db" 2>/dev/null | head -1)

if [ -z "$DB_FILE" ]; then
  echo "Database file not found"
  exit 1
fi

echo "Found database: $DB_FILE"

# Query the clusters table
echo -e "\nQuerying clusters table..."
echo "SELECT name, settings, capacity_providers, default_capacity_provider_strategy FROM clusters WHERE name = 'test-issue-78-complete';" | sqlite3 "$DB_FILE" 2>/dev/null || \
echo "SELECT name, settings, capacity_providers, default_capacity_provider_strategy FROM clusters WHERE name = 'test-issue-78-complete';" | duckdb "$DB_FILE" 2>/dev/null || \
echo "Could not query database (neither sqlite3 nor duckdb command available)"