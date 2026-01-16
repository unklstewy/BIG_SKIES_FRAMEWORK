# Blazor to Flutter Widget Mapping

## Overview
This document provides a comprehensive mapping of UI elements from the ASCOM Alpaca Simulators Blazor interface to Flutter widgets, designed for integration with the BigSkies framework's UI element coordinator.

## Architecture Integration

The UI element coordinator in BigSkies framework will:
1. Track UI element definitions from plugins via MQTT
2. Provide UI element metadata including Flutter widget mappings
3. Enable Flutter frontend to dynamically generate UI from backend API definitions
4. Support real-time updates via MQTT data bindings

## Core UI Element Mappings

### Layout Containers

| Blazor Element | Flutter Widget | BigSkies Element Type | Notes |
|----------------|----------------|----------------------|-------|
| `<fieldset>` | `Card` with `Column` | `panel` | Use `Card` for visual grouping |
| `<legend>` | `Text` in `ListTile` header | N/A | Set as first child in Card |
| `<div class="grid-container-two">` | `GridView` or `Row`/`Column` | `panel` | Use `Row` with `Expanded` for 2-column |
| `<div class="grid-item-left">` | `Expanded` child | N/A | First child in `Row` |
| `<div class="grid-item-right">` | `Expanded` child | N/A | Second child in `Row` |
| `<div class="centered">` | `Row` with `MainAxisAlignment.center` | `panel` | Center-aligned horizontal layout |
| `<body>` | `Column` or `ListView` | `panel` | Main scrollable container |

### Input Controls

| Blazor Element | Flutter Widget | BigSkies Element Type | Notes |
|----------------|----------------|----------------------|-------|
| `<button>` | `ElevatedButton` or `TextButton` | `widget` | Use `onPressed` callback |
| `<input type="checkbox">` | `Checkbox` or `CheckboxListTile` | `widget` | Use `onChanged` callback |
| `<input type="number">` | `TextField` with `TextInputType.number` | `widget` | Or use custom `Stepper` widget |
| `<input type="text">` | `TextField` | `widget` | Single-line text input |
| `<select>` | `DropdownButton<T>` | `widget` | Dropdown selection |
| `<option>` | `DropdownMenuItem<T>` | N/A | Items in dropdown |

### Display Controls

| Blazor Element | Flutter Widget | BigSkies Element Type | Notes |
|----------------|----------------|----------------------|-------|
| `<label>` | `Text` | `widget` | Static text display |
| `<p>` | `Text` | `widget` | Paragraph text |
| `<h2>`, `<h3>` | `Text` with `style: Theme.of(context).textTheme.headline2` | `widget` | Use theme text styles |
| `<svg>` (status circle) | `CustomPaint` with `CustomPainter` | `widget` | Custom drawing with Canvas |
| Dynamic text binding | `Text` with state management | N/A | Update via setState() or Provider |

### Navigation

| Blazor Element | Flutter Widget | BigSkies Element Type | Notes |
|----------------|----------------|----------------------|-------|
| `<NavLink>` | `ListTile` in `Drawer` or `NavigationRail` | `menu` | Navigation item |
| `<ul class="nav flex-column">` | `ListView` in `Drawer` | `menu` | Vertical navigation list |
| `<li class="nav-item">` | `ListTile` | `menu` | Individual nav item |
| Navbar | `AppBar` | `panel` | Application header |

## ASCOM Telescope Control UI Mapping

