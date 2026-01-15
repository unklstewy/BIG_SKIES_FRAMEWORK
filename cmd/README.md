# cmd/

This directory contains the main application entry points for the BIG SKIES Framework.

Each coordinator service will have its own main package here:
- `cmd/message-coordinator/` - Message bus coordinator
- `cmd/security-coordinator/` - Security and authentication coordinator
- `cmd/data-store-coordinator/` - Database management coordinator
- `cmd/application-svc-coordinator/` - Application service coordinator
- `cmd/plugin-coordinator/` - Plugin lifecycle coordinator
- `cmd/telescope-coordinator/` - Telescope and ASCOM coordinator
- `cmd/ui-element-coordinator/` - UI element coordinator
