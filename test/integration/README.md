# Integration Tests

This directory contains integration tests for the BIG SKIES Framework that test complete workflows across multiple services.

## Prerequisites

Before running integration tests, ensure all required services are running:

```bash
# Start all services via docker-compose
cd deployments/docker-compose
docker compose up -d

# Verify services are healthy
docker ps --filter "name=bigskies-" --format "table {{.Names}}\t{{.Status}}"
```

All coordinator services and infrastructure (MQTT, PostgreSQL) must be healthy before running tests.

## Running Tests

### Run All Integration Tests

```bash
# From project root
make test-integration

# Or directly with go test
go test ./test/integration/... -v
```

### Run Specific Test

```bash
# Run only authentication tests
go test ./test/integration/ -run TestAuthenticationFlow -v

# Run only login failure tests
go test ./test/integration/ -run TestLoginFailure -v
```

### Skip Integration Tests (Unit Tests Only)

```bash
go test ./... -short
```

## Test Suites

### Authentication Integration Tests (`auth_integration_test.go`)

#### TestAuthenticationFlow
Tests the complete authentication lifecycle:
1. **Login** - Authenticate with username/password
2. **Validate Token** - Verify token is valid
3. **Logout** - Revoke the token
4. **Validate Revoked Token** - Verify token is now invalid

**Expected Results:**
- ✅ Login returns valid JWT token with user details
- ✅ Token validation succeeds for active tokens
- ✅ Logout successfully revokes token
- ✅ Revoked tokens fail validation with "token has been revoked" error

#### TestLoginFailure
Tests authentication rejection with invalid credentials:
- Invalid password
- Invalid username
- Empty credentials

**Expected Results:**
- ❌ All login attempts fail with appropriate error messages

#### TestTokenValidationInvalid
Tests validation of malformed/invalid tokens:
- Malformed JWT structure
- Empty token
- Invalid signature

**Expected Results:**
- ❌ All tokens fail validation with error messages

#### TestMultipleLogins
Tests concurrent session support:
- Multiple logins generate unique tokens
- Each token has unique JWT ID (jti)
- All tokens are independently valid

**Expected Results:**
- ✅ Each login generates a different token
- ✅ All tokens validate successfully
- ✅ Tokens are independent (revoking one doesn't affect others)

#### TestLogoutInvalidToken
Tests logout error handling:
- Attempting to logout with invalid token

**Expected Results:**
- ❌ Logout fails gracefully with error message

## Test Configuration

### Environment Variables

```bash
# MQTT Broker (default: tcp://localhost:1883)
export MQTT_BROKER="tcp://localhost:1883"

# Test timeout (default: 10 seconds)
export TEST_TIMEOUT="10s"
```

### Test Database

Tests use the same PostgreSQL database as the running services. The default admin user (`admin`/`bigskies_admin_2024`) is used for authentication tests.

## Writing New Integration Tests

### Test Structure

```go
func TestYourFeature(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
    defer cancel()

    // Setup MQTT client
    client := setupMQTTClient(t, "your-test-client-id")
    defer client.Disconnect(250)

    // Your test logic using publishAndWaitForResponse()
}
```

### Helper Functions

- `setupMQTTClient(t, clientID)` - Creates and connects MQTT client
- `publishAndWaitForResponse(t, ctx, client, publishTopic, responseTopic, request)` - Publishes request and waits for response

## Troubleshooting

### Tests Timeout

If tests timeout, check:
1. Services are running: `docker ps`
2. Services are healthy: Check status column
3. MQTT broker is accessible: `mosquitto_pub -h localhost -p 1883 -t test -m hello`
4. Coordinator logs: `docker logs bigskies-security-coordinator`

### Connection Refused

```
Error: Failed to connect to MQTT broker
```

**Solution:** Ensure MQTT broker is running:
```bash
docker ps --filter "name=bigskies-mqtt"
```

### Test Failures

1. **Check coordinator logs:**
   ```bash
   docker logs bigskies-security-coordinator --tail 50
   ```

2. **Verify database schema:**
   ```bash
   docker exec bigskies-postgres psql -U bigskies -d bigskies -c "\dt"
   ```

3. **Reset services if needed:**
   ```bash
   docker compose -f deployments/docker-compose/docker-compose.yml restart
   ```

## CI/CD Integration

Integration tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Start Services
  run: |
    docker compose -f deployments/docker-compose/docker-compose.yml up -d
    sleep 10  # Wait for services to be healthy

- name: Run Integration Tests
  run: make test-integration

- name: Stop Services
  run: docker compose -f deployments/docker-compose/docker-compose.yml down
```

## Test Coverage

Integration tests provide coverage for:
- ✅ End-to-end authentication flows
- ✅ MQTT message bus communication
- ✅ JWT token lifecycle management
- ✅ Token blacklist enforcement
- ✅ Error handling and edge cases
- ✅ Concurrent session management

For unit test coverage of individual components, see the respective package `*_test.go` files.
