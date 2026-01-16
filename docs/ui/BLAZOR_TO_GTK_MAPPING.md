# Blazor to Python GTK UI Element Mapping

## Overview
This document provides a comprehensive mapping of UI elements from the ASCOM Alpaca Simulators Blazor interface to Python GTK components, designed for integration with the BigSkies framework's UI element coordinator.

## Architecture Integration

The UI element coordinator in BigSkies framework will:
1. Track UI element definitions from plugins via MQTT
2. Provide UI element metadata including widget type mappings
3. Enable Python GTK frontend to dynamically generate UI from backend API definitions
4. Support multiple frontend frameworks (Flutter, Unity, Python GTK) through abstraction

## Core UI Element Mappings

### Layout Containers

| Blazor Element | GTK Widget | BigSkies Element Type | Notes |
|----------------|------------|----------------------|-------|
| `<fieldset>` | `Gtk.Frame` | `panel` | Use `set_label()` for legend |
| `<legend>` | Frame label | N/A | Set via parent Frame's `set_label()` |
| `<div class="grid-container-two">` | `Gtk.Grid` | `panel` | 2-column grid layout |
| `<div class="grid-item-left">` | Grid cell | N/A | `attach(widget, 0, row, 1, 1)` |
| `<div class="grid-item-right">` | Grid cell | N/A | `attach(widget, 1, row, 1, 1)` |
| `<div class="centered">` | `Gtk.Box(Gtk.Orientation.HORIZONTAL)` | `panel` | Center-aligned horizontal box |
| `<body>` | `Gtk.Box(Gtk.Orientation.VERTICAL)` | `panel` | Main container |

### Input Controls

| Blazor Element | GTK Widget | BigSkies Element Type | Notes |
|----------------|------------|----------------------|-------|
| `<button>` | `Gtk.Button` | `widget` | Connect to `clicked` signal |
| `<input type="checkbox">` | `Gtk.CheckButton` | `widget` | Bind via `toggled` signal |
| `<input type="number">` | `Gtk.SpinButton` | `widget` | Set min/max/step via `Gtk.Adjustment` |
| `<input type="text">` | `Gtk.Entry` | `widget` | Single-line text input |
| `<select>` | `Gtk.ComboBoxText` | `widget` | Dropdown selection |
| `<option>` | ComboBoxText item | N/A | Add via `append_text()` |

### Display Controls

| Blazor Element | GTK Widget | BigSkies Element Type | Notes |
|----------------|------------|----------------------|-------|
| `<label>` | `Gtk.Label` | `widget` | Static text display |
| `<p>` | `Gtk.Label` | `widget` | Paragraph text |
| `<h2>`, `<h3>` | `Gtk.Label` | `widget` | Use Pango markup for sizing |
| `<svg>` (status circle) | `Gtk.DrawingArea` | `widget` | Custom Cairo drawing |
| Dynamic text binding | `Gtk.Label.set_text()` | N/A | Update via property binding |

### Navigation

| Blazor Element | GTK Widget | BigSkies Element Type | Notes |
|----------------|------------|----------------------|-------|
| `<NavLink>` | `Gtk.Button` or `Gtk.ModelButton` | `menu` | Navigation menu item |
| `<ul class="nav flex-column">` | `Gtk.ListBox` | `menu` | Vertical navigation list |
| `<li class="nav-item">` | `Gtk.ListBoxRow` | `menu` | Individual nav item |
| Navbar | `Gtk.HeaderBar` or custom | `panel` | Application header |

## ASCOM Telescope Control UI Mapping

### Connection Control Section
```
Blazor Structure:
<fieldset>
  <legend>Telescope</legend>
  <div class="grid-container-two">
    <svg circle> + <button>Connect/Disconnect</button>
    <button>Setup</button>
  </div>
</fieldset>

GTK Equivalent:
frame = Gtk.Frame(label="Telescope")
grid = Gtk.Grid()
status_indicator = Gtk.DrawingArea()  # For connection status circle
connect_button = Gtk.Button(label="Connect")
setup_button = Gtk.Button(label="Setup")
grid.attach(status_indicator, 0, 0, 1, 1)
grid.attach(connect_button, 1, 0, 1, 1)
grid.attach(setup_button, 2, 0, 1, 1)
frame.add(grid)
```

### Status Display Section
```
Blazor Structure:
<div class="grid-container-two">
  <label>LST</label><p>@LSTText</p>
  <label>RA</label><p>@RAText</p>
  <label>Dec</label><p>@DecText</p>
  <label>Az</label><p>@AzText</p>
  <label>Alt</label><p>@AltText</p>
</div>

GTK Equivalent:
grid = Gtk.Grid()
grid.set_column_spacing(10)
grid.set_row_spacing(5)

# Add labels and values
labels = ["LST", "RA", "Dec", "Az", "Alt"]
for i, label_text in enumerate(labels):
    label = Gtk.Label(label=f"{label_text}:")
    label.set_halign(Gtk.Align.END)
    value = Gtk.Label(label="00:00:00")
    value.set_halign(Gtk.Align.START)
    grid.attach(label, 0, i, 1, 1)
    grid.attach(value, 1, i, 1, 1)
```

