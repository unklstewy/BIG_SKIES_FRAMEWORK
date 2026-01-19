// Command bridge service for ASCOM Alpaca Simulator plugin
// Receives MQTT commands from framework and translates them to ASCOM API calls
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
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	pluginID     = os.Getenv("PLUGIN_ID")
	mqttBroker   = os.Getenv("MQTT_BROKER")
	ascomBaseURL = os.Getenv("ASCOM_BASE_URL")
)

// Command represents an MQTT command
type Command struct {
	DeviceID       string  `json:"device_id"`
	DeviceType     string  `json:"device_type"`
	DeviceNumber   int     `json:"device_number"`
	Action         string  `json:"action"`
	RightAscension float64 `json:"right_ascension,omitempty"`
	Declination    float64 `json:"declination,omitempty"`
	Tracking       bool    `json:"tracking,omitempty"`
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

	log.Printf("Starting command bridge for plugin: %s", pluginID)
	log.Printf("MQTT Broker: %s", mqttBroker)
	log.Printf("ASCOM Base URL: %s", ascomBaseURL)

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(fmt.Sprintf("plugin-%s-commands", pluginID))
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to MQTT broker")
		subscribeToCommands(client)
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

	log.Println("Command bridge started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Printf("Received signal %v, shutting down...", sig)

	// Disconnect from MQTT
	client.Disconnect(250)
	log.Println("Command bridge stopped")
}

// subscribeToCommands subscribes to command topics
func subscribeToCommands(client mqtt.Client) {
	// Subscribe to plugin-specific command topic
	commandTopic := fmt.Sprintf("bigskies/plugin/%s/command/#", pluginID)

	token := client.Subscribe(commandTopic, 1, handleCommand)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe to commands: %v", token.Error())
	}

	log.Printf("Subscribed to: %s", commandTopic)
}

// handleCommand processes incoming MQTT commands
func handleCommand(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received command on topic: %s", msg.Topic())

	var cmd Command
	if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
		log.Printf("Failed to parse command: %v", err)
		return
	}

	// Extract device type and action from topic
	// Topic format: bigskies/plugin/{id}/command/{device}/{action}
	// e.g., bigskies/plugin/{id}/command/telescope/slew

	// For now, determine from device_id or explicit fields
	deviceType := cmd.DeviceType
	if deviceType == "" && cmd.DeviceID != "" {
		// Parse from device_id (format: host:port-devicetype-number)
		deviceType = "telescope" // Default for now
	}

	// Route to appropriate handler
	switch cmd.Action {
	case "slew", "slew_to_coordinates":
		handleSlewCommand(cmd, deviceType)
	case "park":
		handleParkCommand(cmd, deviceType)
	case "unpark":
		handleUnparkCommand(cmd, deviceType)
	case "set_tracking":
		handleTrackingCommand(cmd, deviceType)
	case "abort":
		handleAbortCommand(cmd, deviceType)
	case "connect":
		handleConnectCommand(cmd, deviceType)
	case "disconnect":
		handleDisconnectCommand(cmd, deviceType)
	default:
		log.Printf("Unknown action: %s", cmd.Action)
	}
}

// handleSlewCommand sends slew command to ASCOM API
func handleSlewCommand(cmd Command, deviceType string) {
	log.Printf("Slewing %s to RA: %.4f, Dec: %.4f", deviceType, cmd.RightAscension, cmd.Declination)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/slewtocoordinatesasync", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("RightAscension", fmt.Sprintf("%.6f", cmd.RightAscension))
	formData.Set("Declination", fmt.Sprintf("%.6f", cmd.Declination))
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create slew request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to slew: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Slew response: %s", string(body))
}

// handleParkCommand sends park command to ASCOM API
func handleParkCommand(cmd Command, deviceType string) {
	log.Printf("Parking %s", deviceType)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/park", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create park request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to park: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Park response: %s", string(body))
}

// handleUnparkCommand sends unpark command to ASCOM API
func handleUnparkCommand(cmd Command, deviceType string) {
	log.Printf("Unparking %s", deviceType)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/unpark", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create unpark request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to unpark: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Unpark response: %s", string(body))
}

// handleTrackingCommand sets tracking on/off
func handleTrackingCommand(cmd Command, deviceType string) {
	log.Printf("Setting tracking to: %v", cmd.Tracking)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/tracking", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("Tracking", fmt.Sprintf("%t", cmd.Tracking))
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create tracking request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to set tracking: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Tracking response: %s", string(body))
}

// handleAbortCommand aborts current operation
func handleAbortCommand(cmd Command, deviceType string) {
	log.Printf("Aborting %s operation", deviceType)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/abortslew", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create abort request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to abort: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Abort response: %s", string(body))
}

// handleConnectCommand connects to device
func handleConnectCommand(cmd Command, deviceType string) {
	log.Printf("Connecting to %s", deviceType)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/connected", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("Connected", "true")
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create connect request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Connect response: %s", string(body))
}

// handleDisconnectCommand disconnects from device
func handleDisconnectCommand(cmd Command, deviceType string) {
	log.Printf("Disconnecting from %s", deviceType)

	deviceNum := cmd.DeviceNumber
	apiURL := fmt.Sprintf("%s/%s/%d/connected", ascomBaseURL, deviceType, deviceNum)

	// Build form data - ALL parameters in body for PUT
	formData := url.Values{}
	formData.Set("Connected", "false")
	formData.Set("ClientID", "1")
	formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))

	req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create disconnect request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to disconnect: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Disconnect response: %s", string(body))
}
