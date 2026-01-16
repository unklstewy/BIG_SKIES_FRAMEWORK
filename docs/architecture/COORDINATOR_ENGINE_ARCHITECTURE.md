# BIG SKIES FRAMEWORK - Coordinator/Engine Architecture Guide

## Document Purpose
This document serves as a comprehensive guide to the coordinator/engine architecture of the BIG SKIES FRAMEWORK. It defines the roles, responsibilities, and separation of concerns for each coordinator and its associated engines. This guide is intended for developers (both human and AI agents) working on the framework.

**Last Updated:** 2026-01-16

---

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Core Concepts](#core-concepts)
3. [Base Infrastructure](#base-infrastructure)
4. [Coordinators](#coordinators)
5. [Engines](#engines)
6. [Separation of Concerns](#separation-of-concerns)
7. [Communication Patterns](#communication-patterns)
8. [Best Practices](#best-practices)

---

## Architecture Overview

The BIG SKIES FRAMEWORK follows a **coordinator-based microservices architecture** where:

- **Coordinators** are high-level orchestrators that manage specific domains
- **Engines** are specialized components that handle specific technical capabilities
- **MQTT Message Bus** provides asynchronous communication with JSON interchange format
- **PostgreSQL** serves as the central data store (including configuration)
- **Health Check System** monitors all components
- **Docker Containers** provide isolation for plugins and supporting services

### Key Principles

1. **Domain Separation**: Each coordinator owns a specific domain (security, telescope, plugins, etc.)
2. **Engine Delegation**: Complex technical operations are delegated to specialized engines
3. **Loose Coupling**: Components communicate via MQTT, minimizing direct dependencies
4. **Health Transparency**: Every component exposes health status for monitoring
5. **Database-Driven Configuration**: Configuration is stored in PostgreSQL for runtime flexibility
6. **Multi-Tenant Architecture**: Support for users, groups, and organizations with RBAC

---

## Core Concepts

### Coordinator
A **coordinator** is a top-level service that:
- Orchestrates operations within a specific domain
- Subscribes to relevant MQTT topics and routes messages to handlers
- Owns one or more specialized **engines**
- Manages its own lifecycle (start/stop)
- Publishes health status periodically
- Loads configuration from database at startup and runtime

### Engine
An **engine** is a specialized component that:
- Handles specific technical capabilities (e.g., ASCOM protocol, JWT tokens, TLS)
- Is owned and managed by a coordinator
- Implements domain-specific business logic
- Reports health status to its parent coordinator
- May manage its own state and resources
- Does NOT directly subscribe to MQTT topics (coordinator handles this)

### Separation Pattern
```
MQTT Topic → Coordinator → Engine → Business Logic
   ↓              ↓            ↓
Subscribe      Route       Execute
               Handle      Manage State
               Validate    Report Health
```

---

## Base Infrastructure

### BaseCoordinator (`internal/coordinators/base.go`)

**Purpose**: Provides common functionality shared by all coordinators.

**Responsibilities**:
- Lifecycle management (Start/Stop)
- MQTT client management
- Health check engine integration
- Health status publishing to MQTT
- Shutdown function registration
- Configuration loading (from database) and validation
- Logger management

**Key Components**:
- `name`: Unique coordinator identifier
- `mqttClient`: MQTT connection for pub/sub
- `healthEngine`: Health check aggregator
- `logger`: Structured logging (zap)
- `shutdownFuncs`: Cleanup functions executed on stop
- `running`: Runtime state tracking
- `startTime`: Uptime tracking

**Usage Pattern**:
```go
baseCoord := NewBaseCoordinator(name, mqttClient, logger)
coordinator := &MyCoordinator{
    BaseCoordinator: baseCoord,
    myEngine: myEngine,
}
coordinator.RegisterHealthCheck(myEngine)
coordinator.RegisterShutdownFunc(cleanupFunc)
```

### Health Check System (`pkg/healthcheck/`)

**Components**:
- `Engine`: Aggregates health checks from multiple components
- `Checker`: Interface that all components must implement
- `Result`: Health status with details
- `Status`: Healthy, Degraded, Unhealthy

**Health Status Flow**:
```
Component.Check() → Result → HealthEngine.CheckAll() → AggregatedResult → MQTT
```

### MQTT Communication (`pkg/mqtt/`)

**Topic Structure**:
```
bigskies/coordinator/{coordinator_name}/{action}/{resource}[/{detail}]
```

**Example Topics**:
- `bigskies/coordinator/security/auth/login`
- `bigskies/coordinator/telescope/device/discover`
- `bigskies/coordinator/health/message-coordinator`

**Message Envelope**:
```json
{
  "message_id": "uuid",
  "type": "command|event|status",
  "source": "coordinator:name",
  "timestamp": "ISO-8601",
  "payload": { ... }
}
```

---

## Coordinators

### 0. Bootstrap Coordinator (`cmd/bootstrap-coordinator/main.go`)

**Domain**: Credential management and database initialization

**Purpose**: Provides database credentials to all coordinators and ensures database schema is properly initialized before coordinator startup.

**Engines**: Uses bootstrap package (`internal/bootstrap/`)
- `credentials.go`: Loads credentials from `.pgpass` file
- `migrations.go`: Runs database schema migrations
- (Note: Process manager removed for container architecture)

**Responsibilities**:
- Load database credentials from shared volume (`.pgpass` file)
- Run database migrations to initialize/update schema
- Publish credentials to coordinators via MQTT (base64-encoded)
- Listen for credential requests and republish
- Stay running to support coordinator restarts

**Container Architecture**:
```
┌────────────────────────────────┐
│ Bootstrap Coordinator          │
│ (First to start)              │
│                               │
│ 1. Load /shared/secrets/.pgpass│
│ 2. Run migrations              │
│ 3. Publish credentials via MQTT│
│ 4. Periodic republish (30s)    │
└────────────────────────────────┘
         │
         │ MQTT Topics
         ├─► bigskies/coordinator/bootstrap/credentials (publish)
         └─◄ bigskies/coordinator/bootstrap/request (subscribe)
         ↓
┌────────────────────────────────┐
│ Other Coordinators             │
│                               │
│ 1. Subscribe to credentials   │
│ 2. Decode base64 path         │
│ 3. Load from shared volume    │
│ 4. Connect to database        │
│ 5. Load config from DB        │
└────────────────────────────────┘
```

**Shared Volume Pattern**:
- All coordinators mount shared volume at `/shared/secrets`
- Volume type: tmpfs (memory-only, no disk persistence)
- Permissions: 0600, uid=1000
- Contains: `.pgpass` file with PostgreSQL credentials

**MQTT Topics**:
- Publish: `bigskies/coordinator/bootstrap/credentials`
  - Payload: `{"pgpass_path": "<base64>", "version": "1.0"}`
  - Frequency: On startup, every 30 seconds, and on request
- Subscribe: `bigskies/coordinator/bootstrap/request`
  - Allows coordinators to request credentials if missed

**Startup Sequence**:
1. **Infrastructure**: PostgreSQL + MQTT broker start (healthchecks pass)
2. **Bootstrap**: Loads credentials, runs migrations, publishes to MQTT
3. **DataStore**: Waits for credentials, connects to database
4. **Security + Message**: Wait for credentials and datastore
5. **Application**: Waits for message and security
6. **Plugins/Telescope/UIElement**: Wait for application

**Command-Line Flags**:
- `--config`: Path to bootstrap.yaml (default: `configs/bootstrap.yaml`)
- `--pgpass`: Path to .pgpass file (default: `/shared/secrets/.pgpass`)
- `--log-level`: Logging level (default: `info`)
- `--validate`: Validate config and credentials only, don't run
- `--skip-migrations`: Skip database migrations (use with caution)
- `--publish-only`: Only publish credentials, skip migrations
- `--version`: Show version and exit

**Configuration** (File-based: `configs/bootstrap.yaml`):
```yaml
database:
  host: postgres
  port: 5432
  database: bigskies
  sslmode: disable
  max_connections: 10
  connection_timeout: 10s

mqtt:
  broker_url: tcp://mqtt-broker
  broker_port: 1883
  client_id: bootstrap-coordinator
  reconnect_interval: 5s
  max_reconnect_attempts: 10

migrations:
  directory: configs/sql
  migrations:
    - name: coordinator_config_schema
      file: coordinator_config_schema.sql
      version: "1.0.0"
    # ... additional migrations
```

**BaseCoordinator Integration**:
All coordinators inherit `WaitForCredentials()` method from BaseCoordinator:

```go
// In coordinator main.go
ctx := context.Background()
creds, err := baseCoord.WaitForCredentials(ctx, 30*time.Second)
if err != nil {
    log.Fatal("Failed to load credentials:", err)
}

dbURL := creds.ConnectionString()
db, err := pgxpool.New(ctx, dbURL)
```

**Fallback Behavior**:
- If credentials not received within timeout, coordinator fails and Docker retries
- Eventually succeeds when bootstrap coordinator is available
- For development, can bypass with `DATABASE_URL` environment variable

**Security Considerations**:
- Credentials encoded in base64 (minor obscurity, NOT encryption)
- `.pgpass` file stored in tmpfs (memory-only, no disk persistence)
- Production: Enable MQTT authentication and PostgreSQL SSL
- Credential rotation: Update `.pgpass`, restart bootstrap, all coordinators update

**Docker Compose Configuration**:
- Service name: `bootstrap-coordinator`
- Depends on: `postgres`, `mqtt-broker` (with health checks)
- Volumes: `shared_secrets:/shared/secrets:ro`, `configs:/app/configs:ro`
- Restart policy: `unless-stopped`

**Separation of Concerns**:
- ✅ Loads and distributes credentials
- ✅ Runs database migrations
- ✅ Publishes credentials via MQTT
- ✅ Handles bootstrap lifecycle
- ❌ Does NOT manage coordinators as processes (container architecture)
- ❌ Does NOT implement business logic
- ❌ Does NOT connect to database after migrations (only for migrations)

**See Also**: `docs/setup/BOOTSTRAP_SETUP.md` for detailed setup instructions

---

### 1. Message Coordinator (`message_coordinator.go`)

**Domain**: Message bus infrastructure and health monitoring

**Purpose**: Manages the MQTT message bus, monitors all coordinator health, and provides message routing diagnostics.

**Engines**: None (handles message bus directly)

**Responsibilities**:
- Subscribe to all coordinator health topics
- Aggregate and monitor system-wide health
- Provide message bus diagnostics
- Track message flow and patterns
- Report MQTT broker connection status

**Key Operations**:
- Subscribe to: `bigskies/coordinator/*/health`
- Publish health to: `bigskies/coordinator/health/message-coordinator`
- Store and aggregate health data (TODO in current implementation)

**Configuration** (Database-stored):
- MQTT broker connection parameters
- Health monitoring intervals
- Reconnection policies
- Message retention policies

**Separation of Concerns**:
- ✅ Handles MQTT subscription/publishing
- ✅ Routes health messages
- ✅ Aggregates health status
- ❌ Does NOT implement security logic
- ❌ Does NOT manage application services

---

### 2. Application Coordinator (`application_coordinator.go`)

**Domain**: Application microservice tracking and monitoring

**Purpose**: Maintains a registry of all microservices in the system, tracks their health, and manages service lifecycle.

**Engines**: None (uses ServiceRegistry data structure)

**Responsibilities**:
- Register/unregister microservices
- Track service health via heartbeats
- Monitor service timeouts
- Provide service discovery
- Report on service availability

**Key Data Structure - ServiceRegistry**:
```go
type ServiceEntry struct {
    ID            string
    Name          string
    Status        healthcheck.Status
    Endpoint      string
    RegisteredAt  time.Time
    LastHeartbeat time.Time
    Metadata      map[string]interface{}
}
```

**MQTT Topics**:
- Subscribe: `bigskies/coordinator/service/event/register`
- Subscribe: `bigskies/coordinator/service/event/heartbeat`
- Publish: Health status

**Health Monitoring**:
- Periodically checks `LastHeartbeat` against `ServiceTimeout`
- Marks services as unhealthy if timeout exceeded
- Reports degraded status if any services are unhealthy

**Configuration** (Database-stored):
- Service timeout thresholds
- Registry check intervals
- Heartbeat requirements

**Separation of Concerns**:
- ✅ Tracks service registry
- ✅ Monitors service health
- ✅ Handles service registration
- ❌ Does NOT implement actual services
- ❌ Does NOT handle authentication/authorization
- ❌ Does NOT manage plugin lifecycle

---

### 3. Security Coordinator (`security_coordinator.go`)

**Domain**: Application security model, authentication, authorization, and TLS/SSL

**Purpose**: Orchestrates all security operations including user authentication, RBAC, and certificate management.

**Engines**:
1. **AppSecurityEngine**: Application-level security (JWT, API keys)
2. **AccountSecurityEngine**: User/group/role/permission management (RBAC)
3. **TLSSecurityEngine**: TLS/SSL certificate management (Let's Encrypt)

**Responsibilities**:
- Route authentication/authorization requests to appropriate engines
- Coordinate security operations across engines
- Manage security-related MQTT topics
- Aggregate security component health
- Interface with database for security data

**MQTT Topics** (Subscribed):
- `bigskies/coordinator/security/auth/*` (login, logout, validate)
- `bigskies/coordinator/security/user/*` (create, update, delete)
- `bigskies/coordinator/security/role/*` (assign)
- `bigskies/coordinator/security/permission/*` (check)
- `bigskies/coordinator/security/cert/*` (request, renew)

**Message Routing Pattern**:
```
MQTT Message → handleMessage() → route by topic → handler method → engine method → response
```

**Configuration** (Database-stored):
- JWT secret key and token duration
- Password policies (complexity, expiration)
- Session timeout policies
- TLS/SSL preferences
- ACME/Let's Encrypt settings
- Multi-factor authentication settings

**Separation of Concerns**:
- ✅ Routes security requests
- ✅ Publishes security responses
- ✅ Coordinates between engines
- ❌ Does NOT implement crypto/hash algorithms (engines do)
- ❌ Does NOT directly query database (engines do)
- ❌ Does NOT manage user sessions (app security engine does)

---

### 4. Telescope Coordinator (`telescope_coordinator.go`)

**Domain**: Telescope configuration management and ASCOM device operations

**Purpose**: Manages multi-tenant telescope configurations, ASCOM device discovery/control, and observing sessions.

**Engines**:
1. **ASCOM Engine**: ASCOM Alpaca protocol and device management

**Responsibilities**:
- CRUD operations for telescope configurations
- Device discovery and connection management
- Telescope control operations (slew, park, track, etc.)
- Session management (start/end observing sessions)
- RBAC integration for telescope access
- Database persistence of configurations

**MQTT Topics** (Subscribed):
- `bigskies/coordinator/telescope/config/*` (create, update, delete, list, get)
- `bigskies/coordinator/telescope/device/*` (discover, connect, disconnect)
- `bigskies/coordinator/telescope/control/*` (slew, park, unpark, track, abort)
- `bigskies/coordinator/telescope/status/get`
- `bigskies/coordinator/telescope/session/*` (start, end)

**Database Schema Integration**:
- `telescope_configurations`: Main configuration table
- `telescope_sessions`: Observing session tracking
- `telescope_permissions`: Multi-tenant access control
- `observatory_sites`: Site location data

**Multi-Tenant Features**:
- Owner-based access (user or group)
- Permission-based sharing
- Site association for multiple observatories

**Configuration** (Database-stored):
- Telescope-specific settings (mount type, capabilities, limits)
- Device endpoint configurations
- ASCOM Alpaca discovery settings
- Observatory site coordinates and time zones
- Session defaults and templates

**Separation of Concerns**:
- ✅ Manages telescope configurations
- ✅ Routes device commands to ASCOM engine
- ✅ Handles sessions and permissions
- ✅ Database CRUD operations
- ❌ Does NOT implement ASCOM protocol (engine does)
- ❌ Does NOT directly communicate with devices (engine does)
- ❌ Does NOT handle authentication (security coordinator does)

---

### 5. Plugin Coordinator (`plugin_coordinator.go`)

**Domain**: Plugin lifecycle management

**Purpose**: Manages plugin installation, versioning, updates, and removal. Tracks plugins by GUID.

**Engines**: None (uses PluginRegistry data structure)

**Responsibilities**:
- Install/remove plugins
- Verify plugin integrity
- Track plugin versions
- Manage plugin state (installed, running, stopped, failed)
- Scan for plugin updates
- Handle Docker container lifecycle for plugins
- Integrate with UI Element Coordinator for plugin UI

**Key Data Structure - PluginRegistry**:
```go
type PluginEntry struct {
    GUID         string          // Unique plugin identifier
    Name         string
    Version      string
    Status       PluginStatus    // installed, running, stopped, failed, updating
    InstalledAt  time.Time
    LastVerified time.Time
    ContainerID  string          // Docker container ID
    Metadata     map[string]interface{}
}
```

**MQTT Topics**:
- Subscribe: `bigskies/coordinator/plugin/command/install`
- Subscribe: `bigskies/coordinator/plugin/command/remove`
- Publish: Health status

**Plugin Lifecycle States**:
1. **Installed**: Plugin installed but not running
2. **Running**: Plugin container is active
3. **Stopped**: Plugin explicitly stopped
4. **Failed**: Plugin encountered an error
5. **Updating**: Plugin update in progress

**Configuration** (Database-stored):
- Plugin directory paths
- Plugin scan intervals
- Docker configuration (network, volumes, resources)
- Plugin repository sources
- Update policies (auto-update, manual, etc.)
- Plugin-specific settings

**Separation of Concerns**:
- ✅ Manages plugin registry
- ✅ Handles plugin lifecycle
- ✅ Docker container management
- ✅ Version tracking
- ❌ Does NOT implement plugin logic
- ❌ Does NOT parse plugin UI APIs (UI element coordinator does)
- ❌ Does NOT handle security (security coordinator does)

---

### 6. Data Store Coordinator (`datastore_coordinator.go`)

**Domain**: PostgreSQL database connection management

**Purpose**: Manages the PostgreSQL connection pool, monitors database health, and provides connection access to other coordinators.

**Engines**: None (manages pgxpool directly)

**Responsibilities**:
- Initialize and manage connection pool
- Monitor connection pool health
- Provide database connections to coordinators
- Handle connection timeouts and retries
- Report database statistics

**Connection Pool Configuration** (Database-stored after bootstrap):
- Max/min connections
- Connection lifetime settings
- Idle timeout policies
- Connection retry policies

**Health Monitoring**:
- Database ping checks
- Connection pool statistics
- Capacity warnings (near max connections)

**Configuration Bootstrap**:
- Initial connection string from environment variable or config file
- Once connected, reads configuration from database
- Supports runtime configuration updates

**Separation of Concerns**:
- ✅ Manages connection pool
- ✅ Provides database access
- ✅ Monitors connection health
- ❌ Does NOT implement business logic
- ❌ Does NOT define database schema
- ❌ Does NOT perform queries (coordinators/engines do)

**Note**: Other coordinators create their own database pools as needed (security, telescope). This coordinator could evolve to provide centralized pool management.

---

### 7. UI Element Coordinator (`uielement_coordinator.go`)

**Domain**: UI element tracking and multi-framework UI provisioning

**Purpose**: Maintains a registry of UI elements provided by plugins, supports multiple UI frameworks (GTK, Flutter, Unity, Qt, WPF, MFC, Blazor), and provides framework-specific widget mappings.

**Engines**: None (uses UIElementRegistry with framework mappings)

**Responsibilities**:
- Register/unregister UI elements from plugins
- Maintain framework-specific widget mappings
- Provide UI element discovery by framework
- Scan plugin APIs for UI definitions
- Enable dynamic UI generation for any supported framework
- Serve UI definitions via MQTT

**Key Data Structure - UIElement**:
```go
type UIElement struct {
    ID                string
    PluginGUID        string
    Type              UIElementType  // menu, panel, widget, tool, dialog
    Title             string
    APIEndpoint       string
    Order             int
    Enabled           bool
    RegisteredAt      time.Time
    Metadata          map[string]interface{}
    FrameworkMappings map[UIFramework]*WidgetMapping
}
```

**Supported Frameworks**:
- **GTK** (Python GTK): Gtk.Frame, Gtk.Button, Gtk.Grid, etc.
- **Flutter**: Card, Column, ElevatedButton, etc.
- **Unity**: Canvas, Panel, Button, etc.
- **Qt**: QGroupBox, QPushButton, QGridLayout, etc.
- **WPF**: GroupBox, Button, StackPanel, etc.
- **MFC**: CDialog, CButton, CStatic, etc.
- **Blazor**: Component mappings

**Widget Mapping Structure**:
```go
type WidgetMapping struct {
    WidgetType  string                      // Framework-specific type
    Layout      string                      // Layout strategy
    Properties  map[string]interface{}      // Widget properties
    Children    []WidgetDefinition          // Child widgets
    DataBinding *DataBinding                // Data source binding
    Actions     map[string]ActionDefinition // Event handlers
}
```

**Data Binding**:
- Property-to-source mapping
- MQTT topic subscription for updates
- Polling or event-driven updates
- Transform expressions

**MQTT Topics**:
- Subscribe: `bigskies/coordinator/uielement/event/register`
- Subscribe: `bigskies/coordinator/uielement/event/unregister`
- Subscribe: `bigskies/coordinator/uielement/command/query`
- Subscribe: `bigskies/coordinator/uielement/command/mapping/*`
- Publish: Query responses with framework-specific mappings

**Frontend Integration Pattern**:
```
Frontend (GTK/Flutter/Unity/etc.)
    ↓
Query uielement-coordinator via MQTT
    ↓
Receive framework-specific mappings
    ↓
Render native widgets dynamically
    ↓
Bind data via MQTT subscriptions
    ↓
Handle actions via MQTT publish
```

**Configuration** (Database-stored):
- UI scan intervals
- Framework preferences and priorities
- Theme and styling defaults
- Layout templates
- Widget library versions

**Separation of Concerns**:
- ✅ Maintains UI element registry
- ✅ Provides framework-agnostic element storage
- ✅ Maps elements to framework-specific widgets
- ✅ Enables dynamic UI generation
- ❌ Does NOT implement UI rendering (frontends do)
- ❌ Does NOT handle plugin installation (plugin coordinator does)
- ❌ Does NOT authenticate requests (security coordinator does)
- ❌ Does NOT implement data sources (other coordinators do)

**See Also**: `docs/ui/BLAZOR_TO_GTK_MAPPING.md` for detailed framework mapping examples

---

## Engines

### 1. AppSecurityEngine (`internal/engines/security/app_security.go`)

**Domain**: Application-level security primitives

**Purpose**: Manages JWT tokens and API keys for application authentication.

**Owned By**: Security Coordinator

**Responsibilities**:
- Generate JWT tokens with claims
- Validate JWT tokens
- Revoke tokens (blacklist)
- Generate API keys
- Validate API keys
- Manage API key lifecycle
- Cleanup expired blacklisted tokens

**Key Features**:
- HMAC-SHA256 signing for JWT
- Token blacklist for revoked tokens
- API key generation with expiry
- Token refresh capability
- In-memory storage for performance

**Configuration** (Loaded from database via coordinator):
- JWT secret key
- Token duration
- Signing algorithm preferences
- API key expiry defaults

**Health Monitoring**:
- Active API key count
- Expired API key count
- JWT configuration status

**Separation of Concerns**:
- ✅ Implements JWT logic
- ✅ Manages token/key storage
- ✅ Handles cryptographic operations
- ❌ Does NOT authenticate users (account security engine does)
- ❌ Does NOT check permissions (account security engine does)
- ❌ Does NOT subscribe to MQTT (coordinator does)

---

### 2. AccountSecurityEngine (`internal/engines/security/account_security.go`)

**Domain**: User, group, role, and permission management (RBAC)

**Purpose**: Implements role-based access control with user/group/role/permission hierarchies.

**Owned By**: Security Coordinator

**Responsibilities**:
- Create/update/delete users
- Hash and verify passwords (bcrypt)
- Authenticate users
- Create/manage roles
- Create/manage groups
- Assign roles to users
- Assign users to groups
- Create/manage permissions
- Evaluate permission checks (RBAC)

**Database Schema Integration**:
- `users`: User accounts
- `roles`: Role definitions
- `groups`: Group definitions
- `permissions`: Permission definitions
- `user_roles`: User-role assignments
- `user_groups`: User-group assignments
- `role_permissions`: Role-permission assignments
- `group_permissions`: Group-permission assignments

**Permission Evaluation Logic**:
1. Query permissions from user's roles
2. Query permissions from user's groups
3. Apply deny-first policy (deny overrides allow)
4. Return decision

**Security Best Practices**:
- Bcrypt with default cost for password hashing
- Soft delete (disable) rather than hard delete
- Timestamp tracking for audit trail

**Configuration** (Loaded from database via coordinator):
- Password complexity requirements
- Password expiration policies
- Account lockout policies
- Session timeout settings

**Separation of Concerns**:
- ✅ Implements RBAC logic
- ✅ Database operations for accounts
- ✅ Password cryptography
- ❌ Does NOT generate JWT tokens (app security engine does)
- ❌ Does NOT manage TLS (TLS security engine does)
- ❌ Does NOT handle MQTT (coordinator does)

---

### 3. TLSSecurityEngine (`internal/engines/security/tls_security.go`)

**Domain**: TLS/SSL certificate management and Let's Encrypt integration

**Purpose**: Generates, requests, stores, and renews TLS certificates for the framework.

**Owned By**: Security Coordinator

**Responsibilities**:
- Generate self-signed certificates for development
- Request Let's Encrypt certificates via ACME
- Store certificates in database
- Monitor certificate expiry
- Automatic renewal for expiring certificates
- Provide certificates for domain lookup
- Manage ACME client and autocert manager

**Certificate Types**:
1. **Self-Signed**: For development/testing, generated locally
2. **Let's Encrypt**: Production certificates via ACME protocol

**Database Schema Integration**:
- `tls_certificates`: Certificate storage (PEM format)

**Certificate Structure**:
```go
type TLSCertificate struct {
    ID             string
    Domain         string
    CertificatePEM string    // PEM-encoded certificate
    PrivateKeyPEM  string    // PEM-encoded private key
    ExpiresAt      time.Time
    Issuer         string    // "self-signed" or "letsencrypt"
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**Renewal Monitoring**:
- Periodic check for certificates expiring within 30 days
- Automatic renewal for Let's Encrypt certificates
- Health status degrades if certificates are expiring

**ACME Integration**:
- Uses `golang.org/x/crypto/acme/autocert`
- Supports Let's Encrypt production and staging
- Directory cache for certificate persistence

**Configuration** (Loaded from database via coordinator):
- ACME directory URL (production/staging)
- Contact email for Let's Encrypt
- Allowed domains
- Cache directory paths
- Renewal thresholds

**Separation of Concerns**:
- ✅ Implements TLS operations
- ✅ ACME protocol handling
- ✅ Certificate cryptography
- ✅ Database persistence
- ❌ Does NOT authenticate users (account security engine does)
- ❌ Does NOT generate tokens (app security engine does)
- ❌ Does NOT handle MQTT (coordinator does)

---

### 4. ASCOM Engine (`internal/engines/ascom/engine.go`)

**Domain**: ASCOM Alpaca protocol and device lifecycle management

**Purpose**: Manages ASCOM device discovery, connection pooling, health monitoring, and coordinated operations.

**Owned By**: Telescope Coordinator

**Responsibilities**:
- Discover ASCOM devices on network
- Register/unregister devices
- Connect/disconnect devices
- Track device connection state
- Perform health checks on connected devices
- Manage telescope device pools (multi-device configs)
- Provide ASCOM client for device operations

**Key Data Structures**:

```go
type managedDevice struct {
    device      *models.AlpacaDevice
    connected   bool
    lastHealthy time.Time
    failCount   int
}

type telescopePool struct {
    telescopeID string
    devices     map[string]*managedDevice  // role -> device
}
```

**Device Roles in Telescope Pool**:
- `telescope`: Mount/OTA
- `camera`: Imaging camera
- `dome`: Observatory dome
- `focuser`: Focuser
- `filterwheel`: Filter wheel
- `rotator`: Field rotator

**Health Check Strategy**:
- Periodic health checks on connected devices
- Track consecutive failure count
- Mark disconnected after 3 failures
- Report health with device statistics

**ASCOM Client Integration** (`internal/engines/ascom/client.go`):
- HTTP-based ASCOM Alpaca REST API
- Device discovery via UDP broadcast
- Commands: Connect, Disconnect, Slew, Park, Track, etc.
- Status queries: Position, Tracking, Slewing, etc.

**Configuration** (Loaded from database via coordinator):
- Discovery port (default 32227)
- Health check intervals
- Timeout thresholds
- Retry policies

**Separation of Concerns**:
- ✅ Implements ASCOM protocol
- ✅ Manages device connections
- ✅ Health monitoring
- ✅ Device state tracking
- ❌ Does NOT manage telescope configurations (coordinator does)
- ❌ Does NOT handle database (coordinator does)
- ❌ Does NOT authenticate (security coordinator does)
- ❌ Does NOT subscribe to MQTT (coordinator does)

---

## Separation of Concerns

### Coordinator Responsibilities

**What Coordinators DO**:
- Subscribe to MQTT topics for their domain
- Route incoming messages to handler methods
- Validate request payloads
- Call engine methods to execute operations
- Publish responses to MQTT
- Manage engine lifecycle (start/stop)
- Aggregate health from engines
- Publish health status
- Load configuration from database
- Interface with database for domain data
- Handle shutdown cleanup

**What Coordinators DO NOT DO**:
- Implement complex technical operations (engines do)
- Directly implement cryptography/protocols (engines do)
- Manage state of technical systems (engines do)
- Perform data transformations (engines do)

### Engine Responsibilities

**What Engines DO**:
- Implement specific technical capabilities
- Manage their own state and resources
- Perform complex operations (crypto, protocols, etc.)
- Interface with external systems (databases, APIs, etc.)
- Report health status
- Provide clear interfaces for coordinators
- Handle technical error conditions

**What Engines DO NOT DO**:
- Subscribe to MQTT topics (coordinators do)
- Publish to MQTT directly (coordinators do)
- Route messages (coordinators do)
- Make domain-level decisions (coordinators do)
- Manage other engines (coordinators do)

### Boundary Examples

| Scenario | Coordinator | Engine |
|----------|-------------|--------|
| JWT Authentication Request | Receives MQTT message, validates payload, calls engine | Generates JWT token with claims |
| Password Check | Receives login request, calls engine with credentials | Hashes password, compares with stored hash |
| Certificate Generation | Receives cert request, calls engine with domain | Generates self-signed cert, stores in DB |
| Device Discovery | Receives discovery request, calls engine | Performs UDP broadcast, returns devices |
| User Creation | Receives create request, calls engine, publishes response | Hashes password, inserts into DB |
| Telescope Slew | Receives slew command, calls engine with coordinates | Sends ASCOM REST API command to device |

---

## Communication Patterns

### 1. Command Pattern (Synchronous-style)

**Flow**: Client → Command Topic → Coordinator → Engine → Response Topic → Client

**Example**: Login request
```
Client publishes: bigskies/coordinator/security/auth/login
  {"username": "user", "password": "pass"}
  
Coordinator receives, routes to handleLogin()
  → Calls engine.AuthenticateUser()
  → Calls engine.GenerateToken()
  
Coordinator publishes: bigskies/coordinator/security/response/auth/login/response
  {"success": true, "token": "...", "expires_at": "..."}
```

### 2. Event Pattern (Fire-and-forget)

**Flow**: Source → Event Topic → Coordinator(s) → Action

**Example**: Service heartbeat
```
Service publishes: bigskies/coordinator/service/event/heartbeat
  {"id": "svc-123", "status": "healthy"}
  
Application Coordinator receives, updates LastHeartbeat
  → No response published
```

### 3. Status Pattern (Periodic broadcast)

**Flow**: Coordinator → Health Engine → Status Topic

**Example**: Health publishing
```
Every 30 seconds:
  Coordinator calls HealthCheck()
  Wraps in MQTT message
  Publishes: bigskies/coordinator/health/telescope-coordinator
    {"status": "healthy", "message": "...", "details": {...}}
```

### 4. Request-Response Pattern (Correlated)

**Flow**: Client → Request Topic (with correlation ID) → Response Topic (with correlation ID) → Client

**Example**: List telescope configs
```
Client publishes: bigskies/coordinator/telescope/config/list
  {"message_id": "req-456", "user_id": "user-123"}
  
Coordinator publishes: bigskies/coordinator/telescope/response/config/list/response
  {"message_id": "req-456", "success": true, "configs": [...]}
```

---

## Best Practices

### For Coordinators

1. **Single Domain Focus**: Each coordinator should manage one cohesive domain
2. **Engine Delegation**: Delegate technical operations to engines
3. **Clear Boundaries**: Don't implement logic that belongs in an engine
4. **MQTT Ownership**: Coordinators own MQTT subscriptions, not engines
5. **Health Aggregation**: Register all engines with health check system
6. **Graceful Shutdown**: Register cleanup functions for proper shutdown
7. **Error Handling**: Validate inputs, handle engine errors, publish error responses
8. **Logging**: Use structured logging with context (zap)
9. **Database-Driven Config**: Load configuration from database for runtime flexibility

### For Engines

1. **Technical Focus**: Implement specific technical capabilities
2. **Stateless When Possible**: Minimize state, or manage it carefully
3. **Clear Interfaces**: Provide simple, clear methods for coordinators
4. **Health Reporting**: Implement healthcheck.Checker interface
5. **Error Context**: Return errors with sufficient context
6. **Resource Management**: Clean up resources properly (connections, files, etc.)
7. **No MQTT**: Never subscribe to or publish MQTT directly
8. **Database Transactions**: Use proper transaction boundaries

### For Both

1. **Logging**: Use structured logging consistently
2. **Configuration**: Support database-driven configuration with validation
3. **Testing**: Write unit tests for business logic
4. **Documentation**: Comment all public interfaces
5. **Health Checks**: Provide meaningful health status
6. **Thread Safety**: Use mutexes for shared state
7. **Context Propagation**: Pass context.Context for cancellation
8. **Error Wrapping**: Use `fmt.Errorf("...: %w", err)` for error chains

### Configuration Management

**Database-First Approach**:
- Store configuration in PostgreSQL tables
- Load configuration at startup and on-demand
- Support runtime configuration updates via MQTT
- Validate configuration before applying
- Maintain configuration history for audit

**Bootstrap Configuration**:
- Minimal startup config (database connection only)
- Database connection string from command-line flag or environment variable
- Once connected, all coordinator config loaded from database

**Configuration Schema**:
See `configs/sql/coordinator_config_schema.sql` for complete schema with:
- `coordinator_config`: Configuration key-value storage with JSONB values
- `coordinator_config_history`: Automatic change tracking via trigger
- Type hints: string, int, bool, float, duration, object
- Secret flagging for sensitive values (passwords, keys)
- User attribution for configuration changes

**Configuration Loader Utility**:
See `internal/config/loader.go` for the configuration loading infrastructure:
- `Loader`: Database-backed configuration loader
- `CoordinatorConfig`: Type-safe configuration accessor with defaults
- `GetString()`, `GetInt()`, `GetBool()`, `GetFloat()`, `GetDuration()`, `GetObject()`: Typed getters
- `UpdateConfigValue()`, `InsertConfigValue()`, `DeleteConfigValue()`: Config management
- `GetConfigHistory()`: Configuration change audit trail

**Runtime Configuration Updates**:
Coordinators subscribe to `bigskies/coordinator/config/update/{coordinator_name}` to receive configuration change notifications. When a config update message is received:
1. Coordinator reloads configuration from database using `ConfigLoader`
2. Parses and validates new configuration values
3. Applies configuration changes (thread-safe with mutex)
4. Logs configuration change with old and new values

**Migration Pattern**:
To migrate a coordinator from struct-based to database-driven configuration:

1. **Database Schema**: Ensure `coordinator_config_schema.sql` is applied
2. **Inject Config Loader**: Add `configLoader *config.Loader` field to coordinator struct
3. **Main Entry Point**: Update `cmd/{coordinator}/main.go`:
```go
// Connect to database
dbPool, err := pgxpool.New(ctx, *databaseURL)
if err != nil {
    logger.Fatal("Failed to connect to database", zap.Error(err))
}
defer dbPool.Close()

// Create config loader
configLoader := config.NewLoader(dbPool)

// Load configuration from database
coordConfig, err := configLoader.LoadCoordinatorConfig(ctx, "my-coordinator")
if err != nil {
    logger.Fatal("Failed to load configuration", zap.Error(err))
}

// Parse config values with defaults
brokerURL, _ := coordConfig.GetString("broker_url", "localhost")
brokerPort, _ := coordConfig.GetInt("broker_port", 1883)

// Create coordinator with config
coordinator, err := coordinators.NewMyCoordinator(cfg, logger)
if err != nil {
    logger.Fatal("Failed to create coordinator", zap.Error(err))
}

// Inject config loader for runtime updates
coordinator.SetConfigLoader(configLoader)
```

4. **Config Update Handler**: Add MQTT subscription handler in coordinator:
```go
// Subscribe to config updates
func (c *MyCoordinator) subscribeConfigTopic() error {
    topic := "bigskies/coordinator/config/update/my-coordinator"
    return c.subscribe(topic, c.handleConfigUpdate)
}

// Handle config update messages
func (c *MyCoordinator) handleConfigUpdate(topic string, payload []byte) error {
    // Reload config from database
    coordConfig, err := c.configLoader.LoadCoordinatorConfig(ctx, "my-coordinator")
    if err != nil {
        return err
    }
    
    // Parse updated values
    newValue, _ := coordConfig.GetInt("some_key", defaultValue)
    
    // Apply configuration (thread-safe)
    c.mu.Lock()
    c.config.SomeField = newValue
    c.mu.Unlock()
    
    c.logger.Info("Configuration reloaded", zap.Int("new_value", newValue))
    return nil
}
```

5. **SetConfigLoader Method**: Add method to inject config loader:
```go
func (c *MyCoordinator) SetConfigLoader(loader *config.Loader) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.configLoader = loader
}
```

**Completed Migrations**:
1. **Message Coordinator** ✅
   - Schema: Default config values in `coordinator_config_schema.sql`
   - Main: `cmd/message-coordinator/main.go` loads config from database
   - Coordinator: `SetConfigLoader()` and `handleConfigUpdate()` methods
   - Test: `scripts/test-config-migration.sh` validates migration

2. **Application Coordinator** ✅
   - Main: `cmd/application-coordinator/main.go` migrated
   - Coordinator: Runtime config reload implemented
   - Config keys: `broker_url`, `broker_port`, `registry_check_interval`, `service_timeout`

3. **Plugin Coordinator** ✅
   - Main: `cmd/plugin-coordinator/main.go` migrated
   - Config keys: `broker_url`, `broker_port`, `plugin_dir`, `scan_interval`
   - Note: Coordinator runtime handler needs implementation

4. **UIElement Coordinator** ✅
   - Main: `cmd/uielement-coordinator/main.go` migrated
   - Coordinator: Runtime config reload implemented
   - Config keys: `broker_url`, `broker_port`, `scan_interval`

5. **Plugin Coordinator** ✅
   - Main: `cmd/plugin-coordinator/main.go` migrated
   - Coordinator: Runtime config reload implemented
   - Config keys: `broker_url`, `broker_port`, `plugin_dir`, `scan_interval`

6. **Telescope Coordinator** ✅
   - Main: `cmd/telescope-coordinator/main.go` migrated
   - Coordinator: Runtime config reload implemented
   - Config keys: `broker_url`, `broker_port`, `discovery_port`, `health_check_interval`
   - Note: Database URL handled as bootstrap parameter

**Remaining Coordinators to Migrate**:
- Security Coordinator (requires database URL bootstrap handling)
- DataStore Coordinator (requires database URL bootstrap handling)

### Architecture Evolution

When extending the framework:

**Adding a New Coordinator**:
1. Embed `BaseCoordinator`
2. Define configuration schema in database
3. Implement MQTT topic subscriptions
4. Create message handlers
5. Integrate engines (if needed)
6. Register health checks
7. Add to `cmd/` with main entry point

**Adding a New Engine**:
1. Define engine struct with dependencies
2. Implement specific technical capability
3. Implement `healthcheck.Checker` interface
4. Provide clear public methods
5. Add to appropriate coordinator
6. Register with coordinator's health check system

**Extending Existing Functionality**:
1. Determine if it's coordinator or engine responsibility
2. Add MQTT topic if new message type needed
3. Add handler method to coordinator
4. Add engine method if technical operation needed
5. Update health checks if appropriate
6. Add database schema changes if needed
7. Add tests

### Anti-Patterns to Avoid

❌ **Engine subscribing to MQTT**: Engines should never interact with MQTT directly

❌ **Coordinator implementing crypto**: Technical operations belong in engines

❌ **Circular dependencies**: Coordinators should not depend on each other directly (use MQTT)

❌ **God coordinator**: Don't create a coordinator that does everything

❌ **Stateless engine with no purpose**: If an engine has no state or logic, it might not be needed

❌ **Direct database access from coordinators**: Use engines for database operations when appropriate

❌ **Missing health checks**: Every component must report health

❌ **Hardcoded configuration**: Store configuration in database for runtime flexibility

❌ **File-based configuration**: Avoid config files except for bootstrap connection parameters

---

## Appendix: Component Inventory

### Coordinators
1. Message Coordinator (`message_coordinator.go`)
2. Application Coordinator (`application_coordinator.go`)
3. Security Coordinator (`security_coordinator.go`)
4. Telescope Coordinator (`telescope_coordinator.go`)
5. Plugin Coordinator (`plugin_coordinator.go`)
6. Data Store Coordinator (`datastore_coordinator.go`)
7. UI Element Coordinator (`uielement_coordinator.go`)

### Engines
1. AppSecurityEngine (`internal/engines/security/app_security.go`)
2. AccountSecurityEngine (`internal/engines/security/account_security.go`)
3. TLSSecurityEngine (`internal/engines/security/tls_security.go`)
4. ASCOM Engine (`internal/engines/ascom/engine.go`)

### Base Infrastructure
1. BaseCoordinator (`internal/coordinators/base.go`)
2. Health Check Engine (`pkg/healthcheck/engine.go`)
3. MQTT Client (`pkg/mqtt/client.go`)

### Entry Points (Main Programs)
- `cmd/message-coordinator/main.go`
- `cmd/application-coordinator/main.go`
- `cmd/security-coordinator/main.go`
- `cmd/telescope-coordinator/main.go`
- `cmd/plugin-coordinator/main.go`
- `cmd/datastore-coordinator/main.go`
- `cmd/uielement-coordinator/main.go`

---

## Document Maintenance

This document should be updated whenever:
- A new coordinator is added
- A new engine is created
- Architecture patterns change
- Separation of concerns is refined
- Communication patterns are modified
- Configuration strategies evolve

**Maintainers**: All developers working on BIG_SKIES_FRAMEWORK

**Review Cycle**: Updated with each significant architectural change

---

**End of Document**