## ASCOM Telescope Setup UI Mapping

### Configuration Sections
```
Blazor Structure:
<fieldset disabled="@Device.Connected">
  <legend>Telescope Settings</legend>
  <input type="checkbox" id="AutoUnpark" @bind="AutoUnpark">
  <label for="AutoUnpark">Auto Unpark / Track on Start</label>
  <input type="number" id="SlewRate" min="0" max="360" step="1" @bind="SlewRate">
</fieldset>

GTK Equivalent:
frame = Gtk.Frame(label="Telescope Settings")
box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=6)

# Checkbox
auto_unpark = Gtk.CheckButton(label="Auto Unpark / Track on Start")
box.pack_start(auto_unpark, False, False, 0)

# Number input with label
slew_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
slew_label = Gtk.Label(label="Slew Rate (deg/sec):")
slew_adj = Gtk.Adjustment(value=10, lower=0, upper=360, step_increment=1)
slew_spin = Gtk.SpinButton(adjustment=slew_adj)
slew_box.pack_start(slew_label, False, False, 0)
slew_box.pack_start(slew_spin, False, False, 0)
box.pack_start(slew_box, False, False, 0)

frame.add(box)

# Disable when connected
frame.set_sensitive(not device_connected)
```

### Site Information Section
```
Blazor Structure:
<fieldset>
  <legend>Site Information</legend>
  <label>Latitude</label>
  <select id="LatitudeSign">
    <option value="1">N</option>
    <option value="-1">S</option>
  </select>
  <input type="number" min="0" max="90" @bind="LatitudeDegrees">
  <input type="number" min="0" max="60" @bind="LatitudeMinutes">
</fieldset>

GTK Equivalent:
frame = Gtk.Frame(label="Site Information")
grid = Gtk.Grid()
grid.set_column_spacing(6)
grid.set_row_spacing(6)

# Latitude
lat_label = Gtk.Label(label="Latitude:")
lat_sign = Gtk.ComboBoxText()
lat_sign.append_text("N")
lat_sign.append_text("S")
lat_sign.set_active(0)
lat_deg_adj = Gtk.Adjustment(value=0, lower=0, upper=90, step_increment=1)
lat_deg = Gtk.SpinButton(adjustment=lat_deg_adj)
lat_min_adj = Gtk.Adjustment(value=0, lower=0, upper=60, step_increment=1)
lat_min = Gtk.SpinButton(adjustment=lat_min_adj)

grid.attach(lat_label, 0, 0, 1, 1)
grid.attach(lat_sign, 1, 0, 1, 1)
grid.attach(lat_deg, 2, 0, 1, 1)
grid.attach(lat_min, 3, 0, 1, 1)

frame.add(grid)
```

## Navigation Menu Mapping

### Blazor NavMenu Structure
```
<div class="top-row navbar navbar-dark">
  <a class="navbar-brand" href="">ASCOM.Alpaca.Simulators</a>
  <button class="navbar-toggler" @onclick="ToggleNavMenu">
    <span class="navbar-toggler-icon"></span>
  </button>
</div>

<ul class="nav flex-column">
  @foreach (var key in DeviceManager.Telescopes)
  {
    <NavLink href=@GetSetupURL("Telescope", key.Key)>
      <span class="oi oi-star"></span> @GetDisplayName("Telescope", key.Key)
    </NavLink>
  }
</ul>

GTK Equivalent:
# Header Bar
header = Gtk.HeaderBar()
header.set_title("BigSkies Framework")
header.set_show_close_button(True)

# Navigation Sidebar
sidebar = Gtk.ListBox()
sidebar.set_selection_mode(Gtk.SelectionMode.SINGLE)

# Populate with devices
for device_id, device_name in telescopes.items():
    row = Gtk.ListBoxRow()
    box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=6)
    icon = Gtk.Image.new_from_icon_name("starred", Gtk.IconSize.BUTTON)
    label = Gtk.Label(label=device_name)
    box.pack_start(icon, False, False, 0)
    box.pack_start(label, True, True, 0)
    row.add(box)
    sidebar.add(row)

sidebar.connect("row-activated", on_nav_item_selected)
```

## BigSkies Framework Integration

### UI Element Definition JSON Schema
```json
{
  "id": "telescope-control-panel",
  "plugin_guid": "ascom-alpaca-telescope-plugin-guid",
  "type": "panel",
  "title": "Telescope Control",
  "api_endpoint": "/api/telescope/control",
  "order": 10,
  "enabled": true,
  "metadata": {
    "ui_framework": "gtk",
    "widget_type": "frame",
    "layout": "grid",
    "children": [
      {
        "id": "connection-status",
        "widget_type": "drawing_area",
        "properties": {
          "width": 30,
          "height": 30,
          "draw_function": "draw_status_circle"
        },
        "data_binding": {
          "property": "connected",
          "source": "device.status.connected",
          "update_interval": 100
        }
      },
      {
        "id": "connect-button",
        "widget_type": "button",
        "properties": {
          "label": "Connect"
        },
        "actions": {
          "clicked": {
            "mqtt_topic": "bigskies/telescope/0/command/connect",
            "payload": {"action": "toggle"}
          }
        },
        "data_binding": {
          "property": "label",
          "source": "device.status.connected",
          "transform": "connected ? 'Disconnect' : 'Connect'"
        }
      }
    ]
  }
}
```

