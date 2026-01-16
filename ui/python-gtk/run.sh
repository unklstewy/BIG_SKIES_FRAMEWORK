#!/bin/bash
# Quick start script for BigSkies GTK Application

cd "$(dirname "$0")"

# Check if dependencies are installed
if [ ! -f "venv/.installed" ]; then
    echo "Dependencies not installed. Running installation script..."
    echo ""
    ./install_macos.sh
fi

# Activate virtual environment
if [ -d "venv" ]; then
    source venv/bin/activate
fi

# Run the application
echo "Starting BigSkies GTK Application..."
python -m bigskies_gtk.main
