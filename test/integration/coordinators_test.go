package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Datastore Coordinator Tests
// =============================================================================

// TestDatastoreCoordinatorHealth tests that datastore coordinator is healthy
func TestDatastoreCoordinatorHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "datastore-health-test")
	defer client.Disconnect(250)

	// Subscribe to datastore health topic
	healthTopic := "bigskies/coordinator/datastore/health/status"
	responseChan := make(chan string, 1)

	token := client.Subscribe(healthTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		select {
		case responseChan <- string(msg.Payload()):
		default:
		}
	})
	token.Wait()
	require.NoError(t, token.Error())

	// Wait for health message
	select {
	case healthMsg := <-responseChan:
		var health map[string]interface{}
		err := json.Unmarshal([]byte(healthMsg), &health)
		require.NoError(t, err)

		assert.NotNil(t, health["payload"], "Health message should have payload")
		t.Logf("Datastore coordinator health: %v", health)

	case <-time.After(35 * time.Second):
		t.Fatal("Timeout waiting for datastore coordinator health update")
	case <-ctx.Done():
		t.Fatal("Context timeout")
	}
}

// TestDatastoreCoordinatorConnection tests database connectivity
func TestDatastoreCoordinatorConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test verifies that datastore coordinator is running and connected
	// by checking that it's publishing health status
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	client := setupMQTTClient(t, "datastore-conn-test")
	defer client.Disconnect(250)

	healthReceived := false
	healthTopic := "bigskies/coordinator/datastore/health/status"

	token := client.Subscribe(healthTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		healthReceived = true
	})
	token.Wait()
	require.NoError(t, token.Error())

	// Wait for health status
	deadline := time.After(35 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if healthReceived {
				t.Log("Datastore coordinator is connected and operational")
				return
			}
		case <-deadline:
			t.Fatal("Datastore coordinator not responding")
		case <-ctx.Done():
			t.Fatal("Context timeout")
		}
	}
}

// =============================================================================
// Plugin Coordinator Tests
// =============================================================================

// TestPluginCoordinatorInstallRequest tests plugin installation request
func TestPluginCoordinatorInstallRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "plugin-install-test")
	defer client.Disconnect(250)

	// Mock plugin install request
	installReq := map[string]interface{}{
		"plugin_id":   uuid.New().String(),
		"name":        "test-plugin",
		"version":     "1.0.0",
		"source":      "docker://test-plugin:1.0.0",
		"permissions": []string{"telescope.read", "telescope.write"},
	}

	payload, err := json.Marshal(installReq)
	require.NoError(t, err)

	// Publish install request
	token := client.Publish("bigskies/coordinator/plugin/cmd/install", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish plugin install request")

	// Give coordinator time to process
	time.Sleep(500 * time.Millisecond)

	t.Logf("Plugin install request sent: %s", installReq["plugin_id"])
}

// TestPluginCoordinatorRemoveRequest tests plugin removal request
func TestPluginCoordinatorRemoveRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "plugin-remove-test")
	defer client.Disconnect(250)

	// Mock plugin remove request
	removeReq := map[string]interface{}{
		"plugin_id": uuid.New().String(),
	}

	payload, err := json.Marshal(removeReq)
	require.NoError(t, err)

	// Publish remove request
	token := client.Publish("bigskies/coordinator/plugin/cmd/remove", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish plugin remove request")

	time.Sleep(500 * time.Millisecond)

	t.Logf("Plugin remove request sent: %s", removeReq["plugin_id"])
}

// TestPluginCoordinatorMultiplePlugins tests multiple plugin operations
func TestPluginCoordinatorMultiplePlugins(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "plugin-multi-test")
	defer client.Disconnect(250)

	// Install multiple mock plugins
	plugins := []map[string]interface{}{
		{
			"plugin_id": uuid.New().String(),
			"name":      "plugin-alpha",
			"version":   "1.0.0",
		},
		{
			"plugin_id": uuid.New().String(),
			"name":      "plugin-beta",
			"version":   "2.0.0",
		},
		{
			"plugin_id": uuid.New().String(),
			"name":      "plugin-gamma",
			"version":   "1.5.0",
		},
	}

	for _, plugin := range plugins {
		payload, err := json.Marshal(plugin)
		require.NoError(t, err)

		token := client.Publish("bigskies/coordinator/plugin/cmd/install", 1, false, payload)
		token.Wait()
		require.NoError(t, token.Error())

		time.Sleep(200 * time.Millisecond)
	}

	t.Logf("Sent install requests for %d plugins", len(plugins))
}

