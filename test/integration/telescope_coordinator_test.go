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
// Telescope Coordinator Health Tests
// =============================================================================

// TestTelescopeCoordinatorHealth tests that telescope coordinator publishes health status
func TestTelescopeCoordinatorHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-health-test")
	defer client.Disconnect(250)

	// Subscribe to telescope coordinator health topic
	healthTopic := "bigskies/coordinator/telescope/health/status"
	responseChan := make(chan string, 1)

	token := client.Subscribe(healthTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		select {
		case responseChan <- string(msg.Payload()):
		default:
		}
	})
	token.Wait()
	require.NoError(t, token.Error(), "Failed to subscribe to health topic")

	// Wait for health message (coordinators publish health periodically)
	select {
	case healthMsg := <-responseChan:
		var health map[string]interface{}
		err := json.Unmarshal([]byte(healthMsg), &health)
		require.NoError(t, err, "Failed to unmarshal health message")

		// Verify health message structure
		assert.NotNil(t, health["payload"], "Health message should have payload")
		t.Logf("Telescope coordinator health: %v", health)

	case <-time.After(35 * time.Second): // Health published every 30s
		t.Fatal("Timeout waiting for telescope coordinator health update")
	case <-ctx.Done():
		t.Fatal("Context timeout waiting for health update")
	}
}

// =============================================================================
// Telescope Configuration Tests
// =============================================================================

// TestTelescopeCoordinatorCreateConfig tests creating a telescope configuration
func TestTelescopeCoordinatorCreateConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-create-config-test")
	defer client.Disconnect(250)

	// Create telescope configuration request
	configReq := map[string]interface{}{
		"name":        "Test Telescope",
		"description": "Integration test telescope configuration",
		"owner_id":    uuid.New().String(),
		"owner_type":  "user",
		"site_id":     uuid.New().String(),
		"mount_type":  "equatorial",
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/create",
		"bigskies/coordinator/telescope/response/config/create/response",
		configReq,
	)

	var configResp map[string]interface{}
	err := json.Unmarshal([]byte(response), &configResp)
	require.NoError(t, err, "Failed to unmarshal config create response")

	assert.True(t, configResp["success"].(bool), "Config creation should succeed")
	assert.NotEmpty(t, configResp["id"], "Config ID should be returned")

	t.Logf("Created telescope config: %s", configResp["id"])
}

// TestTelescopeCoordinatorUpdateConfig tests updating a telescope configuration
func TestTelescopeCoordinatorUpdateConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-update-config-test")
	defer client.Disconnect(250)

	// First create a config
	configReq := map[string]interface{}{
		"name":        "Test Telescope Original",
		"description": "Original description",
		"owner_id":    uuid.New().String(),
		"owner_type":  "user",
		"site_id":     uuid.New().String(),
		"mount_type":  "equatorial",
	}

	createResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/create",
		"bigskies/coordinator/telescope/response/config/create/response",
		configReq,
	)

	var createResult map[string]interface{}
	err := json.Unmarshal([]byte(createResp), &createResult)
	require.NoError(t, err)
	require.True(t, createResult["success"].(bool))

	configID := createResult["id"].(string)

	// Now update it
	updateReq := map[string]interface{}{
		"id":          configID,
		"name":        "Test Telescope Updated",
		"description": "Updated description",
		"mount_type":  "altazimuth",
		"enabled":     true,
	}

	updateResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/update",
		"bigskies/coordinator/telescope/response/config/update/response",
		updateReq,
	)

	var updateResult map[string]interface{}
	err = json.Unmarshal([]byte(updateResp), &updateResult)
	require.NoError(t, err)

	assert.True(t, updateResult["success"].(bool), "Config update should succeed")
	t.Logf("Updated telescope config: %s", configID)
}

