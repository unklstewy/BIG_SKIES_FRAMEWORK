"""
Telescope visual preview widget.
"""
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk
import math
import logging

logger = logging.getLogger(__name__)


class TelescopePreview(Gtk.DrawingArea):
    """Visual telescope orientation preview."""
    
    def __init__(self, mqtt_client):
        super().__init__()
        self.mqtt_client = mqtt_client
        self.set_size_request(400, 400)
        
        # Telescope state
        self.azimuth = 0.0  # degrees
        self.altitude = 45.0  # degrees
        self.connected = False
        self.slewing = False
        
        # Connect draw signal
        self.connect("draw", self._on_draw)
        
        # Subscribe to telescope state
        self.mqtt_client.subscribe("bigskies/telescope/0/state/#", self._on_telescope_state)
        
    def _on_telescope_state(self, topic, payload):
        """Handle telescope state updates."""
        parts = topic.split('/')
        if len(parts) >= 4:
            state_type = parts[3]
            
            if state_type == "connected":
                self.connected = payload.get("value", False)
                self.queue_draw()
                
            elif state_type == "azimuth":
                self.azimuth = payload.get("value", 0.0)
                self.queue_draw()
                
            elif state_type == "altitude":
                self.altitude = payload.get("value", 45.0)
                self.queue_draw()
                
            elif state_type == "slewing":
                self.slewing = payload.get("value", False)
                self.queue_draw()
                
    def _on_draw(self, widget, cr):
        """Draw telescope preview."""
        width = widget.get_allocated_width()
        height = widget.get_allocated_height()
        center_x = width / 2
        center_y = height / 2
        radius = min(width, height) / 2 - 20
        
        # Background
        cr.set_source_rgb(0.1, 0.1, 0.15)
        cr.rectangle(0, 0, width, height)
        cr.fill()
        
        # Draw horizon circle
        cr.set_source_rgb(0.3, 0.3, 0.4)
        cr.set_line_width(2)
        cr.arc(center_x, center_y, radius, 0, 2 * math.pi)
        cr.stroke()
        
        # Draw cardinal directions
        cr.set_source_rgb(0.6, 0.6, 0.7)
        cr.select_font_face("Sans", 0, 0)
        cr.set_font_size(14)
        
        # N, E, S, W
        directions = [
            ("N", 0, -radius - 10),
            ("E", radius + 10, 0),
            ("S", 0, radius + 15),
            ("W", -radius - 15, 0)
        ]
        
        for text, dx, dy in directions:
            extents = cr.text_extents(text)
            cr.move_to(center_x + dx - extents.width / 2, 
                      center_y + dy + extents.height / 2)
            cr.show_text(text)
            
        # Draw altitude circles (30, 60, 90 degrees)
        cr.set_source_rgb(0.2, 0.2, 0.25)
        cr.set_line_width(1)
        for alt in [30, 60]:
            # Radius shrinks as altitude increases
            alt_radius = radius * (90 - alt) / 90
            cr.arc(center_x, center_y, alt_radius, 0, 2 * math.pi)
            cr.stroke()
            
        # Draw azimuth lines (N, NE, E, SE, S, SW, W, NW)
        cr.set_source_rgb(0.2, 0.2, 0.25)
        cr.set_line_width(1)
        for az_deg in range(0, 360, 45):
            az_rad = math.radians(az_deg)
            x = center_x + radius * math.sin(az_rad)
            y = center_y - radius * math.cos(az_rad)
            cr.move_to(center_x, center_y)
            cr.line_to(x, y)
            cr.stroke()
            
        if self.connected:
            # Draw telescope pointer
            az_rad = math.radians(self.azimuth)
            # Altitude: 0° at edge, 90° at center
            alt_radius = radius * (90 - self.altitude) / 90
            
            # Calculate position
            x = center_x + alt_radius * math.sin(az_rad)
            y = center_y - alt_radius * math.cos(az_rad)
            
            # Draw pointer
            if self.slewing:
                cr.set_source_rgb(1.0, 0.6, 0.0)  # Orange when slewing
            else:
                cr.set_source_rgb(0.2, 0.8, 0.2)  # Green when idle
                
            # Draw triangle pointing in azimuth direction
            size = 15
            cr.move_to(x, y)
            cr.line_to(
                x + size * math.sin(az_rad + math.pi + 0.3),
                y - size * math.cos(az_rad + math.pi + 0.3)
            )
            cr.line_to(
                x + size * math.sin(az_rad + math.pi - 0.3),
                y - size * math.cos(az_rad + math.pi - 0.3)
            )
            cr.close_path()
            cr.fill()
            
            # Draw circle at tip
            cr.arc(x, y, 5, 0, 2 * math.pi)
            cr.fill()
            
            # Draw coordinates text
            cr.set_source_rgb(1.0, 1.0, 1.0)
            cr.set_font_size(12)
            coord_text = f"Az: {self.azimuth:.1f}°  Alt: {self.altitude:.1f}°"
            extents = cr.text_extents(coord_text)
            cr.move_to(10, height - 10)
            cr.show_text(coord_text)
        else:
            # Draw "Disconnected" message
            cr.set_source_rgb(0.7, 0.7, 0.7)
            cr.set_font_size(16)
            text = "Telescope Disconnected"
            extents = cr.text_extents(text)
            cr.move_to(center_x - extents.width / 2,
                      center_y + extents.height / 2)
            cr.show_text(text)
