# Blazor to Qt Widget Mapping

## Overview
This document provides a comprehensive mapping of UI elements from the ASCOM Alpaca Simulators Blazor interface to Qt widgets, designed for integration with the BigSkies framework's UI element coordinator. Supports both C++ (Qt Widgets) and Python (PyQt5/PySide6).

## Core UI Element Mappings

### Layout Containers

| Blazor Element | Qt Widget | BigSkies Element Type | Notes |
|----------------|-----------|----------------------|-------|
| `<fieldset>` | `QGroupBox` | `panel` | Group box with title |
| `<legend>` | QGroupBox title | N/A | Set via `setTitle()` |
| `<div class="grid-container-two">` | `QGridLayout` | `panel` | 2-column grid layout |
| `<div class="grid-item-left">` | Grid cell (0, row) | N/A | `addWidget(widget, row, 0)` |
| `<div class="grid-item-right">` | Grid cell (1, row) | N/A | `addWidget(widget, row, 1)` |
| `<div class="centered">` | `QHBoxLayout` with alignment | `panel` | Center-aligned horizontal box |
| `<body>` | `QWidget` or `QMainWindow` | `panel` | Main window container |

### Input Controls

| Blazor Element | Qt Widget | BigSkies Element Type | Notes |
|----------------|-----------|----------------------|-------|
| `<button>` | `QPushButton` | `widget` | Connect to `clicked()` signal |
| `<input type="checkbox">` | `QCheckBox` | `widget` | Connect to `stateChanged()` signal |
| `<input type="number">` | `QSpinBox` or `QDoubleSpinBox` | `widget` | Set min/max via `setRange()` |
| `<input type="text">` | `QLineEdit` | `widget` | Single-line text input |
| `<select>` | `QComboBox` | `widget` | Dropdown selection |
| `<option>` | ComboBox item | N/A | Add via `addItem()` |

### Display Controls

| Blazor Element | Qt Widget | BigSkies Element Type | Notes |
|----------------|-----------|----------------------|-------|
| `<label>` | `QLabel` | `widget` | Static text display |
| `<p>` | `QLabel` | `widget` | Paragraph text with word wrap |
| `<h2>`, `<h3>` | `QLabel` with font | `widget` | Use `QFont` for larger text |
| `<svg>` (status circle) | Custom `QWidget` with `paintEvent` | `widget` | Override `paintEvent()` |
| Dynamic text binding | `setText()` | N/A | Update via slot |

## ASCOM Telescope Control UI Mapping