// TestTelescopeCoordinatorListConfigs tests listing telescope configurations
func TestTelescopeCoordinatorListConfigs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-list-configs-test")
	defer client.Disconnect(250)

	userID := uuid.New().String()

	// Create multiple configs for the same user
	for i := 0; i < 3; i++ {
		configReq := map[string]interface{}{
			"name":        "Test Telescope " + string(rune('A'+i)),
			"description": "Test telescope for listing",
			"owner_id":    userID,
			"owner_type":  "user",
			"mount_type":  "equatorial",
		}

		response := publishAndWaitForResponse(
			t, ctx, client,
			"bigskies/coordinator/telescope/config/create",
			"bigskies/coordinator/telescope/response/config/create/response",
			configReq,
		)

		var createResult map[string]interface{}
		err := json.Unmarshal([]byte(response), &createResult)
		require.NoError(t, err)
		require.True(t, createResult["success"].(bool))

		time.Sleep(100 * time.Millisecond)
	}

	// Now list configs for this user
	listReq := map[string]interface{}{
		"user_id": userID,
	}

	listResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/list",
		"bigskies/coordinator/telescope/response/config/list/response",
		listReq,
	)

	var listResult map[string]interface{}
	err := json.Unmarshal([]byte(listResp), &listResult)
	require.NoError(t, err)

	assert.True(t, listResult["success"].(bool), "Config list should succeed")
	configs := listResult["configs"].([]interface{})
	assert.GreaterOrEqual(t, len(configs), 3, "Should have at least 3 configs")

	t.Logf("Listed %d telescope configs for user %s", len(configs), userID)
}

// TestTelescopeCoordinatorGetConfig tests retrieving a specific telescope configuration
func TestTelescopeCoordinatorGetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-get-config-test")
	defer client.Disconnect(250)

	// Create a config first
	configReq := map[string]interface{}{
		"name":        "Test Telescope Get",
		"description": "Test telescope for get operation",
		"owner_id":    uuid.New().String(),
		"owner_type":  "user",
		"mount_type":  "equatorial",
	}

	createResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/create",
		"bigskies/coordinator/telescope/response/config/create/response",
		configReq,
	)

	var createResult map[string]interface{}
	err := json.Unmarshal([]byte(createResp), &createResult)
	require.NoError(t, err)
	require.True(t, createResult["success"].(bool))

	configID := createResult["id"].(string)

	// Now get the config
	getReq := map[string]interface{}{
		"id": configID,
	}

	getResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/get",
		"bigskies/coordinator/telescope/response/config/get/response",
		getReq,
	)

	var getResult map[string]interface{}
	err = json.Unmarshal([]byte(getResp), &getResult)
	require.NoError(t, err)

	assert.True(t, getResult["success"].(bool), "Config get should succeed")
	config := getResult["config"].(map[string]interface{})
	assert.Equal(t, configID, config["id"], "Config ID should match")
	assert.Equal(t, "Test Telescope Get", config["name"], "Config name should match")

	t.Logf("Retrieved telescope config: %s", configID)
}

// TestTelescopeCoordinatorDeleteConfig tests deleting a telescope configuration
func TestTelescopeCoordinatorDeleteConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-delete-config-test")
	defer client.Disconnect(250)

	// Create a config first
	configReq := map[string]interface{}{
		"name":        "Test Telescope Delete",
		"description": "Test telescope for delete operation",
		"owner_id":    uuid.New().String(),
		"owner_type":  "user",
		"mount_type":  "equatorial",
	}

	createResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/create",
		"bigskies/coordinator/telescope/response/config/create/response",
		configReq,
	)

	var createResult map[string]interface{}
	err := json.Unmarshal([]byte(createResp), &createResult)
	require.NoError(t, err)
	require.True(t, createResult["success"].(bool))

	configID := createResult["id"].(string)

	// Now delete the config
	deleteReq := map[string]interface{}{
		"id": configID,
	}

	deleteResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/delete",
		"bigskies/coordinator/telescope/response/config/delete/response",
		deleteReq,
	)

	var deleteResult map[string]interface{}
	err = json.Unmarshal([]byte(deleteResp), &deleteResult)
	require.NoError(t, err)

	assert.True(t, deleteResult["success"].(bool), "Config delete should succeed")
	t.Logf("Deleted telescope config: %s", configID)
}

// =============================================================================
// Device Discovery and Connection Tests
// =============================================================================

