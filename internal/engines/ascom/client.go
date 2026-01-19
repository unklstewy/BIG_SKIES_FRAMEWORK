// Package ascom provides ASCOM Alpaca protocol client implementation.
package ascom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"go.uber.org/zap"
)

// Client provides methods to interact with ASCOM Alpaca devices.
type Client struct {
	httpClient  *http.Client
	logger      *zap.Logger
	clientID    int32
	transaction int32 // atomic counter for transaction IDs
}

// NewClient creates a new ASCOM Alpaca client.
func NewClient(logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger.With(zap.String("component", "ascom_client")),
		clientID:    1, // Static client ID
		transaction: 0,
	}
}

// nextTransactionID generates the next transaction ID.
func (c *Client) nextTransactionID() int32 {
	return atomic.AddInt32(&c.transaction, 1)
}

// DiscoverDevices performs UDP broadcast discovery to find Alpaca devices.
func (c *Client) DiscoverDevices(ctx context.Context, port int) ([]*models.AlpacaDevice, error) {
	c.logger.Info("Starting Alpaca device discovery", zap.Int("port", port))

	// Create UDP connection for broadcast
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Prepare discovery message
	discoveryMsg := []byte("alpacadiscovery1")
	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: port,
	}

	// Send broadcast
	_, err = conn.WriteToUDP(discoveryMsg, broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send discovery broadcast: %w", err)
	}

	c.logger.Debug("Sent discovery broadcast")

	// Collect responses
	devices := make([]*models.AlpacaDevice, 0)
	buffer := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			c.logger.Warn("Error reading discovery response", zap.Error(err))
			continue
		}

		// Parse discovery response
		var discResp models.DiscoveryResponse
		if err := json.Unmarshal(buffer[:n], &discResp); err != nil {
			c.logger.Warn("Failed to parse discovery response", zap.Error(err))
			continue
		}

		serverURL := fmt.Sprintf("http://%s:%d", addr.IP.String(), discResp.AlpacaPort)
		c.logger.Info("Discovered Alpaca server",
			zap.String("server_url", serverURL),
			zap.Int("port", discResp.AlpacaPort))

		// Query available devices from the server
		serverDevices, err := c.GetConfiguredDevices(ctx, serverURL)
		if err != nil {
			c.logger.Error("Failed to get configured devices",
				zap.String("server_url", serverURL),
				zap.Error(err))
			continue
		}

		devices = append(devices, serverDevices...)
	}

	c.logger.Info("Discovery complete", zap.Int("device_count", len(devices)))
	return devices, nil
}

// GetConfiguredDevices retrieves the list of configured devices from an Alpaca server.
func (c *Client) GetConfiguredDevices(ctx context.Context, serverURL string) ([]*models.AlpacaDevice, error) {
	endpoint := fmt.Sprintf("%s/management/v1/configureddevices", serverURL)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get configured devices: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp struct {
		Value []struct {
			DeviceName   string `json:"DeviceName"`
			DeviceType   string `json:"DeviceType"`
			DeviceNumber int    `json:"DeviceNumber"`
			UniqueID     string `json:"UniqueID"`
		} `json:"Value"`
		models.AlpacaResponse
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.ErrorNumber != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.ErrorNumber, apiResp.ErrorMessage)
	}

	devices := make([]*models.AlpacaDevice, 0, len(apiResp.Value))
	for _, dev := range apiResp.Value {
		device := &models.AlpacaDevice{
			DeviceID:     fmt.Sprintf("%s-%s-%d", serverURL, dev.DeviceType, dev.DeviceNumber),
			DeviceType:   dev.DeviceType,
			DeviceNumber: dev.DeviceNumber,
			Name:         dev.DeviceName,
			ServerURL:    serverURL,
			UUID:         dev.UniqueID,
			LastSeen:     time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// buildURL constructs an Alpaca API endpoint URL.
func (c *Client) buildURL(serverURL, deviceType string, deviceNumber int, method string) string {
	return fmt.Sprintf("%s/api/v1/%s/%d/%s",
		serverURL, deviceType, deviceNumber, method)
}

// Get executes an HTTP GET request to an Alpaca device.
func (c *Client) Get(ctx context.Context, serverURL, deviceType string, deviceNumber int, method string) (*models.AlpacaValueResponse, error) {
	endpoint := c.buildURL(serverURL, deviceType, deviceNumber, method)

	// Add query parameters
	params := url.Values{}
	params.Add("ClientID", fmt.Sprintf("%d", c.clientID))
	params.Add("ClientTransactionID", fmt.Sprintf("%d", c.nextTransactionID()))

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp models.AlpacaValueResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.ErrorNumber != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.ErrorNumber, apiResp.ErrorMessage)
	}

	return &apiResp, nil
}

