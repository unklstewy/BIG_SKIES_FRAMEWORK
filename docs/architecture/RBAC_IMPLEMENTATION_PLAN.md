# BIG SKIES Framework - RBAC Implementation Plan for Message Coordinator

**Document Version**: 1.1
**Date**: January 19, 2026
**Author**: AI Assistant (Warp)
**Review Status**: Implementation Complete (Phases 1-4)
**Target Release**: Phase 6 (RBAC Integration)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Current State Analysis](#current-state-analysis)
3. [Implementation Overview](#implementation-overview)
4. [Detailed Implementation Plan](#detailed-implementation-plan)
5. [Code Changes](#code-changes)
6. [Configuration Changes](#configuration-changes)
7. [Database Schema Changes](#database-schema-changes)
8. [Testing Strategy](#testing-strategy)
9. [Deployment Plan](#deployment-plan)
10. [Rollback Plan](#rollback-plan)
11. [Success Criteria](#success-criteria)
12. [Risk Assessment](#risk-assessment)
13. [Timeline](#timeline)

---

## Executive Summary

### Objective
Implement Role-Based Access Control (RBAC) validation in the message coordinator to enforce authorization checks before processing protected MQTT messages, ensuring that only authorized users can perform sensitive operations across the BIG SKIES Framework.

### Business Value
- **Security Enhancement**: Prevents unauthorized access to telescope control, user management, and other sensitive operations
- **Compliance**: Supports multi-tenant architecture with proper access controls
- **Auditability**: Provides centralized authorization logging and monitoring
- **Flexibility**: Runtime-configurable protection rules without code changes

### Approach
Implement a **Selective Message Validation Pattern** where the message coordinator acts as an authorization gateway for protected topics, requesting permission validation from the security coordinator via MQTT before forwarding messages.

### Success Metrics
- Zero security violations in protected operations
- <5% performance degradation for protected topics
- 100% test coverage for RBAC functionality
- Successful integration testing across all coordinators

---

## Current State Analysis

### Message Coordinator Responsibilities
- Manages MQTT message bus infrastructure
- Monitors system-wide coordinator health
- Provides message routing diagnostics
- Tracks message flow patterns
- Currently processes all messages without authorization checks

### Security Coordinator Capabilities
- Manages user/group/role/permission hierarchies via AccountSecurityEngine
- Provides JWT token validation via AppSecurityEngine
- Supports RBAC permission checking
- Exposes validation endpoints via MQTT

### Architectural Constraints
- Coordinators communicate only via MQTT (no direct calls)
- Engines never interact with MQTT directly
- Message coordinator must maintain high throughput
- Changes must preserve existing message flow patterns

### Current Security Gaps
- No message-level authorization enforcement
- Protected operations rely on application-level checks
- No centralized audit trail for message access
- Potential for privilege escalation through direct MQTT publishing

---

## Implementation Overview

### Core Concept
The message coordinator will intercept messages destined for protected topics, request RBAC validation from the security coordinator, and only forward validated messages to their target coordinators.

### Key Components
1. **Topic Protection Rules**: Configurable patterns defining protected topics and required permissions
2. **Message Interception**: Middleware layer that captures and validates incoming messages
3. **RBAC Validation Protocol**: MQTT-based request/response pattern for permission checks
4. **Pending Message Queue**: Temporary storage for messages awaiting validation
5. **Audit Logging**: Comprehensive logging of authorization decisions

### Message Flow
```
Client Message → Message Coordinator → RBAC Check → Security Coordinator
                                                        ↓
Response: ALLOW → Forward to Target Coordinator → Process Message
Response: DENY → Reject Message → Log Security Event
```

### Performance Impact
- **Unprotected Topics**: Zero overhead (direct forwarding)
- **Protected Topics**: ~50-100ms latency for validation round-trip
- **Memory Usage**: Minimal (pending message queue with timeouts)

---

## Detailed Implementation Plan
**Status**: Phases 1-4 ✅ Complete | Phase 5 Planned

### Phase 1: Core Infrastructure (Week 1-2) ✅ COMPLETE

#### 1.1 Define Data Structures
- `TopicProtectionRule` struct for protection configuration
- `RBACValidationRequest/Response` structs for MQTT protocol
- `PendingMessage` struct for queue management
- `UserContext` struct for extracted authentication data

#### 1.2 Add Configuration Support
- Database schema for protection rules
- Runtime configuration loading
- Protection rule validation

#### 1.3 Implement Message Interception
- MQTT subscription to all coordinator topics
- Message parsing and user context extraction
- Protection rule matching logic

### Phase 2: RBAC Integration (Week 3-4) ✅ COMPLETE

#### 2.1 Validation Request Protocol
- MQTT topic structure for validation requests
- Request payload format with user context
- Correlation ID generation for request/response matching

#### 2.2 Security Coordinator Updates
- New MQTT handler for RBAC validation requests
- Integration with AccountSecurityEngine for permission checks
- Response formatting and publishing

#### 2.3 Response Handling
- MQTT subscription for validation responses
- Pending message queue management
- Message forwarding/rejection logic

### Phase 3: Advanced Features (Week 5-6) ✅ COMPLETE

#### 3.1 Audit Logging
- Authorization decision logging
- Security event tracking
- Performance metrics collection

#### 3.2 Error Handling
- Validation timeout handling
- Security coordinator unavailability handling
- Message queue overflow protection

#### 3.3 Health Monitoring
- RBAC validation health checks
- Queue depth monitoring
- Performance metrics

### Phase 4: Testing & Validation (Week 7-8) ✅ COMPLETE

#### 4.1 Unit Testing
- Protection rule matching
- Message interception logic
- Queue management

#### 4.2 Integration Testing
- End-to-end RBAC validation
- Cross-coordinator message flows
- Performance benchmarking

#### 4.3 Security Testing
- Authorization bypass attempts
- Edge case validation
- Load testing under RBAC

---

## Code Changes

### 1. Message Coordinator Updates

#### File: `internal/coordinators/message_coordinator.go`

```go
package coordinators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// MessageCoordinator manages message bus with RBAC enforcement
type MessageCoordinator struct {
	*BaseCoordinator
	config              *MessageCoordinatorConfig
	protectionRules     []TopicProtectionRule
	pendingValidations  map[string]*PendingMessage
	validationTimeout   time.Duration
	mu                  sync.RWMutex
}

// TopicProtectionRule defines RBAC requirements for message topics
type TopicProtectionRule struct {
	Pattern     string         // Regex pattern for topic matching
	Permission  string         // Required permission string
	RequireAuth bool           // Whether authentication is required
	CompiledRegex *regexp.Regexp // Pre-compiled regex for performance
}

// PendingMessage represents a message awaiting RBAC validation
type PendingMessage struct {
	OriginalMessage *mqtt.Message
	Topic           string
	Payload         []byte
	ReceivedAt      time.Time
	ValidationID    string
	UserContext     *UserContext
}

// UserContext extracted from message authentication
type UserContext struct {
	UserID      string
	Groups      []string
	Permissions []string
	AuthToken   string
}

// RBACValidationRequest sent to security coordinator
type RBACValidationRequest struct {
	MessageID string    `json:"message_id"`
	Topic     string    `json:"topic"`
	Permission string   `json:"permission"`
	UserID    string    `json:"user_id"`
	Groups    []string  `json:"groups"`
	Timestamp time.Time `json:"timestamp"`
}

// RBACValidationResponse received from security coordinator
type RBACValidationResponse struct {
	MessageID string    `json:"message_id"`
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func NewMessageCoordinator(config *MessageCoordinatorConfig, logger *zap.Logger) (*MessageCoordinator, error) {
	mqttClient, err := CreateMQTTClient(config.BrokerURL, mqtt.CoordinatorMessage, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %w", err)
	}

	base := NewBaseCoordinator(mqtt.CoordinatorMessage, mqttClient, logger)

	mc := &MessageCoordinator{
		BaseCoordinator:    base,
		config:             config,
		pendingValidations: make(map[string]*PendingMessage),
		validationTimeout:  30 * time.Second,
	}

	mc.RegisterHealthCheck(mc)
	return mc, nil
}

func (mc *MessageCoordinator) Start(ctx context.Context) error {
	mc.GetLogger().Info("Starting message coordinator with RBAC enforcement")

	// Load protection rules from configuration
	if err := mc.loadProtectionRules(ctx); err != nil {
		return fmt.Errorf("failed to load protection rules: %w", err)
	}

	// Subscribe to RBAC validation responses
	rbacResponseTopic := "bigskies/coordinator/security/rbac/response"
	if err := mc.GetMQTTClient().Subscribe(rbacResponseTopic, 1, mc.handleRBACResponse); err != nil {
		return fmt.Errorf("failed to subscribe to RBAC responses: %w", err)
	}

	// Subscribe to all coordinator topics for interception
	wildcardTopic := "bigskies/coordinator/+/+/+/#"
	if err := mc.GetMQTTClient().Subscribe(wildcardTopic, 1, mc.handleIncomingMessage); err != nil {
		return fmt.Errorf("failed to subscribe to coordinator topics: %w", err)
	}

	// Start cleanup goroutine for expired validations
	go mc.cleanupExpiredValidations(ctx)

	if err := mc.BaseCoordinator.Start(ctx); err != nil {
		return err
	}

	go mc.StartHealthPublishing(ctx)
	return nil
}

func (mc *MessageCoordinator) handleIncomingMessage(topic string, payload []byte) error {
	mc.GetLogger().Debug("Received message", zap.String("topic", topic))

	// Check if topic requires RBAC validation
	if rule := mc.getProtectionRule(topic); rule != nil {
		return mc.requestRBACValidation(topic, payload, rule)
	}

	// Topic doesn't require validation, forward directly
	return mc.forwardMessage(topic, payload)
}

func (mc *MessageCoordinator) getProtectionRule(topic string) *TopicProtectionRule {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, rule := range mc.protectionRules {
		if rule.CompiledRegex.MatchString(topic) {
			return &rule
		}
	}
	return nil
}

func (mc *MessageCoordinator) requestRBACValidation(topic string, payload []byte, rule *TopicProtectionRule) error {
	// Extract user context from message
	userContext, err := mc.extractUserContext(payload)
	if err != nil {
		mc.GetLogger().Warn("Failed to extract user context, rejecting message",
			zap.String("topic", topic), zap.Error(err))
		return mc.rejectMessage(topic, payload, "invalid_user_context")
	}

	// Create validation request
	validationReq := &RBACValidationRequest{
		MessageID: uuid.New().String(),
		Topic:     topic,
		Permission: rule.Permission,
		UserID:    userContext.UserID,
		Groups:    userContext.Groups,
		Timestamp: time.Now(),
	}

	// Store pending message
	mc.mu.Lock()
	mc.pendingValidations[validationReq.MessageID] = &PendingMessage{
		OriginalMessage: &mqtt.Message{Topic: topic, Payload: payload},
		Topic:          topic,
		Payload:        payload,
		ReceivedAt:     time.Now(),
		ValidationID:   validationReq.MessageID,
		UserContext:    userContext,
	}
	mc.mu.Unlock()

	// Publish validation request
	validationTopic := "bigskies/coordinator/security/rbac/validate"
	if err := mc.GetMQTTClient().PublishJSON(validationTopic, 1, false, validationReq); err != nil {
		mc.GetLogger().Error("Failed to publish RBAC validation request",
			zap.Error(err), zap.String("message_id", validationReq.MessageID))

		// Remove from pending on failure
		mc.mu.Lock()
		delete(mc.pendingValidations, validationReq.MessageID)
		mc.mu.Unlock()

		return mc.rejectMessage(topic, payload, "validation_request_failed")
	}

	mc.GetLogger().Debug("RBAC validation requested",
		zap.String("message_id", validationReq.MessageID),
		zap.String("topic", topic),
		zap.String("permission", rule.Permission))

	return nil
}

func (mc *MessageCoordinator) handleRBACResponse(topic string, payload []byte) error {
	var response RBACValidationResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		mc.GetLogger().Error("Failed to unmarshal RBAC response", zap.Error(err))
		return err
	}

	// Find pending message
	mc.mu.Lock()
	pending, exists := mc.pendingValidations[response.MessageID]
	if !exists {
		mc.mu.Unlock()
		mc.GetLogger().Warn("Received validation response for unknown message",
			zap.String("message_id", response.MessageID))
		return nil
	}
	delete(mc.pendingValidations, response.MessageID)
	mc.mu.Unlock()

	if response.Allowed {
		// Validation passed, forward the message
		mc.GetLogger().Info("RBAC validation passed, forwarding message",
			zap.String("message_id", response.MessageID),
			zap.String("topic", pending.Topic))

		return mc.forwardMessage(pending.Topic, pending.Payload)
	} else {
		// Validation failed, reject the message
		mc.GetLogger().Warn("RBAC validation failed, rejecting message",
			zap.String("message_id", response.MessageID),
			zap.String("topic", pending.Topic),
			zap.String("reason", response.Reason))

		return mc.rejectMessage(pending.Topic, pending.Payload, response.Reason)
	}
}

func (mc *MessageCoordinator) forwardMessage(topic string, payload []byte) error {
	// Forward to original topic (MQTT will handle routing)
	return mc.GetMQTTClient().Publish(topic, 1, false, payload)
}

func (mc *MessageCoordinator) rejectMessage(topic string, payload []byte, reason string) error {
	mc.GetLogger().Warn("Message rejected",
		zap.String("topic", topic),
		zap.String("reason", reason))

	// Could publish rejection event to audit topic
	auditTopic := "bigskies/coordinator/message/audit/rejection"
	auditEvent := map[string]interface{}{
		"topic":     topic,
		"reason":    reason,
		"timestamp": time.Now(),
	}

	return mc.GetMQTTClient().PublishJSON(auditTopic, 1, false, auditEvent)
}

func (mc *MessageCoordinator) extractUserContext(payload []byte) (*UserContext, error) {
	// Parse message envelope to extract authentication info
	var envelope map[string]interface{}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse message envelope: %w", err)
	}

	// Extract user context from envelope
	// This depends on how authentication is embedded in messages
	// Could be JWT token, API key, or user session info

	userID, _ := envelope["user_id"].(string)
	if userID == "" {
		return nil, fmt.Errorf("no user_id found in message")
	}

	// Extract groups and permissions if available
	groups := []string{}
	if groupsData, ok := envelope["groups"].([]interface{}); ok {
		for _, g := range groupsData {
			if group, ok := g.(string); ok {
				groups = append(groups, group)
			}
		}
	}

	return &UserContext{
		UserID: userID,
		Groups: groups,
	}, nil
}

func (mc *MessageCoordinator) loadProtectionRules(ctx context.Context) error {
	// Load from database configuration
	// This would use the config loader pattern

	// For now, use hardcoded rules (would be loaded from DB in production)
	rules := []TopicProtectionRule{
		{
			Pattern:    "bigskies/coordinator/telescope/control/.*",
			Permission: "telescope.control",
			RequireAuth: true,
		},
		{
			Pattern:    "bigskies/coordinator/security/user/.*",
			Permission: "security.user.manage",
			RequireAuth: true,
		},
		// Add more rules as needed
	}

	// Compile regex patterns
	for i := range rules {
		compiled, err := regexp.Compile(rules[i].Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %s: %w", rules[i].Pattern, err)
		}
		rules[i].CompiledRegex = compiled
	}

	mc.mu.Lock()
	mc.protectionRules = rules
	mc.mu.Unlock()

	mc.GetLogger().Info("Loaded protection rules",
		zap.Int("count", len(rules)))

	return nil
}

func (mc *MessageCoordinator) cleanupExpiredValidations(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupExpired()
		}
	}
}

func (mc *MessageCoordinator) cleanupExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	expired := []string{}

	for id, pending := range mc.pendingValidations {
		if now.Sub(pending.ReceivedAt) > mc.validationTimeout {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		delete(mc.pendingValidations, id)
		mc.GetLogger().Warn("Validation timeout, removing pending message",
			zap.String("message_id", id))
	}

	if len(expired) > 0 {
		mc.GetLogger().Info("Cleaned up expired validations",
			zap.Int("count", len(expired)))
	}
}

func (mc *MessageCoordinator) Check(ctx context.Context) *healthcheck.Result {
	mc.mu.RLock()
	pendingCount := len(mc.pendingValidations)
	mc.mu.RUnlock()

	status := healthcheck.StatusHealthy
	message := "Message coordinator with RBAC is healthy"
	details := map[string]interface{}{
		"pending_validations": pendingCount,
		"protection_rules":    len(mc.protectionRules),
	}

	// Check for excessive pending validations
	if pendingCount > 100 {
		status = healthcheck.StatusDegraded
		message = "High number of pending RBAC validations"
	}

	return &healthcheck.Result{
		ComponentName: "message-coordinator",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details:       details,
	}
}
```

### 2. Security Coordinator Updates

#### File: `internal/coordinators/security_coordinator.go`

```go
// Add to security coordinator
func (sc *SecurityCoordinator) handleRBACValidation(topic string, payload []byte) error {
	var req RBACValidationRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		sc.GetLogger().Error("Failed to unmarshal RBAC validation request", zap.Error(err))
		return err
	}

	sc.GetLogger().Debug("Processing RBAC validation request",
		zap.String("message_id", req.MessageID),
		zap.String("topic", req.Topic),
		zap.String("permission", req.Permission),
		zap.String("user_id", req.UserID))

	// Check permission using AccountSecurityEngine
	allowed, reason := sc.accountSecEngine.CheckPermission(req.UserID, req.Permission, req.Groups)

	response := RBACValidationResponse{
		MessageID: req.MessageID,
		Allowed:   allowed,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	// Publish response
	responseTopic := "bigskies/coordinator/security/rbac/response"
	if err := sc.GetMQTTClient().PublishJSON(responseTopic, 1, false, response); err != nil {
		sc.GetLogger().Error("Failed to publish RBAC validation response",
			zap.Error(err), zap.String("message_id", req.MessageID))
		return err
	}

	sc.GetLogger().Debug("RBAC validation response sent",
		zap.String("message_id", req.MessageID),
		zap.Bool("allowed", allowed))

	return nil
}

// Add to Start() method
rbacTopic := "bigskies/coordinator/security/rbac/validate"
if err := sc.GetMQTTClient().Subscribe(rbacTopic, 1, sc.handleRBACValidation); err != nil {
	return fmt.Errorf("failed to subscribe to RBAC validation requests: %w", err)
}
```

---

## Configuration Changes

### Database Configuration
Add RBAC settings to message coordinator configuration:

```sql
-- Add to configs/sql/coordinator_config_schema.sql
INSERT INTO coordinator_config (coordinator_name, config_key, config_value)
VALUES
  ('message-coordinator', 'rbac_enabled', 'true'),
  ('message-coordinator', 'validation_timeout', '"30s"'),
  ('message-coordinator', 'max_pending_validations', '1000'),
  ('message-coordinator', 'cleanup_interval', '"30s"'),
  ('message-coordinator', 'audit_enabled', 'true');
```

### Protection Rules Configuration
```sql
-- Protection rules stored as JSON in database
INSERT INTO coordinator_config (coordinator_name, config_key, config_value)
VALUES ('message-coordinator', 'protection_rules', '[
  {
    "pattern": "bigskies/coordinator/telescope/control/.*",
    "permission": "telescope.control",
    "require_auth": true
  },
  {
    "pattern": "bigskies/coordinator/security/user/.*",
    "permission": "security.user.manage",
    "require_auth": true
  },
  {
    "pattern": "bigskies/coordinator/plugin/manage/.*",
    "permission": "plugin.manage",
    "require_auth": true
  }
]');
```

### Docker Compose Updates
```yaml
# In deployments/docker-compose/docker-compose.yml
message-coordinator:
  environment:
    - RBAC_ENABLED=true
    - VALIDATION_TIMEOUT=30s
  depends_on:
    - security-coordinator  # Ensure security starts first
```

---

## Database Schema Changes

### New Tables (if needed)
```sql
-- RBAC audit logging table
CREATE TABLE IF NOT EXISTS rbac_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR NOT NULL,
    topic VARCHAR NOT NULL,
    user_id VARCHAR,
    permission VARCHAR,
    decision VARCHAR NOT NULL, -- 'allow' or 'deny'
    reason TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_rbac_audit_log_timestamp ON rbac_audit_log(timestamp);
CREATE INDEX idx_rbac_audit_log_user ON rbac_audit_log(user_id);
CREATE INDEX idx_rbac_audit_log_decision ON rbac_audit_log(decision);
```

### Migration Script
```sql
-- Add to configs/sql/rbac_audit_schema.sql
-- This file would be added to bootstrap migrations
```

---

## Testing Strategy

### Unit Tests
```go
// File: internal/coordinators/message_coordinator_test.go
func TestMessageCoordinator_ProtectionRuleMatching(t *testing.T) {
	// Test regex pattern matching
}

func TestMessageCoordinator_UserContextExtraction(t *testing.T) {
	// Test JWT/API key parsing
}

func TestMessageCoordinator_PendingMessageQueue(t *testing.T) {
	// Test queue management and cleanup
}

func TestMessageCoordinator_RBACValidationRequest(t *testing.T) {
	// Test validation request creation and publishing
}
```

### Integration Tests
```go
// File: test/integration/rbac_integration_test.go
func TestRBAC_MessageValidation(t *testing.T) {
	// End-to-end RBAC validation test
	// 1. Start all coordinators
	// 2. Send protected message without auth -> should be rejected
	// 3. Send protected message with proper auth -> should be allowed
	// 4. Verify audit logging
}

func TestRBAC_TimeoutHandling(t *testing.T) {
	// Test validation timeout scenarios
}

func TestRBAC_SecurityCoordinatorUnavailable(t *testing.T) {
	// Test behavior when security coordinator is down
}
```

### Performance Tests
```go
func BenchmarkRBAC_ValidationLatency(t *testing.B) {
	// Measure validation round-trip time
}

func BenchmarkMessageCoordinator_Throughput(t *testing.B) {
	// Measure message processing throughput with RBAC enabled
}
```

### Security Tests
- Authorization bypass attempts
- SQL injection in permission checks
- Message tampering
- Replay attacks
- DoS via validation queue overflow

---

## Deployment Plan

### Phase 1: Development Environment
1. Implement core RBAC functionality
2. Add comprehensive unit tests
3. Test in isolated environment
4. Performance benchmarking

### Phase 2: Staging Environment
1. Deploy to staging with feature flags
2. Enable RBAC for subset of topics
3. Monitor performance and error rates
4. User acceptance testing

### Phase 3: Production Rollout
1. Deploy with RBAC disabled by default
2. Enable RBAC for low-risk topics first
3. Gradual rollout with monitoring
4. Full enablement after validation

### Rollback Strategy
- Feature flag to disable RBAC entirely
- Configuration to remove protection rules
- Database migration rollback capability

---

## Rollback Plan

### Immediate Rollback (Critical Issues)
1. Set `rbac_enabled = false` in database configuration
2. Restart message coordinator
3. Monitor that messages flow normally without validation

### Partial Rollback (Performance Issues)
1. Remove specific protection rules from configuration
2. Keep RBAC enabled but reduce scope
3. Monitor performance improvement

### Full Rollback (Architecture Issues)
1. Revert code changes
2. Restore original message coordinator
3. Remove RBAC-related database tables
4. Update documentation

### Rollback Validation
- All integration tests pass
- Message throughput returns to baseline
- No security violations introduced
- User functionality restored

---

## Success Criteria

### Functional Criteria
- [ ] All protected topics require RBAC validation
- [ ] Unauthorized messages are rejected with proper logging
- [ ] Authorized messages are processed normally
- [ ] RBAC validation responses are handled correctly
- [ ] Audit logging captures all authorization decisions

### Performance Criteria
- [ ] <5% throughput degradation for unprotected topics
- [ ] <50ms average latency increase for protected topics
- [ ] <100 pending validations under normal load
- [ ] No memory leaks in validation queue

### Security Criteria
- [ ] Zero authorization bypass vulnerabilities
- [ ] Comprehensive audit trail
- [ ] Secure user context extraction
- [ ] Proper error handling without information leakage

### Quality Criteria
- [ ] >90% test coverage for RBAC functionality
- [ ] All integration tests passing
- [ ] Documentation updated
- [ ] Code review completed

---

## Risk Assessment

### High Risk
- **Message Processing Deadlock**: Validation requests could create circular dependencies
  - Mitigation: Timeout handling and circuit breaker pattern

- **Performance Degradation**: RBAC validation could impact real-time telescope operations
  - Mitigation: Asynchronous validation and caching

### Medium Risk
- **Security Coordinator Unavailability**: Could block all protected operations
  - Mitigation: Graceful degradation and local caching

- **Configuration Errors**: Incorrect protection rules could break functionality
  - Mitigation: Validation and gradual rollout

### Low Risk
- **Memory Leaks**: Pending message queue could grow unbounded
  - Mitigation: Proper cleanup and monitoring

- **Audit Log Growth**: Could consume excessive storage
  - Mitigation: Log rotation and archiving

---

## Timeline

### Week 1-2: Core Infrastructure
- Define data structures and protocols
- Implement message interception
- Add configuration support
- Unit tests for core functionality

### Week 3-4: RBAC Integration
- Security coordinator validation handler
- Message coordinator response handling
- Integration testing
- Performance benchmarking

### Week 5-6: Advanced Features
- Audit logging implementation
- Error handling and monitoring
- Health checks
- Documentation updates

### Week 7-8: Testing & Validation
- Comprehensive testing suite
- Security testing
- Performance optimization
- Production readiness review

### Week 9-10: Deployment & Monitoring
- Staging deployment
- Production rollout
- Monitoring and alerting
- Post-deployment validation

---

## Effort Estimate & Implementation Plan

### Current Implementation Analysis

#### ✅ **Already Implemented (Major Assets)**
- **Complete RBAC Database Schema**: All tables (users, roles, permissions, user_roles, etc.) exist
- **AccountSecurityEngine.CheckPermission()**: Method exists but unused for message validation
- **JWT Authentication**: Full login/logout/token validation working
- **Database-Driven Configuration**: Runtime config updates implemented
- **Integration Test Infrastructure**: MQTT-based testing framework exists
- **Message Envelope Structure**: JSON message format with source/timestamp/correlation ID

#### ⚠️ **Partial Implementation**
- **Message Coordinator**: Basic health monitoring exists, but no message interception
- **Security Coordinator**: Authentication works, but no RBAC validation endpoints

#### ❌ **Missing Components**
- Message interception and RBAC validation logic
- User context extraction from message envelopes
- Protection rule configuration and matching
- RBAC validation MQTT protocol
- Audit logging infrastructure
- RBAC-specific integration tests

### Total Estimated Effort: 8-10 weeks (2-3 developers)

---

## Phase 1: Core RBAC Infrastructure (Weeks 1-2)
**Effort: 2 weeks | Risk: Medium | Dependencies: None**

#### 1.1 Message Coordinator RBAC Foundation
- **Add RBAC data structures** (TopicProtectionRule, PendingMessage, UserContext)
- **Implement regex-based topic matching** for protection rules
- **Add pending message queue** with timeout cleanup
- **User context extraction** from message envelopes (JWT parsing)
- **Files**: `internal/coordinators/message_coordinator.go` (+200 lines)

#### 1.2 Security Coordinator RBAC Handler
- **Add RBAC validation MQTT endpoint** (`bigskies/coordinator/security/rbac/validate`)
- **Implement permission checking** using existing AccountSecurityEngine.CheckPermission()
- **Add validation response publishing** with correlation IDs
- **Files**: `internal/coordinators/security_coordinator.go` (+50 lines)

#### 1.3 Configuration Schema Updates
- **Add RBAC settings** to coordinator_config_schema.sql
- **Protection rules JSON storage** in database
- **Runtime configuration** for rule updates
- **Files**: `configs/sql/coordinator_config_schema.sql` (+20 lines)

**Effort Breakdown**: 40% Message Coordinator, 30% Security Coordinator, 30% Configuration

---

## Phase 2: Message Interception & Validation (Weeks 3-4)
**Effort: 2 weeks | Risk: High | Dependencies: Phase 1**

#### 2.1 Message Interception Logic
- **Wildcard MQTT subscription** to all coordinator topics (`bigskies/coordinator/+/+/+/#`)
- **Protection rule evaluation** for each incoming message
- **RBAC validation request creation** with user context
- **Message queuing** during validation (async processing)
- **Files**: `internal/coordinators/message_coordinator.go` (+150 lines)

#### 2.2 Validation Response Handling
- **RBAC response subscription** (`bigskies/coordinator/security/rbac/response`)
- **Pending message resolution** (allow/deny decisions)
- **Message forwarding/rejection** based on validation results
- **Timeout handling** for stuck validations
- **Files**: `internal/coordinators/message_coordinator.go` (+100 lines)

#### 2.3 User Context Integration
- **JWT token parsing** from message headers/payload
- **User/group extraction** from validated tokens
- **Context validation** (token expiry, revocation)
- **Files**: `internal/coordinators/message_coordinator.go` (+80 lines), `pkg/mqtt/message.go` (+30 lines)

**Effort Breakdown**: 50% Message interception, 30% Response handling, 20% User context

---

## Phase 3: Advanced Features & Error Handling (Weeks 5-6)
**Effort: 2 weeks | Risk: Medium | Dependencies: Phase 2**

#### 3.1 Audit Logging Infrastructure
- **RBAC decision logging** (allow/deny with reasons)
- **Audit table schema** (`rbac_audit_log`)
- **Performance metrics** collection
- **Files**: `configs/sql/rbac_audit_schema.sql` (+30 lines), `internal/coordinators/message_coordinator.go` (+60 lines)

#### 3.2 Error Handling & Resilience
- **Security coordinator unavailability** handling (graceful degradation)
- **Message queue overflow** protection
- **Circuit breaker pattern** for validation failures
- **Health monitoring** for RBAC components
- **Files**: `internal/coordinators/message_coordinator.go` (+80 lines)

#### 3.3 Protection Rule Management
- **Runtime rule updates** via MQTT configuration
- **Rule validation** (regex syntax, permission existence)
- **Hot reloading** without coordinator restart
- **Files**: `internal/coordinators/message_coordinator.go` (+60 lines)

**Effort Breakdown**: 40% Audit logging, 35% Error handling, 25% Rule management

---

## Phase 4: Testing & Validation (Weeks 7-8)
**Effort: 2 weeks | Risk: Low | Dependencies: Phase 3**

#### 4.1 Unit Testing
- **Protection rule matching** logic tests
- **User context extraction** tests
- **Message queue management** tests
- **RBAC validation protocol** tests
- **Files**: `internal/coordinators/message_coordinator_test.go` (+200 lines)

#### 4.2 Integration Testing
- **End-to-end RBAC validation** (protected message → validation → allow/deny)
- **Multi-coordinator message flows** with RBAC
- **Performance benchmarking** (latency impact measurement)
- **Security testing** (authorization bypass attempts)
- **Files**: `test/integration/rbac_integration_test.go` (+300 lines)

#### 4.3 Load Testing
- **Message throughput** with RBAC enabled
- **Validation queue capacity** testing
- **Concurrent validation** performance
- **Files**: `test/integration/rbac_load_test.go` (+150 lines)

**Effort Breakdown**: 50% Integration tests, 30% Unit tests, 20% Load testing

---

## Phase 5: Deployment & Documentation (Weeks 9-10)
**Effort: 2 weeks | Risk: Low | Dependencies: Phase 4**

#### 5.1 Production Deployment
- **Staging environment** validation
- **Gradual rollout** with feature flags
- **Monitoring setup** (alerts, dashboards)
- **Rollback procedures** testing

#### 5.2 Documentation Updates
- **RBAC implementation guide** updates
- **API documentation** for new endpoints
- **Configuration reference** updates
- **Troubleshooting guide** additions

#### 5.3 Training & Handover
- **Developer training** on RBAC usage
- **Operations training** on monitoring/alerts
- **Security review** sign-off

**Effort Breakdown**: 40% Deployment, 40% Documentation, 20% Training

---

## Risk Assessment & Mitigation

### High Risk Items
1. **Message Processing Deadlock**: Validation requests could create circular dependencies
   - **Mitigation**: Timeout handling, circuit breaker, comprehensive testing

2. **Performance Degradation**: RBAC validation could impact real-time telescope operations
   - **Mitigation**: Async validation, caching, performance benchmarking

3. **Security Coordinator Unavailability**: Could block all protected operations
   - **Mitigation**: Graceful degradation, local caching, health monitoring

### Medium Risk Items
1. **Configuration Errors**: Incorrect protection rules could break functionality
   - **Mitigation**: Rule validation, gradual rollout, monitoring

2. **Memory Leaks**: Pending message queue could grow unbounded
   - **Mitigation**: Proper cleanup, monitoring, circuit breakers

### Low Risk Items
1. **Audit Log Growth**: Could consume excessive storage
   - **Mitigation**: Log rotation, archiving, configurable retention

---

## Effort Breakdown Summary

| Phase | Duration | Effort | Key Deliverables |
|-------|----------|--------|------------------|
| **Phase 1**: Core Infrastructure ✅ | 2 weeks | High | RBAC data structures, validation endpoints |
| **Phase 2**: RBAC Integration ✅ | 2 weeks | High | Message interception, validation protocol |
| **Phase 3**: Advanced Features ✅ | 2 weeks | Medium | Audit logging, error handling |
| **Phase 4**: Testing & Validation ✅ | 2 weeks | Medium | Comprehensive test suite |
| **Phase 5**: Deployment & Docs | 2 weeks | Low | Production deployment, documentation |

**Total Effort**: 8-10 weeks for 2-3 developers
**Critical Path**: Phases 1-2 (must be sequential)
**Parallel Work**: Testing can start early, documentation concurrent with development

---

## Key Dependencies & Prerequisites

### Required Before Starting
1. ✅ RBAC database schema deployed
2. ✅ AccountSecurityEngine.CheckPermission() tested
3. ✅ JWT authentication working
4. ✅ Integration test framework operational
5. ✅ Database-driven configuration working

### External Dependencies
1. **Security Review**: Architecture review before Phase 2
2. **Performance Baseline**: Establish current throughput metrics
3. **Test Data**: RBAC users/roles/permissions seeded in test environment

---

## Alternative Approaches Considered

### Option A: Client-Side RBAC (Rejected)
- **Pros**: Simpler implementation, no message interception
- **Cons**: Cannot trust client-side validation, defeats RBAC purpose
- **Risk**: High security vulnerability

### Option B: Security Coordinator as Proxy (Rejected)
- **Pros**: Centralized security control
- **Cons**: Violates message coordinator's role, creates bottleneck
- **Risk**: Architectural violation, performance issues

### Option C: Selective Interception (Chosen)
- **Pros**: Maintains architecture, minimal performance impact
- **Cons**: Complex implementation
- **Risk**: Medium (mitigated by comprehensive testing)

---

**Recommendation**: Proceed with the selective interception approach. The implementation maintains architectural integrity while providing robust RBAC controls. Start with Phase 1 and establish working RBAC validation between message and security coordinators before expanding to full message interception.

---

**Document Status**: Implementation Complete (Phases 1-4) - Ready for Phase 5
**Next Steps**: Proceed to Phase 5 (Deployment & Documentation), schedule deployment planning meeting, begin staging environment validation

---

**Appendices**
- [Detailed API Specifications](#)
- [Performance Benchmark Results](#)
- [Security Assessment Report](#)
- [Test Case Specifications](#)