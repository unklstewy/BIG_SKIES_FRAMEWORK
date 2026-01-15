// Package mqtt provides an MQTT client wrapper with automatic reconnection and JSON message support.
package mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// Client wraps the MQTT client with additional functionality.
type Client struct {
	client mqtt.Client
	logger *zap.Logger
	config *Config
}

// Config holds MQTT client configuration.
type Config struct {
	// BrokerURL is the MQTT broker URL (e.g., "tcp://localhost:1883")
	BrokerURL string
	// ClientID is the unique identifier for this client
	ClientID string
	// Username for MQTT authentication (optional)
	Username string
	// Password for MQTT authentication (optional)
	Password string
	// KeepAlive interval in seconds
	KeepAlive time.Duration
	// ConnectTimeout in seconds
	ConnectTimeout time.Duration
	// AutoReconnect enables automatic reconnection
	AutoReconnect bool
	// MaxReconnectInterval is the maximum time between reconnection attempts
	MaxReconnectInterval time.Duration
}

// MessageHandler is a callback function for handling received messages.
type MessageHandler func(topic string, payload []byte) error

// NewClient creates a new MQTT client with the given configuration.
func NewClient(config *Config, logger *zap.Logger) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.BrokerURL)
	opts.SetClientID(config.ClientID)

	if config.Username != "" {
		opts.SetUsername(config.Username)
	}
	if config.Password != "" {
		opts.SetPassword(config.Password)
	}

	opts.SetKeepAlive(config.KeepAlive)
	opts.SetConnectTimeout(config.ConnectTimeout)
	opts.SetAutoReconnect(config.AutoReconnect)
	opts.SetMaxReconnectInterval(config.MaxReconnectInterval)

	// Connection lost handler
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Error("MQTT connection lost", zap.Error(err))
	})

	// On connect handler
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("MQTT connected", zap.String("broker", config.BrokerURL))
	})

	// Reconnect handler
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		logger.Info("MQTT reconnecting...")
	})

	mqttClient := mqtt.NewClient(opts)

	return &Client{
		client: mqttClient,
		logger: logger,
		config: config,
	}, nil
}

// Connect establishes connection to the MQTT broker.
func (c *Client) Connect() error {
	c.logger.Info("Connecting to MQTT broker", zap.String("broker", c.config.BrokerURL))

	token := c.client.Connect()
	if !token.WaitTimeout(c.config.ConnectTimeout) {
		return fmt.Errorf("connection timeout after %v", c.config.ConnectTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

// Disconnect closes the connection to the MQTT broker.
func (c *Client) Disconnect() {
	c.logger.Info("Disconnecting from MQTT broker")
	c.client.Disconnect(250) // 250ms grace period
}

// IsConnected returns true if the client is connected to the broker.
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Publish sends a message to the specified topic.
func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := c.client.Publish(topic, qos, retained, payload)
	token.Wait()

	if err := token.Error(); err != nil {
		c.logger.Error("Failed to publish message",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("publish failed: %w", err)
	}

	c.logger.Debug("Message published",
		zap.String("topic", topic),
		zap.Int("size", len(payload)))

	return nil
}

// PublishJSON serializes the payload to JSON and publishes it.
func (c *Client) PublishJSON(topic string, qos byte, retained bool, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.Publish(topic, qos, retained, data)
}

// Subscribe subscribes to a topic with the given handler.
func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	callback := func(client mqtt.Client, msg mqtt.Message) {
		c.logger.Debug("Message received",
			zap.String("topic", msg.Topic()),
			zap.Int("size", len(msg.Payload())))

		if err := handler(msg.Topic(), msg.Payload()); err != nil {
			c.logger.Error("Handler error",
				zap.String("topic", msg.Topic()),
				zap.Error(err))
		}
	}

	token := c.client.Subscribe(topic, qos, callback)
	token.Wait()

	if err := token.Error(); err != nil {
		c.logger.Error("Failed to subscribe",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("subscribe failed: %w", err)
	}

	c.logger.Info("Subscribed to topic", zap.String("topic", topic))
	return nil
}

// Unsubscribe unsubscribes from the specified topic.
func (c *Client) Unsubscribe(topic string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := c.client.Unsubscribe(topic)
	token.Wait()

	if err := token.Error(); err != nil {
		c.logger.Error("Failed to unsubscribe",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("unsubscribe failed: %w", err)
	}

	c.logger.Info("Unsubscribed from topic", zap.String("topic", topic))
	return nil
}
