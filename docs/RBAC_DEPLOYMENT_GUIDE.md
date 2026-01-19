# BIG SKIES Framework - RBAC Deployment Guide

## Overview

The BIG SKIES Framework implements Role-Based Access Control (RBAC) at the message bus level through the message coordinator. This provides centralized authorization for all coordinator communications, ensuring that only authorized users can perform sensitive operations.

## Architecture

### RBAC Components

1. **Message Coordinator**: Intercepts and validates messages for protected topics
2. **Security Coordinator**: Provides permission validation via MQTT
3. **Database**: Stores protection rules and user/role/permission hierarchies
4. **Feature Flag**: `rbac_enabled` configuration for gradual rollout

### Message Flow with RBAC

```
Client Message → Message Coordinator → [RBAC Check] → Security Coordinator
                                                        ↓
Response: ALLOW → Forward to Target Coordinator → Process Message
Response: DENY → Reject Message → Log Security Event
```

## Deployment Configuration

### Database Schema

The RBAC system requires the following database schemas to be initialized:

- `security_schema.sql` - User, role, and permission tables
- `message_rbac_schema.sql` - Topic protection rules
- `coordinator_config_schema.sql` - Configuration settings

These are automatically loaded in Docker Compose via PostgreSQL init scripts.

### Environment Variables

```bash
# Enable RBAC validation (default: false for gradual rollout)
RBAC_ENABLED=true

# Standard BIG SKIES environment
LOG_LEVEL=info
POSTGRES_PASSWORD=your_secure_password
JWT_SECRET=your_jwt_secret
```

### Docker Compose

The `docker-compose.yml` includes RBAC configuration:

```yaml
message-coordinator:
  environment:
    - LOG_LEVEL=${LOG_LEVEL:-info}
    - RBAC_ENABLED=${RBAC_ENABLED:-false}
```

## Configuration

### Protection Rules

RBAC protection rules are stored in the `topic_protection_rules` table:

```sql
-- Example protection rules
INSERT INTO topic_protection_rules (id, topic_pattern, resource, action, enabled) VALUES
    (gen_random_uuid(), 'bigskies/coordinator/telescope/+/slew', 'telescope', 'control', true),
    (gen_random_uuid(), 'bigskies/coordinator/security/+/user/create', 'user', 'manage', true);
```

### Coordinator Configuration

RBAC settings are configured via the `coordinator_config` table:

```sql
-- Message coordinator RBAC settings
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type) VALUES
    ('message-coordinator', 'rbac_enabled', 'true', 'bool'),
    ('message-coordinator', 'max_queue_size', '1000', 'int'),
    ('message-coordinator', 'validation_timeout', '30', 'int');
```

## Gradual Rollout Strategy

### Phase 1: RBAC Disabled (Default)
- `RBAC_ENABLED=false`
- All messages flow through normally
- Protection rules loaded but not enforced
- Full backward compatibility

### Phase 2: RBAC Monitoring
- `RBAC_ENABLED=true`
- RBAC validation active
- Audit logging enabled
- Monitor for authorization failures

### Phase 3: RBAC Enforcement
- Protection rules activated
- Unauthorized messages rejected
- Security monitoring active

## Monitoring and Health Checks

### Health Check Endpoints

The message coordinator provides RBAC-specific health metrics:

```json
{
  "component": "message-coordinator",
  "status": "healthy",
  "details": {
    "rbac_enabled": true,
    "pending_validations": 0,
    "protection_rules": 7,
    "validation_errors": 0
  }
}
```

### Key Metrics

- **Pending Validations**: Messages awaiting RBAC validation
- **Validation Errors**: Failed permission checks
- **Queue Depth**: Current validation queue size
- **Response Time**: Average validation latency

### Logging

RBAC events are logged with structured data:

```json
{
  "level": "info",
  "msg": "RBAC validation passed, forwarding message",
  "correlation_id": "rbac-1234567890",
  "topic": "bigskies/coordinator/telescope/control/slew",
  "user_id": "user123",
  "permission": "telescope.control"
}
```

## Troubleshooting

### Common Issues

#### RBAC Validation Timeout
**Symptoms**: Messages delayed or rejected with timeout errors
**Cause**: Security coordinator unavailable or overloaded
**Solution**: Check security coordinator health, increase timeout, monitor queue depth

#### High Queue Depth
**Symptoms**: `pending_validations` > 100 in health check
**Cause**: High message volume or slow validation responses
**Solution**: Scale security coordinator, optimize validation logic

#### Authorization Failures
**Symptoms**: Legitimate messages rejected
**Cause**: Incorrect protection rules or user permissions
**Solution**: Review protection rules, verify user roles, check audit logs

### Rollback Procedure

To disable RBAC immediately:

1. Update configuration: `rbac_enabled = false`
2. Restart message coordinator
3. Verify messages flow normally
4. Monitor for authorization bypasses

### Debug Commands

```bash
# Check message coordinator health
curl http://localhost:8080/health/message

# View RBAC audit logs
docker logs bigskies-message-coordinator | grep rbac-audit

# Check protection rules
docker exec bigskies-postgres psql -U bigskies -d bigskies -c "SELECT * FROM topic_protection_rules;"
```

## Security Considerations

### Authorization Bypass Prevention

- All coordinator-to-coordinator messages are intercepted
- User context extracted from message envelopes
- JWT tokens validated before permission checks
- Failed validations logged with full context

### Performance Impact

- **Unprotected Topics**: Zero overhead
- **Protected Topics**: ~50-100ms validation latency
- **Memory Usage**: Minimal queue management
- **Database Load**: Cached protection rules

### Audit Trail

- All authorization decisions logged
- Correlation IDs for request tracing
- User context and permission details
- Rejection reasons for failed validations

## Testing

### Unit Tests

RBAC functionality includes comprehensive unit tests:

```bash
# Run RBAC-specific tests
go test ./internal/coordinators -run TestMessageCoordinator_RBAC

# Run all coordinator tests
make test
```

### Integration Tests

End-to-end RBAC validation:

```bash
# Run integration tests
make test-integration

# Test with RBAC enabled
RBAC_ENABLED=true make docker-up
```

### Load Testing

Performance validation under load:

```bash
# Load test RBAC validation
go test -bench=BenchmarkRBAC ./test/integration
```

## API Reference

### MQTT Topics

#### Validation Request
- **Topic**: `bigskies/coordinator/security/rbac/validate`
- **Payload**:
```json
{
  "message_id": "uuid",
  "topic": "bigskies/coordinator/telescope/control/slew",
  "permission": "telescope.control",
  "user_id": "user123",
  "groups": ["admin", "operator"],
  "timestamp": "2026-01-19T12:00:00Z"
}
```

#### Validation Response
- **Topic**: `bigskies/coordinator/security/rbac/response`
- **Payload**:
```json
{
  "message_id": "uuid",
  "allowed": true,
  "reason": "permission_granted",
  "timestamp": "2026-01-19T12:00:01Z"
}
```

## Support

For RBAC deployment issues:

1. Check coordinator health endpoints
2. Review audit logs for authorization failures
3. Verify protection rules configuration
4. Ensure security coordinator is operational
5. Check database connectivity and schema

See `docs/architecture/RBAC_IMPLEMENTATION_PLAN.md` for detailed implementation documentation.