#!/bin/bash
# Database restore script for BIG SKIES Framework

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONTAINER_NAME="bigskies-postgres"
DB_NAME="bigskies"
DB_USER="bigskies"

echo "BIG SKIES Framework - Database Restore"
echo "======================================="
echo ""

# Check if backup file was provided
if [ $# -eq 0 ]; then
    echo "ERROR: No backup file specified."
    echo ""
    echo "Usage: $0 <backup_file.sql.gz>"
    echo ""
    echo "Available backups:"
    if [ -d "$PROJECT_ROOT/backups/database" ]; then
        ls -lht "$PROJECT_ROOT/backups/database" | head -11
    else
        echo "  (no backups found)"
    fi
    exit 1
fi

BACKUP_FILE="$1"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo "ERROR: Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "ERROR: PostgreSQL container '$CONTAINER_NAME' is not running."
    echo "Start services with: make docker-up"
    exit 1
fi

echo "⚠️  WARNING: This will replace the current database!"
echo ""
echo "Backup file: $BACKUP_FILE"
echo "Database: $DB_NAME"
echo "Container: $CONTAINER_NAME"
echo ""
read -p "Continue? Type 'yes' to proceed: " confirm

if [ "$confirm" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo ""
echo "Stopping coordinators to prevent database conflicts..."
docker-compose stop \
    bootstrap-coordinator \
    datastore-coordinator \
    security-coordinator \
    message-coordinator \
    application-coordinator \
    plugin-coordinator \
    telescope-coordinator \
    uielement-coordinator 2>/dev/null || true

echo ""
echo "Restoring database..."

# Decompress if gzipped
if [[ "$BACKUP_FILE" == *.gz ]]; then
    echo "Decompressing backup..."
    gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres
else
    cat "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres
fi

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Database restored successfully!"
    echo ""
    echo "Restarting coordinators..."
    docker-compose start \
        bootstrap-coordinator \
        datastore-coordinator \
        security-coordinator \
        message-coordinator \
        application-coordinator \
        plugin-coordinator \
        telescope-coordinator \
        uielement-coordinator 2>/dev/null || true
    
    echo ""
    echo "✅ Coordinators restarted!"
    echo ""
    echo "Verify with:"
    echo "  docker-compose ps"
    echo "  docker logs -f bigskies-bootstrap"
else
    echo ""
    echo "❌ Restore failed!"
    echo ""
    echo "You may need to restart services manually:"
    echo "  make docker-restart"
    exit 1
fi
