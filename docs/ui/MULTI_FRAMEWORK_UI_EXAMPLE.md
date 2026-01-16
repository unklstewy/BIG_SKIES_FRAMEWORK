# Multi-Framework UI Element Mapping Examples

## Overview
This document demonstrates how to define UI elements with mappings for multiple frameworks (GTK, Flutter, MFC, Qt, etc.) in the BigSkies framework.

## Single UI Element, Multiple Framework Mappings

### Telescope Control Panel Example

```json
{
  "id": "telescope-control-panel",
  "plugin_guid": "ascom-alpaca-telescope-00000000-0000-0000-0000-000000000001",
  "type": "panel",
  "title": "Telescope Control",
  "api_endpoint": "/api/v1/telescope/0/control",
  "order": 10,
  "enabled": true,
  "metadata": {
    "description": "Main telescope control interface",
    "icon": "telescope",
    "device_type": "telescope",
    "device_instance": 0
  },
  "framework_mappings": {
    "gtk": {
      "widget_type": "Gtk.Frame",
      "layout": "grid",
      "properties": {
        "label": "Telescope",
        "margin": 10,
        "border_width": 5
      },
      "children": [
        {
          "id": "connection-status-indicator",
          "widget_type": "Gtk.DrawingArea",
          "properties": {
            "width_request": 30,
            "height_request": 30
          },
          "data_binding": {
            "property": "draw",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "update_interval": 0
          }
        },
        {
          "id": "connect-button",
          "widget_type": "Gtk.Button",
          "properties": {
            "label": "Connect"
          },
          "data_binding": {
            "property": "label",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? 'Disconnect' : 'Connect'"
          },
          "actions": {
            "clicked": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {
                "action": "toggle"
              }
            }
          }
        },
        {
          "id": "setup-button",
          "widget_type": "Gtk.Button",
          "properties": {
            "label": "Setup"
          },
          "actions": {
            "clicked": {
              "mqtt_topic": "bigskies/ui/navigate",
              "payload": {
                "target": "telescope-setup-panel"
              }
            }
          }
        }
      ]
    },
    "flutter": {
      "widget_type": "Card",
      "layout": "column",
      "properties": {
        "elevation": 4,
        "margin": "EdgeInsets.all(10)"
      },
      "children": [
        {
          "id": "connection-status-indicator",
          "widget_type": "CustomPaint",
          "properties": {
            "size": "Size(30, 30)",
            "painter": "ConnectionStatusPainter"
          },
          "data_binding": {
            "property": "painter.connected",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected"
          }
        },
        {
          "id": "connect-button",
          "widget_type": "ElevatedButton",
          "properties": {
            "child": "Text('Connect')"
          },
          "data_binding": {
            "property": "child.data",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? 'Disconnect' : 'Connect'"
          },
          "actions": {
            "onPressed": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {
                "action": "toggle"
              }
            }
          }
        },
        {
          "id": "setup-button",
          "widget_type": "TextButton",
          "properties": {
            "child": "Text('Setup')"
          },
          "actions": {
            "onPressed": {
              "mqtt_topic": "bigskies/ui/navigate",
              "payload": {
                "target": "telescope-setup-panel"
              }
            }
          }
        }
      ]
    },
    "mfc": {
      "widget_type": "CDialog",
      "layout": "vertical",
      "properties": {
        "title": "Telescope Control",
        "style": "WS_CHILD | WS_VISIBLE",
        "class": "CTelescopeControlDlg"
      },
      "children": [
        {
          "id": "connection-status-indicator",
          "widget_type": "CStatic",
          "properties": {
            "style": "SS_OWNERDRAW",
            "id": "IDC_CONNECTION_STATUS"
          },
          "data_binding": {
            "property": "custom_draw",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected"
          }
        },
        {
          "id": "connect-button",
          "widget_type": "CButton",
          "properties": {
            "label": "Connect",
            "id": "IDC_CONNECT_BUTTON"
          },
          "data_binding": {
            "property": "text",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? _T(\"Disconnect\") : _T(\"Connect\")"
          },
          "actions": {
            "BN_CLICKED": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {
                "action": "toggle"
              }
            }
          }
        },
        {
          "id": "setup-button",
          "widget_type": "CButton",
          "properties": {
            "label": "Setup",
            "id": "IDC_SETUP_BUTTON"
          },
          "actions": {
            "BN_CLICKED": {
              "mqtt_topic": "bigskies/ui/navigate",
              "payload": {
                "target": "telescope-setup-panel"
              }
            }
          }
        }
      ]
    },
    "qt": {
      "widget_type": "QGroupBox",
      "layout": "QGridLayout",
      "properties": {
        "title": "Telescope",
        "minimumWidth": 300
      },
      "children": [
        {
          "id": "connection-status-indicator",
          "widget_type": "QWidget",
          "properties": {
            "minimumSize": "QSize(30, 30)",
            "maximumSize": "QSize(30, 30)",
            "paintEvent": "paintConnectionStatus"
          },
          "data_binding": {
            "property": "connected",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected"
          }
        },
        {
          "id": "connect-button",
          "widget_type": "QPushButton",
          "properties": {
            "text": "Connect"
          },
          "data_binding": {
            "property": "text",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? \"Disconnect\" : \"Connect\""
          },
          "actions": {
            "clicked": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {
                "action": "toggle"
              }
            }
          }
        },
        {
          "id": "setup-button",
          "widget_type": "QPushButton",
          "properties": {
            "text": "Setup"
          },
          "actions": {
            "clicked": {
              "mqtt_topic": "bigskies/ui/navigate",
              "payload": {
                "target": "telescope-setup-panel"
              }
            }
          }
        }
      ]
    },
    "wpf": {
      "widget_type": "GroupBox",
      "layout": "StackPanel",
      "properties": {
        "Header": "Telescope",
        "Margin": "10",
        "Padding": "5"
      },
      "children": [
        {
          "id": "connection-status-indicator",
          "widget_type": "Ellipse",
          "properties": {
            "Width": "30",
            "Height": "30",
            "Stroke": "Black",
            "StrokeThickness": "2"
          },
          "data_binding": {
            "property": "Fill",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? new SolidColorBrush(Colors.Red) : new SolidColorBrush(Colors.Gray)"
          }
        },
        {
          "id": "connect-button",
          "widget_type": "Button",
          "properties": {
            "Content": "Connect"
          },
          "data_binding": {
            "property": "Content",
            "source": "device.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? \"Disconnect\" : \"Connect\""
          },
          "actions": {
            "Click": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {
                "action": "toggle"
              }
            }
          }
        },
        {
          "id": "setup-button",
          "widget_type": "Button",
          "properties": {
            "Content": "Setup"
          },
          "actions": {
            "Click": {
              "mqtt_topic": "bigskies/ui/navigate",
              "payload": {
                "target": "telescope-setup-panel"
              }
            }
          }
        }
      ]
    }
  }
}
```

