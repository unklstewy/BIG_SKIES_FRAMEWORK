# Glade UI Definition Files

This directory contains Glade XML files that define the GTK+ user interface components for the BigSkies Framework Python GTK application.

## What is Glade?

Glade is a RAD (Rapid Application Development) tool for GTK+ applications. It allows you to design user interfaces visually and saves them as XML files that can be loaded at runtime using `Gtk.Builder`.

## Files

### login_dialog.glade
Defines the login dialog with:
- Server connection field
- Username field
- Password field (masked)
- Login and Cancel buttons

### main_window.glade
Defines the main application window with:
- Header bar with title and MQTT status indicator
- Sidebar navigation with title and buttons
- Content stack for switching between panels
- Paned layout for resizable sidebar

## Editing Glade Files

You can edit these files using:

1. **Glade Designer** - Visual UI editor
   ```bash
   # Install on macOS
   brew install glade
   
   # Open a file
   glade login_dialog.glade
   ```

2. **Text Editor** - Direct XML editing
   - Any text editor can be used to modify the XML directly
   - Useful for fine-tuning properties or version control

## Using Glade Files in Python

To load and use these interfaces in your Python GTK application:

```python
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk

# Load the UI definition
builder = Gtk.Builder()
builder.add_from_file("glade/login_dialog.glade")

# Get widgets by ID
dialog = builder.get_object("login_dialog")
server_entry = builder.get_object("server_entry")
username_entry = builder.get_object("username_entry")
password_entry = builder.get_object("password_entry")

# Connect signals
builder.connect_signals(handler_object)

# Show the dialog
response = dialog.run()
```

## Current Implementation

The current Python implementation builds the UI programmatically in code. These Glade files are provided as:

1. **Documentation** - Visual representation of the UI structure
2. **Alternative Implementation** - Can be used instead of programmatic UI building
3. **Prototyping** - Quick UI mockups and testing
4. **Translation** - Easier to add i18n support with translatable strings marked in XML

## Converting to Glade-Based Implementation

To convert the current code-based UI to use these Glade files:

1. Replace widget creation code with `Gtk.Builder.add_from_file()`
2. Use `builder.get_object(id)` to access widgets instead of instance variables
3. Connect signal handlers using `builder.connect_signals()`
4. Keep dynamic UI elements (tabs, lists) in code

Example refactor for login dialog:

**Before (Programmatic):**
```python
dialog = Gtk.Dialog(title="Login")
grid = Gtk.Grid()
# ... create all widgets ...
```

**After (Glade-based):**
```python
builder = Gtk.Builder()
builder.add_from_file("glade/login_dialog.glade")
dialog = builder.get_object("login_dialog")
```

## Benefits of Glade Files

- **Separation of Concerns**: UI layout separated from business logic
- **Visual Editing**: Easier to design and adjust layouts
- **Maintainability**: Changes to UI don't require code recompilation
- **Translation**: Built-in support for translatable strings
- **Collaboration**: Designers can work on UI without touching Python code

## Limitations

- Dynamic content (list items, tabs, custom widgets) still needs code
- Complex layouts may be easier to understand in code
- Requires Gtk.Builder infrastructure in application
- Changes need to be synced between Glade and code

## Widget IDs Reference

### login_dialog.glade
- `login_dialog` - Main dialog window
- `server_entry` - Server address input
- `username_entry` - Username input
- `password_entry` - Password input
- `login_button` - Submit button
- `cancel_button` - Cancel button

### main_window.glade
- `main_window` - Application window
- `header_bar` - Top header bar
- `mqtt_status_label` - Connection status indicator
- `sidebar` - Left navigation sidebar
- `nav_buttons_box` - Container for navigation buttons
- `about_button` - About dialog button
- `content_stack` - Main content area
- `main_paned` - Resizable pane divider
