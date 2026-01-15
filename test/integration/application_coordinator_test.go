package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestApplicationCoordinatorServiceRegistration tests service registration
func TestApplicationCoordinatorServiceRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "app-register-test")
	defer client.Disconnect(250)

	// Register a mock service
	serviceReg := map[string]interface{}{
		"id":       uuid.New().String(),
		"name":     "test-service",
		"endpoint": "http://test-service:8080",
		"metadata": map[string]interface{}{
			"version": "1.0.0",
			"type":    "api",
		},
	}

	payload, err := json.Marshal(serviceReg)
	require.NoError(t, err)

	// Publish registration
	token := client.Publish("bigskies/coordinator/service/event/register", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish service registration")

	// Give coordinator time to process
	time.Sleep(500 * time.Millisecond)

	// Test passes if no errors occurred
	t.Logf("Service registered: %s", serviceReg["id"])
}

// TestApplicationCoordinatorServiceHeartbeat tests service heartbeat handling
func TestApplicationCoordinatorServiceHeartbeat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "app-heartbeat-test")
	defer client.Disconnect(250)

	// First register a service
	serviceID := uuid.New().String()
	serviceReg := map[string]interface{}{
		"id":       serviceID,
		"name":     "heartbeat-test-service",
		"endpoint": "http://heartbeat-test:8080",
	}

	payload, err := json.Marshal(serviceReg)
	require.NoError(t, err)

	token := client.Publish("bigskies/coordinator/service/event/register", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error())

	time.Sleep(200 * time.Millisecond)

	// Send heartbeat
	heartbeat := map[string]interface{}{
		"id":     serviceID,
		"status": "healthy",
	}

	hbPayload, err := json.Marshal(heartbeat)
	require.NoError(t, err)

	token = client.Publish("bigskies/coordinator/service/event/heartbeat", 1, false, hbPayload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish heartbeat")

	time.Sleep(200 * time.Millisecond)

	t.Logf("Heartbeat sent for service: %s", serviceID)
}

// TestApplicationCoordinatorMultipleServices tests multiple service registrations
func TestApplicationCoordinatorMultipleServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "app-multi-test")
	defer client.Disconnect(250)

	// Register multiple services
	services := []map[string]interface{}{
		{
			"id":       uuid.New().String(),
			"name":     "service-1",
			"endpoint": "http://service-1:8080",
		},
		{
			"id":       uuid.New().String(),
			"name":     "service-2",
			"endpoint": "http://service-2:8080",
		},
		{
			"id":       uuid.New().String(),
			"name":     "service-3",
			"endpoint": "http://service-3:8080",
		},
	}

	for _, svc := range services {
		payload, err := json.Marshal(svc)
		require.NoError(t, err)

		token := client.Publish("bigskies/coordinator/service/event/register", 1, false, payload)
		token.Wait()
		require.NoError(t, token.Error())

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Registered %d services successfully", len(services))
}

// TestApplicationCoordinatorServiceStatus tests different service status values
func TestApplicationCoordinatorServiceStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "app-status-test")
	defer client.Disconnect(250)

	serviceID := uuid.New().String()

	// Register service
	serviceReg := map[string]interface{}{
		"id":       serviceID,
		"name":     "status-test-service",
		"endpoint": "http://status-test:8080",
	}

	payload, err := json.Marshal(serviceReg)
	require.NoError(t, err)

	token := client.Publish("bigskies/coordinator/service/event/register", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error())

	time.Sleep(200 * time.Millisecond)

	// Test different status values
	statuses := []string{"healthy", "degraded", "unhealthy"}

	for _, status := range statuses {
		heartbeat := map[string]interface{}{
			"id":     serviceID,
			"status": status,
		}

		hbPayload, err := json.Marshal(heartbeat)
		require.NoError(t, err)

		token = client.Publish("bigskies/coordinator/service/event/heartbeat", 1, false, hbPayload)
		token.Wait()
		require.NoError(t, token.Error())

		time.Sleep(100 * time.Millisecond)
		t.Logf("Service status updated to: %s", status)
	}
}
