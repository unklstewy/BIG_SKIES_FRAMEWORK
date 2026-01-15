package ascom

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"go.uber.org/zap"
)

// createTestDevice creates a test AlpacaDevice.
func createTestDevice(deviceID, deviceType string, deviceNumber int) *models.AlpacaDevice {
	return &models.AlpacaDevice{
		DeviceID:     deviceID,
		DeviceType:   deviceType,
		DeviceNumber: deviceNumber,
		Name:         "Test " + deviceType,
		ServerURL:    "http://localhost:32323",
		UUID:         "test-uuid-" + deviceID,
		Connected:    false,
		LastSeen:     time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestNewEngine(t *testing.T) {
	t.Run("creates engine with default health check interval", func(t *testing.T) {
		engine := NewEngine(nil, 0)
		assert.NotNil(t, engine)
		assert.Equal(t, 30*time.Second, engine.healthCheck)
		assert.NotNil(t, engine.client)
		assert.NotNil(t, engine.devices)
		assert.NotNil(t, engine.telescopes)
	})

	t.Run("creates engine with custom health check interval", func(t *testing.T) {
		engine := NewEngine(zap.NewNop(), 60*time.Second)
		assert.NotNil(t, engine)
		assert.Equal(t, 60*time.Second, engine.healthCheck)
	})
}

func TestRegisterDevice(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	t.Run("registers a device successfully", func(t *testing.T) {
		device := createTestDevice("device-1", "telescope", 0)
		err := engine.RegisterDevice(device)
		require.NoError(t, err)

		// Verify device is registered
		registered, err := engine.GetDevice("device-1")
		require.NoError(t, err)
		assert.Equal(t, device.DeviceID, registered.DeviceID)
		assert.Equal(t, device.DeviceType, registered.DeviceType)
	})

	t.Run("fails to register nil device", func(t *testing.T) {
		err := engine.RegisterDevice(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("fails to register duplicate device", func(t *testing.T) {
		device := createTestDevice("device-2", "camera", 0)
		err := engine.RegisterDevice(device)
		require.NoError(t, err)

		// Try to register again
		err = engine.RegisterDevice(device)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestUnregisterDevice(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)
	ctx := context.Background()

	device := createTestDevice("device-3", "telescope", 0)
	err := engine.RegisterDevice(device)
	require.NoError(t, err)

	t.Run("unregisters a device successfully", func(t *testing.T) {
		err := engine.UnregisterDevice(ctx, "device-3")
		require.NoError(t, err)

		// Verify device is unregistered
		_, err = engine.GetDevice("device-3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("fails to unregister non-existent device", func(t *testing.T) {
		err := engine.UnregisterDevice(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestGetDevice(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	device := createTestDevice("device-4", "dome", 0)
	err := engine.RegisterDevice(device)
	require.NoError(t, err)

	t.Run("gets a registered device", func(t *testing.T) {
		retrieved, err := engine.GetDevice("device-4")
		require.NoError(t, err)
		assert.Equal(t, "device-4", retrieved.DeviceID)
		assert.Equal(t, "dome", retrieved.DeviceType)
	})

	t.Run("fails to get non-existent device", func(t *testing.T) {
		_, err := engine.GetDevice("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestListDevices(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	t.Run("returns empty list when no devices", func(t *testing.T) {
		devices := engine.ListDevices()
		assert.NotNil(t, devices)
		assert.Len(t, devices, 0)
	})

	t.Run("returns all registered devices", func(t *testing.T) {
		device1 := createTestDevice("device-5", "telescope", 0)
		device2 := createTestDevice("device-6", "camera", 0)
		device3 := createTestDevice("device-7", "focuser", 0)

		require.NoError(t, engine.RegisterDevice(device1))
		require.NoError(t, engine.RegisterDevice(device2))
		require.NoError(t, engine.RegisterDevice(device3))

		devices := engine.ListDevices()
		assert.Len(t, devices, 3)

		// Verify all device IDs are present
		deviceIDs := make(map[string]bool)
		for _, d := range devices {
			deviceIDs[d.DeviceID] = true
		}
		assert.True(t, deviceIDs["device-5"])
		assert.True(t, deviceIDs["device-6"])
		assert.True(t, deviceIDs["device-7"])
	})
}

func TestIsDeviceConnected(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	device := createTestDevice("device-8", "telescope", 0)
	err := engine.RegisterDevice(device)
	require.NoError(t, err)

	t.Run("returns false for newly registered device", func(t *testing.T) {
		connected, err := engine.IsDeviceConnected("device-8")
		require.NoError(t, err)
		assert.False(t, connected)
	})

	t.Run("fails for non-existent device", func(t *testing.T) {
		_, err := engine.IsDeviceConnected("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestRegisterTelescopeDevices(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	// Register individual devices
	telescope := createTestDevice("tel-1", "telescope", 0)
	camera := createTestDevice("cam-1", "camera", 0)
	focuser := createTestDevice("foc-1", "focuser", 0)

	require.NoError(t, engine.RegisterDevice(telescope))
	require.NoError(t, engine.RegisterDevice(camera))
	require.NoError(t, engine.RegisterDevice(focuser))

	t.Run("registers telescope devices successfully", func(t *testing.T) {
		devices := map[string]string{
			"telescope": "tel-1",
			"camera":    "cam-1",
			"focuser":   "foc-1",
		}

		err := engine.RegisterTelescopeDevices("telescope-config-1", devices)
		require.NoError(t, err)

		// Verify devices can be retrieved by role
		telDevice, err := engine.GetTelescopeDevice("telescope-config-1", "telescope")
		require.NoError(t, err)
		assert.Equal(t, "tel-1", telDevice.DeviceID)

		camDevice, err := engine.GetTelescopeDevice("telescope-config-1", "camera")
		require.NoError(t, err)
		assert.Equal(t, "cam-1", camDevice.DeviceID)
	})

	t.Run("fails when device not registered", func(t *testing.T) {
		devices := map[string]string{
			"telescope": "non-existent",
		}

		err := engine.RegisterTelescopeDevices("telescope-config-2", devices)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestUnregisterTelescope(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	// Register devices and telescope
	telescope := createTestDevice("tel-2", "telescope", 0)
	require.NoError(t, engine.RegisterDevice(telescope))

	devices := map[string]string{
		"telescope": "tel-2",
	}
	require.NoError(t, engine.RegisterTelescopeDevices("telescope-config-3", devices))

	t.Run("unregisters telescope successfully", func(t *testing.T) {
		engine.UnregisterTelescope("telescope-config-3")

		// Verify telescope is unregistered
		_, err := engine.GetTelescopeDevice("telescope-config-3", "telescope")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestGetTelescopeDevice(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	// Setup
	telescope := createTestDevice("tel-3", "telescope", 0)
	camera := createTestDevice("cam-3", "camera", 0)

	require.NoError(t, engine.RegisterDevice(telescope))
	require.NoError(t, engine.RegisterDevice(camera))

	devices := map[string]string{
		"telescope": "tel-3",
		"camera":    "cam-3",
	}
	require.NoError(t, engine.RegisterTelescopeDevices("telescope-config-4", devices))

	t.Run("gets telescope device by role", func(t *testing.T) {
		device, err := engine.GetTelescopeDevice("telescope-config-4", "telescope")
		require.NoError(t, err)
		assert.Equal(t, "tel-3", device.DeviceID)

		device, err = engine.GetTelescopeDevice("telescope-config-4", "camera")
		require.NoError(t, err)
		assert.Equal(t, "cam-3", device.DeviceID)
	})

	t.Run("fails for non-existent telescope", func(t *testing.T) {
		_, err := engine.GetTelescopeDevice("non-existent", "telescope")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("fails for non-existent device role", func(t *testing.T) {
		_, err := engine.GetTelescopeDevice("telescope-config-4", "dome")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGetClient(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	t.Run("returns underlying ASCOM client", func(t *testing.T) {
		client := engine.GetClient()
		assert.NotNil(t, client)
		assert.IsType(t, &Client{}, client)
	})
}

func TestHealthCheck(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)
	ctx := context.Background()

	t.Run("healthy with no devices", func(t *testing.T) {
		result := engine.Check(ctx)
		assert.NotNil(t, result)
		assert.Equal(t, "ascom_engine", result.ComponentName)
		assert.Equal(t, healthcheck.StatusHealthy, result.Status)
		assert.Contains(t, result.Message, "healthy")
		assert.Equal(t, 0, result.Details["total_devices"])
		assert.Equal(t, 0, result.Details["connected_devices"])
	})

	t.Run("unhealthy when devices registered but none connected", func(t *testing.T) {
		device := createTestDevice("device-9", "telescope", 0)
		require.NoError(t, engine.RegisterDevice(device))

		result := engine.Check(ctx)
		assert.Equal(t, healthcheck.StatusUnhealthy, result.Status)
		assert.Contains(t, result.Message, "No devices connected")
		assert.Equal(t, 1, result.Details["total_devices"])
		assert.Equal(t, 0, result.Details["connected_devices"])
	})
}

func TestEngineStartStop(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 100*time.Millisecond)
	ctx := context.Background()

	t.Run("starts and stops engine successfully", func(t *testing.T) {
		engine.Start(ctx)

		// Give it a moment to start
		time.Sleep(50 * time.Millisecond)

		// Stop the engine
		engine.Stop()

		// Verify channels are closed (should not block)
		select {
		case <-engine.stopCh:
			// Channel closed as expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("stopCh should be closed")
		}
	})
}

func TestEngineConcurrency(t *testing.T) {
	engine := NewEngine(zap.NewNop(), 10*time.Second)

	t.Run("handles concurrent device registration", func(t *testing.T) {
		done := make(chan bool)

		// Register devices concurrently
		for i := 0; i < 10; i++ {
			go func(idx int) {
				device := createTestDevice(fmt.Sprintf("concurrent-%d", idx), "telescope", idx)
				err := engine.RegisterDevice(device)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all devices are registered
		devices := engine.ListDevices()
		assert.Len(t, devices, 10)
	})
}
