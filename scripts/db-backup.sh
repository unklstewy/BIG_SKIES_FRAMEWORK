#!/bin/bash
# Database backup script for BIG SKIES Framework

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_DIR="$PROJECT_ROOT/backups/database"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="bigskies_backup_${TIMESTAMP}.sql"
CONTAINER_NAME="bigskies-postgres"
DB_NAME="bigskies"
DB_USER="bigskies"

echo "BIG SKIES Framework - Database Backup"
echo "======================================"
echo ""

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "ERROR: PostgreSQL container '$CONTAINER_NAME' is not running."
    echo "Start services with: make docker-up"
    exit 1
fi

echo "Backing up database: $DB_NAME"
echo "Container: $CONTAINER_NAME"
echo "Output: $BACKUP_DIR/$BACKUP_FILE"
echo ""

# Create backup using pg_dump
docker exec -t "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" \
    --clean \
    --if-exists \
    --create \
    --no-owner \
    --no-acl \
    > "$BACKUP_DIR/$BACKUP_FILE"

# Check if backup was successful
if [ $? -eq 0 ] && [ -f "$BACKUP_DIR/$BACKUP_FILE" ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_DIR/$BACKUP_FILE" | cut -f1)
    echo "✅ Backup completed successfully!"
    echo "   File: $BACKUP_FILE"
    echo "   Size: $BACKUP_SIZE"
    echo "   Path: $BACKUP_DIR/$BACKUP_FILE"
    echo ""
    
    # Compress backup
    echo "Compressing backup..."
    gzip "$BACKUP_DIR/$BACKUP_FILE"
    COMPRESSED_SIZE=$(du -h "$BACKUP_DIR/${BACKUP_FILE}.gz" | cut -f1)
    echo "✅ Compressed: ${BACKUP_FILE}.gz ($COMPRESSED_SIZE)"
    echo ""
    
    # List recent backups
    echo "Recent backups:"
    ls -lht "$BACKUP_DIR" | head -6
    echo ""
    
    # Show restore command
    echo "To restore this backup:"
    echo "  ./scripts/db-restore.sh $BACKUP_DIR/${BACKUP_FILE}.gz"
else
    echo "❌ Backup failed!"
    exit 1
fi
