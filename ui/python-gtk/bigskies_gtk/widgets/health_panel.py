"""
Service health monitoring panel for BigSkies coordinators.
"""
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gdk
import logging
from datetime import datetime

logger = logging.getLogger(__name__)


class HealthPanel(Gtk.Box):
    """Panel displaying health status of all BigSkies coordinators."""
    
    COORDINATORS = [
        ("message-coordinator", "Message Bus"),
        ("application-coordinator", "Application Services"),
        ("security-coordinator", "Security"),
        ("telescope-coordinator", "Telescope"),
        ("uielement-coordinator", "UI Elements"),
        ("plugin-coordinator", "Plugins"),
        ("datastore-coordinator", "Data Store"),
    ]
    
    def __init__(self, mqtt_client):
        super().__init__(orientation=Gtk.Orientation.VERTICAL, spacing=6)
        self.mqtt_client = mqtt_client
        self.health_widgets = {}
        
        self.set_margin_start(10)
        self.set_margin_end(10)
        self.set_margin_top(10)
        self.set_margin_bottom(10)
        
        # Title
        title = Gtk.Label()
        title.set_markup("<b>Service Health Status</b>")
        title.set_halign(Gtk.Align.START)
        self.pack_start(title, False, False, 0)
        
        # Coordinator list
        scrolled = Gtk.ScrolledWindow()
        scrolled.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
        scrolled.set_min_content_height(400)
        
        list_box = Gtk.ListBox()
        list_box.set_selection_mode(Gtk.SelectionMode.NONE)
        scrolled.add(list_box)
        
        for coord_id, coord_name in self.COORDINATORS:
            row = self._create_health_row(coord_id, coord_name)
            list_box.add(row)
            
        self.pack_start(scrolled, True, True, 0)
        
        # Subscribe to health topics
        self.mqtt_client.subscribe("bigskies/coordinator/+/health/status", self._on_health_message)
        
    def _create_health_row(self, coord_id, coord_name):
        """Create a health status row for a coordinator."""
        row = Gtk.ListBoxRow()
        box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        box.set_margin_start(10)
        box.set_margin_end(10)
        box.set_margin_top(5)
        box.set_margin_bottom(5)
        
        # Status indicator (colored circle)
        indicator = Gtk.DrawingArea()
        indicator.set_size_request(20, 20)
        indicator.connect("draw", self._draw_indicator, "unknown")
        box.pack_start(indicator, False, False, 0)
        
        # Coordinator name
        name_label = Gtk.Label(label=coord_name)
        name_label.set_halign(Gtk.Align.START)
        name_label.set_hexpand(True)
        box.pack_start(name_label, True, True, 0)
        
        # Status text
        status_label = Gtk.Label(label="Unknown")
        status_label.set_halign(Gtk.Align.END)
        box.pack_start(status_label, False, False, 0)
        
        # Last update time
        time_label = Gtk.Label(label="--:--:--")
        time_label.set_halign(Gtk.Align.END)
        time_label.set_width_chars(10)
        box.pack_start(time_label, False, False, 0)
        
        row.add(box)
        
        self.health_widgets[coord_id] = {
            'indicator': indicator,
            'status_label': status_label,
            'time_label': time_label,
            'status': 'unknown'
        }
        
        return row
        
    def _draw_indicator(self, widget, cr, status):
        """Draw status indicator circle."""
        width = widget.get_allocated_width()
        height = widget.get_allocated_height()
        
        # Set color based on status
        if status == "healthy":
            cr.set_source_rgb(0.2, 0.8, 0.2)  # Green
        elif status == "warning":
            cr.set_source_rgb(1.0, 0.8, 0.0)  # Yellow
        elif status == "unhealthy":
            cr.set_source_rgb(0.9, 0.2, 0.2)  # Red
        else:
            cr.set_source_rgb(0.5, 0.5, 0.5)  # Gray
            
        # Draw filled circle
        cr.arc(width / 2, height / 2, min(width, height) / 2 - 2, 0, 2 * 3.14159)
        cr.fill()
        
        # Draw border
        cr.set_source_rgb(0, 0, 0)
        cr.set_line_width(1)
        cr.arc(width / 2, height / 2, min(width, height) / 2 - 2, 0, 2 * 3.14159)
        cr.stroke()
        
    def _on_health_message(self, topic, payload):
        """Handle health status message."""
        logger.info(f"Received health message on topic: {topic}")
        logger.info(f"Payload: {payload}")
        
        # Extract coordinator ID from topic: bigskies/coordinator/<coord-id>/health/status
        parts = topic.split('/')
        if len(parts) >= 3:
            coord_id = parts[2] + "-coordinator"  # Convert "message" to "message-coordinator"
            logger.info(f"Extracted coordinator ID: {coord_id}")
            logger.info(f"Known coordinators: {list(self.health_widgets.keys())}")
            
            if coord_id in self.health_widgets:
                # Extract status from nested payload structure
                status_payload = payload.get('payload', {})
                status = status_payload.get('status', 'unknown').lower()
                message = status_payload.get('message', 'No message')
                
                widgets = self.health_widgets[coord_id]
                widgets['status'] = status
                widgets['status_label'].set_text(message)
                widgets['time_label'].set_text(datetime.now().strftime("%H:%M:%S"))
                
                # Redraw indicator
                widgets['indicator'].connect("draw", self._draw_indicator, status)
                widgets['indicator'].queue_draw()
                
                logger.debug(f"Updated health for {coord_id}: {status}")