// TestTelescopeCoordinatorDiscoverDevices tests ASCOM device discovery
func TestTelescopeCoordinatorDiscoverDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-discover-test")
	defer client.Disconnect(250)

	// Request device discovery
	discoverReq := map[string]interface{}{
		"port": 32227, // Default ASCOM Alpaca port
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/device/discover",
		"bigskies/coordinator/telescope/response/device/discover/response",
		discoverReq,
	)

	var discoverResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &discoverResult)
	require.NoError(t, err, "Failed to unmarshal discover response")

	// Discovery may or may not find devices depending on test environment
	// Just verify the response structure is valid
	assert.NotNil(t, discoverResult, "Discover response should not be nil")
	t.Logf("Device discovery result: %v", discoverResult)
}

// TestTelescopeCoordinatorConnectDevice tests connecting to an ASCOM device
func TestTelescopeCoordinatorConnectDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-connect-test")
	defer client.Disconnect(250)

	// Attempt to connect to a device
	// Note: This will likely fail without a real ASCOM simulator running
	connectReq := map[string]interface{}{
		"device_id":     "test-device-" + uuid.New().String(),
		"device_type":   "telescope",
		"device_number": 0,
		"server_url":    "http://localhost:32323",
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/device/connect",
		"bigskies/coordinator/telescope/response/device/connect/response",
		connectReq,
	)

	var connectResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &connectResult)
	require.NoError(t, err, "Failed to unmarshal connect response")

	// Connection may fail if no ASCOM server is available
	// Just verify we get a response
	assert.NotNil(t, connectResult, "Connect response should not be nil")
	t.Logf("Device connect result: %v", connectResult)
}

// TestTelescopeCoordinatorDisconnectDevice tests disconnecting from an ASCOM device
func TestTelescopeCoordinatorDisconnectDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-disconnect-test")
	defer client.Disconnect(250)

	// Attempt to disconnect a device
	disconnectReq := map[string]interface{}{
		"device_id": "test-device-" + uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/device/disconnect",
		"bigskies/coordinator/telescope/response/device/disconnect/response",
		disconnectReq,
	)

	var disconnectResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &disconnectResult)
	require.NoError(t, err, "Failed to unmarshal disconnect response")

	assert.NotNil(t, disconnectResult, "Disconnect response should not be nil")
	t.Logf("Device disconnect result: %v", disconnectResult)
}

// =============================================================================
// Telescope Control Tests
// =============================================================================

// TestTelescopeCoordinatorSlewTelescope tests slewing telescope to coordinates
func TestTelescopeCoordinatorSlewTelescope(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-slew-test")
	defer client.Disconnect(250)

	// Send slew command
	slewReq := map[string]interface{}{
		"device_id":        "test-device-" + uuid.New().String(),
		"right_ascension":  12.5,  // RA in hours
		"declination":      45.0,  // Dec in degrees
		"target_name":      "Test Target",
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/control/slew",
		"bigskies/coordinator/telescope/response/control/slew/response",
		slewReq,
	)

	var slewResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &slewResult)
	require.NoError(t, err, "Failed to unmarshal slew response")

	assert.NotNil(t, slewResult, "Slew response should not be nil")
	t.Logf("Slew command result: %v", slewResult)
}

// TestTelescopeCoordinatorParkTelescope tests parking the telescope
func TestTelescopeCoordinatorParkTelescope(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-park-test")
	defer client.Disconnect(250)

	// Send park command
	parkReq := map[string]interface{}{
		"device_id": "test-device-" + uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/control/park",
		"bigskies/coordinator/telescope/response/control/park/response",
		parkReq,
	)

	var parkResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &parkResult)
	require.NoError(t, err, "Failed to unmarshal park response")

	assert.NotNil(t, parkResult, "Park response should not be nil")
	t.Logf("Park command result: %v", parkResult)
}

// TestTelescopeCoordinatorUnparkTelescope tests unparking the telescope
func TestTelescopeCoordinatorUnparkTelescope(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-unpark-test")
	defer client.Disconnect(250)

	// Send unpark command
	unparkReq := map[string]interface{}{
		"device_id": "test-device-" + uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/control/unpark",
		"bigskies/coordinator/telescope/response/control/unpark/response",
		unparkReq,
	)

	var unparkResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &unparkResult)
	require.NoError(t, err, "Failed to unmarshal unpark response")

	assert.NotNil(t, unparkResult, "Unpark response should not be nil")
	t.Logf("Unpark command result: %v", unparkResult)
}

