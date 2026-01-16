// Configuration service for ASCOM Alpaca Simulator plugin
// Handles dynamic configuration loading via MQTT
package main

import (
	"bytes"
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
	pluginID   = os.Getenv("PLUGIN_ID")
	mqttBroker = os.Getenv("MQTT_BROKER")
	logLevel   = os.Getenv("LOG_LEVEL")
)

func main() {
	// Set defaults
	if pluginID == "" {
		pluginID = "f7e8d9c6-b5a4-3210-9876-543210fedcba"
	}
	if mqttBroker == "" {
		mqttBroker = "tcp://mqtt-broker:1883"
	}
	if logLevel == "" {
		logLevel = "info"
	}

	log.Printf("Starting configuration service for plugin: %s", pluginID)
	log.Printf("MQTT Broker: %s", mqttBroker)

	// Create configuration loader
	loader := NewConfigLoader()

	// Load configuration on startup
	loadConfigurationOnStartup(loader)

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(fmt.Sprintf("plugin-%s-config", pluginID))
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to MQTT broker")
		subscribeToTopics(client, pluginID, loader)
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

	log.Println("Configuration service started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Printf("Received signal %v, shutting down...", sig)

	// Disconnect from MQTT
	client.Disconnect(250)
	log.Println("Configuration service stopped")
}

// loadConfigurationOnStartup loads the last used configuration or default on startup.
func loadConfigurationOnStartup(loader *ConfigLoader) {
	const (
		defaultModel     = "s30"
		defaultMountType = "equatorial"
	)

	// Check if there's a previously saved configuration
	status, err := loader.GetCurrentStatus()
	if err != nil {
		log.Printf("Warning: Failed to get current status: %v", err)
	}

	var model, mountType string
	if status != nil {
		// Use last loaded configuration
		model = status.Model
		mountType = status.MountType
		log.Printf("Found previous configuration: %s/%s (loaded at %s)",
			status.Model, status.MountType, status.LoadedAt.Format(time.RFC3339))
	} else {
		// Use default configuration
		model = defaultModel
		mountType = defaultMountType
		log.Printf("No previous configuration found, using default: %s/%s", model, mountType)
	}

	// Load the configuration
	log.Printf("Loading configuration on startup: %s/%s", model, mountType)
	if err := loader.LoadConfiguration(model, mountType, "startup"); err != nil {
		log.Printf("ERROR: Failed to load configuration on startup: %v", err)
		log.Printf("Container will continue without pre-loaded configuration")
		return
	}

	log.Printf("Configuration loaded successfully: %s/%s", model, mountType)

	// Wait a moment for ASCOM to reload
	time.Sleep(3 * time.Second)

	// Connect all devices
	log.Println("Connecting all devices...")
	devices := []string{"telescope", "camera", "filterwheel", "focuser", "switch"}
	connected := 0

	for _, device := range devices {
		if err := connectDevice(device); err != nil {
			log.Printf("Failed to connect %s: %v", device, err)
		} else {
			log.Printf("Connected %s successfully", device)
			connected++
		}
	}

	log.Printf("Startup configuration complete: %d/%d devices connected", connected, len(devices))
}

// connectDevice connects a single ASCOM device via HTTP API.
func connectDevice(deviceType string) error {
	apiURL := fmt.Sprintf("http://localhost/api/v1/%s/0/connected", deviceType)

	// Prepare form data
	data := url.Values{}
	data.Set("Connected", "true")
	data.Set("ClientID", "1")
	data.Set("ClientTransactionID", "1")

	// Create PUT request
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if errorNum, ok := result["ErrorNumber"].(float64); ok && errorNum != 0 {
		errorMsg := result["ErrorMessage"].(string)
		return fmt.Errorf("ASCOM error %v: %s", errorNum, errorMsg)
	}

	return nil
}

// subscribeToTopics subscribes to all configuration-related MQTT topics.
func subscribeToTopics(client mqtt.Client, pluginID string, loader *ConfigLoader) {
	// Create handler
	handler := NewConfigHandler(loader, client, pluginID)

	// Subscribe to configuration topics
	topics := map[string]mqtt.MessageHandler{
		fmt.Sprintf("bigskies/plugin/%s/config/load", pluginID):   handler.HandleMessage,
		fmt.Sprintf("bigskies/plugin/%s/config/list", pluginID):   handler.HandleMessage,
		fmt.Sprintf("bigskies/plugin/%s/config/status", pluginID): handler.HandleMessage,
	}

	for topic, msgHandler := range topics {
		if token := client.Subscribe(topic, 1, msgHandler); token.Wait() && token.Error() != nil {
			log.Printf("Warning: Failed to subscribe to %s: %v", topic, token.Error())
		} else {
			log.Printf("Subscribed to topic: %s", topic)
		}
	}
}
