# BigSkies Framework UI Element Mapping System

## Overview
The BigSkies Framework provides a comprehensive multi-framework UI element mapping system that allows a single UI element definition to be rendered across multiple frontend frameworks (GTK, Flutter, MFC, Qt, WPF, Unity, Blazor) while maintaining consistent backend communication through MQTT.

## Architecture

### UI Element Coordinator
The **ui-element-coordinator** is a backend service that:
- Maintains a registry of UI elements provided by plugins
- Stores framework-specific widget mappings for each UI element
- Provides MQTT-based API for frontends to query UI definitions
- Supports dynamic UI provisioning as plugins are added/removed
- Enables multiple frontend frameworks to coexist

### Framework Separation
Each UI element can have separate mappings for different frameworks stored in the `framework_mappings` field. This allows:
- **Single source of truth**: One UI element definition with multiple framework implementations
- **Framework independence**: Add support for new frameworks without affecting existing ones
- **Optimal native experience**: Each framework uses its native widget types
- **Consistent behavior**: All frameworks communicate via the same MQTT topics

## Supported Frameworks

| Framework | Language | Status | Documentation |
|-----------|----------|--------|---------------|
| **GTK** | Python | Active Development | [Blazor to GTK Mapping](BLAZOR_TO_GTK_MAPPING.md) |
| **Flutter** | Dart | Documented | [Blazor to Flutter Mapping](BLAZOR_TO_FLUTTER_MAPPING.md) |
| **Unity** | C# | Planned | Multi-framework examples |
| **MFC** | C++ | Documented | [Blazor to MFC Mapping](BLAZOR_TO_MFC_MAPPING.md) |
| **Qt** | C++/Python | Documented | [Blazor to Qt Mapping](BLAZOR_TO_QT_MAPPING.md) |
| **WPF** | C#/.NET | Documented | [Blazor to WPF Mapping](BLAZOR_TO_WPF_MAPPING.md) |
| **Blazor** | C#/.NET | Reference (ASCOM Alpaca) | Multi-framework examples |

## Key Documents

### 1. Framework-Specific Mapping Guides

#### [BLAZOR_TO_GTK_MAPPING.md](BLAZOR_TO_GTK_MAPPING.md)
Comprehensive mapping guide from ASCOM Alpaca Simulators Blazor UI to Python GTK:
- Widget type mappings (Blazor → GTK)
- Layout container equivalents
- Data binding strategies with MQTT
- Complete code examples for telescope control and setup
- Best practices and device-specific mappings

#### [BLAZOR_TO_FLUTTER_MAPPING.md](BLAZOR_TO_FLUTTER_MAPPING.md)
Comprehensive mapping guide from Blazor to Flutter (Dart):
- Widget type mappings (Blazor → Flutter)
- Material Design patterns
- StatefulWidget examples with MQTT integration
- Custom painters for graphics
- Provider/Riverpod state management patterns

#### [BLAZOR_TO_MFC_MAPPING.md](BLAZOR_TO_MFC_MAPPING.md)
Comprehensive mapping guide from Blazor to MFC (C++):
- Control type mappings (Blazor → MFC)
- Dialog-based UI with DDX/DDV
- Resource file (.rc) definitions
- Owner-draw controls for custom graphics
- MQTT integration with C++ client

#### [BLAZOR_TO_QT_MAPPING.md](BLAZOR_TO_QT_MAPPING.md)
Comprehensive mapping guide from Blazor to Qt (C++/Python):
- Widget type mappings (Blazor → Qt)
- Signals/slots mechanism
- Both C++ (Qt Widgets) and Python (PyQt5) examples
- Custom widget painting with QPainter
- MQTT integration patterns