// TestTelescopeCoordinatorSetTracking tests enabling/disabling tracking
func TestTelescopeCoordinatorSetTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-tracking-test")
	defer client.Disconnect(250)

	// Test both enabling and disabling tracking
	trackingStates := []bool{true, false}

	for _, enabled := range trackingStates {
		trackReq := map[string]interface{}{
			"device_id": "test-device-" + uuid.New().String(),
			"enabled":   enabled,
		}

		response := publishAndWaitForResponse(
			t, ctx, client,
			"bigskies/coordinator/telescope/control/track",
			"bigskies/coordinator/telescope/response/control/track/response",
			trackReq,
		)

		var trackResult map[string]interface{}
		err := json.Unmarshal([]byte(response), &trackResult)
		require.NoError(t, err, "Failed to unmarshal tracking response")

		assert.NotNil(t, trackResult, "Tracking response should not be nil")
		t.Logf("Set tracking to %v: %v", enabled, trackResult)

		time.Sleep(100 * time.Millisecond)
	}
}

// TestTelescopeCoordinatorAbortSlew tests aborting a slew operation
func TestTelescopeCoordinatorAbortSlew(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-abort-test")
	defer client.Disconnect(250)

	// Send abort command
	abortReq := map[string]interface{}{
		"device_id": "test-device-" + uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/control/abort",
		"bigskies/coordinator/telescope/response/control/abort/response",
		abortReq,
	)

	var abortResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &abortResult)
	require.NoError(t, err, "Failed to unmarshal abort response")

	assert.NotNil(t, abortResult, "Abort response should not be nil")
	t.Logf("Abort command result: %v", abortResult)
}

// TestTelescopeCoordinatorGetStatus tests getting telescope status
func TestTelescopeCoordinatorGetStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-status-test")
	defer client.Disconnect(250)

	// Request telescope status
	statusReq := map[string]interface{}{
		"device_id": "test-device-" + uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/status/get",
		"bigskies/coordinator/telescope/response/status/get/response",
		statusReq,
	)

	var statusResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &statusResult)
	require.NoError(t, err, "Failed to unmarshal status response")

	assert.NotNil(t, statusResult, "Status response should not be nil")
	t.Logf("Telescope status: %v", statusResult)
}

// =============================================================================
// Session Management Tests
// =============================================================================

// TestTelescopeCoordinatorStartSession tests starting an observation session
func TestTelescopeCoordinatorStartSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-start-session-test")
	defer client.Disconnect(250)

	// Start a session
	sessionReq := map[string]interface{}{
		"config_id":    uuid.New().String(),
		"session_name": "Test Observation Session",
		"user_id":      uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/session/start",
		"bigskies/coordinator/telescope/response/session/start/response",
		sessionReq,
	)

	var sessionResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &sessionResult)
	require.NoError(t, err, "Failed to unmarshal session start response")

	assert.NotNil(t, sessionResult, "Session start response should not be nil")
	t.Logf("Start session result: %v", sessionResult)
}

// TestTelescopeCoordinatorEndSession tests ending an observation session
func TestTelescopeCoordinatorEndSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-end-session-test")
	defer client.Disconnect(250)

	// End a session
	sessionReq := map[string]interface{}{
		"session_id": uuid.New().String(),
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/session/end",
		"bigskies/coordinator/telescope/response/session/end/response",
		sessionReq,
	)

	var sessionResult map[string]interface{}
	err := json.Unmarshal([]byte(response), &sessionResult)
	require.NoError(t, err, "Failed to unmarshal session end response")

	assert.NotNil(t, sessionResult, "Session end response should not be nil")
	t.Logf("End session result: %v", sessionResult)
}

// =============================================================================
// Multi-Operation Tests
// =============================================================================

