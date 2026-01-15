package mqtt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewClient(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				BrokerURL:            "tcp://localhost:1883",
				ClientID:             "test-client",
				KeepAlive:            30 * time.Second,
				ConnectTimeout:       5 * time.Second,
				AutoReconnect:        true,
				MaxReconnectInterval: 1 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.client)
				assert.NotNil(t, client.logger)
				assert.Equal(t, tt.config, client.config)
			}
		})
	}
}

func TestClientIsConnected(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		BrokerURL:            "tcp://localhost:1883",
		ClientID:             "test-client",
		KeepAlive:            30 * time.Second,
		ConnectTimeout:       5 * time.Second,
		AutoReconnect:        true,
		MaxReconnectInterval: 1 * time.Minute,
	}

	client, err := NewClient(config, logger)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Should not be connected initially
	assert.False(t, client.IsConnected())
}