// Put executes an HTTP PUT request to an Alpaca device.
func (c *Client) Put(ctx context.Context, serverURL, deviceType string, deviceNumber int, method string, params map[string]interface{}) (*models.AlpacaResponse, error) {
	endpoint := c.buildURL(serverURL, deviceType, deviceNumber, method)

	// Prepare form data
	formData := url.Values{}
	formData.Add("ClientID", fmt.Sprintf("%d", c.clientID))
	formData.Add("ClientTransactionID", fmt.Sprintf("%d", c.nextTransactionID()))

	for key, value := range params {
		formData.Add(key, fmt.Sprintf("%v", value))
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", endpoint, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp models.AlpacaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.ErrorNumber != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.ErrorNumber, apiResp.ErrorMessage)
	}

	return &apiResp, nil
}

// Common device methods

// Connect connects to a device.
func (c *Client) Connect(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "connected", map[string]interface{}{
		"Connected": true,
	})
	return err
}

// Disconnect disconnects from a device.
func (c *Client) Disconnect(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "connected", map[string]interface{}{
		"Connected": false,
	})
	return err
}

// IsConnected checks if a device is connected.
func (c *Client) IsConnected(ctx context.Context, device *models.AlpacaDevice) (bool, error) {
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "connected")
	if err != nil {
		return false, err
	}

	connected, ok := resp.Value.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected value type for connected: %T", resp.Value)
	}

	return connected, nil
}

// GetName retrieves the device name.
func (c *Client) GetName(ctx context.Context, device *models.AlpacaDevice) (string, error) {
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "name")
	if err != nil {
		return "", err
	}

	name, ok := resp.Value.(string)
	if !ok {
		return "", fmt.Errorf("unexpected value type for name: %T", resp.Value)
	}

	return name, nil
}

// GetDescription retrieves the device description.
func (c *Client) GetDescription(ctx context.Context, device *models.AlpacaDevice) (string, error) {
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "description")
	if err != nil {
		return "", err
	}

	description, ok := resp.Value.(string)
	if !ok {
		return "", fmt.Errorf("unexpected value type for description: %T", resp.Value)
	}

	return description, nil
}

// Telescope-specific methods

// GetTelescopeStatus retrieves comprehensive telescope status.
func (c *Client) GetTelescopeStatus(ctx context.Context, device *models.AlpacaDevice) (*models.TelescopeStatus, error) {
	status := &models.TelescopeStatus{}

	// Get connected status
	connected, err := c.IsConnected(ctx, device)
	if err != nil {
		return nil, err
	}
	status.Connected = connected

	if !connected {
		return status, nil
	}

	// Get tracking status
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "tracking")
	if err == nil {
		if tracking, ok := resp.Value.(bool); ok {
			status.Tracking = tracking
		}
	}

	// Get slewing status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "slewing")
	if err == nil {
		if slewing, ok := resp.Value.(bool); ok {
			status.Slewing = slewing
		}
	}

	// Get park status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "atpark")
	if err == nil {
		if atPark, ok := resp.Value.(bool); ok {
			status.AtPark = atPark
		}
	}

	// Get coordinates
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "rightascension")
	if err == nil {
		if ra, ok := resp.Value.(float64); ok {
			status.RightAscension = ra
		}
	}

	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "declination")
	if err == nil {
		if dec, ok := resp.Value.(float64); ok {
			status.Declination = dec
		}
	}

	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "altitude")
	if err == nil {
		if alt, ok := resp.Value.(float64); ok {
			status.Altitude = alt
		}
	}

	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "azimuth")
	if err == nil {
		if az, ok := resp.Value.(float64); ok {
			status.Azimuth = az
		}
	}

	return status, nil
}

