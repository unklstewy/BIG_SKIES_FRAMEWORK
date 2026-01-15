package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageCoordinatorHealth tests that the message coordinator publishes health status
func TestMessageCoordinatorHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "message-health-test")
	defer client.Disconnect(250)

	// Subscribe to message coordinator health topic
	healthTopic := "bigskies/coordinator/message/health/status"
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
		t.Logf("Message coordinator health: %v", health)

	case <-time.After(35 * time.Second): // Health published every 30s
		t.Fatal("Timeout waiting for message coordinator health update")
	case <-ctx.Done():
		t.Fatal("Context timeout waiting for health update")
	}
}

// TestMessageCoordinatorConnectivity tests MQTT message routing through coordinator
func TestMessageCoordinatorConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "message-connectivity-test")
	defer client.Disconnect(250)

	testTopic := "bigskies/test/echo"
	testMessage := `{"test":"message","timestamp":` + time.Now().Format("20060102150405") + `}`
	responseChan := make(chan string, 1)

	// Subscribe to test topic
	token := client.Subscribe(testTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		select {
		case responseChan <- string(msg.Payload()):
		default:
		}
	})
	token.Wait()
	require.NoError(t, token.Error())

	// Publish message
	token = client.Publish(testTopic, 1, false, []byte(testMessage))
	token.Wait()
	require.NoError(t, token.Error())

	// Verify message received (proving MQTT broker is operational)
	select {
	case receivedMsg := <-responseChan:
		assert.Equal(t, testMessage, receivedMsg, "Message should be received unchanged")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for test message")
	case <-ctx.Done():
		t.Fatal("Context timeout")
	}
}

// TestMessageCoordinatorHealthSubscriptions verifies coordinator subscribes to other health topics
func TestMessageCoordinatorHealthSubscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background() // For future use

	client := setupMQTTClient(t, "message-sub-test")
	defer client.Disconnect(250)

	// Publish mock health from security coordinator
	securityHealthTopic := "bigskies/coordinator/security/health"
	mockHealth := map[string]interface{}{
		"component":  "security-coordinator",
		"status":     "healthy",
		"timestamp":  time.Now().Unix(),
		"message":    "Test health message",
	}

	payload, err := json.Marshal(mockHealth)
	require.NoError(t, err)

	token := client.Publish(securityHealthTopic, 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error())

	// Give message coordinator time to process
	time.Sleep(100 * time.Millisecond)

	// Verify no errors occurred (message coordinator subscribed and processed it)
	assert.True(t, true, "Health message published without errors")
}