### Connection Control Section
```dart
// Blazor Structure:
// <fieldset>
//   <legend>Telescope</legend>
//   <div class="grid-container-two">
//     <svg circle> + <button>Connect/Disconnect</button>
//     <button>Setup</button>
//   </div>
// </fieldset>

// Flutter Equivalent:
Card(
  margin: EdgeInsets.all(10),
  child: Padding(
    padding: EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Telescope',
          style: Theme.of(context).textTheme.headline6,
        ),
        SizedBox(height: 10),
        Row(
          children: [
            CustomPaint(
              size: Size(30, 30),
              painter: ConnectionStatusPainter(connected: _isConnected),
            ),
            SizedBox(width: 10),
            ElevatedButton(
              onPressed: _toggleConnection,
              child: Text(_isConnected ? 'Disconnect' : 'Connect'),
            ),
            Spacer(),
            TextButton(
              onPressed: _navigateToSetup,
              child: Text('Setup'),
            ),
          ],
        ),
      ],
    ),
  ),
)

// Custom Painter for connection status circle
class ConnectionStatusPainter extends CustomPainter {
  final bool connected;
  
  ConnectionStatusPainter({required this.connected});
  
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = connected ? Colors.red : Colors.grey
      ..style = PaintingStyle.fill;
    
    final strokePaint = Paint()
      ..color = Colors.black
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3;
    
    final center = Offset(size.width / 2, size.height / 2);
    final radius = size.width / 2 - 2;
    
    canvas.drawCircle(center, radius, paint);
    canvas.drawCircle(center, radius, strokePaint);
  }
  
  @override
  bool shouldRepaint(ConnectionStatusPainter oldDelegate) {
    return connected != oldDelegate.connected;
  }
}
```

### Status Display Section
```dart
// Blazor Structure:
// <div class="grid-container-two">
//   <label>LST</label><p>@LSTText</p>
//   <label>RA</label><p>@RAText</p>
//   <label>Dec</label><p>@DecText</p>
//   <label>Az</label><p>@AzText</p>
//   <label>Alt</label><p>@AltText</p>
// </div>

// Flutter Equivalent:
Card(
  child: Padding(
    padding: EdgeInsets.all(16),
    child: Column(
      children: [
        _buildStatusRow('LST', _lstText),
        _buildStatusRow('RA', _raText),
        _buildStatusRow('Dec', _decText),
        _buildStatusRow('Az', _azText),
        _buildStatusRow('Alt', _altText),
      ],
    ),
  ),
)

Widget _buildStatusRow(String label, String value) {
  return Padding(
    padding: EdgeInsets.symmetric(vertical: 4),
    child: Row(
      children: [
        SizedBox(
          width: 60,
          child: Text(
            '$label:',
            style: TextStyle(fontWeight: FontWeight.bold),
            textAlign: TextAlign.right,
          ),
        ),
        SizedBox(width: 10),
        Text(value),
      ],
    ),
  );
}
```

## ASCOM Telescope Setup UI Mapping

### Configuration Sections
```dart
// Blazor Structure:
// <fieldset disabled="@Device.Connected">
//   <legend>Telescope Settings</legend>
//   <input type="checkbox" id="AutoUnpark" @bind="AutoUnpark">
//   <label for="AutoUnpark">Auto Unpark / Track on Start</label>
//   <input type="number" id="SlewRate" min="0" max="360" step="1" @bind="SlewRate">
// </fieldset>

// Flutter Equivalent:
Card(
  child: Padding(
    padding: EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Telescope Settings',
          style: Theme.of(context).textTheme.headline6,
        ),
        SizedBox(height: 10),
        CheckboxListTile(
          value: _autoUnpark,
          onChanged: _deviceConnected ? null : (value) {
            setState(() {
              _autoUnpark = value ?? false;
            });
          },
          title: Text('Auto Unpark / Track on Start'),
          controlAffinity: ListTileControlAffinity.leading,
        ),
        Row(
          children: [
            Text('Slew Rate (deg/sec):'),
            SizedBox(width: 10),
            Expanded(
              child: Slider(
                value: _slewRate,
                min: 0,
                max: 360,
                divisions: 360,
                label: _slewRate.round().toString(),
                onChanged: _deviceConnected ? null : (value) {
                  setState(() {
                    _slewRate = value;
                  });
                },
              ),
            ),
            SizedBox(
              width: 60,
              child: Text(
                '${_slewRate.round()}',
                textAlign: TextAlign.center,
              ),
            ),
          ],
        ),
      ],
    ),
  ),
)
```