## Usage Patterns

### Frontend Framework Queries UI Coordinator

Each frontend framework can query the UI coordinator for elements with their specific mappings:

#### Python GTK Example
```python
import json
import paho.mqtt.client as mqtt

# Connect to BigSkies MQTT broker
client = mqtt.Client()
client.connect("localhost", 1883, 60)

# Request UI elements with GTK mappings
request = {
    "action": "list_elements",
    "framework": "gtk",
    "type": "panel"
}

client.publish(
    "bigskies/uielement-coordinator/command/query",
    json.dumps(request)
)

# Subscribe to response
def on_message(client, userdata, msg):
    elements = json.loads(msg.payload)
    for element in elements:
        gtk_mapping = element["framework_mappings"]["gtk"]
        # Generate GTK widgets from mapping
        generate_gtk_widget(gtk_mapping)

client.subscribe("bigskies/uielement-coordinator/response/query/+")
client.on_message = on_message
```

#### Flutter Example
```dart
import 'package:mqtt_client/mqtt_client.dart';

final client = MqttClient('localhost', '');
await client.connect();

// Request UI elements with Flutter mappings
final request = {
  'action': 'list_elements',
  'framework': 'flutter',
  'type': 'panel'
};

client.publishMessage(
  'bigskies/uielement-coordinator/command/query',
  MqttQos.atLeastOnce,
  MqttClientPayloadBuilder()
    .addString(jsonEncode(request))
    .payload!
);

// Subscribe and build widgets
client.subscribe('bigskies/uielement-coordinator/response/query/+', MqttQos.atLeastOnce);
client.updates!.listen((List<MqttReceivedMessage<MqttMessage>> messages) {
  final elements = jsonDecode(messages.first.payload.message);
  for (var element in elements) {
    final flutterMapping = element['framework_mappings']['flutter'];
    // Build Flutter widgets from mapping
    buildFlutterWidget(flutterMapping);
  }
});
```

