package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
)

const (
	mqttBroker = "tcp://localhost:1883"
	// testTimeout is set to health publish interval (30s) + 15% = 34.5s
	testTimeout = 35 * time.Second
)

// TestAuthenticationFlow tests the complete authentication lifecycle:
// login -> validate -> logout -> validate (should fail)
func TestAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "auth-flow-test")
	defer client.Disconnect(250)

	// Step 1: Login
	t.Run("Login", func(t *testing.T) {
		loginReq := models.AuthRequest{
			Username: "admin",
			Password: "bigskies_admin_2024",
		}

		response := publishAndWaitForResponse(
			t, ctx, client,
			"bigskies/coordinator/security/auth/login",
			"bigskies/coordinator/security/response/auth/login/response",
			loginReq,
		)

		var authResp models.AuthResponse
		err := json.Unmarshal([]byte(response), &authResp)
		require.NoError(t, err, "Failed to unmarshal login response")

		assert.NotEmpty(t, authResp.Token, "Token should not be empty")
		assert.NotNil(t, authResp.User, "User should not be nil")
		assert.Equal(t, "admin", authResp.User.Username)
		assert.True(t, authResp.User.Enabled, "User should be enabled")
		assert.False(t, authResp.ExpiresAt.IsZero(), "ExpiresAt should be set")

		// Store token for subsequent tests
		testToken := authResp.Token
		
		// Step 2: Validate token (should succeed)
		t.Run("ValidateActiveToken", func(t *testing.T) {
			validateReq := models.TokenValidationRequest{
				Token: testToken,
			}

			response := publishAndWaitForResponse(
				t, ctx, client,
				"bigskies/coordinator/security/auth/validate",
				"bigskies/coordinator/security/response/auth/validate/response",
				validateReq,
			)

			var validateResp models.TokenValidationResponse
			err := json.Unmarshal([]byte(response), &validateResp)
			require.NoError(t, err)

			assert.True(t, validateResp.Valid, "Token should be valid")
			assert.NotEmpty(t, validateResp.UserID, "UserID should not be empty")
			assert.Empty(t, validateResp.Error, "Error should be empty for valid token")
		})

		// Step 3: Logout
		t.Run("Logout", func(t *testing.T) {
			logoutReq := models.TokenValidationRequest{
				Token: testToken,
			}

			response := publishAndWaitForResponse(
				t, ctx, client,
				"bigskies/coordinator/security/auth/logout",
				"bigskies/coordinator/security/response/auth/logout/response",
				logoutReq,
			)

			var logoutResp map[string]interface{}
			err := json.Unmarshal([]byte(response), &logoutResp)
			require.NoError(t, err)

			assert.True(t, logoutResp["success"].(bool), "Logout should succeed")
		})

		// Step 4: Validate revoked token (should fail)
		t.Run("ValidateRevokedToken", func(t *testing.T) {
			validateReq := models.TokenValidationRequest{
				Token: testToken,
			}

			response := publishAndWaitForResponse(
				t, ctx, client,
				"bigskies/coordinator/security/auth/validate",
				"bigskies/coordinator/security/response/auth/validate/response",
				validateReq,
			)

			var validateResp models.TokenValidationResponse
			err := json.Unmarshal([]byte(response), &validateResp)
			require.NoError(t, err)

			assert.False(t, validateResp.Valid, "Revoked token should be invalid")
			assert.Contains(t, validateResp.Error, "revoked", "Error should mention token revocation")
		})
	})
}

// TestLoginFailure tests authentication with invalid credentials
func TestLoginFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "auth-fail-test")
	defer client.Disconnect(250)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{
			name:     "InvalidPassword",
			username: "admin",
			password: "wrong_password",
		},
		{
			name:     "InvalidUsername",
			username: "nonexistent",
			password: "bigskies_admin_2024",
		},
		{
			name:     "EmptyCredentials",
			username: "",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := models.AuthRequest{
				Username: tt.username,
				Password: tt.password,
			}

			response := publishAndWaitForResponse(
				t, ctx, client,
				"bigskies/coordinator/security/auth/login",
				"bigskies/coordinator/security/response/auth/login/response",
				loginReq,
			)

			var authResp map[string]interface{}
			err := json.Unmarshal([]byte(response), &authResp)
			require.NoError(t, err)

			assert.False(t, authResp["success"].(bool), "Login should fail with invalid credentials")
			assert.NotEmpty(t, authResp["error"], "Error message should be present")
		})
	}
}

