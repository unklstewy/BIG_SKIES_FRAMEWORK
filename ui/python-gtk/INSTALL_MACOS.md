# BigSkies GTK - macOS Installation Guide

## The Issue with PyGObject on macOS

PyGObject (the Python bindings for GTK) cannot be easily installed via pip on macOS. It requires system libraries that must be installed via Homebrew.

## Quick Install (Recommended)

### Step 1: Install Homebrew Dependencies

```bash
brew install gtk+3 pygobject3 adwaita-icon-theme
```

### Step 2: Install Python Packages

```bash
pip3 install --user paho-mqtt python-dotenv
```

### Step 3: Run the Application

```bash
python3 -m bigskies_gtk.main
```

## Automated Installation

We've provided an automated installation script:

```bash
./install_macos.sh
```

This script will:
1. Check for Homebrew
2. Install GTK3 and PyGObject via Homebrew
3. Create a virtual environment
4. Install Python dependencies
5. Link system PyGObject to the virtual environment

## Running the Application

### Option 1: Simple (Recommended for macOS)
```bash
./run_simple.sh
```

This uses your system Python with Homebrew's PyGObject.

### Option 2: With Virtual Environment
```bash
./run.sh
```

This creates/uses a virtual environment (requires the install script).

### Option 3: Manual
```bash
python3 -m bigskies_gtk.main
```

## Troubleshooting

### "No module named 'gi'"

This means PyGObject is not installed. Run:
```bash
brew install gtk+3 pygobject3
```

### "No module named 'paho'"

Install paho-mqtt:
```bash
pip3 install --user paho-mqtt
```

### "ImportError: cannot import name '_gi'"

This usually means there's a mismatch between Python versions. Ensure you're using the same Python that Homebrew installed PyGObject for:

```bash
brew info pygobject3
```

Look for the Python version it was built for, then use that Python explicitly:
```bash
python3.11 -m bigskies_gtk.main  # Or whatever version
```

### GTK Warnings about Themes

If you see warnings about missing themes, install:
```bash
brew install adwaita-icon-theme
```

## Why Not Use pip for PyGObject?

PyGObject requires:
- GTK3 system libraries
- GObject introspection
- Cairo graphics library
- Various GI typelibs

These are system-level dependencies that pip cannot install. Homebrew handles all of this correctly for macOS.

## Alternative: Use System Python

The simplest approach on macOS is to:
1. Install GTK via Homebrew (includes PyGObject)
2. Install only paho-mqtt via pip
3. Run with system Python

This avoids virtual environment complexity and works reliably.
