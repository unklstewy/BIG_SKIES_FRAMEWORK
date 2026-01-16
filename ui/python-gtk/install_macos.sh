#!/bin/bash
# Installation script for BigSkies GTK on macOS

set -e

echo "Installing BigSkies GTK Application on macOS..."
echo ""

# Check for Homebrew
if ! command -v brew &> /dev/null; then
    echo "❌ Homebrew is required but not installed."
    echo "Install Homebrew from: https://brew.sh"
    exit 1
fi

echo "✓ Homebrew found"

# Install GTK3 and dependencies via Homebrew
echo ""
echo "Installing GTK3 and dependencies via Homebrew..."
brew install gtk+3 pygobject3 adwaita-icon-theme

echo ""
echo "✓ System dependencies installed"

# Create virtual environment
if [ ! -d "venv" ]; then
    echo ""
    echo "Creating Python virtual environment..."
    python3 -m venv venv
    echo "✓ Virtual environment created"
fi

# Activate virtual environment
source venv/bin/activate

# Install Python packages
echo ""
echo "Installing Python packages..."
pip install --upgrade pip
pip install -r requirements.txt

echo ""
echo "✓ Python packages installed"

# Create symlink to system PyGObject (since it can't be pip installed easily)
SITE_PACKAGES=$(python -c "import sysconfig; print(sysconfig.get_paths()['purelib'])")
BREW_GI_PATH=$(brew --prefix)/lib/python3.*/site-packages/gi

if [ -d "$SITE_PACKAGES" ] && [ -d $(echo $BREW_GI_PATH | head -n 1) ]; then
    echo ""
    echo "Linking system PyGObject to virtual environment..."
    ln -sf $BREW_GI_PATH $SITE_PACKAGES/
    ln -sf $(brew --prefix)/lib/python3.*/site-packages/cairo $SITE_PACKAGES/ 2>/dev/null || true
    ln -sf $(brew --prefix)/lib/python3.*/site-packages/*cairo* $SITE_PACKAGES/ 2>/dev/null || true
    echo "✓ PyGObject linked"
fi

# Mark as installed
touch venv/.installed

echo ""
echo "=========================================="
echo "✓ Installation complete!"
echo "=========================================="
echo ""
echo "To run the application:"
echo "  ./run.sh"
echo ""
echo "Or manually:"
echo "  source venv/bin/activate"
echo "  python -m bigskies_gtk.main"
echo ""
