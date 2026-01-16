#!/bin/bash
# Test script for database-driven configuration migration
# BIG SKIES FRAMEWORK

set -e

echo "=== Testing Database-Driven Configuration Migration ==="

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo "ERROR: PostgreSQL is not running on localhost:5432"
    echo "Please start PostgreSQL first."
    exit 1
fi

# Database connection parameters
DB_USER="${DB_USER:-bigskies}"
DB_PASSWORD="${DB_PASSWORD:-bigskies}"
DB_NAME="${DB_NAME:-bigskies}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

export PGPASSWORD="$DB_PASSWORD"

echo "Step 1: Creating coordinator_config schema..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -f configs/sql/coordinator_config_schema.sql

if [ $? -eq 0 ]; then
    echo "✓ Schema created successfully"
else
    echo "✗ Failed to create schema"
    exit 1
fi

echo ""
echo "Step 2: Verifying configuration data..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
    "SELECT coordinator_name, config_key, config_value, config_type 
     FROM coordinator_config 
     WHERE coordinator_name = 'message-coordinator' 
     ORDER BY config_key;"

echo ""
echo "Step 3: Testing configuration query..."
MESSAGE_CONFIG=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
    "SELECT COUNT(*) FROM coordinator_config WHERE coordinator_name = 'message-coordinator';")

if [ "$MESSAGE_CONFIG" -ge 4 ]; then
    echo "✓ Configuration loaded: $MESSAGE_CONFIG entries found"
else
    echo "✗ Configuration incomplete: only $MESSAGE_CONFIG entries found"
    exit 1
fi

echo ""
echo "Step 4: Testing configuration update..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
    "UPDATE coordinator_config 
     SET config_value = '45'::jsonb 
     WHERE coordinator_name = 'message-coordinator' 
     AND config_key = 'monitor_interval';"

echo "✓ Configuration updated"

echo ""
echo "Step 5: Verifying history tracking..."
sleep 1  # Give trigger time to execute
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
    "SELECT coordinator_name, config_key, old_value, new_value, changed_at 
     FROM coordinator_config_history 
     WHERE coordinator_name = 'message-coordinator' 
     AND config_key = 'monitor_interval' 
     ORDER BY changed_at DESC 
     LIMIT 1;"

HISTORY_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
    "SELECT COUNT(*) FROM coordinator_config_history;")

if [ "$HISTORY_COUNT" -ge 1 ]; then
    echo "✓ History tracking works: $HISTORY_COUNT entries"
else
    echo "✗ History tracking failed"
    exit 1
fi

echo ""
echo "Step 6: Reverting test change..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
    "UPDATE coordinator_config 
     SET config_value = '30'::jsonb 
     WHERE coordinator_name = 'message-coordinator' 
     AND config_key = 'monitor_interval';"

echo "✓ Configuration reverted"

echo ""
echo "=== Configuration Migration Tests Passed ==="
echo ""
echo "To build and test the message-coordinator with database config:"
echo "  1. Build: make build"
echo "  2. Run: ./bin/message-coordinator --database-url='postgresql://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable'"
echo ""
echo "To test runtime config updates, publish MQTT message:"
echo "  Topic: bigskies/coordinator/config/update/message-coordinator"
echo "  Payload: {\"message_id\":\"test\",\"type\":\"command\",\"source\":\"test\",\"timestamp\":\"$(date -Iseconds)\",\"payload\":{\"config_key\":\"monitor_interval\",\"config_value\":60}}"
