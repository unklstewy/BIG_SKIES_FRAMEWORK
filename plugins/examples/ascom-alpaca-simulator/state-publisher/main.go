// State publisher service for ASCOM Alpaca Simulator plugin
// Polls ASCOM API and publishes device state to MQTT
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	pluginID        = os.Getenv("PLUGIN_ID")
	mqttBroker      = os.Getenv("MQTT_BROKER")
	ascomBaseURL    = os.Getenv("ASCOM_BASE_URL")
	publishInterval = 1 * time.Second // Poll every second
)

// DeviceState represents the state of an ASCOM device
type DeviceState struct {
	Connected bool   `json:"connected"`
	Name      string `json:"name,omitempty"`
}

// TelescopeState represents extended telescope state
type TelescopeState struct {
	DeviceState
	RightAscension float64 `json:"right_ascension,omitempty"`
	Declination    float64 `json:"declination,omitempty"`
	Altitude       float64 `json:"altitude,omitempty"`
	Azimuth        float64 `json:"azimuth,omitempty"`
	Slewing        bool    `json:"slewing,omitempty"`
	Tracking       bool    `json:"tracking,omitempty"`
	AtPark         bool    `json:"at_park,omitempty"`
}

func main() {
	// Set defaults
	if pluginID == "" {
		pluginID = "f7e8d9c6-b5a4-3210-9876-543210fedcba"
	}
	if mqttBroker == "" {
		mqttBroker = "tcp://mqtt-broker:1883"
	}
	if ascomBaseURL == "" {
		ascomBaseURL = "http://localhost/api/v1"
	}

	log.Printf("Starting state publisher for plugin: %s", pluginID)
	log.Printf("MQTT Broker: %s", mqttBroker)
	log.Printf("ASCOM Base URL: %s", ascomBaseURL)

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(fmt.Sprintf("plugin-%s-state", pluginID))
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to MQTT broker")
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("Lost connection to MQTT broker: %v", err)
	})

	client := mqtt.NewClient(opts)

	// Connect to MQTT broker
	log.Println("Connecting to MQTT broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Println("State publisher started successfully")

	// Start publishing state
	stopChan := make(chan struct{})
	go publishStateLoop(client, stopChan)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Printf("Received signal %v, shutting down...", sig)
	close(stopChan)

	// Disconnect from MQTT
	client.Disconnect(250)
	log.Println("State publisher stopped")
}

// publishStateLoop continuously polls and publishes device state
func publishStateLoop(client mqtt.Client, stopChan chan struct{}) {
	ticker := time.NewTicker(publishInterval)
	defer ticker.Stop()

	devices := []string{"telescope", "camera", "filterwheel", "focuser", "switch"}

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Publish state for each device
			for _, device := range devices {
				if device == "telescope" {
					publishTelescopeState(client, device)
				} else {
					publishDeviceState(client, device)
				}
			}
		}
	}
}

// publishDeviceState publishes basic device state (connected/disconnected)
func publishDeviceState(client mqtt.Client, deviceType string) {
	connected, err := getDeviceConnected(deviceType)
	if err != nil {
		// Device not available, skip
		return
	}

	state := DeviceState{
		Connected: connected,
		Name:      deviceType,
	}

	// Publish to bigskies/plugin/{plugin_id}/device/{device_type}/state
	topic := fmt.Sprintf("bigskies/plugin/%s/device/%s/state", pluginID, deviceType)
	publishJSON(client, topic, state)
}

// publishTelescopeState publishes detailed telescope state
func publishTelescopeState(client mqtt.Client, deviceType string) {
	connected, err := getDeviceConnected(deviceType)
	if err != nil {
		// Device not available, skip
		return
	}

	state := TelescopeState{
		DeviceState: DeviceState{
			Connected: connected,
			Name:      deviceType,
		},
	}

	// If connected, get additional telescope properties
	if connected {
		if ra, err := getTelescopeProperty("rightascension"); err == nil {
			state.RightAscension = ra
		}
		if dec, err := getTelescopeProperty("declination"); err == nil {
			state.Declination = dec
		}
		if alt, err := getTelescopeProperty("altitude"); err == nil {
			state.Altitude = alt
		}
		if az, err := getTelescopeProperty("azimuth"); err == nil {
			state.Azimuth = az
		}
		if slewing, err := getTelescopeBool("slewing"); err == nil {
			state.Slewing = slewing
		}
		if tracking, err := getTelescopeBool("tracking"); err == nil {
			state.Tracking = tracking
		}
		if atPark, err := getTelescopeBool("atpark"); err == nil {
			state.AtPark = atPark
		}
	}

	// Publish to bigskies/plugin/{plugin_id}/device/telescope/state
	topic := fmt.Sprintf("bigskies/plugin/%s/device/telescope/state", pluginID)
	publishJSON(client, topic, state)
}

// getDeviceConnected checks if a device is connected
func getDeviceConnected(deviceType string) (bool, error) {
	apiURL := fmt.Sprintf("%s/%s/0/connected", ascomBaseURL, deviceType)

	params := url.Values{}
	params.Set("ClientID", "1")
	params.Set("ClientTransactionID", "1")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	// Check for ASCOM errors
	if errorNum, ok := result["ErrorNumber"].(float64); ok && errorNum != 0 {
		return false, fmt.Errorf("ASCOM error %v", errorNum)
	}

	if value, ok := result["Value"].(bool); ok {
		return value, nil
	}

	return false, fmt.Errorf("unexpected response format")
}

// getTelescopeProperty gets a float64 property from telescope
func getTelescopeProperty(property string) (float64, error) {
	apiURL := fmt.Sprintf("%s/telescope/0/%s", ascomBaseURL, property)

	params := url.Values{}
	params.Set("ClientID", "1")
	params.Set("ClientTransactionID", "1")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	// Check for ASCOM errors
	if errorNum, ok := result["ErrorNumber"].(float64); ok && errorNum != 0 {
		return 0, fmt.Errorf("ASCOM error %v", errorNum)
	}

	if value, ok := result["Value"].(float64); ok {
		return value, nil
	}

	return 0, fmt.Errorf("unexpected response format")
}

// getTelescopeBool gets a boolean property from telescope
func getTelescopeBool(property string) (bool, error) {
	apiURL := fmt.Sprintf("%s/telescope/0/%s", ascomBaseURL, property)

	params := url.Values{}
	params.Set("ClientID", "1")
	params.Set("ClientTransactionID", "1")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	// Check for ASCOM errors
	if errorNum, ok := result["ErrorNumber"].(float64); ok && errorNum != 0 {
		return false, fmt.Errorf("ASCOM error %v", errorNum)
	}

	if value, ok := result["Value"].(bool); ok {
		return value, nil
	}

	return false, fmt.Errorf("unexpected response format")
}

// publishJSON publishes a JSON payload to MQTT
func publishJSON(client mqtt.Client, topic string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return
	}

	token := client.Publish(topic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		log.Printf("Error publishing to %s: %v", topic, token.Error())
	}
}