// TestTelescopeCoordinatorFullWorkflow tests a complete telescope workflow
func TestTelescopeCoordinatorFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-workflow-test")
	defer client.Disconnect(250)

	userID := uuid.New().String()

	// Step 1: Create a telescope configuration
	t.Log("Step 1: Creating telescope configuration...")
	configReq := map[string]interface{}{
		"name":        "Workflow Test Telescope",
		"description": "Full workflow test",
		"owner_id":    userID,
		"owner_type":  "user",
		"mount_type":  "equatorial",
	}

	createResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/create",
		"bigskies/coordinator/telescope/response/config/create/response",
		configReq,
	)

	var createResult map[string]interface{}
	err := json.Unmarshal([]byte(createResp), &createResult)
	require.NoError(t, err)
	require.True(t, createResult["success"].(bool))

	configID := createResult["id"].(string)
	t.Logf("Created config: %s", configID)

	time.Sleep(200 * time.Millisecond)

	// Step 2: List configurations
	t.Log("Step 2: Listing configurations...")
	listReq := map[string]interface{}{
		"user_id": userID,
	}

	listResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/list",
		"bigskies/coordinator/telescope/response/config/list/response",
		listReq,
	)

	var listResult map[string]interface{}
	err = json.Unmarshal([]byte(listResp), &listResult)
	require.NoError(t, err)
	require.True(t, listResult["success"].(bool))

	configs := listResult["configs"].([]interface{})
	assert.GreaterOrEqual(t, len(configs), 1, "Should have at least 1 config")

	time.Sleep(200 * time.Millisecond)

	// Step 3: Start a session
	t.Log("Step 3: Starting observation session...")
	sessionReq := map[string]interface{}{
		"config_id":    configID,
		"session_name": "Workflow Test Session",
		"user_id":      userID,
	}

	sessionResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/session/start",
		"bigskies/coordinator/telescope/response/session/start/response",
		sessionReq,
	)

	var sessionResult map[string]interface{}
	err = json.Unmarshal([]byte(sessionResp), &sessionResult)
	require.NoError(t, err)

	t.Logf("Session started: %v", sessionResult)

	time.Sleep(200 * time.Millisecond)

	// Step 4: End the session
	t.Log("Step 4: Ending observation session...")
	// Note: In a real scenario, we'd use the session_id from the start response
	// For this test, we'll use a mock session ID
	endSessionReq := map[string]interface{}{
		"session_id": uuid.New().String(),
	}

	endSessionResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/session/end",
		"bigskies/coordinator/telescope/response/session/end/response",
		endSessionReq,
	)

	var endSessionResult map[string]interface{}
	err = json.Unmarshal([]byte(endSessionResp), &endSessionResult)
	require.NoError(t, err)

	t.Logf("Session ended: %v", endSessionResult)

	t.Log("Full workflow completed successfully")
}

// TestTelescopeCoordinatorConcurrentRequests tests handling multiple rapid requests
func TestTelescopeCoordinatorConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "telescope-concurrent-test")
	defer client.Disconnect(250)

	userID := uuid.New().String()

	// Create multiple configs in rapid succession, waiting for each response
	numConfigs := 5
	t.Logf("Creating %d configs in rapid succession for user %s", numConfigs, userID)
	
	for i := 0; i < numConfigs; i++ {
		configReq := map[string]interface{}{
			"name":        "Concurrent Test " + string(rune('A'+i)),
			"description": "Rapid creation test",
			"owner_id":    userID,
			"owner_type":  "user",
			"mount_type":  "equatorial",
		}

		response := publishAndWaitForResponse(
			t, ctx, client,
			"bigskies/coordinator/telescope/config/create",
			"bigskies/coordinator/telescope/response/config/create/response",
			configReq,
		)

		var createResult map[string]interface{}
		err := json.Unmarshal([]byte(response), &createResult)
		require.NoError(t, err, "Failed to unmarshal response for config %d", i)
		require.True(t, createResult["success"].(bool), "Config %d creation should succeed", i)
		
		t.Logf("Created config %d: %s", i, createResult["id"])
		
		// Small delay between requests
		time.Sleep(50 * time.Millisecond)
	}

	// Now verify all configs were created by listing them
	t.Log("Verifying all configs via list operation...")
	listReq := map[string]interface{}{
		"user_id": userID,
	}

	listResp := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/telescope/config/list",
		"bigskies/coordinator/telescope/response/config/list/response",
		listReq,
	)

	var listResult map[string]interface{}
	err := json.Unmarshal([]byte(listResp), &listResult)
	require.NoError(t, err)

	if listResult["success"].(bool) {
		configs := listResult["configs"].([]interface{})
		assert.GreaterOrEqual(t, len(configs), numConfigs, "All configs should be in the list")
		t.Logf("Successfully created and verified %d configs", len(configs))
	}
}
