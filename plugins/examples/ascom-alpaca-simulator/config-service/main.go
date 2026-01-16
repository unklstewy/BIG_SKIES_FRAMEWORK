// Configuration service for ASCOM Alpaca Simulator plugin
// Handles dynamic configuration loading via MQTT
package main

import (
	"fmt"
	"log"
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

	// Log current configuration status on startup
	status, err := loader.GetCurrentStatus()
	if err != nil {
		log.Printf("Warning: Failed to get current status: %v", err)
	} else if status != nil {
		log.Printf("Current configuration: %s/%s (loaded at %s)",
			status.Model, status.MountType, status.LoadedAt.Format(time.RFC3339))
	} else {
		log.Println("No configuration currently loaded")
	}

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
