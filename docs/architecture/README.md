# BIG SKIES Framework Architecture

This directory contains architecture diagrams and specifications for the BIG SKIES Framework.

## Files

### 1. big_skies_architecture_gojs.json
**System Architecture Overview**

GoJS-compatible JSON diagram representing the complete system architecture.

**Status**: Complete ✅ (All 7 backend coordinators implemented)

**Structure**:
- Frontend options (Flutter, Unity, Python GTK) - On hold pending backend completion
- Backend services with MQTT message bus architecture
- 7 coordinators with health check engines:
  1. **message-coordinator** - MQTT message bus management
  2. **application-svc-coordinator** - Microservice tracking and monitoring
  3. **security-coordinator** - Authentication, authorization, RBAC, TLS/mTLS
  4. **telescope-coordinator** - ASCOM Alpaca integration, telescope configs
  5. **ui-element-coordinator** - Plugin UI element registry
  6. **plugin-coordinator** - Plugin lifecycle management
  7. **data-store-coordinator** - PostgreSQL database management

Each coordinator includes:
- Health check engine (API, reporter, diagnostics)
- Specialized engines/services for domain logic
- Setup wizard APIs (where applicable)

---

### 2. mqtt_message_flows_gojs.json
**MQTT Message Flow Diagram**

**Status**: Complete ✅

**Content**:
- Complete mapping of all MQTT topics used by coordinators
- Publish/Subscribe relationships between components
- Topic naming patterns and conventions
- Message flow between coordinators and MQTT broker
- Health status publishing patterns

**Key Features**:
- Topic pattern: `bigskies/coordinator/{component}/{action}/{resource}`
- Response pattern: `bigskies/coordinator/{component}/response/{action}/{resource}/response`
- Health pattern: `bigskies/coordinator/{component}/health/status`
- Message format: JSON
- QoS: 1 (at least once delivery)

**Coordinators Covered**:
- Message Coordinator (health monitoring via wildcard subscription)
- Security Coordinator (auth, user, role, permission, certificate topics)
- DataStore Coordinator (query, backup, restore)
- Application Coordinator (service registration, heartbeat, status)
- Plugin Coordinator (install, remove, verify, list)
- UIElement Coordinator (register, unregister, list, filter)
- Telescope Coordinator (config, device, control, status, session management)

---

### 3. database_schema_gojs.json
**Database Schema Diagram**

**Status**: Complete ✅

**Content**:
- Complete PostgreSQL database schema
- 14 tables across 2 coordinators
- Table relationships and foreign keys
- Indexes for performance optimization

**Security Coordinator Tables** (9 tables):
1. `users` - User accounts with authentication credentials
2. `groups` - User groups for organizing permissions
3. `roles` - User roles defining sets of permissions
4. `permissions` - Individual permissions (resource + action + effect)
5. `user_groups` - Many-to-many: users ↔ groups
6. `user_roles` - Many-to-many: users ↔ roles
7. `group_permissions` - Many-to-many: groups ↔ permissions
8. `role_permissions` - Many-to-many: roles ↔ permissions
9. `tls_certificates` - TLS/SSL certificates with expiration tracking

**Telescope Coordinator Tables** (5 tables):
1. `observatory_sites` - Physical observatory locations with coordinates
2. `telescope_configurations` - Multi-tenant telescope configurations
3. `telescope_devices` - ASCOM Alpaca device assignments
4. `telescope_permissions` - Fine-grained per-telescope access control
5. `telescope_sessions` - Active and historical telescope usage sessions

**Features**:
- RBAC (Role-Based Access Control) implementation
- Multi-tenant ownership model
- Deny-first permission evaluation policy
- UUID primary keys throughout
- Comprehensive indexing strategy

---

### 4. docker_deployment_gojs.json
**Docker Deployment Architecture**

**Status**: Complete ✅