### Python GTK UI Generator Usage
```python
from bigskies.ui.gtk_generator import GTKUIGenerator
from bigskies.mqtt.client import MQTTClient

# Connect to BigSkies MQTT broker
mqtt_client = MQTTClient("localhost:1883")
mqtt_client.connect()

# Initialize UI generator
ui_generator = GTKUIGenerator(mqtt_client)

# Subscribe to UI element registrations
ui_generator.subscribe_to_coordinator()

# Generate UI from registered elements
window = ui_generator.generate_main_window()
window.show_all()

# UI will auto-update as plugins register/unregister elements
Gtk.main()
```

## Widget Property Mappings

### Common Properties

| Blazor Property | GTK Property/Method | Notes |
|----------------|---------------------|-------|
| `@bind` | Signal connection | Use appropriate signal (toggled, value-changed, etc.) |
| `disabled` | `set_sensitive(False)` | Disables widget interaction |
| `@onclick` | `connect("clicked", handler)` | Button click handler |
| `style="color:red"` | CSS provider or `override_color()` | GTK CSS or direct styling |
| `id` | Widget name | `set_name("id")` |
| `class` | CSS class | `get_style_context().add_class("class")` |
| `min`, `max`, `step` | `Gtk.Adjustment` | For numeric inputs |

### Data Binding Strategy

Blazor uses `@bind` for two-way data binding. In GTK with BigSkies:

1. **Backend → UI Updates**: MQTT messages on state topics trigger UI updates
2. **UI → Backend Updates**: Widget signals trigger MQTT command messages
3. **Polling Updates**: Timer-based polling for real-time data (LST, RA, Dec, etc.)

Example:
```python
# Subscribe to telescope state updates
mqtt_client.subscribe("bigskies/telescope/0/state/+", on_state_update)

def on_state_update(topic, payload):
    data = json.loads(payload)
    if "ra" in data:
        ra_label.set_text(data["ra"])
    if "dec" in data:
        dec_label.set_text(data["dec"])

# Send commands on button click
def on_connect_clicked(button):
    mqtt_client.publish(
        "bigskies/telescope/0/command/connect",
        json.dumps({"action": "toggle"})
    )

connect_button.connect("clicked", on_connect_clicked)
```

## Style Mappings

### CSS Classes
Blazor CSS classes can be mapped to GTK CSS classes:

```css
/* Blazor CSS */
.grid-container-two {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
}

/* GTK CSS equivalent */
.grid-container-two {
    padding: 10px;
}
```

GTK Implementation:
```python
css_provider = Gtk.CssProvider()
css_provider.load_from_path("bigskies-theme.css")
Gtk.StyleContext.add_provider_for_screen(
    Gdk.Screen.get_default(),
    css_provider,
    Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
)
```

## Device-Specific Mappings

### Camera Control
- Image display: `Gtk.Image` with `GdkPixbuf`
- Exposure controls: `Gtk.Scale` for duration
- Binning: `Gtk.ComboBoxText`

### Dome Control
- Azimuth control: `Gtk.Scale` (circular dial using custom widget)
- Shutter: `Gtk.Switch`

### Focuser Control
- Position: `Gtk.Scale` with `Gtk.SpinButton`
- Absolute/Relative: `Gtk.RadioButton` group

### Switch Control
- Multiple switches: `Gtk.ListBox` with `Gtk.Switch` per row

## Best Practices

1. **Separation of Concerns**: Keep UI generation logic separate from business logic
2. **Responsive Design**: Use `Gtk.Paned` and `Gtk.Stack` for responsive layouts
3. **Accessibility**: Set proper tooltips and accessible names
4. **Theme Support**: Use GTK themes and CSS for consistent styling
5. **MQTT Integration**: All device interactions via MQTT, no direct HTTP calls
6. **Error Handling**: Display errors via `Gtk.MessageDialog` or status bar
7. **Performance**: Use `GLib.idle_add()` for UI updates from background threads
8. **Validation**: Validate input before sending MQTT commands

## Future Enhancements

1. **Hot Reload**: Dynamic UI updates when plugins change
2. **Custom Widgets**: Telescope position display, sky chart integration
3. **Multi-Device**: Tabbed or stacked views for multiple devices
4. **Responsive**: Adaptive layouts for different screen sizes
5. **Themes**: Dark/light mode support
6. **Localization**: i18n support for multiple languages

## References

- GTK 3 Python Documentation: https://python-gtk-3-tutorial.readthedocs.io/
- ASCOM Alpaca API: https://ascom-standards.org/api/
- BigSkies UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
