#!/usr/bin/env python3
"""
BigSkies GTK Application Entry Point
"""
import sys
import gi

gi.require_version('Gtk', '3.0')
from gi.repository import Gtk

from .app import BigSkiesApp


def main():
    """Main entry point."""
    app = BigSkiesApp()
    return app.run(sys.argv)


if __name__ == '__main__':
    sys.exit(main())
