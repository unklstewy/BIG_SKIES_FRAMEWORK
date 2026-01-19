#!/bin/bash
# Helper script to update .pgpass file in Docker shared volume

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PGPASS_FILE="$PROJECT_ROOT/.pgpass"
VOLUME_NAME="big_skies_framework_shared_secrets"

echo "BIG SKIES Framework - Update .pgpass in Docker Volume"
echo "======================================================"

# Check if .pgpass exists
if [ ! -f "$PGPASS_FILE" ]; then
    echo "ERROR: .pgpass file not found at $PGPASS_FILE"
    echo "Please create it first with your PostgreSQL credentials."
    exit 1
fi

# Check permissions
PERMS=$(stat -f "%Lp" "$PGPASS_FILE" 2>/dev/null || stat -c "%a" "$PGPASS_FILE" 2>/dev/null)
if [ "$PERMS" != "600" ]; then
    echo "WARNING: .pgpass has incorrect permissions ($PERMS), fixing..."
    chmod 0600 "$PGPASS_FILE"
fi

# Create volume if it doesn't exist
echo "Creating volume if needed: $VOLUME_NAME"
docker volume create "$VOLUME_NAME" > /dev/null 2>&1 || true

# Copy to volume
echo "Copying .pgpass to Docker volume..."
docker run --rm \
    -v "${VOLUME_NAME}:/shared/secrets" \
    -v "${PROJECT_ROOT}:/host" \
    alpine sh -c "
        mkdir -p /shared/secrets &&
        cp /host/.pgpass /shared/secrets/.pgpass &&
        chmod 0600 /shared/secrets/.pgpass &&
        chown 1000:1000 /shared/secrets/.pgpass &&
        echo 'File copied successfully:' &&
        ls -la /shared/secrets/.pgpass
    "

echo ""
echo "✅ .pgpass file updated in Docker volume successfully!"
echo ""
echo "To apply changes to running containers:"
echo "  docker-compose restart bootstrap-coordinator"
echo ""
echo "Or restart all coordinators:"
echo "  docker-compose restart"
