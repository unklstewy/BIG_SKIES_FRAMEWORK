# Building Cross-Platform UIs with Fyne
## A Practical Guide to Desktop Applications for Big Skies

---

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Lesson 1: Your First Fyne Window](#lesson-1-your-first-fyne-window)
4. [Lesson 2: Layout Basics](#lesson-2-layout-basics)
5. [Lesson 3: Working with Widgets](#lesson-3-working-with-widgets)
6. [Lesson 4: Forms and Data Entry](#lesson-4-forms-and-data-entry)
7. [Lesson 5: Visual Design with Scene Builder](#lesson-5-visual-design-with-scene-builder)
8. [Lesson 6: Connecting to Big Skies Backend](#lesson-6-connecting-to-big-skies-backend)
9. [Lesson 7: Advanced Features](#lesson-7-advanced-features)
10. [Lesson 8: Building and Distribution](#lesson-8-building-and-distribution)
11. [Resources and Next Steps](#resources-and-next-steps)

---

## Introduction

### What is Fyne?

Fyne is a modern, easy-to-use UI toolkit written in pure Go. Unlike GTK or Qt which require C bindings, Fyne is 100% Go and incredibly easy to learn.

**Why Fyne for Big Skies?**
- ✅ Pure Go - integrates seamlessly with your backend
- ✅ Cross-platform - macOS, Windows, Linux from one codebase
- ✅ No dependencies - no CGO headaches
- ✅ Visual designer - drag-and-drop UI building
- ✅ Modern look - Material Design inspired

### What You'll Build

By the end of this tutorial, you'll create a **Big Skies Telescope Controller** desktop app that:
- Displays real-time telescope status
- Sends commands via MQTT
- Has a professional, native-looking interface
- Runs on any platform

### Prerequisites

- Go 1.21+ installed
- Basic Go knowledge (variables, functions, structs)
- Familiarity with Big Skies MQTT concepts (helpful but not required)

**Estimated Time:** 3-4 hours

---

## Getting Started

### Installation

```bash
# Install Fyne toolkit
go get fyne.io/fyne/v2

# Install Fyne command-line tools (includes bundler, packager)
go install fyne.io/fyne/v2/cmd/fyne@latest

# Verify installation
fyne version
```

#### System Dependencies

<div class="os-tabs">
  <div class="os-tab-buttons">
    <button class="os-tab-button active" data-os="macos">macOS</button>
    <button class="os-tab-button" data-os="windows">Windows</button>
    <button class="os-tab-button" data-os="debian">Debian/Ubuntu</button>
    <button class="os-tab-button" data-os="fedora">Fedora/RHEL</button>
    <button class="os-tab-button" data-os="arch">Arch Linux</button>
  </div>
  <div class="os-tab-content">
    <div class="os-tab-pane active" data-os="macos">

You may need Xcode command-line tools:
```bash
xcode-select --install
```

> 💡 **Stuck?** [Google: "install fyne on macos"](https://www.google.com/search?q=install+fyne+on+macos)

</div>
<div class="os-tab-pane" data-os="windows">

Fyne requires a C compiler on Windows. Install one of:

**Option 1: TDM-GCC (Recommended)**
- Download from: https://jmeubank.github.io/tdm-gcc/
- Run installer and select default options
- Add to PATH: `C:\TDM-GCC-64\bin`

**Option 2: MSYS2**
```powershell
# Install MSYS2 from https://www.msys2.org/
# Then in MSYS2 terminal:
pacman -S mingw-w64-x86_64-gcc
```

Verify gcc installation:
```powershell
gcc --version
```

> 💡 **Stuck?** [Google: "install fyne on windows"](https://www.google.com/search?q=install+fyne+on+windows)

</div>
<div class="os-tab-pane" data-os="debian">

Install required development packages:
```bash
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev
```

> 💡 **Stuck?** [Google: "install fyne on ubuntu"](https://www.google.com/search?q=install+fyne+on+ubuntu)

</div>
<div class="os-tab-pane" data-os="fedora">

Install required development packages:
```bash
sudo dnf install -y gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

For older RHEL/CentOS:
```bash
sudo yum install -y gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

> 💡 **Stuck?** [Google: "install fyne on fedora"](https://www.google.com/search?q=install+fyne+on+fedora)

</div>
<div class="os-tab-pane" data-os="arch">

Install required development packages:
```bash
sudo pacman -S gcc libgl libxcursor libxrandr libxinerama libxi libxxf86vm
```

> 💡 **Stuck?** [Google: "install fyne on arch linux"](https://www.google.com/search?q=install+fyne+on+arch+linux)

</div>
</div>
</div>

### Project Setup

```bash
# Create new project
mkdir -p ~/Development/bigskies-ui-tutorial
cd ~/Development/bigskies-ui-tutorial

# Initialize Go module
go mod init github.com/yourusername/bigskies-ui-tutorial

# Install Fyne
go get fyne.io/fyne/v2
```

---

## Lesson 1: Your First Fyne Window

### Goal
Create a simple "Hello Big Skies" window.

### Code

Create `main.go`:

```go
package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Create new application
	myApp := app.New()
	
	// Create window
	myWindow := myApp.NewWindow("Big Skies")
	
	// Add content
	myWindow.SetContent(widget.NewLabel("Hello Big Skies!"))
	
	// Show and run (blocks until window closed)
	myWindow.ShowAndRun()
}
```

### Run It

```bash
go run main.go
```

You should see a window with "Hello Big Skies!" text.

### Understanding the Code

**`app.New()`** - Creates the application instance. Every Fyne app starts here.

**`NewWindow()`** - Creates a window. You can have multiple windows.

**`SetContent()`** - Defines what's displayed in the window. Takes any `fyne.CanvasObject`.

**`ShowAndRun()`** - Shows the window and starts the event loop. Blocks until app closes.

### Experiment

Try changing:
- Window title
- Label text
- Add `myWindow.Resize(fyne.NewSize(400, 300))` before ShowAndRun()

> 💡 **Stuck?** [Google: "fyne tutorial hello world"](https://www.google.com/search?q=fyne+tutorial+hello+world)

---

## Lesson 2: Layout Basics

### Goal
Understand Fyne's container-based layout system.

### Concept

Fyne uses **containers** to arrange widgets. Think of them as invisible boxes that organize content.

**Main Container Types:**
- `VBox` - Vertical stack (top to bottom)
- `HBox` - Horizontal row (left to right)
- `Grid` - Grid layout (rows and columns)
- `Border` - Five regions (Top, Bottom, Left, Right, Center)
- `Split` - Resizable split view

### Example: Vertical Layout

```go
package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Telescope Status")
	
	// Create widgets
	title := widget.NewLabel("Big Skies Telescope")
	status := widget.NewLabel("Status: Disconnected")
	button := widget.NewButton("Connect", func() {
		status.SetText("Status: Connecting...")
	})
	
	// Arrange vertically with VBox
	content := container.NewVBox(
		title,
		status,
		button,
	)
	
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(300, 200))
	myWindow.ShowAndRun()
}
```

### Example: Horizontal Layout

```go
// Create button row
buttons := container.NewHBox(
	widget.NewButton("Connect", func() { /* ... */ }),
	widget.NewButton("Disconnect", func() { /* ... */ }),
	widget.NewButton("Park", func() { /* ... */ }),
)
```

### Example: Grid Layout

```go
// 2-column grid
grid := container.NewGridWithColumns(2,
	widget.NewLabel("RA:"), widget.NewLabel("12.5h"),
	widget.NewLabel("Dec:"), widget.NewLabel("45.0°"),
	widget.NewLabel("Status:"), widget.NewLabel("Tracking"),
)
```

### Nesting Containers

Containers can be nested for complex layouts:

```go
content := container.NewVBox(
	widget.NewLabel("Telescope Control"),
	container.NewHBox(  // Nested horizontal container
		widget.NewButton("Slew", nil),
		widget.NewButton("Park", nil),
	),
	widget.NewLabel("Status: Ready"),
)
```

### Challenge

Create a layout with:
- Title at top
- 2x2 grid in middle (showing RA, Dec, Alt, Az)
- Button row at bottom

> 💡 **Stuck?** [Google: "fyne container layout examples"](https://www.google.com/search?q=fyne+container+layout+examples)

---

## Lesson 3: Working with Widgets

### Goal
Learn common widgets and their usage.

### Essential Widgets

#### Label
Displays text.

```go
label := widget.NewLabel("Static text")
label.SetText("Update text")  // Change after creation

// Rich text label with styling
richLabel := widget.NewRichTextFromMarkdown("**Bold** and *italic*")
```

#### Button
Clickable button.

```go
button := widget.NewButton("Click Me", func() {
	// Called when clicked
	fmt.Println("Button pressed!")
})

// Disable button
button.Disable()
button.Enable()
```

#### Entry
Single-line text input.

```go
entry := widget.NewEntry()
entry.SetPlaceHolder("Enter RA (hours)...")

// Get value
value := entry.Text

// Validate on change
entry.OnChanged = func(content string) {
	fmt.Println("User typed:", content)
}

// Number-only entry
numEntry := widget.NewEntry()
numEntry.Validator = func(s string) error {
	_, err := strconv.ParseFloat(s, 64)
	return err
}
```

#### Check and Radio
Boolean and single-choice selections.

```go
// Checkbox
check := widget.NewCheck("Enable tracking", func(checked bool) {
	fmt.Println("Tracking:", checked)
})

// Radio group (single choice)
radio := widget.NewRadioGroup(
	[]string{"Manual", "Tracking", "Parking"},
	func(selected string) {
		fmt.Println("Mode:", selected)
	},
)
radio.SetSelected("Manual")
```

#### Select (Dropdown)
Dropdown menu.

```go
dropdown := widget.NewSelect(
	[]string{"Telescope 1", "Telescope 2", "Simulator"},
	func(selected string) {
		fmt.Println("Selected:", selected)
	},
)
dropdown.SetSelected("Simulator")
```

#### Progress Bar
Shows progress or activity.

```go
// Determinate (0-100%)
progress := widget.NewProgressBar()
progress.SetValue(0.75)  // 75%

// Indeterminate (spinner)
spinner := widget.NewProgressBarInfinite()
```

### Example: Status Display

```go
package main

import (
	"fmt"
	"time"
	
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Telescope Dashboard")
	
	// Status labels
	raLabel := widget.NewLabel("--")
	decLabel := widget.NewLabel("--")
	statusLabel := widget.NewLabel("Disconnected")
	
	// Control buttons
	connectBtn := widget.NewButton("Connect", func() {
		statusLabel.SetText("Connected")
		raLabel.SetText("12.5h")
		decLabel.SetText("45.0°")
	})
	
	// Layout
	grid := container.NewGridWithColumns(2,
		widget.NewLabel("RA:"), raLabel,
		widget.NewLabel("Dec:"), decLabel,
		widget.NewLabel("Status:"), statusLabel,
	)
	
	content := container.NewVBox(
		widget.NewLabel("Telescope Status"),
		grid,
		connectBtn,
	)
	
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(300, 200))
	myWindow.ShowAndRun()
}
```

### Challenge

Add:
1. An entry field for target RA
2. An entry field for target Dec
3. A "Slew" button that prints the values

> 💡 **Stuck?** [Google: "fyne widgets tutorial"](https://www.google.com/search?q=fyne+widgets+tutorial)

---

## Lesson 4: Forms and Data Entry

### Goal
Build a proper form for telescope control with validation.

### Form Widget

Fyne has a built-in `Form` widget that handles labels, validation, and submission.

```go
package main

import (
	"fmt"
	"strconv"
	
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Slew Telescope")
	
	// Entry fields
	raEntry := widget.NewEntry()
	raEntry.SetPlaceHolder("0-24")
	
	decEntry := widget.NewEntry()
	decEntry.SetPlaceHolder("-90 to 90")
	
	// Create form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Right Ascension (hours)", Widget: raEntry},
			{Text: "Declination (degrees)", Widget: decEntry},
		},
		OnSubmit: func() {
			// Validate RA
			ra, err := strconv.ParseFloat(raEntry.Text, 64)
			if err != nil || ra < 0 || ra > 24 {
				dialog.ShowError(fmt.Errorf("RA must be 0-24"), myWindow)
				return
			}
			
			// Validate Dec
			dec, err := strconv.ParseFloat(decEntry.Text, 64)
			if err != nil || dec < -90 || dec > 90 {
				dialog.ShowError(fmt.Errorf("Dec must be -90 to 90"), myWindow)
				return
			}
			
			// Success
			msg := fmt.Sprintf("Slewing to RA: %.2f, Dec: %.2f", ra, dec)
			dialog.ShowInformation("Command Sent", msg, myWindow)
		},
		OnCancel: func() {
			raEntry.SetText("")
			decEntry.SetText("")
		},
	}
	
	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("Telescope Control"),
		form,
	))
	
	myWindow.Resize(fyne.NewSize(400, 250))
	myWindow.ShowAndRun()
}
```

### Dialogs

Fyne provides several dialog types:

```go
// Information
dialog.ShowInformation("Title", "Message", myWindow)

// Error
dialog.ShowError(fmt.Errorf("Connection failed"), myWindow)

// Confirmation
dialog.ShowConfirm("Park Telescope?", "Are you sure?", 
	func(confirmed bool) {
		if confirmed {
			// Park telescope
		}
	}, myWindow)

// Custom dialog
content := widget.NewLabel("Custom content")
customDialog := dialog.NewCustom("Title", "Close", content, myWindow)
customDialog.Show()
```

### Data Binding

Fyne supports data binding for automatic UI updates:

```go
import "fyne.io/fyne/v2/data/binding"

// Create bound string
status := binding.NewString()
status.Set("Disconnected")

// Bind to label (updates automatically)
label := widget.NewLabelWithData(status)

// Update (label updates automatically)
status.Set("Connected")

// Bind entry to string
raValue := binding.NewString()
entry := widget.NewEntryWithData(raValue)

// Get value
val, _ := raValue.Get()
```

### Example: Complete Control Form

```go
package main

import (
	"fmt"
	
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Telescope Controller")
	
	// Bound data
	statusText := binding.NewString()
	statusText.Set("Disconnected")
	
	raText := binding.NewString()
	decText := binding.NewString()
	
	// Status display
	statusLabel := widget.NewLabelWithData(statusText)
	
	// Control form
	raEntry := widget.NewEntryWithData(raText)
	raEntry.SetPlaceHolder("12.5")
	
	decEntry := widget.NewEntryWithData(decText)
	decEntry.SetPlaceHolder("45.0")
	
	// Buttons
	connectBtn := widget.NewButton("Connect", func() {
		statusText.Set("Connected")
	})
	
	slewBtn := widget.NewButton("Slew", func() {
		ra, _ := raText.Get()
		dec, _ := decText.Get()
		statusText.Set(fmt.Sprintf("Slewing to RA:%s Dec:%s", ra, dec))
	})
	
	// Layout
	content := container.NewVBox(
		widget.NewLabel("Big Skies Telescope"),
		container.NewHBox(
			widget.NewLabel("Status:"),
			statusLabel,
		),
		widget.NewSeparator(),
		widget.NewLabel("Target Coordinates:"),
		container.NewGridWithColumns(2,
			widget.NewLabel("RA:"), raEntry,
			widget.NewLabel("Dec:"), decEntry,
		),
		container.NewHBox(
			connectBtn,
			slewBtn,
		),
	)
	
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.ShowAndRun()
}
```

### Challenge

Add validation that shows red border on invalid input.

> 💡 **Hint:** Look into `entry.Validator` and use `validation.NewRegexp()`

> 💡 **Stuck?** [Google: "fyne form validation"](https://www.google.com/search?q=fyne+form+validation)

---

## Lesson 5: Visual Design with Scene Builder

### Goal
Use drag-and-drop to design UIs visually.

### What is Fyne Scene Builder?

A visual tool for designing Fyne UIs without writing code. Similar to Interface Builder (iOS) or Android Studio's layout editor.

### Installation

<div class="os-tabs">
  <div class="os-tab-buttons">
    <button class="os-tab-button active" data-os="macos">macOS</button>
    <button class="os-tab-button" data-os="windows">Windows</button>
    <button class="os-tab-button" data-os="linux">Linux</button>
    <button class="os-tab-button" data-os="source">Build from Source</button>
  </div>
  <div class="os-tab-content">
    <div class="os-tab-pane active" data-os="macos">

**Option 1: Homebrew**
```bash
brew install defyne
```

**Option 2: Download Binary**
- Download `.dmg` from: https://github.com/fyne-io/defyne/releases
- Drag Defyne.app to Applications folder

</div>
<div class="os-tab-pane" data-os="windows">

**Download Binary**
- Download `defyne-windows-amd64.zip` from: https://github.com/fyne-io/defyne/releases
- Extract and run `defyne.exe`
- Optionally add to PATH for command-line access

</div>
<div class="os-tab-pane" data-os="linux">

**Download Binary**
- Download `defyne-linux-amd64.tar.gz` from: https://github.com/fyne-io/defyne/releases
- Extract: `tar -xzf defyne-linux-amd64.tar.gz`
- Run: `./defyne`
- Optionally move to `/usr/local/bin` for system-wide access

</div>
<div class="os-tab-pane" data-os="source">

**Build from Source (All Platforms)**
```bash
go install github.com/fyne-io/defyne@latest
```

Ensure `$GOPATH/bin` is in your PATH.

</div>
</div>
</div>

### Using Defyne

1. **Launch Defyne**
   ```bash
   defyne
   ```

2. **Create New Project**
   - File → New
   - Choose layout type (VBox, HBox, etc.)

3. **Drag Widgets**
   - Widget palette on left
   - Drag onto canvas
   - Nest containers for complex layouts

4. **Set Properties**
   - Select widget
   - Edit properties in right panel
   - Set text, sizes, callbacks

5. **Generate Code**
   - File → Export → Go Code
   - Copy into your project

### Example Workflow

**Step 1:** Design in Defyne
- Drag VBox as root
- Add Label "Telescope Status"
- Add Grid (2 columns)
- Add Button "Connect"

**Step 2:** Export code

Defyne generates:

```go
func makeUI() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Telescope Status"),
		container.NewGridWithColumns(2,
			widget.NewLabel("RA:"),
			widget.NewLabel("--"),
			widget.NewLabel("Dec:"),
			widget.NewLabel("--"),
		),
		widget.NewButton("Connect", nil),
	)
}
```

**Step 3:** Integrate into your app

```go
func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Big Skies")
	
	// Use designed UI
	myWindow.SetContent(makeUI())
	
	myWindow.ShowAndRun()
}
```

**Step 4:** Add functionality

Update the button callback:

```go
widget.NewButton("Connect", func() {
	// Your connection logic
	connectToTelescope()
})
```

### Tips

- **Start Simple:** Design basic layout, then add complexity
- **Use Containers:** Nest VBox/HBox for flexible layouts
- **Name Widgets:** Use meaningful names in Defyne
- **Iterate:** Design → Export → Test → Refine

### Alternative: JSON Layout

Defyne can also export to JSON:

```json
{
	"Type": "VBox",
	"Children": [
		{"Type": "Label", "Text": "Telescope Status"},
		{"Type": "Button", "Text": "Connect"}
	]
}
```

Load JSON at runtime (advanced):

```go
// Load UI from JSON file
data, _ := os.ReadFile("layout.json")
content := parseLayout(data)  // Your parser
myWindow.SetContent(content)
```

> 💡 **Stuck?** [Google: "defyne fyne visual designer tutorial"](https://www.google.com/search?q=defyne+fyne+visual+designer+tutorial)

---

## Lesson 6: Connecting to Big Skies Backend

### Goal
Integrate your UI with Big Skies MQTT backend.

### Architecture

```
┌─────────────────┐
│   Fyne UI       │
│  (This Lesson)  │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│  MQTT Client    │
│  (From Novice   │
│   Tutorial)     │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│ Big Skies MQTT  │
│     Broker      │
└─────────────────┘
```

### Setup

```bash
# Add MQTT dependency
go get github.com/eclipse/paho.mqtt.golang
```

### MQTT Client Wrapper

Create `mqtt_client.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"
	
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	client mqtt.Client
}

type Message struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

func NewMQTTClient(brokerURL, clientID string) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)
	
	client := mqtt.NewClient(opts)
	token := client.Connect()
	
	if !token.WaitTimeout(5 * time.Second) {
		return nil, fmt.Errorf("connection timeout")
	}
	
	if err := token.Error(); err != nil {
		return nil, err
	}
	
	return &MQTTClient{client: client}, nil
}

func (m *MQTTClient) Publish(topic string, payload map[string]interface{}) error {
	msg := Message{
		ID:        fmt.Sprintf("%d", time.Now().Unix()),
		Source:    "fyne-ui",
		Type:      "command",
		Timestamp: time.Now().Format(time.RFC3339),
		Payload:   payload,
	}
	
	data, _ := json.Marshal(msg)
	token := m.client.Publish(topic, 0, false, data)
	token.Wait()
	
	return token.Error()
}

func (m *MQTTClient) Subscribe(topic string, callback func([]byte)) error {
	token := m.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		callback(msg.Payload())
	})
	token.Wait()
	
	return token.Error()
}

func (m *MQTTClient) Close() {
	m.client.Disconnect(250)
}
```

### Integrated Application

Create `main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Big Skies Telescope Control")
	
	// MQTT client
	var mqttClient *MQTTClient
	
	// Bound data for UI updates
	statusText := binding.NewString()
	statusText.Set("Disconnected")
	
	raText := binding.NewString()
	decText := binding.NewString()
	
	// UI Elements
	statusLabel := widget.NewLabelWithData(statusText)
	
	raEntry := widget.NewEntry()
	raEntry.SetPlaceHolder("12.5")
	
	decEntry := widget.NewEntry()
	decEntry.SetPlaceHolder("45.0")
	
	// Connect button
	connectBtn := widget.NewButton("Connect", func() {
		var err error
		mqttClient, err = NewMQTTClient("tcp://localhost:1883", "fyne-ui")
		
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		
		statusText.Set("Connected")
		
		// Subscribe to status updates
		mqttClient.Subscribe("bigskies/telescope/status", func(payload []byte) {
			var msg Message
			json.Unmarshal(payload, &msg)
			
			// Update UI on main thread
			if ra, ok := msg.Payload["right_ascension"].(float64); ok {
				raText.Set(fmt.Sprintf("%.2f", ra))
			}
			if dec, ok := msg.Payload["declination"].(float64); ok {
				decText.Set(fmt.Sprintf("%.2f", dec))
			}
		})
	})
	
	// Slew button
	slewBtn := widget.NewButton("Slew", func() {
		if mqttClient == nil {
			dialog.ShowError(fmt.Errorf("Not connected"), myWindow)
			return
		}
		
		ra, _ := strconv.ParseFloat(raEntry.Text, 64)
		dec, _ := strconv.ParseFloat(decEntry.Text, 64)
		
		payload := map[string]interface{}{
			"action": "slew_to_coordinates",
			"ra":     ra,
			"dec":    dec,
		}
		
		err := mqttClient.Publish("bigskies/telescope/command/slew", payload)
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		
		statusText.Set("Slewing...")
	})
	
	// Layout
	content := container.NewVBox(
		widget.NewLabel("🔭 Big Skies Telescope Control"),
		widget.NewSeparator(),
		
		container.NewHBox(
			widget.NewLabel("Status:"),
			statusLabel,
		),
		
		widget.NewSeparator(),
		widget.NewLabel("Current Position:"),
		container.NewGridWithColumns(2,
			widget.NewLabel("RA:"), widget.NewLabelWithData(raText),
			widget.NewLabel("Dec:"), widget.NewLabelWithData(decText),
		),
		
		widget.NewSeparator(),
		widget.NewLabel("Target Coordinates:"),
		container.NewGridWithColumns(2,
			widget.NewLabel("RA (hours):"), raEntry,
			widget.NewLabel("Dec (degrees):"), decEntry,
		),
		
		container.NewHBox(
			connectBtn,
			slewBtn,
		),
	)
	
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 400))
	
	// Cleanup on close
	myWindow.SetOnClosed(func() {
		if mqttClient != nil {
			mqttClient.Close()
		}
	})
	
	myWindow.ShowAndRun()
}
```

### Run It

```bash
# Make sure Big Skies services are running
make docker-up

# Run the app
go run .
```

### Testing

1. Click "Connect" - should connect to MQTT broker
2. Enter coordinates (e.g., RA: 10.5, Dec: 45)
3. Click "Slew" - command sent to telescope coordinator
4. Status updates automatically from MQTT messages

### Real-Time Updates

For polling telescope status every 3 seconds:

```go
// Start status update timer
go func() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		if mqttClient != nil {
			payload := map[string]interface{}{
				"action": "get_status",
			}
			mqttClient.Publish("bigskies/telescope/query/status", payload)
		}
	}
}()
```

> 💡 **Stuck?** [Google: "fyne mqtt integration"](https://www.google.com/search?q=fyne+mqtt+integration)

---

## Lesson 7: Advanced Features

### Tabs

Organize complex UIs with tabs:

```go
import "fyne.io/fyne/v2/container"

tabs := container.NewAppTabs(
	container.NewTabItem("Control", makeControlPanel()),
	container.NewTabItem("Status", makeStatusPanel()),
	container.NewTabItem("Settings", makeSettingsPanel()),
)

myWindow.SetContent(tabs)
```

### Menus

Add menu bar:

```go
mainMenu := fyne.NewMainMenu(
	fyne.NewMenu("File",
		fyne.NewMenuItem("Connect", func() { /* ... */ }),
		fyne.NewMenuItem("Disconnect", func() { /* ... */ }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { myApp.Quit() }),
	),
	fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() {
			// Open browser
		}),
	),
)

myWindow.SetMainMenu(mainMenu)
```

### Tables

Display tabular data:

```go
data := [][]string{
	{"Telescope 1", "Connected", "Tracking"},
	{"Telescope 2", "Connected", "Parked"},
	{"Simulator", "Disconnected", "N/A"},
}

table := widget.NewTable(
	func() (int, int) { return len(data), len(data[0]) },
	func() fyne.CanvasObject { return widget.NewLabel("") },
	func(id widget.TableCellID, cell fyne.CanvasObject) {
		cell.(*widget.Label).SetText(data[id.Row][id.Col])
	},
)
```

### Custom Themes

```go
import "fyne.io/fyne/v2/theme"

type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.RGBA{R: 42, G: 82, B: 152, A: 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// Apply theme
myApp.Settings().SetTheme(&myTheme{})
```

### System Tray

Add system tray icon:

```go
import "fyne.io/fyne/v2/driver/desktop"

if desk, ok := myApp.(desktop.App); ok {
	menu := fyne.NewMenu("Big Skies",
		fyne.NewMenuItem("Show", func() {
			myWindow.Show()
		}),
		fyne.NewMenuItem("Quit", func() {
			myApp.Quit()
		}),
	)
	desk.SetSystemTrayMenu(menu)
}
```

### Notifications

```go
import "fyne.io/fyne/v2"

myApp.SendNotification(&fyne.Notification{
	Title:   "Slew Complete",
	Content: "Telescope reached target position",
})
```

> 💡 **Stuck?** [Google: "fyne advanced features"](https://www.google.com/search?q=fyne+advanced+features)

---

## Lesson 8: Building and Distribution

### Development Build

```bash
# Run directly
go run .

# Build binary
go build -o bigskies-ui

# Run binary
./bigskies-ui
```

### Production Build

```bash
# Build optimized binary
go build -ldflags="-s -w" -o bigskies-ui
```

**Flags explained:**
- `-s` - Strip symbol table
- `-w` - Strip debug info
- Result: Smaller binary size

### Cross-Platform Build

```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o bigskies-ui-mac

# Windows
GOOS=windows GOARCH=amd64 go build -o bigskies-ui.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o bigskies-ui-linux
```

### Packaging with Fyne

Fyne provides packaging tools for native bundles:

<div class="os-tabs">
  <div class="os-tab-buttons">
    <button class="os-tab-button active" data-os="macos">macOS</button>
    <button class="os-tab-button" data-os="windows">Windows</button>
    <button class="os-tab-button" data-os="linux">Linux</button>
  </div>
  <div class="os-tab-content">
    <div class="os-tab-pane active" data-os="macos">

**Package as .app Bundle**

```bash
# Package as .app
fyne package -os darwin -icon icon.png

# Creates: BigSkies.app
# Double-clickable application with icon
```

**With App Name and Release Build**
```bash
fyne package -os darwin -icon icon.png -name "BigSkies" -release
```

**Distribution:**
- Zip the .app for distribution
- Or create .dmg installer (requires additional tools)
- For App Store: Use `fyne release -os darwin -appID com.yourcompany.bigskies`

</div>
<div class="os-tab-pane" data-os="windows">

**Package as .exe with Icon**

```bash
# Package as .exe
fyne package -os windows -icon icon.png

# Creates: BigSkies.exe
```

**With App Name and Release Build**
```bash
fyne package -os windows -icon icon.png -name "BigSkies" -release
```

**Distribution:**
- Distribute .exe directly
- Or create installer with NSIS/Inno Setup
- Can be code-signed for Windows SmartScreen

</div>
<div class="os-tab-pane" data-os="linux">

**Package as AppImage**

```bash
# Package as AppImage
fyne package -os linux -icon icon.png

# Creates: BigSkies.AppImage
```

**With App Name and Release Build**
```bash
fyne package -os linux -icon icon.png -name "BigSkies" -release
```

**Distribution Options:**
- AppImage (portable, works everywhere)
- Create .deb package for Debian/Ubuntu
- Create .rpm package for Fedora/RHEL
- Submit to Snap Store or Flathub

**Making AppImage Executable:**
```bash
chmod +x BigSkies.AppImage
./BigSkies.AppImage
```

</div>
</div>
</div>

### Creating Icon

Icon should be 512x512 PNG. Example icon creation:

```bash
# Generate from SVG (requires ImageMagick)
convert -background none -resize 512x512 icon.svg icon.png

# Or use Fyne's bundler
fyne bundle -o bundled.go icon.png
```

### Bundling Assets

Bundle images, configs, etc., into binary:

```bash
# Bundle resource
fyne bundle -o bundled.go icon.png

# Bundle multiple files
fyne bundle -o bundled.go icon.png
fyne bundle -append -o bundled.go config.json
```

Use in code:

```go
import "yourpackage/bundled"

icon := bundled.ResourceIconPng
image := canvas.NewImageFromResource(icon)
```

### Release Checklist

- [ ] Test on target platforms
- [ ] Update version number
- [ ] Create icon.png (512x512)
- [ ] Build with optimizations
- [ ] Package for each platform
- [ ] Test packaged apps
- [ ] Write release notes
- [ ] Upload to GitHub releases

### Example Release Script

```bash
#!/bin/bash
VERSION="1.0.0"

# Build for all platforms
echo "Building v${VERSION}..."

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/bigskies-ui-mac
fyne package -os darwin -icon icon.png -name "BigSkies" -release

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/bigskies-ui.exe
fyne package -os windows -icon icon.png -name "BigSkies" -release

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/bigskies-ui-linux
fyne package -os linux -icon icon.png -name "BigSkies" -release

echo "Build complete!"
```

> 💡 **Stuck?** [Google: "fyne packaging distribution"](https://www.google.com/search?q=fyne+packaging+distribution)

---

## Resources and Next Steps

### Official Documentation

- **Fyne Documentation**: https://developer.fyne.io/
- **API Reference**: https://pkg.go.dev/fyne.io/fyne/v2
- **Fyne Examples**: https://github.com/fyne-io/examples
- **Tutorial Videos**: https://www.youtube.com/c/FyneIO

### Community

- **Discord**: https://discord.gg/fyne
- **GitHub Discussions**: https://github.com/fyne-io/fyne/discussions
- **Reddit**: r/golang, r/Fyne

### Helpful Searches

- [Google: "fyne best practices"](https://www.google.com/search?q=fyne+best+practices)
- [Google: "fyne performance optimization"](https://www.google.com/search?q=fyne+performance+optimization)
- [Google: "fyne custom widgets"](https://www.google.com/search?q=fyne+custom+widgets)
- [Google: "fyne responsive layout"](https://www.google.com/search?q=fyne+responsive+layout)

### Related Tutorials

- **Big Skies Novice Guide**: Complete backend integration tutorial
- **MQTT Documentation**: Understanding message patterns
- **ASCOM Alpaca**: Telescope control protocol (supports telescope, camera, dome, focuser, filterwheel, rotator)

### Next Steps

**Beginner:**
1. Build the basic telescope controller from Lesson 6
2. Add error handling and validation
3. Experiment with different layouts
4. Add device discovery UI showing all ASCOM device types

**Intermediate:**
1. Add multiple telescope support with device pools
2. Implement configuration save/load for telescope + accessories
3. Create custom widgets for camera, focuser, and filterwheel controls
4. Add charting for telescope position
5. Display status for all device types (mount, camera, dome, focuser, etc.)

**Advanced:**
1. Build plugin system for custom device controls
2. Implement telescope path visualization with dome tracking
3. Add camera image capture preview with focus graph
4. Create mobile version with Fyne
5. Implement automated imaging sequences coordinating mount, camera, focuser, and filterwheel

### Project Ideas

- **Telescope Planner**: Plan observation sessions with equipment profiles
- **Sky Chart**: Display current sky with telescope overlay and dome position
- **Session Logger**: Track observations with notes and device telemetry
- **Weather Monitor**: Display seeing conditions and auto-close dome
- **Remote Control**: Web interface + desktop app for all device types
- **Multi-Device Dashboard**: Control telescope, camera, dome, focuser, filterwheel, and rotator from one interface
- **Automated Imaging**: Coordinate slewing, focusing, filter changes, and camera exposures

### Tips for Success

1. **Start Simple**: Get basic UI working before adding features
2. **Test Often**: Run frequently to catch issues early
3. **Read Examples**: Fyne examples repo has great patterns
4. **Ask Community**: Discord is very helpful
5. **Keep It Clean**: Organize code into packages early

---

## Conclusion

Congratulations! You now know how to:

✅ Create Fyne desktop applications  
✅ Design layouts with containers and widgets  
✅ Build forms with validation  
✅ Use visual design tools (Defyne)  
✅ Connect to Big Skies MQTT backend  
✅ Build and distribute cross-platform apps  

**Your Next Challenge**: Build a complete Big Skies control center with:
- Multi-telescope support
- Real-time status dashboard
- Configuration management
- Session planning and logging

Happy coding! 🚀

---

**Tutorial Version**: 1.0  
**Last Updated**: January 2026  
**Big Skies Framework**: Compatible with all versions  
**Fyne Version**: 2.x
