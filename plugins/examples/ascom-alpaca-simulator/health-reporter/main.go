package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// HealthStatus represents the health status message format for BIG SKIES Framework
type HealthStatus struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   Payload   `json:"payload"`
}

// Payload contains the health status details
type Payload struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"` // healthy, degraded, unhealthy
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
}

// ASCOMVersions represents the ASCOM API versions response
type ASCOMVersions struct {
	Value         []int  `json:"Value"`
	ErrorNumber   int    `json:"ErrorNumber"`
	ErrorMessage  string `json:"ErrorMessage"`
	ClientTransID uint32 `json:"ClientTransactionID"`
	ServerTransID uint32 `json:"ServerTransactionID"`
}

var (
	pluginID   = os.Getenv("PLUGIN_ID")
	pluginName = os.Getenv("PLUGIN_NAME")
	mqttBroker = os.Getenv("MQTT_BROKER")
	logLevel   = os.Getenv("LOG_LEVEL")
	
	startTime = time.Now()
	ascomURL  = "http://localhost/api/v1/management/apiversions"
)

func main() {
	// Set defaults
	if pluginID == "" {
		pluginID = "f7e8d9c6-b5a4-3210-9876-543210fedcba"
	}
	if pluginName == "" {
		pluginName = "ASCOM Alpaca Simulators"
	}
	if mqttBroker == "" {
		mqttBroker = "tcp://mqtt-broker:1883"
	}
	if logLevel == "" {
		logLevel = "info"
	}

	log.Printf("Starting health reporter for plugin: %s", pluginName)
	log.Printf("Plugin ID: %s", pluginID)
	log.Printf("MQTT Broker: %s", mqttBroker)

	// Configure MQTT client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(fmt.Sprintf("plugin-%s-health", pluginID))
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
	
	// Wait for ASCOM to be ready before connecting to MQTT
	log.Println("Waiting for ASCOM Alpaca API to be ready...")
	waitForASCOM(30 * time.Second)

	// Connect to MQTT
	log.Println("Connecting to MQTT broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Println("Health reporter started successfully")

	// Start health reporting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go publishHealth(ctx, client)

	// Subscribe to control topics
	controlTopic := fmt.Sprintf("bigskies/plugin/%s/control/#", pluginID)
	if token := client.Subscribe(controlTopic, 1, handleControlMessage); token.Wait() && token.Error() != nil {
		log.Printf("Warning: Failed to subscribe to control topic: %v", token.Error())
	} else {
		log.Printf("Subscribed to control topic: %s", controlTopic)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Printf("Received signal %v, shutting down...", sig)
	
	// Send final unhealthy status
	sendHealthStatus(client, "unhealthy", "Plugin shutting down")
	
	time.Sleep(500 * time.Millisecond) // Allow final message to send
	client.Disconnect(250)
	log.Println("Health reporter stopped")
}

// waitForASCOM waits for the ASCOM API to become available
func waitForASCOM(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if checkASCOMHealth() {
			log.Println("ASCOM Alpaca API is ready")
			return
		}
		<-ticker.C
	}
	
	log.Println("Warning: ASCOM API not responding, continuing anyway...")
}

// publishHealth publishes health status every 30 seconds
func publishHealth(ctx context.Context, client mqtt.Client) {
	// Send initial status immediately
	sendHealthStatus(client, "healthy", "Plugin started and operational")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check ASCOM health
			if checkASCOMHealth() {
				sendHealthStatus(client, "healthy", "Plugin operational, ASCOM API responding")
			} else {
				sendHealthStatus(client, "degraded", "Plugin operational, but ASCOM API not responding")
			}
		}
	}
}

// sendHealthStatus publishes a health status message
func sendHealthStatus(client mqtt.Client, status, message string) {
	uptime := time.Since(startTime).Seconds()
	
	health := HealthStatus{
		ID:        fmt.Sprintf("%s.%06d", time.Now().Format("20060102150405"), time.Now().Nanosecond()/1000),
		Source:    fmt.Sprintf("plugin:%s", pluginID),
		Type:      "status",
		Timestamp: time.Now(),
		Payload: Payload{
			Component: pluginName,
			Status:    status,
			Message:   message,
			Details: map[string]interface{}{
				"running":        true,
				"uptime_seconds": uptime,
				"ascom_api_url":  ascomURL,
			},
		},
	}

	payload, err := json.Marshal(health)
	if err != nil {
		log.Printf("Error marshaling health status: %v", err)
		return
	}

	topic := fmt.Sprintf("bigskies/plugin/%s/health", pluginID)
	token := client.Publish(topic, 1, false, payload)
	token.Wait()
	
	if token.Error() != nil {
		log.Printf("Error publishing health status: %v", token.Error())
	} else if logLevel == "debug" {
		log.Printf("Published health status: %s", status)
	}
}

// checkASCOMHealth checks if ASCOM API is responding
func checkASCOMHealth() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ascomURL, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var versions ASCOMVersions
	if err := json.Unmarshal(body, &versions); err != nil {
		return false
	}

	// Check if we got valid API versions
	return len(versions.Value) > 0 && versions.ErrorNumber == 0
}

// handleControlMessage handles control messages from the framework
func handleControlMessage(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received control message on topic %s: %s", msg.Topic(), string(msg.Payload()))
	
	// Parse control message
	var control map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &control); err != nil {
		log.Printf("Error parsing control message: %v", err)
		return
	}

	// Handle different control commands
	command, ok := control["command"].(string)
	if !ok {
		log.Println("Control message missing 'command' field")
		return
	}

	switch command {
	case "ping":
		log.Println("Received ping, sending health status...")
		sendHealthStatus(client, "healthy", "Responding to ping command")
	case "status":
		log.Println("Received status request, sending health status...")
		sendHealthStatus(client, "healthy", "Responding to status request")
	default:
		log.Printf("Unknown control command: %s", command)
	}
}
