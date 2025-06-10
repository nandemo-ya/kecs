#!/bin/bash
echo "SELECT * FROM task_definitions;" | duckdb /tmp/kecs-test/kecs.db