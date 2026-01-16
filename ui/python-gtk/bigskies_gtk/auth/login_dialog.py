"""
Authentication dialog for BigSkies framework.
"""
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk
import logging

logger = logging.getLogger(__name__)


class LoginDialog(Gtk.Dialog):
    """Login dialog for BigSkies authentication."""
    
    def __init__(self, parent):
        super().__init__(title="BigSkies Login", parent=parent, flags=0)
        self.add_buttons(
            Gtk.STOCK_CANCEL, Gtk.ResponseType.CANCEL,
            Gtk.STOCK_OK, Gtk.ResponseType.OK
        )
        
        self.set_default_size(350, 150)
        self.set_border_width(10)
        
        box = self.get_content_area()
        grid = Gtk.Grid()
        grid.set_column_spacing(10)
        grid.set_row_spacing(10)
        box.add(grid)
        
        # Username
        label = Gtk.Label(label="Username:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 0, 1, 1)
        
        self.username_entry = Gtk.Entry()
        self.username_entry.set_text("admin")
        grid.attach(self.username_entry, 1, 0, 1, 1)
        
        # Password
        label = Gtk.Label(label="Password:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 1, 1, 1)
        
        self.password_entry = Gtk.Entry()
        self.password_entry.set_visibility(False)
        self.password_entry.set_text("password")
        grid.attach(self.password_entry, 1, 1, 1, 1)
        
        # Server
        label = Gtk.Label(label="MQTT Server:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 2, 1, 1)
        
        self.server_entry = Gtk.Entry()
        self.server_entry.set_text("localhost")
        grid.attach(self.server_entry, 1, 2, 1, 1)
        
        self.show_all()
        
    def get_credentials(self):
        """Return entered credentials as dict."""
        return {
            'username': self.username_entry.get_text(),
            'password': self.password_entry.get_text(),
            'server': self.server_entry.get_text()
        }