**Content**:
- Complete Docker Compose deployment architecture
- 9 containerized services (2 infrastructure + 7 coordinators)
- Network topology and container communication
- Volume mappings and persistent storage
- Service dependencies and startup order

**Infrastructure Services**:
1. **mqtt-broker** (Eclipse Mosquitto 2.0)
   - Ports: 1883 (MQTT), 9001 (WebSocket)
   - Volumes: mqtt-data, mqtt-logs, config

2. **postgres** (PostgreSQL 16 Alpine)
   - Port: 5432
   - Volumes: postgres-data, SQL schema files
   - Health check: pg_isready every 10s

**Coordinator Services**:
- All built from same Dockerfile (multi-stage)
- Alpine-based (~10-16MB each)
- Connected via bigskies-network bridge
- Restart policy: unless-stopped

**Startup Order**:
1. mqtt-broker, postgres (no dependencies)
2. message-coordinator (after mqtt-broker)
3. datastore-coordinator (after postgres healthy + mqtt-broker)
4. application-coordinator, plugin-coordinator (after mqtt-broker)
5. security-coordinator (after postgres healthy + mqtt-broker)
6. telescope-coordinator (after postgres healthy + mqtt-broker + security)
7. uielement-coordinator (after mqtt-broker + plugin-coordinator)

**Persistent Volumes**:
- mqtt-data, mqtt-logs, postgres-data, plugin-data, cert-data

---

### 5. security_auth_flows_gojs.json
**Security & Authentication Flow Diagram**

**Status**: Complete ✅

**Content**:
- Complete authentication flow (login)
- Authorization flow (RBAC permission checking)
- Token lifecycle (generation, validation, revocation)
- JWT token management

**Login Flow** (5 steps):
1. Client sends credentials to security coordinator
2. Authenticate user (query users table, verify bcrypt hash)
3. Generate JWT token (HMAC-SHA256, 24h expiration)
4. Return token + user info to client
5. Client stores token for subsequent requests

**Authorization Flow** (5 steps):
1. Client includes JWT in protected request
2. Validate token (signature, expiration, revocation)
3. Extract user_id from token claims
4. Check RBAC permissions (query roles, groups, permissions tables)
5. Apply deny-first policy, return allow/deny

**Logout Flow** (4 steps):
1. Client sends logout request with token
2. Security coordinator receives request
3. Add token to in-memory revocation list
4. Return success response

**Security Features**:
- JWT tokens signed with HMAC-SHA256
- Password hashing with bcrypt (cost 10)
- Token expiration (default 24 hours, configurable)
- In-memory token revocation for logout
- Deny-first RBAC evaluation
- Many-to-many: users ↔ groups ↔ permissions
- Many-to-many: users ↔ roles ↔ permissions

**Default Roles**:
- admin (all permissions)
- operator (telescope + plugin read/write/control)
- observer (read-only access)
- developer (plugin management)

## Viewing the Diagram

The GoJS diagram can be viewed using:
- [GoJS Diagram Editor](https://gojs.net/latest/samples/index.html)
- Any GoJS-compatible visualization tool
- Import into GoJS to create interactive diagrams

## Implementation Status

**Last Updated**: January 16, 2026

- ✅ All 7 backend coordinators implemented
- ✅ MQTT message bus operational
- ✅ PostgreSQL database integration complete
- ✅ Health check infrastructure in all coordinators
- ✅ Integration tests passing for all services
- ✅ Docker orchestration functional
- ⏸️ Frontend development on hold

## Future Architecture Additions

Potential future diagrams:
- Plugin architecture and SDK specifications
- ASCOM Alpaca device communication flows
- WebSocket real-time update patterns
- Frontend-to-backend integration architecture
- Distributed deployment architecture (Kubernetes)
- Monitoring and observability architecture
- Backup and disaster recovery flows

## Related Documentation

- Main README: `../../README.md`
- Implementation roadmap: `../../next_steps.txt`
- Coordinator docs: `../coordinators/`
- WARP development guide: `../../WARP.md`