// SlewToCoordinates slews telescope to specified RA/Dec coordinates.
func (c *Client) SlewToCoordinates(ctx context.Context, device *models.AlpacaDevice, ra, dec float64) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "slewtocoordinates", map[string]interface{}{
		"RightAscension": ra,
		"Declination":    dec,
	})
	return err
}

// Park parks the telescope.
func (c *Client) Park(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "park", map[string]interface{}{})
	return err
}

// Unpark unparks the telescope.
func (c *Client) Unpark(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "unpark", map[string]interface{}{})
	return err
}

// SetTracking enables or disables telescope tracking.
func (c *Client) SetTracking(ctx context.Context, device *models.AlpacaDevice, tracking bool) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "tracking", map[string]interface{}{
		"Tracking": tracking,
	})
	return err
}

// AbortSlew aborts the current slew operation.
func (c *Client) AbortSlew(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "abortslew", map[string]interface{}{})
	return err
}

// Camera-specific methods

// GetCameraStatus retrieves comprehensive camera status.
func (c *Client) GetCameraStatus(ctx context.Context, device *models.AlpacaDevice) (*models.CameraStatus, error) {
	status := &models.CameraStatus{}

	// Get connected status
	connected, err := c.IsConnected(ctx, device)
	if err != nil {
		return nil, err
	}
	status.Connected = connected

	if !connected {
		return status, nil
	}

	// Get camera state
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "camerastate")
	if err == nil {
		if state, ok := resp.Value.(float64); ok {
			// ASCOM camera state enum
			stateMap := map[int]string{
				0: "Idle", 1: "Waiting", 2: "Exposing",
				3: "Reading", 4: "Download", 5: "Error",
			}
			status.CameraState = stateMap[int(state)]
		}
	}

	// Get CCD temperature
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "ccdtemperature")
	if err == nil {
		if temp, ok := resp.Value.(float64); ok {
			status.CCDTemperature = temp
		}
	}

	// Get cooler status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "cooleron")
	if err == nil {
		if coolerOn, ok := resp.Value.(bool); ok {
			status.CoolerOn = coolerOn
		}
	}

	// Get cooler power
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "coolerpower")
	if err == nil {
		if power, ok := resp.Value.(float64); ok {
			status.CoolerPower = power
		}
	}

	// Get image ready status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "imageready")
	if err == nil {
		if ready, ok := resp.Value.(bool); ok {
			status.ImageReady = ready
		}
	}

	// Get percent completed
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "percentcompleted")
	if err == nil {
		if pct, ok := resp.Value.(float64); ok {
			status.PercentCompleted = int(pct)
		}
	}

	return status, nil
}

// StartExposure starts a camera exposure.
func (c *Client) StartExposure(ctx context.Context, device *models.AlpacaDevice, duration float64, light bool) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "startexposure", map[string]interface{}{
		"Duration": duration,
		"Light":    light,
	})
	return err
}

// StopExposure stops the current exposure.
func (c *Client) StopExposure(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "stopexposure", map[string]interface{}{})
	return err
}

// AbortExposure aborts the current exposure.
func (c *Client) AbortExposure(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "abortexposure", map[string]interface{}{})
	return err
}

// SetCoolerOn enables or disables the CCD cooler.
func (c *Client) SetCoolerOn(ctx context.Context, device *models.AlpacaDevice, coolerOn bool) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "cooleron", map[string]interface{}{
		"CoolerOn": coolerOn,
	})
	return err
}

// Dome-specific methods

// GetDomeStatus retrieves comprehensive dome status.
func (c *Client) GetDomeStatus(ctx context.Context, device *models.AlpacaDevice) (*models.DomeStatus, error) {
	status := &models.DomeStatus{}

	// Get connected status
	connected, err := c.IsConnected(ctx, device)
	if err != nil {
		return nil, err
	}
	status.Connected = connected

	if !connected {
		return status, nil
	}

	// Get at home status
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "athome")
	if err == nil {
		if atHome, ok := resp.Value.(bool); ok {
			status.AtHome = atHome
		}
	}

	// Get at park status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "atpark")
	if err == nil {
		if atPark, ok := resp.Value.(bool); ok {
			status.AtPark = atPark
		}
	}

	// Get slewing status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "slewing")
	if err == nil {
		if slewing, ok := resp.Value.(bool); ok {
			status.Slewing = slewing
		}
	}

	// Get azimuth
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "azimuth")
	if err == nil {
		if az, ok := resp.Value.(float64); ok {
			status.Azimuth = az
		}
	}

	// Get shutter status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "shutterstatus")
	if err == nil {
		if shutterState, ok := resp.Value.(float64); ok {
			// ASCOM shutter state enum
			shutterMap := map[int]string{
				0: "Open", 1: "Closed", 2: "Opening",
				3: "Closing", 4: "Error",
			}
			status.ShutterStatus = shutterMap[int(shutterState)]
		}
	}

	return status, nil
}

