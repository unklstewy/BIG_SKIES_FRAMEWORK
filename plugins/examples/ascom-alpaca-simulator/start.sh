#!/bin/bash
# Startup script for ASCOM Alpaca Simulators Plugin
# Launches both the ASCOM server and the health reporter

set -e

echo "========================================="
echo "ASCOM Alpaca Simulators Plugin"
echo "BIG SKIES Framework"
echo "========================================="
echo ""

# Environment variables
export PLUGIN_ID="${PLUGIN_ID:-f7e8d9c6-b5a4-3210-9876-543210fedcba}"
export PLUGIN_NAME="${PLUGIN_NAME:-ASCOM Alpaca Simulators}"
export MQTT_BROKER="${MQTT_BROKER:-tcp://mqtt-broker:1883}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export ASPNETCORE_URLS="${ASPNETCORE_URLS:-http://+:80}"

echo "Configuration:"
echo "  Plugin ID: $PLUGIN_ID"
echo "  Plugin Name: $PLUGIN_NAME"
echo "  MQTT Broker: $MQTT_BROKER"
echo "  Log Level: $LOG_LEVEL"
echo "  ASCOM URLs: $ASPNETCORE_URLS"
echo ""

# Function to handle shutdown
shutdown() {
    echo ""
    echo "Shutting down ASCOM Alpaca Simulators Plugin..."
    
    # Kill config service
    if [ ! -z "$CONFIG_PID" ]; then
        echo "Stopping config service (PID: $CONFIG_PID)..."
        kill -TERM "$CONFIG_PID" 2>/dev/null || true
        wait "$CONFIG_PID" 2>/dev/null || true
    fi
    
    # Kill health reporter
    if [ ! -z "$HEALTH_PID" ]; then
        echo "Stopping health reporter (PID: $HEALTH_PID)..."
        kill -TERM "$HEALTH_PID" 2>/dev/null || true
        wait "$HEALTH_PID" 2>/dev/null || true
    fi
    
    # Kill ASCOM server
    if [ ! -z "$ASCOM_PID" ]; then
        echo "Stopping ASCOM server (PID: $ASCOM_PID)..."
        kill -TERM "$ASCOM_PID" 2>/dev/null || true
        wait "$ASCOM_PID" 2>/dev/null || true
    fi
    
    echo "Shutdown complete"
    exit 0
}

# Trap signals for graceful shutdown
trap shutdown SIGTERM SIGINT

echo "Starting ASCOM Alpaca Simulators..."
cd /app/ascom

# Start ASCOM server in background
dotnet ascom.alpaca.simulators.dll \
    --urls "$ASPNETCORE_URLS" \
    2>&1 | sed 's/^/[ASCOM] /' &

ASCOM_PID=$!
echo "ASCOM server started (PID: $ASCOM_PID)"
echo ""

# Give ASCOM a moment to start
sleep 2

echo "Starting health reporter..."
# Start health reporter in background
/usr/local/bin/health-reporter 2>&1 | sed 's/^/[Health] /' &
HEALTH_PID=$!
echo "Health reporter started (PID: $HEALTH_PID)"
echo ""

echo "Starting config service..."
# Start config service in background
/usr/local/bin/config-service 2>&1 | sed 's/^/[Config] /' &
CONFIG_PID=$!
echo "Config service started (PID: $CONFIG_PID)"
echo ""

echo "========================================="
echo "Plugin is now running"
echo "  ASCOM Web UI: http://localhost:32323"
echo "  ASCOM API: http://localhost:32323/api/v1"
echo "  Swagger Docs: http://localhost:32323/swagger"
echo "========================================="
echo ""

# Wait for any process to exit
wait -n $ASCOM_PID $HEALTH_PID $CONFIG_PID

# If we get here, one of the processes died
EXIT_CODE=$?

if ps -p $ASCOM_PID > /dev/null 2>&1; then
    if ps -p $HEALTH_PID > /dev/null 2>&1; then
        echo "Config service died unexpectedly (exit code: $EXIT_CODE)"
    else
        echo "Health reporter died unexpectedly (exit code: $EXIT_CODE)"
    fi
else
    echo "ASCOM server died unexpectedly (exit code: $EXIT_CODE)"
fi

# Trigger shutdown
shutdown