// TestTokenValidationInvalid tests validation of malformed/invalid tokens
func TestTokenValidationInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "token-invalid-test")
	defer client.Disconnect(250)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "MalformedToken",
			token: "not.a.valid.jwt.token",
		},
		{
			name:  "EmptyToken",
			token: "",
		},
		{
			name:  "InvalidSignature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidGVzdCJ9.invalid_signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateReq := models.TokenValidationRequest{
				Token: tt.token,
			}

			response := publishAndWaitForResponse(
				t, ctx, client,
				"bigskies/coordinator/security/auth/validate",
				"bigskies/coordinator/security/response/auth/validate/response",
				validateReq,
			)

			var validateResp models.TokenValidationResponse
			err := json.Unmarshal([]byte(response), &validateResp)
			require.NoError(t, err)

			assert.False(t, validateResp.Valid, "Invalid token should not validate")
			assert.NotEmpty(t, validateResp.Error, "Error message should be present")
		})
	}
}

// TestMultipleLogins tests that multiple logins generate different tokens
func TestMultipleLogins(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "multi-login-test")
	defer client.Disconnect(250)

	loginReq := models.AuthRequest{
		Username: "admin",
		Password: "bigskies_admin_2024",
	}

	// First login
	response1 := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/security/auth/login",
		"bigskies/coordinator/security/response/auth/login/response",
		loginReq,
	)

	var authResp1 models.AuthResponse
	err := json.Unmarshal([]byte(response1), &authResp1)
	require.NoError(t, err)

	// Second login
	time.Sleep(100 * time.Millisecond) // Ensure different timestamp
	response2 := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/security/auth/login",
		"bigskies/coordinator/security/response/auth/login/response",
		loginReq,
	)

	var authResp2 models.AuthResponse
	err = json.Unmarshal([]byte(response2), &authResp2)
	require.NoError(t, err)

	assert.NotEqual(t, authResp1.Token, authResp2.Token, "Each login should generate a unique token")
	
	// Both tokens should be valid
	for i, token := range []string{authResp1.Token, authResp2.Token} {
		validateReq := models.TokenValidationRequest{Token: token}
		response := publishAndWaitForResponse(
			t, ctx, client,
			"bigskies/coordinator/security/auth/validate",
			"bigskies/coordinator/security/response/auth/validate/response",
			validateReq,
		)

		var validateResp models.TokenValidationResponse
		err := json.Unmarshal([]byte(response), &validateResp)
		require.NoError(t, err)
		assert.True(t, validateResp.Valid, fmt.Sprintf("Token %d should be valid", i+1))
	}
}

// TestLogoutInvalidToken tests logout with invalid token
func TestLogoutInvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client := setupMQTTClient(t, "logout-invalid-test")
	defer client.Disconnect(250)

	logoutReq := models.TokenValidationRequest{
		Token: "invalid.token.value",
	}

	response := publishAndWaitForResponse(
		t, ctx, client,
		"bigskies/coordinator/security/auth/logout",
		"bigskies/coordinator/security/response/auth/logout/response",
		logoutReq,
	)

	var logoutResp map[string]interface{}
	err := json.Unmarshal([]byte(response), &logoutResp)
	require.NoError(t, err)

	assert.False(t, logoutResp["success"].(bool), "Logout should fail with invalid token")
}

// Helper Functions

func setupMQTTClient(t *testing.T, clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	require.NoError(t, token.Error(), "Failed to connect to MQTT broker")

	return client
}

func publishAndWaitForResponse(
	t *testing.T,
	ctx context.Context,
	client mqtt.Client,
	publishTopic string,
	responseTopic string,
	request interface{},
) string {
	responseChan := make(chan string, 1)

	// Subscribe to response topic
	token := client.Subscribe(responseTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		select {
		case responseChan <- string(msg.Payload()):
		default:
			// Channel already has a message
		}
	})
	token.Wait()
	require.NoError(t, token.Error(), "Failed to subscribe to response topic")

	// Publish request
	payload, err := json.Marshal(request)
	require.NoError(t, err, "Failed to marshal request")

	token = client.Publish(publishTopic, 1, false, payload)
	token.Wait()
	require.NoError(t, token.Error(), "Failed to publish request")

	// Wait for response
	select {
	case response := <-responseChan:
		return response
	case <-ctx.Done():
		t.Fatal("Timeout waiting for response")
		return ""
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for MQTT response")
		return ""
	}
}