### C++ Implementation
```cpp
// telescopecontrol.h
#include <QWidget>
#include <QGroupBox>
#include <QPushButton>
#include <QLabel>
#include <QGridLayout>
#include <QPainter>
#include "mqttclient.h"

class ConnectionStatusWidget : public QWidget {
    Q_OBJECT
public:
    ConnectionStatusWidget(QWidget *parent = nullptr);
    void setConnected(bool connected);
    
protected:
    void paintEvent(QPaintEvent *event) override;
    
private:
    bool m_connected;
};

class TelescopeControlWidget : public QWidget {
    Q_OBJECT
public:
    TelescopeControlWidget(QWidget *parent = nullptr);
    
private slots:
    void onConnectClicked();
    void onSetupClicked();
    void onMqttMessage(const QString &topic, const QByteArray &payload);
    
private:
    QGroupBox *m_groupBox;
    ConnectionStatusWidget *m_statusIndicator;
    QPushButton *m_connectBtn;
    QPushButton *m_setupBtn;
    MqttClient *m_mqttClient;
    bool m_connected;
};

// telescopecontrol.cpp
#include "telescopecontrol.h"

ConnectionStatusWidget::ConnectionStatusWidget(QWidget *parent)
    : QWidget(parent), m_connected(false) {
    setFixedSize(30, 30);
}

void ConnectionStatusWidget::setConnected(bool connected) {
    if (m_connected != connected) {
        m_connected = connected;
        update();
    }
}

void ConnectionStatusWidget::paintEvent(QPaintEvent *event) {
    QPainter painter(this);
    painter.setRenderHint(QPainter::Antialiasing);
    
    // Draw circle
    QColor fillColor = m_connected ? Qt::red : Qt::gray;
    painter.setBrush(QBrush(fillColor));
    painter.setPen(QPen(Qt::black, 3));
    painter.drawEllipse(2, 2, 26, 26);
}

TelescopeControlWidget::TelescopeControlWidget(QWidget *parent)
    : QWidget(parent), m_connected(false) {
    
    // Create UI elements
    m_groupBox = new QGroupBox("Telescope", this);
    m_statusIndicator = new ConnectionStatusWidget(this);
    m_connectBtn = new QPushButton("Connect", this);
    m_setupBtn = new QPushButton("Setup", this);
    
    // Layout
    QGridLayout *gridLayout = new QGridLayout;
    gridLayout->addWidget(m_statusIndicator, 0, 0);
    gridLayout->addWidget(m_connectBtn, 0, 1);
    gridLayout->addWidget(m_setupBtn, 0, 2);
    m_groupBox->setLayout(gridLayout);
    
    QVBoxLayout *mainLayout = new QVBoxLayout(this);
    mainLayout->addWidget(m_groupBox);
    
    // Connect signals
    connect(m_connectBtn, &QPushButton::clicked, this, &TelescopeControlWidget::onConnectClicked);
    connect(m_setupBtn, &QPushButton::clicked, this, &TelescopeControlWidget::onSetupClicked);
    
    // MQTT setup
    m_mqttClient = new MqttClient(this);
    m_mqttClient->connectToHost("localhost", 1883);
    m_mqttClient->subscribe("bigskies/telescope/0/state/+");
    connect(m_mqttClient, &MqttClient::messageReceived, this, &TelescopeControlWidget::onMqttMessage);
}

void TelescopeControlWidget::onConnectClicked() {
    QJsonObject payload;
    payload["action"] = "toggle";
    m_mqttClient->publish("bigskies/telescope/0/command/connect", 
                          QJsonDocument(payload).toJson());
}

void TelescopeControlWidget::onMqttMessage(const QString &topic, const QByteArray &payload) {
    if (topic == "bigskies/telescope/0/state/connected") {
        QJsonDocument doc = QJsonDocument::fromJson(payload);
        m_connected = doc.object()["connected"].toBool();
        m_connectBtn->setText(m_connected ? "Disconnect" : "Connect");
        m_statusIndicator->setConnected(m_connected);
    }
}
```

### Python (PyQt5) Implementation
```python
from PyQt5.QtWidgets import (QWidget, QGroupBox, QPushButton, QLabel,
                             QGridLayout, QVBoxLayout)
from PyQt5.QtCore import Qt, pyqtSignal
from PyQt5.QtGui import QPainter, QColor, QPen, QBrush
import paho.mqtt.client as mqtt
import json

class ConnectionStatusWidget(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.connected = False
        self.setFixedSize(30, 30)
    
    def setConnected(self, connected):
        if self.connected != connected:
            self.connected = connected
            self.update()
    
    def paintEvent(self, event):
        painter = QPainter(self)
        painter.setRenderHint(QPainter.Antialiasing)
        
        # Draw circle
        color = QColor(255, 0, 0) if self.connected else QColor(128, 128, 128)
        painter.setBrush(QBrush(color))
        painter.setPen(QPen(Qt.black, 3))
        painter.drawEllipse(2, 2, 26, 26)

class TelescopeControlWidget(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.connected = False
        self.init_ui()
        self.init_mqtt()
    
    def init_ui(self):
        # Create UI elements
        self.group_box = QGroupBox("Telescope")
        self.status_indicator = ConnectionStatusWidget()
        self.connect_btn = QPushButton("Connect")
        self.setup_btn = QPushButton("Setup")
        
        # Layout
        grid_layout = QGridLayout()
        grid_layout.addWidget(self.status_indicator, 0, 0)
        grid_layout.addWidget(self.connect_btn, 0, 1)
        grid_layout.addWidget(self.setup_btn, 0, 2)
        self.group_box.setLayout(grid_layout)
        
        main_layout = QVBoxLayout(self)
        main_layout.addWidget(self.group_box)
        
        # Connect signals
        self.connect_btn.clicked.connect(self.on_connect_clicked)
        self.setup_btn.clicked.connect(self.on_setup_clicked)
    
    def init_mqtt(self):
        self.mqtt_client = mqtt.Client()
        self.mqtt_client.on_message = self.on_mqtt_message
        self.mqtt_client.connect("localhost", 1883, 60)
        self.mqtt_client.subscribe("bigskies/telescope/0/state/+")
        self.mqtt_client.loop_start()
    
    def on_connect_clicked(self):
        payload = json.dumps({"action": "toggle"})
        self.mqtt_client.publish("bigskies/telescope/0/command/connect", payload)
    
    def on_setup_clicked(self):
        # Navigate to setup page
        pass
    
    def on_mqtt_message(self, client, userdata, msg):
        if msg.topic == "bigskies/telescope/0/state/connected":
            data = json.loads(msg.payload)
            self.connected = data.get("connected", False)
            self.connect_btn.setText("Disconnect" if self.connected else "Connect")
            self.status_indicator.setConnected(self.connected)
```

