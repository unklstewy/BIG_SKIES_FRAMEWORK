package coordinators

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/engines/ascom"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/mqtt"
	"go.uber.org/zap"
)

// testableTelescopeCoordinator wraps TelescopeCoordinator for testing
type testableTelescopeCoordinator struct {
	*TelescopeCoordinator
}

func newTestableTelescopeCoordinator(logger *zap.Logger) *testableTelescopeCoordinator {
	tc := &TelescopeCoordinator{
		BaseCoordinator: &BaseCoordinator{
			name:          mqtt.CoordinatorTelescope,
			mqttClient:    nil, // No MQTT client in unit tests
			logger:        logger,
			shutdownFuncs: make([]func(context.Context) error, 0),
		},
		ascomEngine: ascom.NewEngine(logger, 30*time.Second),
		config: &TelescopeConfig{
			BaseConfig: BaseConfig{
				Name:                "telescope",
				HealthCheckInterval: 30 * time.Second,
			},
			DiscoveryPort:       32227,
			HealthCheckInterval: 30 * time.Second,
		},
	}

	return &testableTelescopeCoordinator{
		TelescopeCoordinator: tc,
	}
}

// TestTelescopeCoordinator_Creation tests basic coordinator initialization
func TestTelescopeCoordinator_Creation(t *testing.T) {
	logger := zap.NewNop()
	ttc := newTestableTelescopeCoordinator(logger)

	t.Run("coordinator is properly initialized", func(t *testing.T) {
		assert.NotNil(t, ttc)
		assert.NotNil(t, ttc.ascomEngine)
		assert.NotNil(t, ttc.config)
		assert.Equal(t, 32227, ttc.config.DiscoveryPort)
	})

	t.Run("ASCOM engine is accessible", func(t *testing.T) {
		assert.NotNil(t, ttc.ascomEngine)
		assert.NotNil(t, ttc.ascomEngine.GetClient())
	})

	t.Run("coordinator name is correct", func(t *testing.T) {
		assert.Equal(t, mqtt.CoordinatorTelescope, ttc.Name())
	})
}

// TestTelescopeCoordinator_ASCOMEngineIntegration tests integration with ASCOM engine
func TestTelescopeCoordinator_ASCOMEngineIntegration(t *testing.T) {
	logger := zap.NewNop()
	ttc := newTestableTelescopeCoordinator(logger)

	t.Run("can register devices with engine", func(t *testing.T) {
		device := &models.AlpacaDevice{
			DeviceID:     "test-device-1",
			DeviceType:   "telescope",
			DeviceNumber: 0,
			Name:         "Test Telescope",
			ServerURL:    "http://localhost:32323",
			UUID:         "test-uuid",
			Connected:    false,
		}

		err := ttc.ascomEngine.RegisterDevice(device)
		assert.NoError(t, err)

		// Verify device was registered
		devices := ttc.ascomEngine.ListDevices()
		assert.Len(t, devices, 1)
		assert.Equal(t, "test-device-1", devices[0].DeviceID)
	})

	t.Run("can connect and disconnect devices", func(t *testing.T) {
		device := &models.AlpacaDevice{
			DeviceID:     "test-device-2",
			DeviceType:   "telescope",
			DeviceNumber: 0,
			Name:         "Test Telescope 2",
			ServerURL:    "http://localhost:32323",
			UUID:         "test-uuid-2",
			Connected:    false,
		}

		// Register device
		err := ttc.ascomEngine.RegisterDevice(device)
		assert.NoError(t, err)

		// Note: Actual connection will fail without a real ASCOM server
		// This test just verifies the methods exist and can be called
		ctx := context.Background()
		assert.NotPanics(t, func() {
			_ = ttc.ascomEngine.ConnectDevice(ctx, "test-device-2")
			_ = ttc.ascomEngine.DisconnectDevice(ctx, "test-device-2")
		})
	})

	t.Run("health check returns results", func(t *testing.T) {
		result := ttc.ascomEngine.Check(context.Background())
		assert.NotNil(t, result)
		assert.Equal(t, "ascom_engine", result.ComponentName)
	})
}

// TestTelescopeCoordinator_Config tests configuration management
func TestTelescopeCoordinator_Config(t *testing.T) {
	logger := zap.NewNop()
	ttc := newTestableTelescopeCoordinator(logger)

	t.Run("default configuration is set", func(t *testing.T) {
		config := ttc.config
		assert.NotNil(t, config)
		assert.Equal(t, "telescope", config.Name)
		assert.Equal(t, 32227, config.DiscoveryPort)
		assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
	})

	t.Run("can update configuration", func(t *testing.T) {
		newConfig := &TelescopeConfig{
			BaseConfig: BaseConfig{
				Name:                "telescope-updated",
				HealthCheckInterval: 60 * time.Second,
			},
			DiscoveryPort:       11111,
			HealthCheckInterval: 60 * time.Second,
		}

		err := ttc.LoadConfig(newConfig)
		assert.NoError(t, err)

		retrieved := ttc.GetConfig().(*TelescopeConfig)
		assert.Equal(t, "telescope-updated", retrieved.Name)
		assert.Equal(t, 11111, retrieved.DiscoveryPort)
	})
}
