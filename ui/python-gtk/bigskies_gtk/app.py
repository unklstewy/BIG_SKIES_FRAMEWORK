"""
Main application class for BigSkies GTK UI.
"""
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gio
import logging
import sys

from .mqtt.client import BigSkiesMQTTClient
from .auth.login_dialog import LoginDialog
from .widgets.health_panel import HealthPanel
from .widgets.control_panel import ControlPanel
from .widgets.telescope_preview import TelescopePreview

logger = logging.getLogger(__name__)


class BigSkiesApp(Gtk.Application):
    """Main BigSkies GTK application."""
    
    def __init__(self):
        super().__init__(
            application_id="dev.bigskies.gtk",
            flags=Gio.ApplicationFlags.FLAGS_NONE
        )
        self.window = None
        self.mqtt_client = None
        self.username = None
        
    def do_startup(self):
        """Application startup."""
        Gtk.Application.do_startup(self)
        
        # Setup logging
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        
        logger.info("BigSkies GTK Application starting...")
        
    def do_activate(self):
        """Application activation."""
        if not self.window:
            self._show_login()
            
    def _show_login(self):
        """Show login dialog."""
        dialog = LoginDialog(None)
        response = dialog.run()
        
        if response == Gtk.ResponseType.OK:
            creds = dialog.get_credentials()
            dialog.destroy()
            
            # Connect to MQTT
            self.mqtt_client = BigSkiesMQTTClient(
                broker_host=creds['server'],
                broker_port=1883
            )
            
            self.mqtt_client.set_callbacks(
                on_connect=self._on_mqtt_connected,
                on_disconnect=self._on_mqtt_disconnected
            )
            
            if self.mqtt_client.connect():
                self.username = creds['username']
                self._build_main_window()
            else:
                self._show_error("Failed to connect to MQTT broker")
                self.quit()
        else:
            dialog.destroy()
            self.quit()
            
    def _build_main_window(self):
        """Build main application window."""
        self.window = Gtk.ApplicationWindow(application=self)
        self.window.set_title("BigSkies Framework")
        self.window.set_default_size(1200, 800)
        self.window.set_border_width(0)
        
        # Header bar
        header = Gtk.HeaderBar()
        header.set_show_close_button(True)
        header.set_title("BigSkies Framework")
        header.set_subtitle(f"User: {self.username}")
        self.window.set_titlebar(header)
        
        # Status indicator in header
        self.mqtt_status_label = Gtk.Label(label="● Connected")
        self.mqtt_status_label.set_markup("<span color='green'>● Connected</span>")
        header.pack_end(self.mqtt_status_label)
        
        # Main container
        paned = Gtk.Paned(orientation=Gtk.Orientation.HORIZONTAL)
        self.window.add(paned)
        
        # Left sidebar (navigation)
        sidebar = self._build_sidebar()
        paned.pack1(sidebar, False, False)
        
        # Right content area
        self.content_stack = Gtk.Stack()
        self.content_stack.set_transition_type(Gtk.StackTransitionType.SLIDE_LEFT_RIGHT)
        
        # Add pages
        self.health_panel = HealthPanel(self.mqtt_client)
        self.content_stack.add_titled(self.health_panel, "health", "Health")
        
        self.control_panel = ControlPanel(self.mqtt_client)
        self.content_stack.add_titled(self.control_panel, "control", "Control")
        
        self.telescope_preview = TelescopePreview(self.mqtt_client)
        preview_box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL)
        preview_box.set_halign(Gtk.Align.CENTER)
        preview_box.set_valign(Gtk.Align.CENTER)
        preview_box.pack_start(self.telescope_preview, False, False, 0)
        self.content_stack.add_titled(preview_box, "preview", "Preview")
        
        paned.pack2(self.content_stack, True, False)
        
        # Set initial position
        paned.set_position(250)
        
        self.window.show_all()
        
    def _build_sidebar(self):
        """Build navigation sidebar."""
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        box.set_size_request(250, -1)
        
        # Style
        style_provider = Gtk.CssProvider()
        style_provider.load_from_data(b"""
            .sidebar { background-color: #2d2d2d; }
            .nav-button { 
                background: transparent; 
                border: none; 
                border-radius: 0;
                padding: 12px;
                color: #ffffff;
            }
            .nav-button:hover { background-color: #3d3d3d; }
            .nav-button:checked { background-color: #4d4d4d; }
        """)
        Gtk.StyleContext.add_provider_for_screen(
            self.window.get_screen(),
            style_provider,
            Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
        )
        
        box.get_style_context().add_class("sidebar")
        
        # Logo/title
        title_box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=5)
        title_box.set_margin_start(10)
        title_box.set_margin_end(10)
        title_box.set_margin_top(20)
        title_box.set_margin_bottom(20)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>BigSkies</span>")
        title.set_halign(Gtk.Align.START)
        title_box.pack_start(title, False, False, 0)
        
        subtitle = Gtk.Label(label="Framework")
        subtitle.set_halign(Gtk.Align.START)
        title_box.pack_start(subtitle, False, False, 0)
        
        box.pack_start(title_box, False, False, 0)
        
        # Navigation buttons
        nav_buttons = [
            ("Health Status", "health"),
            ("Telescope Control", "control"),
            ("Telescope Preview", "preview"),
        ]
        
        first_button = None
        for label, page_name in nav_buttons:
            button = Gtk.RadioButton.new_with_label_from_widget(first_button, label)
            if first_button is None:
                first_button = button
                button.set_active(True)  # Select first button by default
            button.set_mode(False)  # Act like toggle button
            button.get_style_context().add_class("nav-button")
            button.set_halign(Gtk.Align.FILL)
            button.connect("toggled", self._on_nav_button_toggled, page_name)
            box.pack_start(button, False, False, 0)
            
        # Spacer
        box.pack_start(Gtk.Box(), True, True, 0)
        
        # About button at bottom
        about_button = Gtk.Button(label="About")
        about_button.get_style_context().add_class("nav-button")
        about_button.connect("clicked", self._on_about_clicked)
        box.pack_end(about_button, False, False, 0)
        
        return box
        
    def _on_nav_button_toggled(self, button, page_name):
        """Handle navigation button toggle."""
        if button.get_active():
            self.content_stack.set_visible_child_name(page_name)
            
    def _on_mqtt_connected(self):
        """Handle MQTT connection."""
        logger.info("MQTT connected")
        if self.mqtt_status_label:
            self.mqtt_status_label.set_markup("<span color='green'>● Connected</span>")
            
    def _on_mqtt_disconnected(self):
        """Handle MQTT disconnection."""
        logger.warning("MQTT disconnected")
        if self.mqtt_status_label:
            self.mqtt_status_label.set_markup("<span color='red'>● Disconnected</span>")
            
    def _on_about_clicked(self, button):
        """Show about dialog."""
        dialog = Gtk.AboutDialog()
        dialog.set_transient_for(self.window)
        dialog.set_modal(True)
        dialog.set_program_name("BigSkies Framework")
        dialog.set_version("1.0.0")
        dialog.set_comments("Plugin-extensible backend framework for telescope operations")
        dialog.set_website("https://github.com/unklstewy/BIG_SKIES_FRAMEWORK")
        dialog.set_authors(["BigSkies Team", "Co-Authored-By: Warp <agent@warp.dev>"])
        dialog.run()
        dialog.destroy()
        
    def _show_error(self, message):
        """Show error dialog."""
        dialog = Gtk.MessageDialog(
            transient_for=self.window,
            flags=0,
            message_type=Gtk.MessageType.ERROR,
            buttons=Gtk.ButtonsType.OK,
            text="Error"
        )
        dialog.format_secondary_text(message)
        dialog.run()
        dialog.destroy()
        
    def do_shutdown(self):
        """Application shutdown."""
        logger.info("BigSkies GTK Application shutting down...")
        if self.mqtt_client:
            self.mqtt_client.disconnect()
        Gtk.Application.do_shutdown(self)