### Site Information Section
```dart
// Blazor Structure:
// <fieldset>
//   <legend>Site Information</legend>
//   <label>Latitude</label>
//   <select id="LatitudeSign">
//     <option value="1">N</option>
//     <option value="-1">S</option>
//   </select>
//   <input type="number" min="0" max="90" @bind="LatitudeDegrees">
//   <input type="number" min="0" max="60" @bind="LatitudeMinutes">
// </fieldset>

// Flutter Equivalent:
Card(
  child: Padding(
    padding: EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Site Information',
          style: Theme.of(context).textTheme.headline6,
        ),
        SizedBox(height: 10),
        Row(
          children: [
            SizedBox(
              width: 80,
              child: Text('Latitude:'),
            ),
            DropdownButton<int>(
              value: _latitudeSign,
              items: [
                DropdownMenuItem(value: 1, child: Text('N')),
                DropdownMenuItem(value: -1, child: Text('S')),
              ],
              onChanged: (value) {
                setState(() {
                  _latitudeSign = value ?? 1;
                });
              },
            ),
            SizedBox(width: 10),
            SizedBox(
              width: 60,
              child: TextField(
                controller: _latDegreesController,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  hintText: '0-90',
                  isDense: true,
                ),
              ),
            ),
            Text('°'),
            SizedBox(width: 10),
            SizedBox(
              width: 60,
              child: TextField(
                controller: _latMinutesController,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  hintText: '0-60',
                  isDense: true,
                ),
              ),
            ),
            Text('\''),
          ],
        ),
      ],
    ),
  ),
)
```

## Navigation Menu Mapping

### Blazor NavMenu Structure
```dart
// Blazor Structure:
// <div class="top-row navbar navbar-dark">
//   <a class="navbar-brand" href="">ASCOM.Alpaca.Simulators</a>
//   <button class="navbar-toggler" @onclick="ToggleNavMenu">
//     <span class="navbar-toggler-icon"></span>
//   </button>
// </div>
// <ul class="nav flex-column">
//   @foreach (var key in DeviceManager.Telescopes)
//   {
//     <NavLink href=@GetSetupURL("Telescope", key.Key)>
//       <span class="oi oi-star"></span> @GetDisplayName("Telescope", key.Key)
//     </NavLink>
//   }
// </ul>

// Flutter Equivalent:
Scaffold(
  appBar: AppBar(
    title: Text('BigSkies Framework'),
    leading: Builder(
      builder: (context) => IconButton(
        icon: Icon(Icons.menu),
        onPressed: () => Scaffold.of(context).openDrawer(),
      ),
    ),
  ),
  drawer: Drawer(
    child: ListView(
      padding: EdgeInsets.zero,
      children: [
        DrawerHeader(
          decoration: BoxDecoration(
            color: Theme.of(context).primaryColor,
          ),
          child: Text(
            'BigSkies Framework',
            style: TextStyle(
              color: Colors.white,
              fontSize: 24,
            ),
          ),
        ),
        ...telescopes.entries.map((entry) => ListTile(
          leading: Icon(Icons.star),
          title: Text(_getDisplayName('Telescope', entry.key)),
          onTap: () {
            Navigator.pop(context);
            _navigateToDevice('telescope', entry.key);
          },
        )),
        Divider(),
        ListTile(
          leading: Icon(Icons.settings),
          title: Text('Setup'),
          onTap: () {
            Navigator.pop(context);
            _navigateToSetup();
          },
        ),
      ],
    ),
  ),
  body: _currentPage,
)

String _getDisplayName(String deviceType, int index) {
  return index == 0 ? deviceType : '$deviceType - $index';
}
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
  "framework_mappings": {
    "flutter": {
      "widget_type": "Card",
      "layout": "column",
      "properties": {
        "margin": "EdgeInsets.all(10)",
        "elevation": 4
      },
      "children": [
        {
          "id": "connection-status",
          "widget_type": "CustomPaint",
          "properties": {
            "size": "Size(30, 30)",
            "painter": "ConnectionStatusPainter"
          },
          "data_binding": {
            "property": "painter.connected",
            "source": "device.status.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "update_interval": 0
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
            "source": "device.status.connected",
            "mqtt_topic": "bigskies/telescope/0/state/connected",
            "transform": "connected ? 'Disconnect' : 'Connect'"
          },
          "actions": {
            "onPressed": {
              "mqtt_topic": "bigskies/telescope/0/command/connect",
              "payload": {"action": "toggle"}
            }
          }
        }
      ]
    }
  }
}
```

