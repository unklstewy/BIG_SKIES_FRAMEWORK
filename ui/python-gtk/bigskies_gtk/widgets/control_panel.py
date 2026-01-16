"""
Telescope control panel for BigSkies framework.
"""
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk
import logging
import time

logger = logging.getLogger(__name__)


class ControlPanel(Gtk.Box):
    """Telescope control panel widget."""
    
    def __init__(self, mqtt_client):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=10)
        self.mqtt_client = mqtt_client
        self.connected = False
        self.available_configs = []
        self.selected_model = None
        self.selected_mount_type = None
        self.plugin_id = "f7e8d9c6-b5a4-3210-9876-543210fedcba"  # ASCOM Alpaca Simulator plugin ID
        
        self.set_margin_start(10)
        self.set_margin_end(10)
        self.set_margin_top(10)
        self.set_margin_bottom(10)
        
        # Configuration selection
        self._build_configuration_section()
        
        # Connection control
        self._build_connection_section()
        
        # Status display
        self._build_status_section()
        
        # Slew controls
        self._build_slew_section()
        
        # Subscribe to telescope state and plugin config responses
        self.mqtt_client.subscribe("bigskies/telescope/0/state/#", self._on_telescope_state)
        self.mqtt_client.subscribe(f"bigskies/plugin/{self.plugin_id}/config/response", self._on_config_response)
        self.mqtt_client.subscribe(f"bigskies/plugin/{self.plugin_id}/config/event", self._on_config_event)
        
        # Request available configurations on startup
        self._request_configurations()
        
    def _build_configuration_section(self):
        """Build configuration selection section."""
        frame = Gtk.Frame(label="Simulator Configuration (ASCOM Alpaca)")
        frame.set_margin_bottom(10)
        
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=10)
        box.set_margin_start(10)
        box.set_margin_end(10)
        box.set_margin_top(10)
        box.set_margin_bottom(10)
        
        # Model selection
        model_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        
        model_label = Gtk.Label(label="Model:")
        model_label.set_width_chars(12)
        model_label.set_halign(Gtk.Align.START)
        model_box.pack_start(model_label, False, False, 0)
        
        self.model_combo = Gtk.ComboBoxText()
        self.model_combo.append_text("-- Select Model --")
        self.model_combo.set_active(0)
        self.model_combo.connect("changed", self._on_selection_changed)
        model_box.pack_start(self.model_combo, True, True, 0)
        
        box.pack_start(model_box, False, False, 0)
        
        # Mount type selection
        mount_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        
        mount_label = Gtk.Label(label="Mount Type:")
        mount_label.set_width_chars(12)
        mount_label.set_halign(Gtk.Align.START)
        mount_box.pack_start(mount_label, False, False, 0)
        
        self.mount_combo = Gtk.ComboBoxText()
        self.mount_combo.append_text("-- Select Mount Type --")
        self.mount_combo.set_active(0)
        self.mount_combo.connect("changed", self._on_selection_changed)
        mount_box.pack_start(self.mount_combo, True, True, 0)
        
        box.pack_start(mount_box, False, False, 0)
        
        # Button row
        button_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        
        # Refresh button
        refresh_button = Gtk.Button(label="Refresh")
        refresh_button.connect("clicked", lambda b: self._request_configurations())
        button_box.pack_start(refresh_button, False, False, 0)
        
        # Spacer
        button_box.pack_start(Gtk.Box(), True, True, 0)
        
        # Load button
        self.load_button = Gtk.Button(label="Load Configuration")
        self.load_button.set_sensitive(False)
        self.load_button.connect("clicked", self._on_load_config_clicked)
        button_box.pack_end(self.load_button, False, False, 0)
        
        box.pack_start(button_box, False, False, 0)
        
        frame.add(box)
        self.pack_start(frame, False, False, 0)
        
    def _build_connection_section(self):
        """Build connection control section."""
        frame = Gtk.Frame(label="Telescope Connection")
        frame.set_margin_bottom(10)
        
        box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        box.set_margin_start(10)
        box.set_margin_end(10)
        box.set_margin_top(10)
        box.set_margin_bottom(10)
        
        # Status indicator
        self.status_indicator = Gtk.DrawingArea()
        self.status_indicator.set_size_request(30, 30)
        self.status_indicator.connect("draw", self._draw_status_indicator)
        box.pack_start(self.status_indicator, False, False, 0)
        
        # Connect button
        self.connect_button = Gtk.Button(label="Connect")
        self.connect_button.connect("clicked", self._on_connect_clicked)
        box.pack_start(self.connect_button, False, False, 0)
        
        # Setup button
        setup_button = Gtk.Button(label="Setup")
        setup_button.connect("clicked", self._on_setup_clicked)
        box.pack_end(setup_button, False, False, 0)
        
        frame.add(box)
        self.pack_start(frame, False, False, 0)
        
    def _build_status_section(self):
        """Build status display section."""
        frame = Gtk.Frame(label="Position")
        frame.set_margin_bottom(10)
        
        grid = Gtk.Grid()
        grid.set_column_spacing(10)
        grid.set_row_spacing(5)
        grid.set_margin_start(10)
        grid.set_margin_end(10)
        grid.set_margin_top(10)
        grid.set_margin_bottom(10)
        
        # LST
        label = Gtk.Label(label="LST:")
        label.set_halign(Gtk.Align.END)
        label.set_markup("<b>LST:</b>")
        grid.attach(label, 0, 0, 1, 1)
        
        self.lst_value = Gtk.Label(label="00:00:00")
        self.lst_value.set_halign(Gtk.Align.START)
        grid.attach(self.lst_value, 1, 0, 1, 1)
        
        # RA
        label = Gtk.Label()
        label.set_markup("<b>RA:</b>")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 1, 1, 1)
        
        self.ra_value = Gtk.Label(label="00:00:00")
        self.ra_value.set_halign(Gtk.Align.START)
        grid.attach(self.ra_value, 1, 1, 1, 1)
        
        # Dec
        label = Gtk.Label()
        label.set_markup("<b>Dec:</b>")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 2, 1, 1)
        
        self.dec_value = Gtk.Label(label="+00:00:00")
        self.dec_value.set_halign(Gtk.Align.START)
        grid.attach(self.dec_value, 1, 2, 1, 1)
        
        # Az
        label = Gtk.Label()
        label.set_markup("<b>Az:</b>")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 3, 1, 1)
        
        self.az_value = Gtk.Label(label="0.00°")
        self.az_value.set_halign(Gtk.Align.START)
        grid.attach(self.az_value, 1, 3, 1, 1)
        
        # Alt
        label = Gtk.Label()
        label.set_markup("<b>Alt:</b>")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 4, 1, 1)
        
        self.alt_value = Gtk.Label(label="0.00°")
        self.alt_value.set_halign(Gtk.Align.START)
        grid.attach(self.alt_value, 1, 4, 1, 1)
        
        frame.add(grid)
        self.pack_start(frame, False, False, 0)
        
    def _build_slew_section(self):
        """Build slew control section."""
        frame = Gtk.Frame(label="Slew Status")
        
        self.slew_label = Gtk.Label(label="Idle")
        self.slew_label.set_margin_start(10)
        self.slew_label.set_margin_end(10)
        self.slew_label.set_margin_top(10)
        self.slew_label.set_margin_bottom(10)
        
        frame.add(self.slew_label)
        self.pack_start(frame, False, False, 0)
        
    def _draw_status_indicator(self, widget, cr):
        """Draw connection status indicator."""
        width = widget.get_allocated_width()
        height = widget.get_allocated_height()
        
        # Set color based on connection status
        if self.connected:
            cr.set_source_rgb(1.0, 0.0, 0.0)  # Red for connected
        else:
            cr.set_source_rgb(0.5, 0.5, 0.5)  # Gray for disconnected
            
        # Draw filled circle
        cr.arc(width / 2, height / 2, min(width, height) / 2 - 2, 0, 2 * 3.14159)
        cr.fill()
        
        # Draw border
        cr.set_source_rgb(0, 0, 0)
        cr.set_line_width(3)
        cr.arc(width / 2, height / 2, min(width, height) / 2 - 2, 0, 2 * 3.14159)
        cr.stroke()
        
    def _on_connect_clicked(self, button):
        """Handle connect button click."""
        self.mqtt_client.publish(
            "bigskies/telescope/0/command/connect",
            {"action": "toggle"}
        )
        
    def _on_setup_clicked(self, button):
        """Handle setup button click."""
        logger.info("Setup button clicked")
        # TODO: Navigate to setup view
        
    def _on_telescope_state(self, topic, payload):
        """Handle telescope state updates."""
        # Extract state type from topic
        parts = topic.split('/')
        if len(parts) >= 4:
            state_type = parts[3]
            
            if state_type == "connected":
                self.connected = payload.get("value", False)
                self.connect_button.set_label("Disconnect" if self.connected else "Connect")
                self.status_indicator.queue_draw()
                
            elif state_type == "lst":
                self.lst_value.set_text(payload.get("value", "00:00:00"))
                
            elif state_type == "ra":
                self.ra_value.set_text(payload.get("value", "00:00:00"))
                
            elif state_type == "dec":
                self.dec_value.set_text(payload.get("value", "+00:00:00"))
                
            elif state_type == "azimuth":
                az = payload.get("value", 0.0)
                self.az_value.set_text(f"{az:.2f}°")
                
            elif state_type == "altitude":
                alt = payload.get("value", 0.0)
                self.alt_value.set_text(f"{alt:.2f}°")
                
            elif state_type == "slewing":
                slewing = payload.get("value", False)
                if slewing:
                    self.slew_label.set_markup("<span color='orange'><b>SLEWING</b></span>")
                else:
                    self.slew_label.set_text("Idle")
                    
    def _request_configurations(self):
        """Request list of available configurations from ASCOM plugin."""
        logger.info("Requesting available configurations from ASCOM plugin")
        self.mqtt_client.publish(
            f"bigskies/plugin/{self.plugin_id}/config/list",
            {
                "command": "list_configs",
                "request_id": "gui-request-" + str(int(time.time() * 1000))
            }
        )
        
    def _on_config_response(self, topic, payload):
        """Handle configuration response from ASCOM plugin."""
        logger.info(f"Received config response: {payload}")
        
        command = payload.get("command", "")
        success = payload.get("success", False)
        
        if not success:
            logger.error(f"Config command failed: {payload.get('message', 'Unknown error')}")
            return
            
        if command == "list_configs":
            self._handle_config_list(payload.get("data", {}))
        elif command == "load_config":
            self._handle_config_loaded(payload.get("data", {}))
            
    def _handle_config_list(self, data):
        """Handle list of available configurations."""
        configs = data.get("available_configs", [])
        models = data.get("models", {})
        mount_types = data.get("mount_types", {})
        current = data.get("current")
        
        self.available_configs = configs
        
        # Clear and populate model dropdown
        self.model_combo.remove_all()
        self.model_combo.append_text("-- Select Model --")
        
        for config in configs:
            model = config["model"]
            description = config["description"]
            self.model_combo.append(model, f"{model} - {description}")
            
        # Clear and populate mount type dropdown
        self.mount_combo.remove_all()
        self.mount_combo.append_text("-- Select Mount Type --")
        
        for mount_type, description in mount_types.items():
            self.mount_combo.append(mount_type, f"{mount_type} - {description}")
            
        self.model_combo.set_active(0)
        self.mount_combo.set_active(0)
        
        logger.info(f"Loaded {len(configs)} models and {len(mount_types)} mount types")
        
        # Show current configuration if available
        if current:
            logger.info(f"Current config: {current['model']}/{current['mount_type']}")
            
    def _handle_config_loaded(self, data):
        """Handle successful configuration load."""
        model = data.get("model")
        mount_type = data.get("mount_type")
        logger.info(f"Configuration loaded successfully: {model}/{mount_type}")
        
        # Show success message in status
        self.slew_label.set_markup(f"<span color='green'>Config loaded: {model}/{mount_type}</span>")
        
    def _on_config_event(self, topic, payload):
        """Handle configuration events from ASCOM plugin."""
        event_type = payload.get("event_type", "")
        message = payload.get("message", "")
        logger.info(f"Config event: {event_type} - {message}")
        
    def _on_selection_changed(self, combo):
        """Handle model or mount type selection change."""
        model_id = self.model_combo.get_active_id()
        mount_id = self.mount_combo.get_active_id()
        
        if model_id and mount_id:
            self.selected_model = model_id
            self.selected_mount_type = mount_id
            self.load_button.set_sensitive(True)
            logger.info(f"Selected: {model_id} / {mount_id}")
        else:
            self.load_button.set_sensitive(False)
            
    def _on_load_config_clicked(self, button):
        """Handle load configuration button click."""
        if not self.selected_model or not self.selected_mount_type:
            logger.warning("Model and mount type must be selected")
            return
            
        logger.info(f"Loading configuration: {self.selected_model}/{self.selected_mount_type}")
        
        # Publish load config command to ASCOM plugin
        self.mqtt_client.publish(
            f"bigskies/plugin/{self.plugin_id}/config/load",
            {
                "command": "load_config",
                "model": self.selected_model,
                "mount_type": self.selected_mount_type,
                "request_id": "gui-load-" + str(int(time.time() * 1000))
            }
        )
