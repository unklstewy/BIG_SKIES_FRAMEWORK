# BigSkies Framework - Python GTK UI

A standalone Python GTK application for the BigSkies Framework, featuring authentication, service health monitoring, telescope control, and visual telescope preview.

## Features

- **Authentication**: Login dialog with MQTT server configuration
- **Service Health Monitoring**: Real-time status of all BigSkies coordinators
- **Telescope Control**: ASCOM Alpaca-compatible telescope control interface
- **Visual Preview**: Real-time telescope orientation display with Alt/Az visualization
- **MQTT Integration**: Full integration with BigSkies MQTT message bus

## Requirements

- Python 3.8+
- GTK 3
- PyGObject
- paho-mqtt
- cairo

## Installation

### macOS (Recommended)

**Option 1: Automated Installation**
```bash
cd ui/python-gtk
./install_macos.sh
```

**Option 2: Manual Installation**
```bash
# Install system dependencies
brew install gtk+3 pygobject3 adwaita-icon-theme

# Install Python packages
pip3 install --user paho-mqtt python-dotenv
```

### Linux

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install python3-gi python3-gi-cairo gir1.2-gtk-3.0
pip3 install --user paho-mqtt python-dotenv
```

**Fedora:**
```bash
sudo dnf install python3-gobject gtk3
pip3 install --user paho-mqtt python-dotenv
```

## Running the Application

**Option 1: Quick Start (macOS)**
```bash
./run.sh
```

**Option 2: Simple Run (if GTK already installed)**
```bash
./run_simple.sh
```

**Option 3: Manual**
```bash
python3 -m bigskies_gtk.main
```

## Usage

### Login
1. Enter your username and password (defaults: admin/password)
2. Configure MQTT server (default: localhost)
3. Click OK to connect

### Navigation
The application features three main views accessible via the sidebar:

1. **Health Status**: Monitor all BigSkies coordinators
   - Green: Healthy
   - Yellow: Warning
   - Red: Unhealthy
   - Gray: Unknown/No data

2. **Telescope Control**: Control telescope operations
   - Connect/Disconnect telescope
   - View real-time position (LST, RA, Dec, Az, Alt)
   - Monitor slewing status
   - Access setup (future)

3. **Telescope Preview**: Visual representation
   - Alt/Az sky chart with cardinal directions
   - Real-time telescope pointer
   - Green pointer when idle, orange when slewing
   - Shows azimuth and altitude coordinates

## Architecture

The application conforms to BigSkies architecture:

- **MQTT Topics Used**:
  - `bigskies/+/health/status` - Coordinator health
  - `bigskies/telescope/0/state/#` - Telescope state
  - `bigskies/telescope/0/command/+` - Telescope commands
  - `bigskies/uielement-coordinator/command/query` - UI element queries

- **Components**:
  - `mqtt/client.py` - MQTT client wrapper with topic matching
  - `auth/login_dialog.py` - Authentication dialog
  - `widgets/health_panel.py` - Service health monitoring
  - `widgets/control_panel.py` - Telescope control interface
  - `widgets/telescope_preview.py` - Visual telescope display
  - `app.py` - Main application with sidebar navigation

## Configuration

Default configuration (can be changed in login dialog):
- **MQTT Broker**: localhost:1883
- **Username**: admin
- **Password**: password

## Development

### Project Structure
```
bigskies_gtk/
├── __init__.py
├── main.py              # Entry point
├── app.py               # Main application
├── auth/
│   ├── __init__.py
│   └── login_dialog.py  # Authentication
├── mqtt/
│   ├── __init__.py
│   └── client.py        # MQTT client
├── widgets/
│   ├── __init__.py
│   ├── health_panel.py  # Health monitoring
│   ├── control_panel.py # Telescope control
│   └── telescope_preview.py  # Visual display
├── coordinators/
│   ├── __init__.py
│   └── ui_element_manager.py  # UI coordinator client (future)
└── utils/
    ├── __init__.py
    └── config.py        # Configuration (future)
```

### Extending the Application

#### Adding New Views
1. Create widget in `widgets/` directory
2. Import in `app.py`
3. Add to content_stack in `_build_main_window()`
4. Add navigation button in `_build_sidebar()`

#### Adding MQTT Subscriptions
```python
self.mqtt_client.subscribe("topic/pattern", callback_function)

def callback_function(self, topic, payload):
    # Handle message
    value = payload.get('key')
    # Update UI
```

## Troubleshooting

### GTK Not Found
Install GTK3 and PyGObject:
- **macOS**: `brew install gtk+3 pygobject3`
- **Ubuntu**: `apt-get install python3-gi python3-gi-cairo gir1.2-gtk-3.0`
- **Fedora**: `dnf install python3-gobject gtk3`

### MQTT Connection Failed
- Ensure MQTT broker is running on specified host/port
- Check firewall settings
- Verify broker accepts anonymous connections or configure auth

### No Telescope Data
- Ensure telescope coordinator is running
- Verify telescope is publishing to correct MQTT topics
- Check MQTT broker connectivity

## Co-Authored-By
Warp <agent@warp.dev>

## License
Part of BigSkies Framework