// SlewDomeToAzimuth slews dome to specified azimuth.
func (c *Client) SlewDomeToAzimuth(ctx context.Context, device *models.AlpacaDevice, azimuth float64) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "slewtoazimuth", map[string]interface{}{
		"Azimuth": azimuth,
	})
	return err
}

// OpenDomeShutter opens the dome shutter.
func (c *Client) OpenDomeShutter(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "openshutter", map[string]interface{}{})
	return err
}

// CloseDomeShutter closes the dome shutter.
func (c *Client) CloseDomeShutter(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "closeshutter", map[string]interface{}{})
	return err
}

// Focuser-specific methods

// GetFocuserStatus retrieves comprehensive focuser status.
func (c *Client) GetFocuserStatus(ctx context.Context, device *models.AlpacaDevice) (*models.FocuserStatus, error) {
	status := &models.FocuserStatus{}

	// Get connected status
	connected, err := c.IsConnected(ctx, device)
	if err != nil {
		return nil, err
	}
	status.Connected = connected

	if !connected {
		return status, nil
	}

	// Get is moving status
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "ismoving")
	if err == nil {
		if isMoving, ok := resp.Value.(bool); ok {
			status.IsMoving = isMoving
		}
	}

	// Get position
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "position")
	if err == nil {
		if pos, ok := resp.Value.(float64); ok {
			status.Position = int(pos)
		}
	}

	// Get max step
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "maxstep")
	if err == nil {
		if maxStep, ok := resp.Value.(float64); ok {
			status.MaxStep = int(maxStep)
		}
	}

	// Get temperature compensation status
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "tempcomp")
	if err == nil {
		if tempComp, ok := resp.Value.(bool); ok {
			status.TempComp = tempComp
		}
	}

	// Get temperature
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "temperature")
	if err == nil {
		if temp, ok := resp.Value.(float64); ok {
			status.Temperature = temp
		}
	}

	return status, nil
}

// MoveFocuser moves focuser to absolute position.
func (c *Client) MoveFocuser(ctx context.Context, device *models.AlpacaDevice, position int) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "move", map[string]interface{}{
		"Position": position,
	})
	return err
}

// HaltFocuser stops any focuser movement.
func (c *Client) HaltFocuser(ctx context.Context, device *models.AlpacaDevice) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "halt", map[string]interface{}{})
	return err
}

// FilterWheel-specific methods

// GetFilterWheelStatus retrieves comprehensive filter wheel status.
func (c *Client) GetFilterWheelStatus(ctx context.Context, device *models.AlpacaDevice) (*models.FilterWheelStatus, error) {
	status := &models.FilterWheelStatus{}

	// Get connected status
	connected, err := c.IsConnected(ctx, device)
	if err != nil {
		return nil, err
	}
	status.Connected = connected

	if !connected {
		return status, nil
	}

	// Get position
	resp, err := c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "position")
	if err == nil {
		if pos, ok := resp.Value.(float64); ok {
			status.Position = int(pos)
		}
	}

	// Get filter names
	resp, err = c.Get(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "names")
	if err == nil {
		if names, ok := resp.Value.([]interface{}); ok {
			status.Names = make([]string, len(names))
			for i, name := range names {
				if nameStr, ok := name.(string); ok {
					status.Names[i] = nameStr
				}
			}
		}
	}

	return status, nil
}

// SetFilterWheelPosition sets the filter wheel to a specific position.
func (c *Client) SetFilterWheelPosition(ctx context.Context, device *models.AlpacaDevice, position int) error {
	_, err := c.Put(ctx, device.ServerURL, device.DeviceType, device.DeviceNumber, "position", map[string]interface{}{
		"Position": position,
	})
	return err
}
