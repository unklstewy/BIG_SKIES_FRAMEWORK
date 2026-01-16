#!/bin/bash
# Deploy Seestar telescope configuration to ASCOM simulator
# Usage: ./deploy-config.sh <model> <mount-type>
#   model: s30, s30-pro, s50
#   mount-type: altaz, equatorial, german-equatorial

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}"
TARGET_DIR="/tmp/ascom-config/alpaca/ascom-alpaca-simulator"
CONTAINER_NAME="ascom-alpaca-simulator"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Usage function
usage() {
    echo "Usage: $0 <model> <mount-type>"
    echo ""
    echo "Models:"
    echo "  s30         - Seestar S30 (30mm f/5, 150mm FL)"
    echo "  s30-pro     - Seestar S30 Pro (30mm f/5.3, 160mm FL, 4K)"
    echo "  s50         - Seestar S50 (50mm f/5, 250mm FL)"
    echo ""
    echo "Mount Types:"
    echo "  altaz               - Altitude-Azimuth mount"
    echo "  equatorial          - Equatorial mount (polar-aligned)"
    echo "  german-equatorial   - German Equatorial mount (with meridian flip)"
    echo ""
    echo "Example:"
    echo "  $0 s50 altaz"
    exit 1
}

# Check arguments
if [ $# -ne 2 ]; then
    usage
fi

MODEL="$1"
MOUNT_TYPE="$2"

# Validate model
if [[ ! "$MODEL" =~ ^(s30|s30-pro|s50)$ ]]; then
    echo -e "${RED}Error: Invalid model '$MODEL'${NC}"
    usage
fi

# Validate mount type
if [[ ! "$MOUNT_TYPE" =~ ^(altaz|equatorial|german-equatorial)$ ]]; then
    echo -e "${RED}Error: Invalid mount type '$MOUNT_TYPE'${NC}"
    usage
fi

SOURCE_DIR="${CONFIG_DIR}/${MODEL}/${MOUNT_TYPE}"

# Check if source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}Error: Configuration directory not found: $SOURCE_DIR${NC}"
    exit 1
fi

# Display configuration info
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  ASCOM Seestar Configuration Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  Model:      ${GREEN}$(echo "$MODEL" | tr '[:lower:]' '[:upper:]')${NC}"
echo -e "  Mount Type: ${GREEN}${MOUNT_TYPE}${NC}"
echo -e "  Source:     ${SOURCE_DIR}"
echo -e "  Target:     ${TARGET_DIR}"
echo ""

# Check if Docker container exists
if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}Warning: Container '${CONTAINER_NAME}' not found.${NC}"
    echo -e "${YELLOW}Make sure the ASCOM simulator container is created before deploying.${NC}"
    echo ""
fi

# Create target directory if it doesn't exist
echo -e "${BLUE}[1/4]${NC} Creating target directory..."
mkdir -p "$TARGET_DIR"

# Backup existing configuration if it exists
if [ "$(ls -A $TARGET_DIR 2>/dev/null)" ]; then
    BACKUP_DIR="${TARGET_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${BLUE}[2/4]${NC} Backing up existing configuration to: ${BACKUP_DIR}"
    cp -r "$TARGET_DIR" "$BACKUP_DIR"
else
    echo -e "${BLUE}[2/4]${NC} No existing configuration to backup"
fi

# Copy new configuration
echo -e "${BLUE}[3/4]${NC} Deploying configuration..."
cp -r "$SOURCE_DIR"/* "$TARGET_DIR/"

# Count deployed files
FILE_COUNT=$(find "$TARGET_DIR" -name "instance-0.xml" | wc -l | tr -d ' ')
echo -e "${GREEN}      ✓ Deployed $FILE_COUNT device configurations${NC}"

# Restart container if it's running
echo -e "${BLUE}[4/5]${NC} Restarting ASCOM simulator..."
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    docker restart "$CONTAINER_NAME" > /dev/null 2>&1
    echo -e "${GREEN}      ✓ Container restarted${NC}"
    sleep 2
    
    # Check if container is healthy
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo -e "${GREEN}      ✓ Container is running${NC}"
    else
        echo -e "${RED}      ✗ Container failed to start. Check logs: docker logs ${CONTAINER_NAME}${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}      ! Container not running. Start it with: docker start ${CONTAINER_NAME}${NC}"
fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Configuration deployed successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}[5/5]${NC} Connecting all devices..."

# List of device types to connect
DEVICE_TYPES=("telescope" "camera" "filterwheel" "focuser" "switch")
CONNECTED_COUNT=0
FAILED_DEVICES=()

# Wait a bit more for API to be fully ready
sleep 3

# Connect each device
for DEVICE in "${DEVICE_TYPES[@]}"; do
    echo -ne "      Connecting ${DEVICE}..."
    
    # Attempt to connect the device
    RESPONSE=$(curl -s -X PUT "http://localhost:32323/api/v1/${DEVICE}/0/connected" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "Connected=true&ClientID=1&ClientTransactionID=1" 2>&1)
    
    # Check if connection was successful
    if echo "$RESPONSE" | grep -q '"ErrorNumber":0' 2>/dev/null; then
        echo -e "${GREEN} ✓${NC}"
        ((CONNECTED_COUNT++))
    else
        echo -e "${YELLOW} ⚠${NC}"
        FAILED_DEVICES+=("$DEVICE")
    fi
done

echo ""
if [ ${CONNECTED_COUNT} -eq ${#DEVICE_TYPES[@]} ]; then
    echo -e "${GREEN}      ✓ All ${CONNECTED_COUNT} devices connected successfully${NC}"
elif [ ${CONNECTED_COUNT} -gt 0 ]; then
    echo -e "${YELLOW}      ! ${CONNECTED_COUNT}/${#DEVICE_TYPES[@]} devices connected${NC}"
    echo -e "${YELLOW}        Failed: ${FAILED_DEVICES[*]}${NC}"
else
    echo -e "${RED}      ✗ Failed to connect devices. Check API availability.${NC}"
fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Configuration deployed successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "Access points:"
echo -e "  Web UI:     ${BLUE}http://localhost:32323${NC}"
echo -e "  API:        ${BLUE}http://localhost:32323/api/v1${NC}"
echo -e "  Swagger:    ${BLUE}http://localhost:32323/swagger${NC}"
echo ""
echo -e "Verify configuration:"
echo -e "  ${YELLOW}curl -s http://localhost:32323/api/v1/telescope/0/aperturearea | python3 -m json.tool${NC}"
echo -e "  ${YELLOW}curl -s http://localhost:32323/api/v1/camera/0/cameraxsize | python3 -m json.tool${NC}"
echo ""
