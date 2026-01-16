#!/bin/bash
# Generate Dockerfiles for all coordinators from template

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_FILE="$PROJECT_ROOT/deployments/docker/Dockerfile.template"
OUTPUT_DIR="$PROJECT_ROOT/deployments/docker"

# Coordinators to generate (name:binary)
COORDINATORS=(
    "DataStore:datastore-coordinator"
    "Security:security-coordinator"
    "Message:message-coordinator"
    "Application:application-coordinator"
    "Plugin:plugin-coordinator"
    "Telescope:telescope-coordinator"
    "UIElement:uielement-coordinator"
)

echo "Generating Dockerfiles from template..."

for coord in "${COORDINATORS[@]}"; do
    IFS=: read -r name binary <<< "$coord"
    output_file="$OUTPUT_DIR/Dockerfile.${binary}"
    
    echo "  - Generating $output_file"
    
    sed -e "s/{{COORDINATOR_NAME}}/$name/g" \
        -e "s/{{COORDINATOR_BINARY}}/$binary/g" \
        "$TEMPLATE_FILE" > "$output_file"
done

echo "Done! Generated ${#COORDINATORS[@]} Dockerfiles"