### Flutter MQTT Integration
```dart
import 'package:mqtt_client/mqtt_client.dart';
import 'package:mqtt_client/mqtt_server_client.dart';
import 'dart:convert';

class BigSkiesUIGenerator {
  late MqttServerClient client;
  final Map<String, dynamic> uiElements = {};
  
  Future<void> connect() async {
    client = MqttServerClient('localhost', 'flutter_client');
    client.port = 1883;
    client.keepAlivePeriod = 60;
    
    await client.connect();
    
    // Subscribe to UI element updates
    client.subscribe(
      'bigskies/uielement-coordinator/response/query/+',
      MqttQos.atLeastOnce,
    );
    
    client.updates!.listen(_handleMessage);
  }
  
  Future<void> queryUIElements() async {
    final request = {
      'action': 'list_elements',
      'framework': 'flutter',
      'type': 'panel',
    };
    
    final builder = MqttClientPayloadBuilder();
    builder.addString(jsonEncode(request));
    
    client.publishMessage(
      'bigskies/uielement-coordinator/command/query',
      MqttQos.atLeastOnce,
      builder.payload!,
    );
  }
  
  void _handleMessage(List<MqttReceivedMessage<MqttMessage>> messages) {
    final recMess = messages[0].payload as MqttPublishMessage;
    final payload = MqttPublishPayload.bytesToStringAsString(
      recMess.payload.message,
    );
    
    final elements = jsonDecode(payload) as List;
    for (var element in elements) {
      uiElements[element['id']] = element;
    }
  }
  
  Widget buildWidget(String elementId) {
    final element = uiElements[elementId];
    if (element == null) return Container();
    
    final flutterMapping = element['framework_mappings']['flutter'];
    return _buildFromMapping(flutterMapping);
  }
  
  Widget _buildFromMapping(Map<String, dynamic> mapping) {
    final widgetType = mapping['widget_type'];
    
    switch (widgetType) {
      case 'Card':
        return _buildCard(mapping);
      case 'ElevatedButton':
        return _buildElevatedButton(mapping);
      case 'Text':
        return _buildText(mapping);
      default:
        return Container();
    }
  }
  
  Widget _buildCard(Map<String, dynamic> mapping) {
    final children = (mapping['children'] as List?)
        ?.map((child) => _buildFromMapping(child))
        .toList() ?? [];
    
    return Card(
      elevation: mapping['properties']?['elevation'] ?? 1,
      child: Column(children: children),
    );
  }
  
  Widget _buildElevatedButton(Map<String, dynamic> mapping) {
    return ElevatedButton(
      onPressed: () => _handleAction(mapping['actions']?['onPressed']),
      child: Text(mapping['properties']?['child'] ?? 'Button'),
    );
  }
  
  Widget _buildText(Map<String, dynamic> mapping) {
    return Text(mapping['properties']?['data'] ?? '');
  }
  
  void _handleAction(Map<String, dynamic>? action) {
    if (action == null) return;
    
    final topic = action['mqtt_topic'];
    final payload = action['payload'];
    
    final builder = MqttClientPayloadBuilder();
    builder.addString(jsonEncode(payload));
    
    client.publishMessage(topic, MqttQos.atLeastOnce, builder.payload!);
  }
}
```

## Widget Property Mappings

### Common Properties

| Blazor Property | Flutter Property/Method | Notes |
|----------------|------------------------|-------|
| `@bind` | State management (setState, Provider, Riverpod) | Two-way data binding |
| `disabled` | `onChanged: null` or `enabled: false` | Disables widget interaction |
| `@onclick` | `onPressed`, `onTap` | Callback for user interaction |
| `style="color:red"` | `style: TextStyle(color: Colors.red)` | Widget styling |
| `id` | `key: Key('id')` or `ValueKey('id')` | Widget identification |
| `class` | Theme or custom styling | Use ThemeData for consistent styling |
| `min`, `max`, `step` | Slider or TextField validation | For numeric inputs |

