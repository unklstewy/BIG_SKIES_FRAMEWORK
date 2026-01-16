#!/bin/bash
# Simple run script using system Python (if GTK already installed)

cd "$(dirname "$0")"

# Check if GTK is available
python3 -c "import gi; gi.require_version('Gtk', '3.0'); from gi.repository import Gtk" 2>/dev/null

if [ $? -ne 0 ]; then
    echo "❌ GTK3 not found in system Python"
    echo ""
    echo "Install with:"
    echo "  brew install gtk+3 pygobject3"
    echo ""
    echo "Or run the full installation:"
    echo "  ./install_macos.sh"
    exit 1
fi

# Install minimal Python dependencies
if ! python3 -c "import paho.mqtt.client" 2>/dev/null; then
    echo "Installing paho-mqtt..."
    pip3 install --user paho-mqtt python-dotenv
fi

echo "Starting BigSkies GTK Application..."
python3 -m bigskies_gtk.main
