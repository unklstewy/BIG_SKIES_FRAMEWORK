# Security Coordinator

The Security Coordinator manages authentication, authorization (RBAC), and TLS/SSL certificate management for the BIG SKIES Framework.

## Features

### 1. Application Security Engine
- JWT token generation and validation
- Configurable token expiration (default: 24 hours)
- Token refresh capability
- API key management (generation, validation, revocation)
- API key expiration tracking

### 2. Account Security Engine  
- User account management (CRUD operations)
- bcrypt password hashing
- Group and role management
- Permission assignment (resource + action + effect)
- RBAC policy evaluation with deny-first priority
- Database-backed persistence

### 3. TLS Security Engine
- Self-signed certificate generation for development
- Let's Encrypt ACME integration with autocert
- Certificate storage and caching
- Automatic renewal monitoring (30-day warning)
- Certificate expiration tracking

## Database Schema

The security coordinator uses 9 PostgreSQL tables:

- `users` - User accounts with credentials
- `groups` - User groups
- `roles` - Named permission sets
- `permissions` - Resource/action/effect tuples
- `user_groups` - User-to-group associations
- `user_roles` - User-to-role associations
- `group_permissions` - Group-to-permission associations
- `role_permissions` - Role-to-permission associations
- `tls_certificates` - SSL/TLS certificates

### Default Data

**Admin User:**
- Username: `admin`
- Password: `bigskies_admin_2024` (⚠️ **CHANGE IMMEDIATELY IN PRODUCTION**)
- Email: `admin@bigskies.local`

**Default Roles:**
- `admin` - Full system administrator
- `operator` - Telescope operator
- `observer` - Read-only observer
- `developer` - Plugin developer

**Default Groups:**
- `administrators`
- `operators`
- `observers`

**Default Permissions:**
- User management: read, write, delete
- Telescope operations: read, write, control
- Plugin management: read, write, install, delete
- Security management: read, write
- Certificate management: read, write

## Usage

### Command-Line Options

```bash
security-coordinator \
  --broker-host=localhost \
  --broker-port=1883 \
  --database-url="postgres://user:pass@localhost:5432/bigskies" \
  --jwt-secret="your-secret-key" \
  --token-duration=24h \
  --log-level=info \
  --acme-directory="https://acme-v02.api.letsencrypt.org/directory" \
  --acme-email="admin@example.com" \
  --acme-cache-dir="./certs"
```

### Environment Variables

- `MQTT_USERNAME` - MQTT broker username
- `MQTT_PASSWORD` - MQTT broker password
- `JWT_SECRET` - JWT signing secret (required)
- `POSTGRES_PASSWORD` - PostgreSQL password

### Initialize Database

Apply the schema to your PostgreSQL database:

```bash
psql -U bigskies -d bigskies -f configs/sql/security_schema.sql
```

## MQTT Topics

The security coordinator subscribes to and publishes on the following topics:

### Authentication
- `bigskies/coordinator/security/auth/login` - User login
- `bigskies/coordinator/security/auth/validate` - Token validation
- `bigskies/coordinator/security/response/auth/login/response` - Login response
- `bigskies/coordinator/security/response/auth/validate/response` - Validation response

### User Management
- `bigskies/coordinator/security/user/create` - Create user
- `bigskies/coordinator/security/user/update` - Update user
- `bigskies/coordinator/security/user/delete` - Delete user
- `bigskies/coordinator/security/response/user/*/response` - User operation responses

### Role & Permission Management
- `bigskies/coordinator/security/role/assign` - Assign role to user
- `bigskies/coordinator/security/permission/check` - Check user permission
- `bigskies/coordinator/security/response/role/assign/response` - Role assignment response
- `bigskies/coordinator/security/response/permission/check/response` - Permission check response

### Certificate Management
- `bigskies/coordinator/security/cert/request` - Request certificate (self-signed or Let's Encrypt)
- `bigskies/coordinator/security/cert/renew` - Renew certificate
- `bigskies/coordinator/security/response/cert/*/response` - Certificate operation responses

### Health Status
- `bigskies/coordinator/security/health/status` - Published every 30 seconds

## Message Examples

### Login Request
```json
{
  "username": "admin",
  "password": "bigskies_admin_2024"
}
```

### Login Response
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-01-16T12:00:00Z",
  "user": {
    "id": "a0000000-0000-0000-0000-000000000001",
    "username": "admin",
    "email": "admin@bigskies.local",
    "enabled": true
  }
}
```

### Permission Check Request
```json
{
  "user_id": "a0000000-0000-0000-0000-000000000001",
  "resource": "telescope",
  "action": "control"
}
```

### Permission Check Response
```json
{
  "allowed": true,
  "reason": ""
}
```

### Certificate Request
```json
{
  "domain": "telescope.example.com",
  "email": "admin@example.com",
  "type": "self-signed"
}
```

### Certificate Response
```json
{
  "success": true,
  "domain": "telescope.example.com",
  "expires_at": "2027-01-15T12:00:00Z"
}
```

## Security Considerations

1. **Change Default Password**: The default admin password MUST be changed immediately in production
2. **JWT Secret**: Use a strong, random JWT secret (minimum 32 characters)
3. **Database Credentials**: Store database credentials securely (environment variables, secrets manager)
4. **HTTPS Only**: Use TLS/SSL for all network communications in production
5. **Certificate Management**: For production, use Let's Encrypt or trusted CA certificates
6. **Password Policy**: Implement password complexity requirements for user accounts
7. **Audit Logging**: Consider adding audit logging for security-critical operations
8. **Rate Limiting**: Implement rate limiting for authentication attempts

## Health Checks

The security coordinator reports health status for all three engines:

- **Application Security Engine**: Reports active API keys and JWT configuration
- **Account Security Engine**: Reports enabled user count and database connectivity
- **TLS Security Engine**: Reports certificate counts and expiration warnings

Health status is published to MQTT every 30 seconds.

## Development

### Building
```bash
go build -o bin/security-coordinator ./cmd/security-coordinator
```

### Running Locally
```bash
./bin/security-coordinator \
  --jwt-secret="dev-secret-key-change-in-production" \
  --database-url="postgres://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable"
```

### Docker
```bash
docker-compose up security-coordinator
```

## Future Enhancements

- OAuth2/OpenID Connect integration
- Multi-factor authentication (MFA)
- Session management with Redis
- Password reset workflow
- User invitation system
- Audit log persistence
- Certificate management UI
- LDAP/Active Directory integration
- Rate limiting per user/IP
- Brute force protection