## Telescope Setup UI Mapping

### C++ Implementation
```cpp
class TelescopeSetupWidget : public QWidget {
    Q_OBJECT
public:
    TelescopeSetupWidget(QWidget *parent = nullptr);
    void setDeviceConnected(bool connected);
    
private:
    QGroupBox *m_settingsGroup;
    QCheckBox *m_autoUnparkCheck;
    QSpinBox *m_slewRateSpinBox;
    QComboBox *m_latitudeSignCombo;
    QSpinBox *m_latDegreesSpinBox;
    QSpinBox *m_latMinutesSpinBox;
    bool m_deviceConnected;
};

TelescopeSetupWidget::TelescopeSetupWidget(QWidget *parent)
    : QWidget(parent), m_deviceConnected(false) {
    
    // Settings Group
    m_settingsGroup = new QGroupBox("Telescope Settings");
    m_autoUnparkCheck = new QCheckBox("Auto Unpark / Track on Start");
    
    QLabel *slewLabel = new QLabel("Slew Rate (deg/sec):");
    m_slewRateSpinBox = new QSpinBox();
    m_slewRateSpinBox->setRange(0, 360);
    m_slewRateSpinBox->setValue(10);
    
    QVBoxLayout *settingsLayout = new QVBoxLayout;
    settingsLayout->addWidget(m_autoUnparkCheck);
    
    QHBoxLayout *slewLayout = new QHBoxLayout;
    slewLayout->addWidget(slewLabel);
    slewLayout->addWidget(m_slewRateSpinBox);
    settingsLayout->addLayout(slewLayout);
    
    m_settingsGroup->setLayout(settingsLayout);
    
    // Site Information
    QGroupBox *siteGroup = new QGroupBox("Site Information");
    m_latitudeSignCombo = new QComboBox();
    m_latitudeSignCombo->addItem("N", 1);
    m_latitudeSignCombo->addItem("S", -1);
    
    m_latDegreesSpinBox = new QSpinBox();
    m_latDegreesSpinBox->setRange(0, 90);
    
    m_latMinutesSpinBox = new QSpinBox();
    m_latMinutesSpinBox->setRange(0, 60);
    
    QHBoxLayout *latLayout = new QHBoxLayout;
    latLayout->addWidget(new QLabel("Latitude:"));
    latLayout->addWidget(m_latitudeSignCombo);
    latLayout->addWidget(m_latDegreesSpinBox);
    latLayout->addWidget(new QLabel("°"));
    latLayout->addWidget(m_latMinutesSpinBox);
    latLayout->addWidget(new QLabel("'"));
    
    siteGroup->setLayout(latLayout);
    
    // Main layout
    QVBoxLayout *mainLayout = new QVBoxLayout(this);
    mainLayout->addWidget(m_settingsGroup);
    mainLayout->addWidget(siteGroup);
}

void TelescopeSetupWidget::setDeviceConnected(bool connected) {
    m_deviceConnected = connected;
    m_autoUnparkCheck->setEnabled(!connected);
    m_slewRateSpinBox->setEnabled(!connected);
}
```

