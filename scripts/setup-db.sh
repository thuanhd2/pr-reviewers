#!/bin/bash
set -e

echo "Creating database user and database..."
psql -U postgres -c "CREATE USER pr_reviewer WITH PASSWORD 'pr_reviewer_dev';" 2>/dev/null || true
psql -U postgres -c "CREATE DATABASE pr_reviewer OWNER pr_reviewer;" 2>/dev/null || true
echo "Database ready."