// =============================================================================
// UI Element Coordinator Tests
// =============================================================================

// TestUIElementCoordinatorRegistration tests UI element registration
func TestUIElementCoordinatorRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "ui-register-test")
	defer client.Disconnect(250)

	// Mock UI element registration
	uiElement := map[string]interface{}{
		"element_id": uuid.New().String(),
		"plugin_id":  uuid.New().String(),
		"type":       "control-panel",
		"name":       "Test Control Panel",
		"metadata": map[string]interface{}{
			"width":  800,
			"height": 600,
			"layout": "grid",
		},
	}

	payload, err := json.Marshal(uiElement)
	require.NoError(t, err)

	// Publish registration
	token := client.Publish("bigskies/coordinator/uielement/event/register", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish UI element registration")

	time.Sleep(500 * time.Millisecond)

	t.Logf("UI element registered: %s", uiElement["element_id"])
}

// TestUIElementCoordinatorUnregistration tests UI element unregistration
func TestUIElementCoordinatorUnregistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "ui-unregister-test")
	defer client.Disconnect(250)

	elementID := uuid.New().String()

	// First register
	uiElement := map[string]interface{}{
		"element_id": elementID,
		"plugin_id":  uuid.New().String(),
		"type":       "widget",
		"name":       "Test Widget",
	}

	payload, err := json.Marshal(uiElement)
	require.NoError(t, err)

	token := client.Publish("bigskies/coordinator/uielement/event/register", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error())

	time.Sleep(200 * time.Millisecond)

	// Then unregister
	unregister := map[string]interface{}{
		"element_id": elementID,
	}

	payload, err = json.Marshal(unregister)
	require.NoError(t, err)

	token = client.Publish("bigskies/coordinator/uielement/event/unregister", 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish UI element unregistration")

	time.Sleep(200 * time.Millisecond)

	t.Logf("UI element unregistered: %s", elementID)
}

// TestUIElementCoordinatorMultipleElements tests multiple UI element registrations
func TestUIElementCoordinatorMultipleElements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "ui-multi-test")
	defer client.Disconnect(250)

	// Register multiple UI elements
	elements := []map[string]interface{}{
		{
			"element_id": uuid.New().String(),
			"plugin_id":  uuid.New().String(),
			"type":       "dashboard",
			"name":       "Main Dashboard",
		},
		{
			"element_id": uuid.New().String(),
			"plugin_id":  uuid.New().String(),
			"type":       "chart",
			"name":       "Telemetry Chart",
		},
		{
			"element_id": uuid.New().String(),
			"plugin_id":  uuid.New().String(),
			"type":       "control",
			"name":       "Telescope Control",
		},
	}

	for _, elem := range elements {
		payload, err := json.Marshal(elem)
		require.NoError(t, err)

		token := client.Publish("bigskies/coordinator/uielement/event/register", 1, false, payload)
		token.Wait()
		require.NoError(t, token.Error())

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Registered %d UI elements successfully", len(elements))
}

// TestUIElementCoordinatorDifferentTypes tests various UI element types
func TestUIElementCoordinatorDifferentTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "ui-types-test")
	defer client.Disconnect(250)

	// Test different UI element types
	types := []string{"panel", "widget", "dialog", "menu", "toolbar", "status-bar"}

	for _, elemType := range types {
		uiElement := map[string]interface{}{
			"element_id": uuid.New().String(),
			"plugin_id":  uuid.New().String(),
			"type":       elemType,
			"name":       "Test " + elemType,
		}

		payload, err := json.Marshal(uiElement)
		require.NoError(t, err)

		token := client.Publish("bigskies/coordinator/uielement/event/register", 1, false, payload)
		token.Wait()
		require.NoError(t, token.Error())

		time.Sleep(100 * time.Millisecond)
		t.Logf("Registered UI element type: %s", elemType)
	}
}