#### MFC Example (C++)
```cpp
#include "mqtt/async_client.h"
#include <nlohmann/json.hpp>

mqtt::async_client client("localhost:1883", "bigskies_mfc_client");
client.connect()->wait();

// Request UI elements with MFC mappings
nlohmann::json request = {
    {"action", "list_elements"},
    {"framework", "mfc"},
    {"type", "panel"}
};

client.publish(
    "bigskies/uielement-coordinator/command/query",
    request.dump()
)->wait();

// Subscribe and create MFC controls
client.subscribe("bigskies/uielement-coordinator/response/query/+", 1)->wait();

client.set_message_callback([](mqtt::const_message_ptr msg) {
    auto elements = nlohmann::json::parse(msg->to_string());
    for (auto& element : elements) {
        auto mfcMapping = element["framework_mappings"]["mfc"];
        // Create MFC controls from mapping
        CreateMFCControl(mfcMapping);
    }
});
```

## Framework-Specific Widget Type Reference

### GTK Widget Types
- `Gtk.Frame` - Container with border and label
- `Gtk.Box` - Linear layout container
- `Gtk.Grid` - Grid layout container
- `Gtk.Button` - Push button
- `Gtk.CheckButton` - Checkbox
- `Gtk.SpinButton` - Numeric input
- `Gtk.ComboBoxText` - Dropdown selection
- `Gtk.Label` - Text display
- `Gtk.Entry` - Text input
- `Gtk.DrawingArea` - Custom drawing
- `Gtk.ListBox` - List container

### Flutter Widget Types
- `Container` - Basic container
- `Card` - Material design card
- `Column` - Vertical layout
- `Row` - Horizontal layout
- `ElevatedButton` - Material button
- `TextButton` - Flat button
- `Checkbox` - Checkbox
- `TextField` - Text input
- `DropdownButton` - Dropdown
- `Text` - Text display
- `CustomPaint` - Custom drawing
- `ListView` - Scrollable list

### MFC Widget Types
- `CDialog` - Dialog window
- `CStatic` - Static text/graphics
- `CButton` - Button control
- `CEdit` - Text edit
- `CComboBox` - Combo box
- `CListBox` - List box
- `CSpinButtonCtrl` - Spin control
- `CCheckBox` - Checkbox

### Qt Widget Types
- `QWidget` - Base widget
- `QGroupBox` - Group box
- `QVBoxLayout` - Vertical layout
- `QHBoxLayout` - Horizontal layout
- `QGridLayout` - Grid layout
- `QPushButton` - Push button
- `QCheckBox` - Checkbox
- `QSpinBox` - Spin box
- `QComboBox` - Combo box
- `QLabel` - Label
- `QLineEdit` - Line edit

### WPF Widget Types
- `GroupBox` - Group box
- `StackPanel` - Stack layout
- `Grid` - Grid layout
- `Button` - Button
- `CheckBox` - Checkbox
- `TextBox` - Text input
- `ComboBox` - Combo box
- `Label` - Label
- `Ellipse` - Ellipse shape
- `Rectangle` - Rectangle shape

## Best Practices

1. **Maintain Semantic Consistency**: Each framework mapping should represent the same logical UI, just using different widgets
2. **Share Common Metadata**: Use the top-level `metadata` field for framework-agnostic information
3. **Use Appropriate Widgets**: Choose widgets that best match the target framework's design patterns
4. **Consistent IDs**: Use the same child widget IDs across all framework mappings for easier cross-reference
5. **MQTT Topics**: Keep MQTT topics identical across frameworks for consistent backend communication
6. **Data Binding**: Use the same data sources and transforms across frameworks when possible
7. **Progressive Enhancement**: Not all frameworks need all features - it's OK to have simpler mappings for some frameworks

## Adding New Framework Mappings

When adding support for a new UI framework:

1. Add the framework constant to `UIFramework` type in `uielement_coordinator.go`
2. Document framework-specific widget types
3. Create mapping examples
4. Implement framework-specific UI generator in the appropriate language
5. Update this document with examples

## API Endpoints

### Query Elements by Framework
```
Topic: bigskies/uielement-coordinator/command/query
Payload: {
  "action": "list_elements",
  "framework": "gtk|flutter|mfc|qt|wpf",
  "type": "panel|widget|menu|tool|dialog" (optional)
}

Response Topic: bigskies/uielement-coordinator/response/query/{request_id}
```

### Get Supported Frameworks
```
Topic: bigskies/uielement-coordinator/command/frameworks
Payload: {
  "action": "get_supported"
}

Response Topic: bigskies/uielement-coordinator/response/frameworks/{request_id}
Response: {
  "frameworks": ["gtk", "flutter", "mfc"]
}
```

### Add Framework Mapping
```
Topic: bigskies/uielement-coordinator/command/mapping/add
Payload: {
  "element_id": "telescope-control-panel",
  "framework": "gtk",
  "mapping": { ... }
}
```

## References
- UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
- Blazor to GTK Mapping: `docs/ui/BLAZOR_TO_GTK_MAPPING.md`
- MQTT Topic Structure: `pkg/mqtt/topics.go`