## BigSkies Framework Integration

### MQTT Client Wrapper (C++)
```cpp
// mqttclient.h
#include <QObject>
#include <mqtt/async_client.h>

class MqttClient : public QObject {
    Q_OBJECT
public:
    explicit MqttClient(QObject *parent = nullptr);
    void connectToHost(const QString &host, int port = 1883);
    void subscribe(const QString &topic);
    void publish(const QString &topic, const QByteArray &payload);
    
signals:
    void messageReceived(const QString &topic, const QByteArray &payload);
    
private:
    mqtt::async_client *m_client;
    QString m_clientId;
};
```

### Query UI Elements (Python)
```python
import json
import paho.mqtt.client as mqtt

class BigSkiesUIGenerator:
    def __init__(self):
        self.mqtt_client = mqtt.Client()
        self.ui_elements = {}
        
    def connect(self):
        self.mqtt_client.on_message = self.on_message
        self.mqtt_client.connect("localhost", 1883)
        self.mqtt_client.subscribe("bigskies/uielement-coordinator/response/query/+")
        self.mqtt_client.loop_start()
        
    def query_ui_elements(self):
        request = {
            "action": "list_elements",
            "framework": "qt",
            "type": "panel"
        }
        self.mqtt_client.publish(
            "bigskies/uielement-coordinator/command/query",
            json.dumps(request)
        )
    
    def on_message(self, client, userdata, msg):
        elements = json.loads(msg.payload)
        for element in elements:
            self.ui_elements[element['id']] = element
```

## Widget Property Mappings

| Blazor Property | Qt Property/Method | Notes |
|----------------|-------------------|-------|
| `@bind` | Signals/slots | Connect `valueChanged`, `textChanged`, etc. |
| `disabled` | `setEnabled(false)` | Disables widget |
| `@onclick` | `clicked()` signal | Connect to slot |
| `style="color:red"` | `setStyleSheet()` or `QPalette` | Qt stylesheet or palette |
| `id` | `setObjectName()` | Widget name for finding |
| `class` | StyleSheet selector | Use in QSS |
| `min`, `max`, `step` | `setRange()`, `setSingleStep()` | For spin boxes |

## Best Practices

1. **Signals/Slots**: Use Qt's signal/slot mechanism for event handling
2. **Layouts**: Use layout managers (`QVBoxLayout`, `QHBoxLayout`, `QGridLayout`)
3. **Thread Safety**: Use `QMetaObject::invokeMethod()` for cross-thread UI updates
4. **MQTT Integration**: Run MQTT client on background thread
5. **Resource Files**: Use `.qrc` files for bundling resources
6. **Internationalization**: Use `tr()` for translatable strings
7. **Model/View**: Use `QAbstractItemModel` for complex data
8. **Style Sheets**: Use QSS for consistent styling

## Device-Specific Mappings

- **Camera**: `QLabel` with `QPixmap` for image display
- **Dome**: Custom widget with `paintEvent()` for azimuth dial
- **Focuser**: `QSlider` with `QSpinBox` buddy
- **Switch**: `QListWidget` with checkable items

## References

- Qt Documentation: https://doc.qt.io/
- PyQt5 Documentation: https://www.riverbankcomputing.com/static/Docs/PyQt5/
- MQTT C++ Client: https://github.com/eclipse/paho.mqtt.cpp
- ASCOM Alpaca API: https://ascom-standards.org/api/
- BigSkies UI Element Coordinator: `internal/coordinators/uielement_coordinator.go`