#### [BLAZOR_TO_WPF_MAPPING.md](BLAZOR_TO_WPF_MAPPING.md)
Comprehensive mapping guide from Blazor to WPF (C#/.NET):
- Control type mappings (Blazor → WPF)
- MVVM pattern with INotifyPropertyChanged
- XAML markup and data binding
- Command pattern for user actions
- MQTTnet integration

### 2. [MULTI_FRAMEWORK_UI_EXAMPLE.md](MULTI_FRAMEWORK_UI_EXAMPLE.md)
Complete example showing a single UI element (telescope control panel) with mappings for all supported frameworks:
- GTK (Python)
- Flutter (Dart)
- MFC (C++)
- Qt (C++/Python)
- WPF (C#)

Includes query patterns and framework-specific widget type references.

## Quick Start

### Backend: Register UI Element
```go
// Create UI element with multiple framework mappings
element := &UIElement{
    ID:         "telescope-control",
    PluginGUID: "ascom-telescope-plugin-guid",
    Type:       UIElementTypePanel,
    Title:      "Telescope Control",
    FrameworkMappings: map[UIFramework]*WidgetMapping{
        UIFrameworkGTK: {
            WidgetType: "Gtk.Frame",
            Properties: map[string]interface{}{
                "label": "Telescope",
            },
            // ... children definitions
        },
        UIFrameworkFlutter: {
            WidgetType: "Card",
            Properties: map[string]interface{}{
                "elevation": 4,
            },
            // ... children definitions
        },
    },
}

coordinator.RegisterUIElement(element)
```

### Frontend: Query UI Elements (Python GTK Example)
```python
import paho.mqtt.client as mqtt
import json

client = mqtt.Client()
client.connect("localhost", 1883)

# Request GTK-specific UI elements
request = {
    "action": "list_elements",
    "framework": "gtk",
    "type": "panel"
}

client.publish(
    "bigskies/uielement-coordinator/command/query",
    json.dumps(request)
)

# Handle response and generate UI
def on_message(client, userdata, msg):
    elements = json.loads(msg.payload)
    for element in elements:
        gtk_mapping = element["framework_mappings"]["gtk"]
        build_gtk_widget(gtk_mapping)

client.subscribe("bigskies/uielement-coordinator/response/query/+")
client.on_message = on_message
```

## Implementation Details

### Backend Components
- **Location**: `internal/coordinators/uielement_coordinator.go`
- **Key Types**:
  - `UIElement` - Main element definition with framework mappings
  - `UIFramework` - Enum of supported frameworks (gtk, flutter, mfc, qt, wpf, unity, blazor)
  - `WidgetMapping` - Framework-specific widget hierarchy
  - `WidgetDefinition` - Individual widget definition
  - `DataBinding` - MQTT-based data binding configuration
  - `ActionDefinition` - User interaction → MQTT command mapping

- **Key Methods**:
  - `ListUIElementsByFramework()` - Get elements for specific framework
  - `GetFrameworkMapping()` - Retrieve framework-specific mapping
  - `AddFrameworkMapping()` - Add/update framework mapping
  - `RemoveFrameworkMapping()` - Remove framework mapping
  - `GetSupportedFrameworks()` - List all supported frameworks

### MQTT Topics
- **Query Elements**: `bigskies/uielement-coordinator/command/query`
- **Query Response**: `bigskies/uielement-coordinator/response/query/{request_id}`
- **Register Element**: `bigskies/uielement-coordinator/event/register`
- **Unregister Element**: `bigskies/uielement-coordinator/event/unregister`
- **Add Mapping**: `bigskies/uielement-coordinator/command/mapping/add`
- **Get Frameworks**: `bigskies/uielement-coordinator/command/frameworks`

## Data Flow

```
┌─────────────┐
│   Plugin    │ Registers UI elements with framework mappings
└──────┬──────┘
       │ MQTT
       ↓
┌──────────────────────┐
│ UI Element           │ Stores elements with framework-specific
│ Coordinator          │ widget definitions
└──────┬───────────────┘
       │ MQTT
       ↓
┌──────────────────────┐
│  Frontend (GTK/      │ Queries coordinator for framework-specific
│  Flutter/MFC/etc.)   │ mappings and generates native UI
└──────────────────────┘
       │
       ↓
┌──────────────────────┐
│  User Interaction    │ UI events → MQTT commands
└──────┬───────────────┘
       │ MQTT
       ↓
┌──────────────────────┐
│  Backend Services    │ Process commands, update state
│  (Telescope, etc.)   │
└──────┬───────────────┘
       │ MQTT
       ↓
┌──────────────────────┐
│  Frontend Updates    │ State changes → UI updates via data binding
└──────────────────────┘
```

## Design Principles

1. **Framework Agnostic Core**: UI element semantics are framework-independent
2. **Native Experience**: Each framework uses its native widgets and patterns
3. **Separation of Concerns**: UI mappings are separate from business logic
4. **MQTT-Centric**: All communication through message bus
5. **Plugin Extensibility**: Plugins can provide UI elements dynamically
6. **Multi-Frontend Support**: Multiple frontend frameworks can run simultaneously

## Use Cases

### ASCOM Alpaca Integration
The primary reference implementation comes from the ASCOM Alpaca Simulators:
- **Source**: Blazor-based telescope control interfaces
- **Target**: Python GTK for BigSkies framework
- **Devices**: Telescope, Camera, Dome, Focuser, FilterWheel, Switch, etc.
- **Features**: Device control, configuration, status monitoring

### Future Use Cases
- **Terrestrial Astronomy**: Aircraft tracking (ADS-B), wildlife monitoring
- **Deep Sky Surveys**: Distributed workload coordination (DeepSky@Home)
- **VR Integration**: Spatial visualization through Unity
- **Mobile Control**: Flutter-based mobile apps
- **Desktop Admin**: Qt/WPF/MFC desktop applications

## Contributing

### Adding a New Framework
1. Add framework constant to `UIFramework` in `uielement_coordinator.go`
2. Document framework-specific widget types
3. Create example mappings in `MULTI_FRAMEWORK_UI_EXAMPLE.md`
4. Implement frontend UI generator in appropriate language
5. Update architecture diagram

### Adding New UI Elements
1. Define element with `id`, `type`, `title`, and `api_endpoint`
2. Create framework mappings for desired frameworks
3. Define data bindings to MQTT state topics
4. Define actions that publish to MQTT command topics
5. Register element with coordinator via MQTT or API

## References

- **Architecture Diagram**: `docs/architecture/big_skies_architecture_gojs.json`
- **Coordinator Implementation**: `internal/coordinators/uielement_coordinator.go`
- **GTK Mapping Guide**: `docs/ui/BLAZOR_TO_GTK_MAPPING.md`
- **Multi-Framework Examples**: `docs/ui/MULTI_FRAMEWORK_UI_EXAMPLE.md`
- **ASCOM Alpaca Source**: `external/ASCOM.Alpaca.Simulators`
- **Project Overview**: `WARP.md`, `README.md`

## Status

- ✅ **Completed**: 
  - UI element coordinator Go implementation with multi-framework support
  - Comprehensive Blazor mapping documentation for all frameworks:
    - ✅ Python GTK
    - ✅ Flutter (Dart)
    - ✅ MFC (C++)
    - ✅ Qt (C++/Python)
    - ✅ WPF (C#/.NET)
  - Multi-framework example definitions (GTK, Flutter, MFC, Qt, WPF)
  - Architecture diagram updates
  - Code examples for all supported frameworks

- 🚧 **In Progress**:
  - Python GTK UI generator implementation
  - MQTT query/response handlers in coordinator
  - Framework-specific UI generators

- 📋 **Planned**:
  - Unity UI mapping and generator
  - Complete data binding implementation
  - Plugin UI discovery and auto-registration
  - UI hot reload support
  - Testing and validation of all framework mappings

## License & Attribution

BigSkies Framework is developed with contributions from Warp AI. 
When committing changes, include: `Co-Authored-By: Warp <agent@warp.dev>`