### Data Binding Strategy

Flutter uses reactive state management. In BigSkies with MQTT:

1. **Backend → UI Updates**: MQTT messages trigger setState() or Provider updates
2. **UI → Backend Updates**: Widget callbacks publish MQTT messages
3. **Real-time Updates**: Stream-based updates for continuous data

Example:
```dart
class TelescopeControlWidget extends StatefulWidget {
  @override
  _TelescopeControlWidgetState createState() => _TelescopeControlWidgetState();
}

class _TelescopeControlWidgetState extends State<TelescopeControlWidget> {
  late MqttServerClient _mqttClient;
  bool _connected = false;
  String _ra = '00:00:00';
  String _dec = '00:00:00';
  
  @override
  void initState() {
    super.initState();
    _initMqtt();
  }
  
  Future<void> _initMqtt() async {
    _mqttClient = MqttServerClient('localhost', 'telescope_client');
    await _mqttClient.connect();
    
    // Subscribe to state updates
    _mqttClient.subscribe(
      'bigskies/telescope/0/state/+',
      MqttQos.atLeastOnce,
    );
    
    _mqttClient.updates!.listen((List<MqttReceivedMessage<MqttMessage>> c) {
      final recMess = c[0].payload as MqttPublishMessage;
      final payload = MqttPublishPayload.bytesToStringAsString(
        recMess.payload.message,
      );
      
      final data = jsonDecode(payload);
      setState(() {
        if (data['connected'] != null) _connected = data['connected'];
        if (data['ra'] != null) _ra = data['ra'];
        if (data['dec'] != null) _dec = data['dec'];
      });
    });
  }
  
  void _toggleConnection() {
    final builder = MqttClientPayloadBuilder();
    builder.addString(jsonEncode({'action': 'toggle'}));
    
    _mqttClient.publishMessage(
      'bigskies/telescope/0/command/connect',
      MqttQos.atLeastOnce,
      builder.payload!,
    );
  }
  
  @override
  Widget build(BuildContext context) {
    return Card(
      child: Column(
        children: [
          ElevatedButton(
            onPressed: _toggleConnection,
            child: Text(_connected ? 'Disconnect' : 'Connect'),
          ),
          Text('RA: $_ra'),
          Text('Dec: $_dec'),
        ],
      ),
    );
  }
}
```

## Device-Specific Mappings

### Camera Control
- Image display: `Image.memory()` for byte data
- Exposure controls: `Slider` for duration
- Binning: `DropdownButton<int>`

### Dome Control
- Azimuth control: Custom circular slider widget
- Shutter: `Switch` widget

### Focuser Control
- Position: `Slider` with value display
- Absolute/Relative: `SegmentedButton` or `ToggleButtons`

### Switch Control
- Multiple switches: `ListView` with `SwitchListTile` per item

## Best Practices

1. **State Management**: Use Provider, Riverpod, or BLoC for complex state
2. **Responsive Design**: Use `LayoutBuilder` and `MediaQuery` for adaptive layouts
3. **Accessibility**: Set semantic labels for screen readers
4. **Theme Support**: Use `Theme.of(context)` for consistent styling
5. **MQTT Integration**: Use streams for reactive updates
6. **Error Handling**: Show `SnackBar` or `AlertDialog` for errors
7. **Performance**: Use `const` constructors where possible
8. **Platform Support**: Test on iOS, Android, Web, Desktop

## Future Enhancements

1. **Hot Reload**: Dynamic UI updates when plugins change
2. **Custom Widgets**: Telescope position display, sky chart
3. **Multi-Device**: Tabbed views with `TabBar` and `TabBarView`
4. **Responsive**: Adaptive layouts for phone/tablet/desktop
5. **Themes**: Material Design dark/light themes
6. **Localization**: Built-in l10n support

## References

- Flutter Documentation: https://flutter.dev/docs
- MQTT Client: https://pub.dev/packages/mqtt_client
- ASCOM Alpaca API: https://ascom-standards.org/api/
- BigSkies UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
