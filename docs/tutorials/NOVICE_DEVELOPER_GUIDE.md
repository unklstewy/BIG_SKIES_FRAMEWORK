# Big Skies Framework - Complete Novice Developer Guide
## Extensively Commented Edition for Beginners

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Lesson 1: Understanding Big Skies Architecture](#lesson-1-understanding-big-skies-architecture)
4. [Lesson 2: Setting Up Your Development Environment](#lesson-2-setting-up-your-development-environment)
5. [Lesson 3: Your First MQTT Client](#lesson-3-your-first-mqtt-client)
6. [Lesson 4: Building a Simple Web Service](#lesson-4-building-a-simple-web-service)
7. [Lesson 5: Adding Security and Authentication](#lesson-5-adding-security-and-authentication)
8. [Lesson 6: Integrating Telescope Control](#lesson-6-integrating-telescope-control)
9. [Lesson 7: Complete Web Application](#lesson-7-complete-web-application)
10. [Appendix: Common Patterns and Best Practices](#appendix-common-patterns-and-best-practices)

---

## Introduction

Welcome to the Big Skies Framework! This guide will take you from a novice Go developer to building a complete web application that interfaces with the Big Skies backend for telescope control.

**What You'll Build**: A web application that:
- Authenticates users with JWT tokens
- Controls telescopes through the ASCOM Alpaca Simulator
- Monitors system health
- Communicates via MQTT message bus

**Learning Path**: 7 progressive lessons, each building on the previous one.

---

## Prerequisites

### Required Software
- **Go 1.25.5+**: [Download](https://golang.org/dl/)
- **Docker Desktop**: [Download](https://www.docker.com/products/docker-desktop)
- **Git**: [Download](https://git-scm.com/downloads)
- **Code Editor**: VS Code, GoLand, or your preference

### Recommended Knowledge
- Basic programming concepts (variables, functions, loops)
- Basic command line usage
- Understanding of HTTP and REST APIs (helpful but not required)

### Framework Dependencies
The Big Skies Framework uses:
- **MQTT** for message passing
- **PostgreSQL** for data storage
- **Docker** for containerization
- **Gin** for HTTP routing

---

## Lesson 1: Understanding Big Skies Architecture

### The Big Picture

Big Skies is a **coordinator-based microservices architecture**. Think of it like an orchestra:
- **Coordinators** are like section leaders (strings, brass, etc.)
- **MQTT Message Bus** is the conductor
- **Your Application** is a musician playing along

### The 7 Core Coordinators

1. **message-coordinator**: Manages the MQTT broker (the conductor)
2. **security-coordinator**: Handles authentication and authorization
3. **telescope-coordinator**: Controls telescopes via ASCOM Alpaca
4. **datastore-coordinator**: Manages PostgreSQL database
5. **application-svc-coordinator**: Tracks running services
6. **plugin-coordinator**: Manages plugin lifecycle
7. **ui-element-coordinator**: Provides UI element definitions

### Communication Pattern

Everything communicates via **MQTT topics** with **JSON messages**:

```
Your Web App → MQTT Topic → Coordinator → Response Topic → Your Web App
```

Example:
```
bigskies/telescope/command/slew  →  Telescope Coordinator  →  bigskies/telescope/response/slew/{id}
```

### Key Concepts

**MQTT Topics**: Hierarchical message channels like file paths
- `bigskies/telescope/command/slew` - Command to slew telescope
- `bigskies/security/query/users` - Query user information

**JSON Messages**: All data exchanged in JSON format
```json
{
  "id": "unique-request-id",
  "source": "my-web-app",
  "type": "command",
  "timestamp": "2026-01-16T15:00:00Z",
  "payload": {
    "action": "slew",
    "ra": 12.5,
    "dec": 45.0
  }
}
```

---

## Lesson 2: Setting Up Your Development Environment

### Step 1: Clone the Repository

```bash
# Navigate to your development directory
# This is where you keep all your coding projects
cd ~/Development

# Clone the Big Skies repository from GitHub
# This downloads all the framework code to your local machine
git clone git@github.com:unklstewy/BIG_SKIES_FRAMEWORK.git

# Enter the newly created directory
# All subsequent commands will be run from here
cd BIG_SKIES_FRAMEWORK
```

### Step 2: Install Development Tools

```bash
# Run the make command to install required development tools
# This includes linters, formatters, and test runners
make install-tools
```

This installs:
- `golangci-lint` - Code linter (finds bugs and style issues)
- `goimports` - Import organizer (keeps imports clean)
- `staticcheck` - Static analysis (detects potential errors)
- `gotestsum` - Better test output (prettier test results)

### Step 3: Download Dependencies

```bash
# Download all Go module dependencies
# This fetches all external libraries the framework needs
# Go modules are like npm packages or pip packages
go mod download
```

### Step 4: Start the Framework Services

```bash
# Start all Big Skies services using Docker Compose
# This launches MQTT broker, PostgreSQL, and all coordinators
# The -d flag would run in background, but make does this for you
make docker-up
```

This starts:
- MQTT Broker (port 1883) - Message bus for all communication
- PostgreSQL (port 5432) - Database for persistent storage
- All 7 coordinators - Core framework services

Verify services are running:
```bash
# List all running Docker containers
# You should see containers for each service
docker ps
```

You should see containers for:
- `bigskies-mqtt` - MQTT message broker
- `bigskies-postgres` - PostgreSQL database
- `message-coordinator` - Message bus manager
- `security-coordinator` - Authentication service
- `telescope-coordinator` - Telescope control service
- (and others)

### Step 5: View Logs

```bash
# View live logs from all services
# This helps you see what's happening in real-time
# Press Ctrl+C to exit log view
make docker-logs
```

### Step 6: Start ASCOM Simulator

```bash
# Start the ASCOM Alpaca telescope simulator
# This provides a virtual telescope for testing
make plugin-ascom-up
```

The simulator will be available at: http://localhost:32323

---

## Lesson 3: Your First MQTT Client

In this lesson, you'll create a simple Go program that connects to the MQTT broker and publishes/subscribes to messages.

### Project Structure

Create a new directory for your tutorial project:

```bash
# Create a nested directory structure for lesson 3
# -p flag creates parent directories if they don't exist
mkdir -p tutorials/lesson3-mqtt-client

# Navigate into the new directory
cd tutorials/lesson3-mqtt-client
```

### Step 1: Initialize Go Module

```bash
# Initialize a new Go module
# This creates a go.mod file that tracks dependencies
# Replace 'yourusername' with your actual GitHub username
go mod init github.com/yourusername/bigskies-lesson3

# Download and add the MQTT client library to your project
# Eclipse Paho is the standard MQTT library for Go
go get github.com/eclipse/paho.mqtt.golang

# Download and add Zap logging library
# Zap is a fast, structured logger used throughout Big Skies
go get go.uber.org/zap
```

### Step 2: Create main.go

```go
// tutorials/lesson3-mqtt-client/main.go

// Package declaration - every Go file starts with this
// 'main' is a special package name that creates an executable program
package main

// Import block - brings in external libraries we need
// Each import provides specific functionality
import (
	"encoding/json"   // For converting Go structs to/from JSON
	"fmt"             // For formatted printing (like printf in C)
	"os"              // For operating system functions (signals, exit)
	"os/signal"       // For catching OS signals like Ctrl+C
	"syscall"         // For low-level system calls
	"time"            // For time-related functions (timestamps, delays)

	mqtt "github.com/eclipse/paho.mqtt.golang"  // MQTT client library
	"go.uber.org/zap"                          // Structured logging library
)

// Message represents a Big Skies MQTT message
// This struct defines the standard message format used throughout the framework
// All fields have JSON tags that specify how they map to JSON keys
type Message struct {
	// ID is a unique identifier for this message
	// Used to correlate requests with responses
	ID        string                 `json:"id"`
	
	// Source identifies who sent this message
	// Could be "telescope-coordinator", "web-app", etc.
	Source    string                 `json:"source"`
	
	// Type indicates the message category
	// Common values: "command", "query", "response", "event", "status"
	Type      string                 `json:"type"`
	
	// Timestamp records when the message was created
	// Uses RFC3339 format: "2026-01-16T15:00:00Z"
	Timestamp string                 `json:"timestamp"`
	
	// Payload contains the actual message data
	// map[string]interface{} means keys are strings, values can be anything
	// This allows flexible nested JSON structures
	Payload   map[string]interface{} `json:"payload"`
}

// main is the entry point of the program
// Go always starts executing from the main() function in the main package
func main() {
	// Initialize logger - this creates a development-mode logger
	// Development mode includes more verbose output with line numbers
	// The underscore (_) discards the error return value (not recommended for production)
	logger, _ := zap.NewDevelopment()
	
	// Defer the Sync() call - this ensures logs are flushed before exit
	// defer means "run this when the function exits, no matter how it exits"
	// This is important because logs are buffered for performance
	defer logger.Sync()

	// Log that we're starting - first thing we tell the user
	// zap.Info() is for informational messages (not errors or warnings)
	logger.Info("Starting MQTT Client Tutorial")

	// Configure MQTT options
	// NewClientOptions() creates a default configuration
	// We'll customize this configuration with various settings
	opts := mqtt.NewClientOptions()
	
	// AddBroker tells the client where to connect
	// "tcp://localhost:1883" means:
	//   - tcp:// = use TCP protocol (not websockets or TLS)
	//   - localhost = connect to the same machine
	//   - 1883 = standard MQTT port (like 80 for HTTP, 443 for HTTPS)
	opts.AddBroker("tcp://localhost:1883")
	
	// SetClientID gives this client a unique name
	// The broker uses this to track connections
	// Must be unique among all connected clients
	opts.SetClientID("tutorial-lesson3")
	
	// SetKeepAlive configures heartbeat interval
	// Every 60 seconds, client sends a ping to broker
	// If broker doesn't respond, connection is considered lost
	opts.SetKeepAlive(60 * time.Second)
	
	// SetAutoReconnect enables automatic reconnection
	// If connection drops, client will automatically try to reconnect
	// Very important for production systems
	opts.SetAutoReconnect(true)

	// Connection handlers - these are callback functions
	// They're called when specific events happen
	
	// SetOnConnectHandler is called when connection succeeds
	// Takes a function that receives the mqtt.Client as parameter
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		// This code runs when we successfully connect to the broker
		logger.Info("Connected to MQTT broker")
	})

	// SetConnectionLostHandler is called when connection is lost
	// Takes a function that receives the client and an error
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		// This code runs when connection is lost unexpectedly
		// The err parameter tells us why the connection was lost
		logger.Error("Connection lost", zap.Error(err))
	})

	// Create MQTT client
	// This doesn't connect yet - it just creates the client object
	// The client has all the methods we need to publish/subscribe
	client := mqtt.NewClient(opts)

	// Connect to broker
	// Connect() returns a "token" - this represents an async operation
	// MQTT operations are asynchronous for better performance
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		// token.Wait() blocks until the connection attempt completes
		// token.Error() returns nil if successful, or an error if it failed
		// If connection fails, log a fatal error and exit
		logger.Fatal("Failed to connect", zap.Error(token.Error()))
	}

	// Subscribe to health status messages from all coordinators
	// The topic pattern "bigskies/+/health" uses a wildcard
	// + matches exactly one level (any coordinator name)
	// So this matches:
	//   - bigskies/telescope/health
	//   - bigskies/security/health
	//   - bigskies/message/health
	// But NOT: bigskies/telescope/status/health (too many levels)
	topic := "bigskies/+/health"
	logger.Info("Subscribing to topic", zap.String("topic", topic))

	// Subscribe to the topic with QoS 0 and our message handler
	// Parameters:
	//   - topic: the MQTT topic pattern to subscribe to
	//   - 0: QoS (Quality of Service) level
	//        0 = at most once (fire and forget)
	//        1 = at least once (may get duplicates)
	//        2 = exactly once (slowest but guaranteed)
	//   - messageHandler(logger): callback function for received messages
	if token := client.Subscribe(topic, 0, messageHandler(logger)); token.Wait() && token.Error() != nil {
		// If subscription fails, log error and exit
		logger.Fatal("Failed to subscribe", zap.Error(token.Error()))
	}

	// Publish a test message every 5 seconds
	// time.NewTicker creates a ticker that fires every 5 seconds
	// Like a metronome - it ticks at regular intervals
	ticker := time.NewTicker(5 * time.Second)
	
	// Start a goroutine (concurrent function) to publish messages
	// go keyword means "run this function concurrently"
	// This allows the function to run in the background while main continues
	go func() {
		// range ticker.C loops forever, receiving from the ticker channel
		// Each time the ticker fires (every 5 seconds), this loop body runs
		for range ticker.C {
			// Build a message struct with current data
			msg := Message{
				// Create a unique ID using current Unix timestamp
				// fmt.Sprintf formats a string (like printf in C)
				// %d is a placeholder for an integer
				ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
				
				// Identify ourselves as the source
				Source:    "tutorial-lesson3",
				
				// This is a ping message
				Type:      "ping",
				
				// Current time in RFC3339 format (standard for JSON)
				Timestamp: time.Now().Format(time.RFC3339),
				
				// The actual message content
				Payload: map[string]interface{}{
					"message": "Hello from tutorial!",
				},
			}

			// Convert the message struct to JSON bytes
			// json.Marshal takes any Go value and converts to JSON
			// Returns ([]byte, error) - we ignore the error with _
			data, _ := json.Marshal(msg)
			
			// Publish the message to the topic
			// Parameters:
			//   - "bigskies/tutorial/ping": topic to publish to
			//   - 0: QoS level (at most once)
			//   - false: retained flag (false means don't store message on broker)
			//   - data: the actual message bytes
			token := client.Publish("bigskies/tutorial/ping", 0, false, data)
			
			// Wait for the publish operation to complete
			// This blocks until the broker acknowledges receipt
			token.Wait()

			// Log that we published a message
			logger.Info("Published message", zap.String("topic", "bigskies/tutorial/ping"))
		}
		// This goroutine continues running until ticker is stopped
	}()

	// Wait for interrupt signal (Ctrl+C)
	// Create a channel to receive OS signals
	// Channels are Go's way of communicating between goroutines
	// Buffer size of 1 means it can hold one signal without blocking
	sigChan := make(chan os.Signal, 1)
	
	// Tell the OS to send signals to our channel
	// We're interested in SIGINT (Ctrl+C) and SIGTERM (graceful shutdown)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Client running, press Ctrl+C to stop")

	// Block here waiting for a signal
	// <-sigChan means "receive from channel sigChan"
	// This line waits forever until a signal arrives
	// When user presses Ctrl+C, execution continues past this line
	<-sigChan

	// Cleanup code - runs after Ctrl+C is pressed
	logger.Info("Shutting down...")
	
	// Stop the ticker so it doesn't fire anymore
	ticker.Stop()
	
	// Disconnect from MQTT broker
	// 250 means wait up to 250ms for graceful disconnect
	client.Disconnect(250)
	
	// main() exits here, defer statements run (logger.Sync())
}

// messageHandler creates a message handler function
// This is a "factory function" - it creates and returns another function
// Why? Because we want the handler to have access to the logger
// Go's closures allow the returned function to "remember" the logger
func messageHandler(logger *zap.Logger) mqtt.MessageHandler {
	// Return an anonymous function that matches the MessageHandler signature
	// mqtt.MessageHandler is: func(mqtt.Client, mqtt.Message)
	return func(client mqtt.Client, msg mqtt.Message) {
		// This code runs every time a message is received on a subscribed topic
		
		// Log basic message info
		// msg.Topic() returns the topic the message was published to
		// msg.Payload() returns the message content as []byte
		// len() gives us the size in bytes
		logger.Info("Received message",
			zap.String("topic", msg.Topic()),
			zap.Int("size", len(msg.Payload())))

		// Try to parse as JSON
		// Declare a variable to hold the parsed message
		var message Message
		
		// json.Unmarshal converts JSON bytes to a Go struct
		// If successful, err will be nil and message will be populated
		if err := json.Unmarshal(msg.Payload(), &message); err == nil {
			// Parsing succeeded - log the structured data
			logger.Info("Parsed message",
				zap.String("id", message.ID),
				zap.String("source", message.Source),
				zap.String("type", message.Type))
		}
		// If parsing fails, we just skip this part
		// Real applications should handle parsing errors explicitly
	}
}
```

### Step 3: Run Your Client

```bash
# Run the Go program
# go run compiles and runs in one step (convenient for development)
# The dot (.) means "compile all .go files in current directory"
go run main.go
```

You should see output like:
```
2026-01-16T15:00:00.000Z	INFO	Connected to MQTT broker
2026-01-16T15:00:00.001Z	INFO	Subscribing to topic	{"topic": "bigskies/+/health"}
2026-01-16T15:00:00.002Z	INFO	Client running, press Ctrl+C to stop
2026-01-16T15:00:05.000Z	INFO	Published message	{"topic": "bigskies/tutorial/ping"}
2026-01-16T15:00:10.123Z	INFO	Received message	{"topic": "bigskies/telescope/health", "size": 234}
```

### Understanding the Code Flow

1. **Initialization Phase**:
   - Logger is created for outputting information
   - MQTT options are configured
   - Connection handlers are registered

2. **Connection Phase**:
   - Client connects to broker
   - OnConnect handler fires when successful

3. **Subscription Phase**:
   - Client subscribes to health topics
   - messageHandler is registered for incoming messages

4. **Runtime Phase**:
   - Goroutine publishes messages every 5 seconds
   - Main goroutine waits for Ctrl+C
   - messageHandler processes incoming messages

5. **Shutdown Phase**:
   - Ctrl+C signal is received
   - Ticker is stopped
   - Client disconnects gracefully
   - Logger flushes any buffered messages

### Experiment

Try modifying the code:
- Change the subscribe topic to `bigskies/telescope/#`
- Add more fields to the message payload
- Subscribe to multiple topics
- Change the publish interval to 10 seconds

---

## Lesson 4: Building a Simple Web Service

Now let's build a web server using the Gin framework that exposes HTTP endpoints and communicates with the Big Skies framework via MQTT.

### Project Structure

```bash
# Create directory for lesson 4
mkdir -p tutorials/lesson4-web-service

# Navigate to the new directory
cd tutorials/lesson4-web-service
```

### Step 1: Initialize and Install Dependencies

```bash
# Initialize Go module for this project
go mod init github.com/yourusername/bigskies-lesson4

# Install Gin web framework
# Gin provides HTTP routing, middleware, and request handling
go get github.com/gin-gonic/gin

# Install MQTT client library (same as lesson 3)
go get github.com/eclipse/paho.mqtt.golang

# Install Zap logging library
go get go.uber.org/zap

# Install UUID library for generating unique IDs
# UUIDs are better than sequential IDs for distributed systems
go get github.com/google/uuid
```

### Step 2: Create MQTTClient Wrapper (mqtt_client.go)

This file creates a wrapper around the MQTT client that makes request/response patterns easier.

```go
// tutorials/lesson4-web-service/mqtt_client.go

// Package main - this file is part of the main package
package main

// Import necessary libraries
import (
	"encoding/json"  // For JSON marshaling/unmarshaling
	"fmt"            // For error formatting
	"sync"           // For thread-safe map access (mutex)
	"time"           // For timeouts and timestamps

	mqtt "github.com/eclipse/paho.mqtt.golang"  // MQTT client
	"github.com/google/uuid"                    // UUID generation
	"go.uber.org/zap"                          // Logging
)

// MQTTClient wraps the MQTT client with request/response handling
// This makes it easier to send a message and wait for a response
// Without this, you'd have to manually manage correlating requests and responses
type MQTTClient struct {
	// client is the underlying MQTT client from Paho library
	// We wrap this to add our custom functionality
	client       mqtt.Client
	
	// logger for outputting diagnostic information
	logger       *zap.Logger
	
	// pendingReqs tracks requests waiting for responses
	// Key: request ID (UUID string)
	// Value: channel to send the response to
	// When a response arrives, we look up the channel by ID and send the response
	pendingReqs  map[string]chan []byte
	
	// mu is a mutex (mutual exclusion lock)
	// RWMutex allows multiple readers OR one writer
	// This prevents race conditions when multiple goroutines access pendingReqs
	mu           sync.RWMutex
}

// BigSkiesMessage represents the standard message format
// This is the same structure we used in Lesson 3
// All Big Skies coordinators use this format
type BigSkiesMessage struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// NewMQTTClient creates a new MQTT client
// This is a "constructor function" - Go doesn't have constructors like Java/C++
// Instead, we use functions that start with "New" and return a pointer to a struct
func NewMQTTClient(brokerURL, clientID string, logger *zap.Logger) (*MQTTClient, error) {
	// Create the MQTTClient struct
	// &MQTTClient{} creates a new instance and returns a pointer to it
	mc := &MQTTClient{
		logger:      logger,
		
		// make() creates a new map
		// This map will hold all pending requests
		pendingReqs: make(map[string]chan []byte),
	}

	// Configure MQTT options (same as Lesson 3)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)

	// OnConnectHandler fires when connection succeeds
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("Connected to MQTT broker")
	})

	// ConnectionLostHandler fires when connection drops
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Error("Connection lost", zap.Error(err))
	})

	// Create the underlying MQTT client
	mc.client = mqtt.NewClient(opts)

	// Attempt to connect to the broker
	if token := mc.client.Connect(); token.Wait() && token.Error() != nil {
		// If connection fails, return nil and an error
		// fmt.Errorf creates a formatted error message
		// %w wraps the original error (allows error unwrapping)
		return nil, fmt.Errorf("failed to connect: %w", token.Error())
	}

	// Subscribe to ALL response topics from ALL coordinators
	// "bigskies/+/response/#" means:
	//   - bigskies/ = namespace
	//   - + = any coordinator name (wildcard for one level)
	//   - /response/ = response topic category
	//   - # = any number of levels after this (wildcard for remaining path)
	// Examples matched:
	//   - bigskies/telescope/response/slew/abc-123
	//   - bigskies/security/response/authenticate/def-456
	mc.client.Subscribe("bigskies/+/response/#", 0, mc.responseHandler)

	// Return the fully initialized client
	return mc, nil
}

// PublishAndWait publishes a message and waits for response
// This is the key feature of our wrapper - it makes request/response easy
// Without this, you'd need to:
//   1. Publish message
//   2. Subscribe to response topic
//   3. Wait for response
//   4. Match response to request by ID
// This function does all of that for you
func (mc *MQTTClient) PublishAndWait(topic string, payload map[string]interface{}, timeout time.Duration) ([]byte, error) {
	// Generate unique request ID using UUID
	// UUID (Universally Unique Identifier) is a 128-bit number
	// Example: "550e8400-e29b-41d4-a716-446655440000"
	// Virtually guaranteed to be unique across all systems and time
	requestID := uuid.New().String()

	// Create response channel
	// This channel will receive exactly one response (buffer size 1)
	// Buffered channels allow sending without blocking if no one is receiving yet
	respChan := make(chan []byte, 1)
	
	// Lock the mutex before modifying pendingReqs map
	// Lock() blocks if another goroutine has the lock
	// This prevents race conditions where two goroutines modify the map simultaneously
	mc.mu.Lock()
	
	// Register this request in the pending requests map
	// When a response arrives with this ID, it will be sent to respChan
	mc.pendingReqs[requestID] = respChan
	
	// Unlock the mutex - we're done modifying the map
	mc.mu.Unlock()

	// Clean up on exit using defer
	// defer runs when the function exits, regardless of how it exits
	// This ensures we always clean up, even if there's an error or timeout
	defer func() {
		// Lock again to modify the map
		mc.mu.Lock()
		
		// Remove this request from pending map
		// delete() is a built-in function for removing map entries
		delete(mc.pendingReqs, requestID)
		
		// Unlock
		mc.mu.Unlock()
		
		// Close the channel to signal no more values will be sent
		// This prevents goroutine leaks
		close(respChan)
	}()

	// Build the message structure
	msg := BigSkiesMessage{
		ID:        requestID,
		Source:    "web-service",        // Identify ourselves
		Type:      "query",               // This is a query message
		Timestamp: time.Now().Format(time.RFC3339),  // Current time
		Payload:   payload,               // The actual data
	}

	// Convert message struct to JSON bytes
	data, err := json.Marshal(msg)
	if err != nil {
		// If JSON marshaling fails, return error
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Publish message to MQTT broker
	// Parameters:
	//   - topic: where to publish
	//   - 0: QoS level (at most once)
	//   - false: not retained (broker doesn't store it)
	//   - data: the message bytes
	token := mc.client.Publish(topic, qos, retained, data)
	
	// Wait for publish to complete
	token.Wait()

	// Check if publish had an error
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	// Log that we sent the request
	// Debug level is less important than Info
	// Won't show in production logs unless debug logging is enabled
	mc.logger.Debug("Published request", 
		zap.String("id", requestID), 
		zap.String("topic", topic))

	// Wait for response with timeout
	// select statement lets us wait on multiple channels
	// Whichever case happens first will execute
	select {
	case response := <-respChan:
		// Case 1: We received a response on our channel
		// This means responseHandler got a message with our ID
		return response, nil
		
	case <-time.After(timeout):
		// Case 2: The timeout expired before we got a response
		// time.After() creates a channel that sends after the timeout
		// This prevents hanging forever if the coordinator doesn't respond
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
	// Only one case will execute, then the function returns
}

// responseHandler handles incoming response messages
// This is called by the MQTT client whenever a message arrives on a subscribed topic
// It's called from a different goroutine, so we need to be thread-safe
func (mc *MQTTClient) responseHandler(client mqtt.Client, msg mqtt.Message) {
	// Try to parse the message as JSON
	var response BigSkiesMessage
	
	// json.Unmarshal converts JSON bytes to a struct
	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		// If parsing fails, log error and return
		// This might happen if someone publishes malformed JSON
		mc.logger.Error("Failed to parse response", zap.Error(err))
		return
	}

	// Log that we received a response
	mc.logger.Debug("Received response", zap.String("id", response.ID))

	// Look up the request channel using RLock (read lock)
	// RLock allows multiple readers simultaneously
	// This is safe because we're only reading, not modifying the map
	mc.mu.RLock()
	
	// Look up the channel for this request ID
	// respChan will be nil if the ID doesn't exist in the map
	// exists will be false if the key isn't in the map
	respChan, exists := mc.pendingReqs[response.ID]
	
	// Release the read lock
	mc.mu.RUnlock()

	// If there's a waiting request with this ID
	if exists {
		// Send the response to the waiting goroutine
		// The goroutine in PublishAndWait() is waiting on this channel
		// <- operator sends to a channel
		respChan <- msg.Payload()
		// The waiting goroutine will now receive the response and return it
	}
	// If exists is false, this might be a late response for a timed-out request
	// We just ignore it - the request has already been cleaned up
}

// Publish sends a message without waiting for response
// This is for "fire and forget" messages where you don't need a response
// Example: sending a status update or event notification
func (mc *MQTTClient) Publish(topic string, payload interface{}) error {
	// Marshal the payload to JSON
	// payload can be any type - struct, map, slice, etc.
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// Publish to MQTT broker
	token := mc.client.Publish(topic, 0, false, data)
	
	// Wait for completion
	token.Wait()

	// Return any error that occurred
	return token.Error()
}

// Close disconnects from MQTT broker
// Should be called when shutting down to cleanup resources
func (mc *MQTTClient) Close() {
	// Disconnect with 250ms grace period
	// This gives the broker time to process any pending messages
	mc.client.Disconnect(250)
}
```

### Step 3: Create Web Server (main.go)

Now we'll create the web server that uses our MQTT client.

```go
// tutorials/lesson4-web-service/main.go

// Package main - executable program
package main

import (
	"context"        // For handling cancellation and timeouts
	"encoding/json"  // For parsing JSON responses
	"net/http"       // For HTTP constants and types
	"os"             // For OS signals
	"os/signal"      // For signal handling
	"syscall"        // For signal constants
	"time"           // For timeouts and timestamps

	"github.com/gin-gonic/gin"  // Web framework
	"go.uber.org/zap"           // Logging
)

// main is the entry point
func main() {
	// Initialize logger
	// NewDevelopment() creates a logger with debug output
	// For production, use NewProduction()
	logger, _ := zap.NewDevelopment()
	
	// Ensure logs are flushed before exit
	defer logger.Sync()

	logger.Info("Starting Big Skies Web Service")

	// Create MQTT client
	// This connects to the MQTT broker running on localhost
	// The client will handle all communication with coordinators
	mqttClient, err := NewMQTTClient("tcp://localhost:1883", "web-service", logger)
	if err != nil {
		// If we can't connect to MQTT, we can't function
		// Fatal logs the error and exits the program
		logger.Fatal("Failed to create MQTT client", zap.Error(err))
	}
	
	// Ensure MQTT client is closed when main() exits
	defer mqttClient.Close()

	// Create Gin router
	// Default() creates a router with logger and recovery middleware
	// Recovery middleware prevents panics from crashing the server
	router := gin.Default()

	// Add MQTT client to context
	// This middleware runs for every request
	// It adds the MQTT client and logger to the request context
	// This allows handlers to access these without passing them as parameters
	router.Use(func(c *gin.Context) {
		// c.Set stores a value in the request context
		// "mqtt" is the key we'll use to retrieve it later
		c.Set("mqtt", mqttClient)
		
		// Store logger in context too
		c.Set("logger", logger)
		
		// c.Next() calls the next handler in the chain
		// Without this, the request would stop here
		c.Next()
	})

	// Define routes
	// Each route maps an HTTP method + path to a handler function
	
	// GET /health - Health check endpoint
	// Used by load balancers and monitoring to check if service is alive
	router.GET("/health", healthHandler)
	
	// GET /api/coordinators - List all coordinators
	router.GET("/api/coordinators", listCoordinatorsHandler)
	
	// GET /api/telescope/status - Get telescope status
	router.GET("/api/telescope/status", telescopeStatusHandler)

	// Create HTTP server
	// This gives us more control than just router.Run()
	// We can configure timeouts, graceful shutdown, etc.
	srv := &http.Server{
		Addr:    ":8080",      // Listen on port 8080
		Handler: router,        // Use our Gin router
	}

	// Start server in goroutine
	// We start it in a goroutine so main() can continue
	// This allows us to handle shutdown signals
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", srv.Addr))
		
		// ListenAndServe blocks until server is stopped
		// It returns an error when it stops
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// ErrServerClosed is expected when we shut down gracefully
			// Any other error is a problem
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	// This creates a channel for OS signals
	quit := make(chan os.Signal, 1)
	
	// Register to receive SIGINT (Ctrl+C) and SIGTERM (kill command)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Block until we receive a signal
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	// Create a context with 5 second timeout
	// This gives the server 5 seconds to finish processing requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	
	// Cancel the context when we're done
	// This releases resources associated with the context
	defer cancel()

	// Shutdown stops the server gracefully
	// It stops accepting new connections and waits for existing requests to complete
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
	// defer statements run here: mqttClient.Close(), logger.Sync()
}

// healthHandler returns service health status
// This is a simple endpoint that always returns 200 OK
// Load balancers use this to check if the service is running
func healthHandler(c *gin.Context) {
	// c.JSON sends a JSON response
	// First parameter: HTTP status code
	// Second parameter: data to serialize as JSON
	// gin.H is a shortcut for map[string]interface{}
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "big-skies-web-service",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// listCoordinatorsHandler queries and lists all coordinators
// This demonstrates how to query a coordinator via MQTT
func listCoordinatorsHandler(c *gin.Context) {
	// Retrieve MQTT client from context
	// c.MustGet panics if the key doesn't exist
	// This is safe because we know the middleware set it
	// Type assertion .(*MQTTClient) converts interface{} to *MQTTClient
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	
	// Get logger from context
	logger := c.MustGet("logger").(*zap.Logger)

	// Build payload for the query
	// This tells the application coordinator what we want
	payload := map[string]interface{}{
		"action": "list_services",
	}

	// Send query and wait for response
	// PublishAndWait handles:
	//   1. Generating unique request ID
	//   2. Publishing the message
	//   3. Waiting for response with matching ID
	//   4. Timing out if no response
	response, err := mqtt.PublishAndWait(
		"bigskies/application/query",  // Topic to publish to
		payload,                        // Message payload
		5*time.Second,                  // Timeout duration
	)

	// Check if the request failed or timed out
	if err != nil {
		logger.Error("Failed to query coordinators", zap.Error(err))
		
		// Return error response to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query coordinators",
		})
		return
	}

	// Parse response JSON
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		// Response wasn't valid JSON
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return the payload from the response
	// The coordinator's response payload contains the list of services
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeStatusHandler gets telescope status
// Similar to listCoordinatorsHandler but queries telescope coordinator
func telescopeStatusHandler(c *gin.Context) {
	// Get dependencies from context
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Build query payload
	payload := map[string]interface{}{
		"action": "get_status",
	}

	// Send query to telescope coordinator
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/query",  // Telescope coordinator's query topic
		payload,
		5*time.Second,
	)

	if err != nil {
		logger.Error("Failed to query telescope", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query telescope status",
		})
		return
	}

	// Parse and return response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return telescope status data
	c.JSON(http.StatusOK, msg.Payload)
}
```

### Step 4: Run Your Web Service

```bash
# Run the web service
# It will compile both main.go and mqtt_client.go
go run .
```

Test the endpoints in a new terminal:
```bash
# Check health
# Should return: {"status":"healthy","service":"big-skies-web-service","timestamp":"..."}
curl http://localhost:8080/health

# List coordinators
# Should return list of running coordinators
curl http://localhost:8080/api/coordinators

# Get telescope status
# Should return telescope state information
curl http://localhost:8080/api/telescope/status
```

### Understanding the Request Flow

1. **HTTP Request arrives** at Gin router
2. **Middleware runs** - adds MQTT client and logger to context
3. **Handler executes** - retrieves dependencies from context
4. **MQTT query sent** - PublishAndWait sends message with unique ID
5. **Request registered** - ID stored in pendingReqs map
6. **Handler waits** - select statement blocks on response channel
7. **Response arrives** - responseHandler receives MQTT message
8. **Response matched** - ID looked up in pendingReqs map
9. **Response delivered** - sent to response channel
10. **Handler resumes** - select receives response from channel
11. **HTTP Response sent** - JSON returned to client

This completes Lesson 4! You now have a working web service that communicates with Big Skies via MQTT.

[Continuing with Lessons 5-7 in the next part...]

---

## Lesson 5: Adding Security and Authentication

In this lesson, we'll add JWT (JSON Web Token) based authentication to protect our API endpoints.

### Understanding JWT

**What is JWT?**: A compact, URL-safe token that contains claims (user info) and is cryptographically signed.

**Structure**: Three parts separated by dots (.)
- **Header**: Token type and signing algorithm
- **Payload**: Claims (user ID, username, expiration)
- **Signature**: Cryptographic signature to prevent tampering

**Example JWT**:
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIiwidXNlcm5hbWUiOiJqb2huIn0.xyz...
```

**Why JWT?**: 
- Stateless (server doesn't need to store sessions)
- Can be verified without database lookup
- Contains user information
- Has built-in expiration

### Step 1: Install Dependencies

```bash
# Navigate to lesson 4 directory (we'll extend it)
cd tutorials/lesson4-web-service

# Install JWT library
# This handles creating and validating JWT tokens
go get github.com/golang-jwt/jwt/v5

# Install bcrypt for password hashing
# bcrypt is a secure one-way hash function for passwords
# It's designed to be slow to resist brute-force attacks
go get golang.org/x/crypto/bcrypt
```

### Step 2: Create Authentication Middleware (auth.go)

```go
// tutorials/lesson4-web-service/auth.go

package main

import (
	"encoding/json"  // For parsing JSON responses
	"fmt"            // For error formatting
	"net/http"       // For HTTP status codes
	"strings"        // For string manipulation
	"time"           // For token expiration

	"github.com/gin-gonic/gin"            // Web framework
	"github.com/golang-jwt/jwt/v5"        // JWT library
	"go.uber.org/zap"                     // Logging
)

// jwtSecret is the secret key used to sign JWT tokens
// IMPORTANT: In production, this MUST be:
//   1. Long and random (at least 32 bytes)
//   2. Stored in environment variables or secrets manager
//   3. Never committed to version control
// The same secret must be used to create and validate tokens
var jwtSecret = []byte("your-secret-key-change-in-production")

// Claims represents JWT claims
// This struct holds the data we embed in the JWT token
// It extends jwt.RegisteredClaims which includes standard fields like expiration
type Claims struct {
	// Custom fields - these are specific to our application
	
	// UserID is the unique identifier for the user
	// Used to look up user information when needed
	UserID   string `json:"user_id"`
	
	// Username is the human-readable username
	// Included for convenience so we don't always need to query the database
	Username string `json:"username"`
	
	// RegisteredClaims includes standard JWT fields:
	//   - ExpiresAt: when the token expires
	//   - IssuedAt: when the token was created
	//   - Issuer: who created the token
	//   - Subject: who the token is for
	jwt.RegisteredClaims
}

// LoginRequest represents login credentials sent from client
// The binding tags specify validation rules:
//   - required: field must be present in JSON
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response with JWT token
// This is what we send back to the client after successful login
type LoginResponse struct {
	// Token is the JWT that client will use for authentication
	// Client must send this in Authorization header for protected endpoints
	Token     string    `json:"token"`
	
	// ExpiresAt tells client when token expires
	// Client can use this to refresh token before expiration
	ExpiresAt time.Time `json:"expires_at"`
	
	// User contains non-sensitive user information
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information
// This is safe to send to client (no password hash)
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// authMiddleware validates JWT tokens
// This is a Gin middleware function that runs before protected handlers
// It checks if the request has a valid JWT token
// If valid, it allows the request to continue
// If invalid, it returns 401 Unauthorized
func authMiddleware() gin.HandlerFunc {
	// This returns a function that will be called for each request
	// The returned function is the actual middleware
	return func(c *gin.Context) {
		// Get logger from context (set by our earlier middleware)
		logger := c.MustGet("logger").(*zap.Logger)

		// Get token from Authorization header
		// Header format: "Authorization: Bearer <token>"
		// GetHeader returns empty string if header doesn't exist
		authHeader := c.GetHeader("Authorization")
		
		// Check if Authorization header is present
		if authHeader == "" {
			// No header = no authentication
			// Return error and abort the request chain
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			
			// c.Abort() prevents any further handlers from running
			// Without this, the request would continue to the protected handler
			c.Abort()
			return
		}

		// Extract token from header
		// Split on space character to separate "Bearer" from token
		// SplitN splits into at most N parts (2 in this case)
		parts := strings.SplitN(authHeader, " ", 2)
		
		// Validate header format
		// len(parts) != 2 means no space was found
		// parts[0] != "Bearer" means wrong prefix
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Extract the token string (everything after "Bearer ")
		tokenString := parts[1]

		// Parse and validate token
		// jwt.ParseWithClaims does several things:
		//   1. Parses the token string
		//   2. Validates the signature using the provided key function
		//   3. Checks expiration
		//   4. Populates the Claims struct
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// This function is called to get the signing key
			// It's called by the JWT library during verification
			
			// Check that signing method is HMAC
			// token.Method.Alg() returns the algorithm used
			// We only accept HMAC-based algorithms (HS256, HS384, HS512)
			// This prevents attacks where someone changes the algorithm to "none"
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				// Wrong signing method - this is an attack attempt
				return nil, fmt.Errorf("unexpected signing method")
			}
			
			// Return the secret key used to verify the signature
			// JWT library will use this to check if signature is valid
			return jwtSecret, nil
		})

		// Check for errors during parsing/validation
		// token.Valid checks if token hasn't expired
		if err != nil || !token.Valid {
			logger.Warn("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract claims from the validated token
		// Type assertion to convert interface{} to *Claims
		if claims, ok := token.Claims.(*Claims); ok {
			// Token is valid - add user info to context
			// Following handlers can access this info using c.GetString("user_id")
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
		}

		// Call next handler in the chain
		// This allows the request to proceed to the protected handler
		c.Next()
	}
	// If we reach here without calling Abort(), the request proceeds
}

// loginHandler handles user login and JWT generation
// This endpoint:
//   1. Receives username/password
//   2. Validates credentials with security coordinator
//   3. Generates JWT token
//   4. Returns token to client
func loginHandler(c *gin.Context) {
	// Get dependencies from context
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Parse request body into LoginRequest struct
	var req LoginRequest
	
	// ShouldBindJSON:
	//   1. Parses JSON from request body
	//   2. Validates using binding tags (required)
	//   3. Returns error if validation fails
	if err := c.ShouldBindJSON(&req); err != nil {
		// Validation failed - return 400 Bad Request
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Authenticate via security coordinator
	// Build payload with credentials
	payload := map[string]interface{}{
		"action":   "authenticate",
		"username": req.Username,
		"password": req.Password,
	}

	// Send authentication request to security coordinator
	// This coordinator will:
	//   1. Look up user in database
	//   2. Compare password hash using bcrypt
	//   3. Return success/failure and user info
	response, err := mqtt.PublishAndWait(
		"bigskies/security/authenticate",  // Security coordinator topic
		payload,
		5*time.Second,                      // 5 second timeout
	)

	// Check if request failed (timeout or error)
	if err != nil {
		logger.Error("Authentication request failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Authentication service unavailable",
		})
		return
	}

	// Parse response from security coordinator
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Check authentication result
	// The payload contains a "success" field indicating if auth succeeded
	// Type assertion .(bool) converts interface{} to bool
	// If the field doesn't exist or isn't a bool, success will be false
	success, _ := msg.Payload["success"].(bool)
	
	if !success {
		// Authentication failed - wrong username or password
		// Return 401 Unauthorized
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Extract user info from response payload
	// Type assertions convert interface{} to string
	userID, _ := msg.Payload["user_id"].(string)
	username, _ := msg.Payload["username"].(string)
	email, _ := msg.Payload["email"].(string)

	// Generate JWT token
	// Token will expire in 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)
	
	// Create claims struct with user info and expiration
	claims := &Claims{
		// Custom fields
		UserID:   userID,
		Username: username,
		
		// Standard JWT fields
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt is when the token becomes invalid
			// After this time, authMiddleware will reject the token
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			
			// IssuedAt records when token was created
			// Can be used for token refresh logic
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			
			// Issuer identifies who created the token
			// Useful if multiple services issue tokens
			Issuer:    "big-skies-web-service",
		},
	}

	// Create token with claims
	// NewWithClaims creates a new token with the specified signing method and claims
	// SigningMethodHS256 uses HMAC-SHA256 for signing
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Sign the token with our secret key
	// This creates the complete JWT string with signature
	// Only someone with the secret can create valid tokens
	tokenString, err := token.SignedString(jwtSecret)
	
	if err != nil {
		logger.Error("Failed to sign token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	// Return token and user info to client
	// Client should store the token and include it in future requests
	c.JSON(http.StatusOK, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       userID,
			Username: username,
			Email:    email,
		},
	})
}

// registerHandler handles user registration
// This creates a new user account
func registerHandler(c *gin.Context) {
	// Get dependencies
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Define request structure inline
	// This is an anonymous struct - no type name
	// Used when you only need the struct in one place
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`  // email validation
		Password string `json:"password" binding:"required,min=8"` // minimum 8 characters
	}

	// Parse and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		// Validation failed - return error message
		// err.Error() describes what validation failed
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user via security coordinator
	payload := map[string]interface{}{
		"action":   "create_user",
		"username": req.Username,
		"email":    req.Email,
		"password": req.Password,  // Coordinator will hash this before storing
	}

	// Send user creation request
	response, err := mqtt.PublishAndWait(
		"bigskies/security/users/create",
		payload,
		5*time.Second,
	)

	if err != nil {
		logger.Error("User creation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	// Parse response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return created user info (minus password)
	// StatusCreated (201) indicates resource was created successfully
	c.JSON(http.StatusCreated, msg.Payload)
}

// profileHandler returns current user profile
// This demonstrates how protected endpoints access user info from context
func profileHandler(c *gin.Context) {
	// Get user info from context
	// authMiddleware set these values after validating the JWT
	userID := c.GetString("user_id")
	username := c.GetString("username")

	// Return user information
	// In a real application, you might query more details from database
	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"message":  "This is a protected endpoint",
	})
}
```

### Step 3: Update main.go to Include Auth Routes

Update your `main.go` file to add authentication routes:

```go
// Add these routes AFTER the existing middleware setup
// and BEFORE the existing routes

	// Public routes (no authentication required)
	// Anyone can access these endpoints
	router.POST("/api/auth/login", loginHandler)
	router.POST("/api/auth/register", registerHandler)

	// Protected routes (authentication required)
	// Create a route group that uses the auth middleware
	// All routes in this group will check JWT tokens
	protected := router.Group("/api")
	
	// Use applies middleware to all routes in this group
	// authMiddleware() will run before any handler in this group
	protected.Use(authMiddleware())
	{
		// Inside the braces, all routes require authentication
		// The authMiddleware will run first, then the handler
		
		protected.GET("/profile", profileHandler)
		protected.GET("/telescope/status", telescopeStatusHandler)
		protected.POST("/telescope/slew", telescopeSlewHandler)  // We'll create this in Lesson 6
		protected.GET("/coordinators", listCoordinatorsHandler)
	}

// Remove or comment out the old unprotected routes
// router.GET("/api/coordinators", listCoordinatorsHandler)
// router.GET("/api/telescope/status", telescopeStatusHandler)
```

### Step 4: Test Authentication

```bash
# Start the web service
go run .
```

In another terminal:

```bash
# Try accessing protected endpoint without token
# Should return: {"error":"Authorization header required"}
curl http://localhost:8080/api/profile

# Register a new user
# Should return: {"id":"...","username":"testuser",...}
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'

# Login to get token
# Should return: {"token":"eyJ...","expires_at":"...","user":{...}}
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'

# Copy the token from the response
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lk..." # Your actual token here

# Access protected endpoint with token
# Should return: {"user_id":"...","username":"testuser","message":"..."}
curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer $TOKEN"
```

### Understanding JWT Flow

1. **Registration**:
   - Client sends username/email/password
   - Server sends to security coordinator
   - Coordinator hashes password with bcrypt
   - Coordinator stores user in database
   - Server returns user info (without password)

2. **Login**:
   - Client sends username/password
   - Server sends to security coordinator
   - Coordinator compares with stored hash
   - If valid, server generates JWT with user claims
   - Server signs JWT with secret key
   - Server returns JWT to client

3. **Authenticated Request**:
   - Client includes JWT in Authorization header
   - authMiddleware extracts token from header
   - Middleware verifies signature using secret
   - Middleware checks expiration
   - If valid, middleware adds user info to context
   - Handler accesses user info from context
   - Handler processes request with user identity

4. **Security Features**:
   - **Password Hashing**: Passwords are never stored in plain text
   - **Token Signing**: Tokens can't be forged without the secret
   - **Expiration**: Tokens automatically become invalid after 24 hours
   - **Stateless**: Server doesn't need to store session data
   - **Middleware**: Authentication logic is centralized and reusable

---

## Lesson 6: Integrating Telescope Control

In this lesson, we'll add complete telescope control functionality, allowing users to command a telescope through the ASCOM Alpaca Simulator.

### Understanding ASCOM Alpaca

**ASCOM** (Astronomy Common Object Model) is a standard interface for astronomy equipment. **Alpaca** is the cross-platform, network-based version that works over HTTP.

**Key Concepts**:
- **Device Number**: Multiple telescopes can be connected; each has a device number (usually 0 for the first)
- **Connected State**: Must connect to telescope before issuing commands
- **Coordinates**:
  - **RA (Right Ascension)**: Like longitude in the sky, measured in hours (0-24)
  - **Dec (Declination)**: Like latitude in the sky, measured in degrees (-90 to +90)
- **Slewing**: Moving the telescope to point at coordinates
- **Tracking**: Following objects as Earth rotates
- **Parking**: Moving telescope to safe storage position

### Step 1: Create Telescope Controller (telescope.go)

Add this file to your `tutorials/lesson4-web-service/` directory:

```go
// tutorials/lesson4-web-service/telescope.go

// Package main - this is part of our web service
package main

import (
	"encoding/json"  // For parsing JSON responses from coordinators
	"net/http"       // For HTTP status codes
	"time"           // For timeouts (slewing can take time)

	"github.com/gin-gonic/gin"  // Web framework
	"go.uber.org/zap"           // Logging
)

// TelescopeStatus represents the current state of the telescope
// This struct mirrors the information returned by ASCOM Alpaca
// All fields use JSON tags so they serialize properly for API responses
type TelescopeStatus struct {
	// Connected indicates if we have an active connection to the telescope
	// Must be true before issuing any commands
	Connected    bool    `json:"connected"`
	
	// Tracking indicates if telescope is actively tracking (following sky motion)
	// When tracking is on, telescope compensates for Earth's rotation
	Tracking     bool    `json:"tracking"`
	
	// Slewing indicates if telescope is currently moving to a new position
	// Cannot issue new movement commands while slewing
	Slewing      bool    `json:"slewing"`
	
	// AtPark indicates if telescope is in its parked (safe storage) position
	// Should park telescope when done observing
	AtPark       bool    `json:"at_park"`
	
	// RightAscension is the RA coordinate where telescope is currently pointing
	// Measured in hours (0-24), like longitude on the celestial sphere
	// Example: 12.5 means 12 hours 30 minutes
	RightAscen   float64 `json:"right_ascension"`
	
	// Declination is the Dec coordinate where telescope is currently pointing
	// Measured in degrees (-90 to +90), like latitude on the celestial sphere
	// Example: 45.0 means 45 degrees north of celestial equator
	Declination  float64 `json:"declination"`
	
	// Altitude is the angle above the horizon (0-90 degrees)
	// 0 = horizon, 90 = straight up (zenith)
	// Useful for avoiding pointing at ground or obstructions
	Altitude     float64 `json:"altitude"`
	
	// Azimuth is the compass direction (0-360 degrees)
	// 0/360 = North, 90 = East, 180 = South, 270 = West
	Azimuth      float64 `json:"azimuth"`
	
	// SiderealTime is the local sidereal time
	// This is the RA that's currently on the meridian (directly overhead)
	// Used for calculating what objects are visible
	SiderealTime float64 `json:"sidereal_time"`
}

// SlewRequest represents a command to slew (move) the telescope
// This is the data structure clients send when commanding a slew
// The binding tags provide validation - both fields are required
type SlewRequest struct {
	// RA is the target Right Ascension in hours (0-24)
	// binding:"required" means the request will be rejected if this is missing
	RA  float64 `json:"ra" binding:"required"`
	
	// Dec is the target Declination in degrees (-90 to +90)
	Dec float64 `json:"dec" binding:"required"`
}

// telescopeStatusHandler gets detailed telescope status
// This endpoint queries the telescope coordinator for complete state information
// It's a protected endpoint - user must be authenticated (JWT token required)
func telescopeStatusHandler(c *gin.Context) {
	// Retrieve MQTT client from Gin context
	// MustGet panics if key doesn't exist - safe because middleware sets it
	// Type assertion .(*MQTTClient) converts from interface{} to concrete type
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	
	// Get logger for diagnostic output
	logger := c.MustGet("logger").(*zap.Logger)

	// Build the query payload
	// This tells the telescope coordinator what information we want
	// The "action" field is a convention - coordinators use it to route requests
	payload := map[string]interface{}{
		"action": "get_status",  // Request full status information
	}

	// Send query via MQTT and wait for response
	// PublishAndWait handles:
	//   1. Generating unique request ID
	//   2. Publishing to the topic
	//   3. Waiting for response with matching ID
	//   4. Timing out if no response received
	// Topic: "bigskies/telescope/query/status" - telescope coordinator's query endpoint
	// Timeout: 5 seconds should be plenty for a status query
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/query/status",
		payload,
		5*time.Second,  // 5 second timeout
	)

	// Check if the MQTT request failed
	// Failures can occur due to:
	//   - Timeout (coordinator didn't respond in 5 seconds)
	//   - MQTT connection lost
	//   - Coordinator not running
	if err != nil {
		// Log the error with context for debugging
		// Include enough information to diagnose the problem
		logger.Error("Failed to get telescope status", 
			zap.Error(err),  // The actual error
			zap.String("user", c.GetString("username")))  // Who made the request
		
		// Return error response to client
		// 500 Internal Server Error because this is a server-side issue
		// Client can't fix this - they need to retry or report it
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query telescope",
		})
		return  // Stop processing this request
	}

	// Parse the MQTT response into a BigSkiesMessage struct
	// Declare a variable to hold the parsed message
	var msg BigSkiesMessage
	
	// json.Unmarshal converts JSON bytes to Go struct
	// response is []byte (raw bytes)
	// &msg is a pointer to where we want the data stored
	if err := json.Unmarshal(response, &msg); err != nil {
		// Parsing failed - response wasn't valid JSON
		// This shouldn't happen with Big Skies coordinators
		// but we handle it defensively
		logger.Error("Failed to parse telescope response",
			zap.Error(err),
			zap.String("response", string(response)))  // Log the malformed response
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return the payload to the client
	// msg.Payload contains the actual status data from the coordinator
	// Gin automatically serializes this to JSON
	// StatusOK = 200, meaning success
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeSlewHandler commands the telescope to slew to specific coordinates
// Slewing is the process of moving the telescope to point at a target
// This is a protected endpoint - requires authentication
// This operation can take 10-30 seconds depending on how far the telescope moves
func telescopeSlewHandler(c *gin.Context) {
	// Get dependencies from context
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Parse and validate the request body
	// ShouldBindJSON will:
	//   1. Parse JSON from request body
	//   2. Check that all required fields are present
	//   3. Validate data types match
	var req SlewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Validation failed - return 400 Bad Request
		// This means the client sent invalid data
		// err.Error() will describe what was wrong
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),  // e.g., "Key: 'SlewRequest.RA' Error:Field validation for 'RA' failed"
		})
		return
	}

	// Validate coordinate ranges
	// These checks ensure physically valid coordinates
	// Invalid coordinates could damage telescope or point at ground
	
	// RA must be between 0 and 24 (hours)
	// RA is like longitude - wraps around the sky
	// 0 hours = 24 hours (same position)
	if req.RA < 0 || req.RA > 24 {
		// Return error explaining the constraint
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "RA must be between 0 and 24 hours",
		})
		return
	}
	
	// Dec must be between -90 and +90 (degrees)
	// -90 = south celestial pole
	// 0 = celestial equator
	// +90 = north celestial pole
	if req.Dec < -90 || req.Dec > 90 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Dec must be between -90 and +90 degrees",
		})
		return
	}

	// Build command payload
	// This tells the coordinator to execute a slew command
	payload := map[string]interface{}{
		"action": "slew_to_coordinates",  // The command to execute
		"ra":     req.RA,                   // Target RA
		"dec":    req.Dec,                  // Target Dec
	}

	// Log the slew command for audit trail
	// Important to log telescope movements for safety and debugging
	logger.Info("Slewing telescope",
		zap.String("user", c.GetString("username")),  // Who commanded it
		zap.Float64("ra", req.RA),                     // Where to
		zap.Float64("dec", req.Dec))

	// Send slew command via MQTT
	// Note: longer timeout (30 seconds) because slewing takes time
	// The actual slew might take 10-30 seconds depending on distance
	// Coordinator will respond when slew is complete (or fails)
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/command/slew",  // Command topic (not query)
		payload,
		30*time.Second,  // Generous timeout for physical movement
	)

	// Check for errors
	if err != nil {
		// Slew command failed or timed out
		// Could be:
		//   - Timeout (telescope took too long)
		//   - Mechanical error
		//   - Safety limit hit
		logger.Error("Slew command failed", 
			zap.Error(err),
			zap.String("user", c.GetString("username")),
			zap.Float64("ra", req.RA),
			zap.Float64("dec", req.Dec))
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute slew command",
		})
		return
	}

	// Parse response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return success response with any data from coordinator
	// Payload might include final position, slew duration, etc.
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeConnectHandler establishes connection to telescope
// Must be called before any other telescope operations
// This is like "turning on" the telescope control
func telescopeConnectHandler(c *gin.Context) {
	// Get dependencies
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Parse request body
	// Device number is optional - defaults to 0 if not provided
	// Multiple telescopes can be controlled; each has a device number
	var req struct {
		// DeviceNumber identifies which telescope to connect to
		// Usually 0 for the first/only telescope
		DeviceNumber int `json:"device_number"`
	}

	// Try to parse JSON, but don't fail if body is empty or invalid
	// We'll use a default value if parsing fails
	if err := c.ShouldBindJSON(&req); err != nil {
		// No device number provided or invalid JSON
		// Use device 0 as default (first telescope)
		req.DeviceNumber = 0
		
		// Log that we're using default
		logger.Debug("Using default device number",
			zap.Int("device", req.DeviceNumber))
	}

	// Build command payload
	payload := map[string]interface{}{
		"action":        "connect",         // Command to connect
		"device_number": req.DeviceNumber,  // Which device to connect to
	}

	// Log connection attempt
	logger.Info("Connecting to telescope",
		zap.String("user", c.GetString("username")),
		zap.Int("device", req.DeviceNumber))

	// Send connect command
	// 10 second timeout - connecting can take a few seconds
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/command/connect",
		payload,
		10*time.Second,  // Connecting might take time
	)

	if err != nil {
		logger.Error("Connect command failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to connect to telescope",
		})
		return
	}

	// Parse response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return success - telescope is now connected and ready for commands
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeDisconnectHandler closes connection to telescope
// Should be called when done observing to release resources
// This is like "turning off" the telescope control
func telescopeDisconnectHandler(c *gin.Context) {
	// Get MQTT client
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Build command payload
	payload := map[string]interface{}{
		"action": "disconnect",  // Command to disconnect
	}

	// Log disconnect
	logger.Info("Disconnecting from telescope",
		zap.String("user", c.GetString("username")))

	// Send disconnect command
	// Note: We use Publish (not PublishAndWait) here
	// Disconnect is a "fire and forget" operation
	// We don't need to wait for confirmation
	mqtt.Publish("bigskies/telescope/command/disconnect", payload)

	// Return immediate success
	// The disconnect will happen asynchronously
	c.JSON(http.StatusOK, gin.H{
		"message": "Disconnect command sent",
	})
}

// telescopeParkHandler parks the telescope at its safe storage position
// Parking moves telescope to a predefined safe position
// Should always park when done observing to protect telescope
func telescopeParkHandler(c *gin.Context) {
	// Get dependencies
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Build command payload
	payload := map[string]interface{}{
		"action": "park",  // Command to park telescope
	}

	// Log park command
	logger.Info("Parking telescope",
		zap.String("user", c.GetString("username")))

	// Send park command
	// 30 second timeout - parking involves physical movement
	// Telescope will slew to park position before confirming
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/command/park",
		payload,
		30*time.Second,  // Parking can take time (moving to park position)
	)

	if err != nil {
		logger.Error("Park command failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to park telescope",
		})
		return
	}

	// Parse response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return success - telescope is now parked
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeUnparkHandler unparks the telescope
// Must unpark before you can slew to targets
// This readies the telescope for observation
func telescopeUnparkHandler(c *gin.Context) {
	// Get dependencies
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Build command payload
	payload := map[string]interface{}{
		"action": "unpark",  // Command to unpark telescope
	}

	// Log unpark command
	logger.Info("Unparking telescope",
		zap.String("user", c.GetString("username")))

	// Send unpark command
	// 10 second timeout - unpark is usually quick
	response, err := mqtt.PublishAndWait(
		"bigskies/telescope/command/unpark",
		payload,
		10*time.Second,
	)

	if err != nil {
		logger.Error("Unpark command failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to unpark telescope",
		})
		return
	}

	// Parse response
	var msg BigSkiesMessage
	if err := json.Unmarshal(response, &msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse response",
		})
		return
	}

	// Return success - telescope is now unparked and ready
	c.JSON(http.StatusOK, msg.Payload)
}

// telescopeAbortHandler aborts the current telescope operation
// Use this to stop a slew in progress (emergency stop)
// This is a safety feature - always available regardless of state
func telescopeAbortHandler(c *gin.Context) {
	// Get MQTT client
	mqtt := c.MustGet("mqtt").(*MQTTClient)
	logger := c.MustGet("logger").(*zap.Logger)

	// Build command payload
	payload := map[string]interface{}{
		"action": "abort_slew",  // Emergency stop command
	}

	// Log abort command
	// Important for safety audit trail
	logger.Warn("Aborting telescope operation",  // Warn level - this is unusual
		zap.String("user", c.GetString("username")))

	// Send abort command
	// Fire and forget - abort must be immediate
	// Don't wait for confirmation - action is time-critical
	mqtt.Publish("bigskies/telescope/command/abort", payload)

	// Return immediate success
	// Abort command has been sent to telescope
	c.JSON(http.StatusOK, gin.H{
		"message": "Abort command sent",
	})
}
```

### Step 2: Add Telescope Routes to main.go

Update the protected routes section in your `main.go`:

```go
// In main.go, update the protected routes group:

	// Protected routes (authentication required)
	// Create a route group that uses the auth middleware
	protected := router.Group("/api")
	protected.Use(authMiddleware())  // Apply authentication to all routes in this group
	{
		// User profile endpoint
		protected.GET("/profile", profileHandler)
		
		// Coordinator management
		protected.GET("/coordinators", listCoordinatorsHandler)
		
		// Telescope status and control endpoints
		// GET endpoints query state without changing anything
		protected.GET("/telescope/status", telescopeStatusHandler)
		
		// POST endpoints issue commands that change telescope state
		// Connection management
		protected.POST("/telescope/connect", telescopeConnectHandler)
		protected.POST("/telescope/disconnect", telescopeDisconnectHandler)
		
		// Movement commands
		protected.POST("/telescope/slew", telescopeSlewHandler)
		protected.POST("/telescope/abort", telescopeAbortHandler)
		
		// Parking commands
		protected.POST("/telescope/park", telescopeParkHandler)
		protected.POST("/telescope/unpark", telescopeUnparkHandler)
	}
```

### Step 3: Test Telescope Control

First, make sure the ASCOM simulator is running:

```bash
# Start the ASCOM Alpaca Simulator (if not already running)
make plugin-ascom-up

# Verify it's accessible
curl http://localhost:32323
```

Now start your web service:

```bash
# Navigate to your project directory
cd tutorials/lesson4-web-service

# Run the web service
go run .
```

In another terminal, test the telescope endpoints:

```bash
# Get JWT token first (from Lesson 5)
# Login to get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' \
  | grep -o '"token":"[^"]*' | cut -d'"' -f4)

echo "Token: $TOKEN"

# 1. Connect to telescope
# This establishes connection to the ASCOM simulator
curl -X POST http://localhost:8080/api/telescope/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"device_number": 0}'

# Expected response: {"connected": true, ...}

# 2. Get telescope status
# This queries current position and state
curl http://localhost:8080/api/telescope/status \
  -H "Authorization: Bearer $TOKEN"

# Expected response includes:
# - connected: true
# - tracking: false/true
# - right_ascension: current RA
# - declination: current Dec

# 3. Unpark telescope (if it's parked)
# Telescope must be unparked before slewing
curl -X POST http://localhost:8080/api/telescope/unpark \
  -H "Authorization: Bearer $TOKEN"

# 4. Slew to coordinates
# Move telescope to point at specific sky coordinates
# Example: RA=10.5 hours, Dec=45 degrees
# This would point at a location in the constellation Leo
curl -X POST http://localhost:8080/api/telescope/slew \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ra": 10.5, "dec": 45.0}'

# This will take 10-30 seconds to complete
# The curl command will wait for the slew to finish

# 5. Check status again to see new position
curl http://localhost:8080/api/telescope/status \
  -H "Authorization: Bearer $TOKEN"

# Should show new RA and Dec coordinates

# 6. Park telescope when done
# Always park to protect the telescope
curl -X POST http://localhost:8080/api/telescope/park \
  -H "Authorization: Bearer $TOKEN"

# 7. Test abort (emergency stop)
# Start a slew, then abort it mid-movement
# First, start a slew in background:
curl -X POST http://localhost:8080/api/telescope/slew \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ra": 5.0, "dec": -30.0}' &

# Immediately abort:
curl -X POST http://localhost:8080/api/telescope/abort \
  -H "Authorization: Bearer $TOKEN"

# 8. Disconnect when completely done
curl -X POST http://localhost:8080/api/telescope/disconnect \
  -H "Authorization: Bearer $TOKEN"
```

### Understanding Telescope Operation Flow

**Normal Operation Sequence**:
1. **Connect** - Establish connection to telescope
2. **Unpark** - Take telescope out of storage position
3. **Slew** - Move to target coordinates (repeat as needed)
4. **Park** - Return to safe storage position
5. **Disconnect** - Release telescope connection

**Error Handling**:
- All commands check for errors and return appropriate HTTP status codes
- Timeouts are set appropriately for each operation type
- Logging provides audit trail of all telescope movements

**Safety Features**:
- Coordinate validation prevents invalid targets
- Abort command available as emergency stop
- Parking enforced before disconnect (in production)
- User authentication required for all commands

---

## Lesson 7: Complete Web Application

Now let's build a complete single-page web application with an HTML/JavaScript frontend that provides a full-featured telescope control interface.

### Understanding the Frontend Architecture

**Technology Stack**:
- **HTML5**: Structure and content
- **CSS3**: Styling and layout (no external frameworks needed)
- **Vanilla JavaScript**: All functionality (no React/Vue/Angular)
- **Fetch API**: For HTTP requests to our backend
- **LocalStorage**: For storing JWT token

**Key Concepts**:
- **SPA (Single Page Application)**: No page reloads, all content swapped via JavaScript
- **JWT Token Management**: Store token, include in every API request
- **Real-time Updates**: Poll telescope status every 3 seconds
- **Responsive Design**: Works on desktop and mobile

### Step 1: Create Static Directory Structure

```bash
# Navigate to your project directory
cd tutorials/lesson4-web-service

# Create directory for static files (HTML, CSS, JS)
mkdir -p static

# This directory will hold all frontend files
# Gin will serve these files to browsers
```

### Step 2: Create HTML Frontend (static/index.html)

Create the complete web interface:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <!-- Meta tags for proper rendering and character encoding -->
    <meta charset="UTF-8">
    <!-- Viewport meta tag makes site responsive on mobile devices -->
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <!-- Page title shown in browser tab -->
    <title>Big Skies Telescope Control</title>
    
    <!-- Embedded CSS styles -->
    <!-- We use embedded CSS to keep everything in one file -->
    <!-- In production, you'd move this to a separate .css file -->
    <style>
        /* Universal selector - applies to all elements */
        /* Reset default browser styles for consistency */
        * {
            margin: 0;           /* Remove default margins */
            padding: 0;          /* Remove default padding */
            box-sizing: border-box;  /* Include padding/border in element width */
        }
        
        /* Body styles - applies to entire page */
        body {
            /* Sans-serif fonts for clean, modern look */
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            
            /* Gradient background from dark blue to lighter blue */
            /* 135deg = diagonal from bottom-left to top-right */
            background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
            
            /* Minimum height of viewport - prevents background from cutting off */
            min-height: 100vh;
            
            /* Padding around entire page */
            padding: 20px;
        }
        
        /* Container for all content - centers and constrains width */
        .container {
            max-width: 1200px;     /* Don't exceed 1200px even on wide screens */
            margin: 0 auto;        /* Center horizontally */
        }
        
        /* Header section styling */
        header {
            /* Semi-transparent white background with blur effect */
            /* This creates a "frosted glass" appearance */
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);  /* Blur what's behind the header */
            
            padding: 20px;
            border-radius: 10px;   /* Rounded corners */
            margin-bottom: 20px;   /* Space below header */
            color: white;          /* White text */
        }
        
        /* Main heading in header */
        header h1 {
            font-size: 2.5rem;     /* Large font size (40px) */
            margin-bottom: 10px;   /* Space below heading */
        }
        
        /* Login and dashboard cards - these contain main content */
        .login-card, .dashboard-card {
            background: white;           /* Solid white background */
            padding: 30px;              /* Internal spacing */
            border-radius: 10px;        /* Rounded corners */
            /* Shadow creates depth - makes card appear to float */
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
        }
        
        /* Hidden class - used to show/hide sections via JavaScript */
        .hidden {
            display: none !important;  /* !important overrides other display rules */
        }
        
        /* Form group - container for each form field */
        .form-group {
            margin-bottom: 20px;  /* Space between form fields */
        }
        
        /* Labels for form inputs */
        label {
            display: block;       /* Take full width, start on new line */
            margin-bottom: 5px;   /* Space between label and input */
            font-weight: bold;    /* Make labels stand out */
            color: #333;          /* Dark gray text */
        }
        
        /* All text and number inputs */
        input[type="text"],
        input[type="password"],
        input[type="email"],
        input[type="number"] {
            width: 100%;          /* Full width of container */
            padding: 12px;        /* Internal spacing */
            border: 2px solid #ddd;  /* Light gray border */
            border-radius: 5px;   /* Slightly rounded corners */
            font-size: 1rem;      /* Normal text size */
        }
        
        /* Button styling */
        button {
            background: #2a5298;  /* Blue background */
            color: white;         /* White text */
            border: none;         /* Remove default border */
            padding: 12px 24px;   /* Vertical and horizontal padding */
            border-radius: 5px;   /* Rounded corners */
            font-size: 1rem;      /* Normal text size */
            cursor: pointer;      /* Hand cursor on hover */
            /* Smooth transition when hovering */
            transition: background 0.3s;
        }
        
        /* Button hover state - darker blue */
        button:hover {
            background: #1e3c72;
        }
        
        /* Disabled button state - gray and non-interactive */
        button:disabled {
            background: #ccc;     /* Light gray */
            cursor: not-allowed;  /* "Not allowed" cursor */
        }
        
        /* Grid layout for status cards */
        .status-grid {
            /* CSS Grid - automatically sizes columns */
            display: grid;
            /* auto-fit: create as many columns as fit */
            /* minmax(250px, 1fr): each column min 250px, grow to fill space */
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;            /* Space between grid items */
            margin-bottom: 30px;  /* Space below grid */
        }
        
        /* Individual status card */
        .status-card {
            background: #f8f9fa;  /* Very light gray background */
            padding: 20px;
            border-radius: 8px;
            /* Blue left border for accent */
            border-left: 4px solid #2a5298;
        }
        
        /* Status card heading */
        .status-card h3 {
            margin-bottom: 10px;
            color: #333;
        }
        
        /* Status value - the actual data displayed */
        .status-value {
            font-size: 1.5rem;    /* Larger text */
            font-weight: bold;    /* Bold */
            color: #2a5298;       /* Blue color */
        }
        
        /* Control section - groups related controls */
        .control-section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        
        /* Control section heading */
        .control-section h2 {
            margin-bottom: 15px;
            color: #333;
        }
        
        /* Button group - horizontal row of buttons */
        .button-group {
            display: flex;        /* Flexbox layout */
            gap: 10px;           /* Space between buttons */
            flex-wrap: wrap;     /* Wrap to new line if needed */
        }
        
        /* Alert messages */
        .alert {
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        
        /* Success alert - green */
        .alert-success {
            background: #d4edda;  /* Light green background */
            color: #155724;       /* Dark green text */
            border: 1px solid #c3e6cb;  /* Green border */
        }
        
        /* Error alert - red */
        .alert-error {
            background: #f8d7da;  /* Light red background */
            color: #721c24;       /* Dark red text */
            border: 1px solid #f5c6cb;  /* Red border */
        }
        
        /* Coordinate input grid - side-by-side RA and Dec inputs */
        .coordinate-input {
            display: grid;
            grid-template-columns: 1fr 1fr;  /* Two equal columns */
            gap: 15px;  /* Space between columns */
        }
        
        /* Logout button - red for danger/warning */
        .logout-btn {
            background: #dc3545;  /* Red background */
            float: right;         /* Position on right side */
        }
        
        /* Logout button hover - darker red */
        .logout-btn:hover {
            background: #c82333;
        }
    </style>
</head>
<body>
    <!-- Main container holds all page content -->
    <div class="container">
        <!-- Header section - always visible -->
        <header>
            <!-- 🔭 is telescope emoji -->
            <h1>🔭 Big Skies Telescope Control</h1>
            <p>Web Interface for ASCOM Alpaca Telescope Control</p>
        </header>

        <!-- Login Section -->
        <!-- This is shown when user is not logged in -->
        <!-- Initially visible, hidden after login -->
        <div id="loginSection" class="login-card">
            <h2>Login</h2>
            <!-- Alert container for login messages -->
            <!-- Populated by JavaScript when login succeeds/fails -->
            <div id="loginAlert"></div>
            
            <!-- Login form -->
            <!-- onsubmit would normally submit to server -->
            <!-- We handle it with JavaScript instead -->
            <form id="loginForm">
                <div class="form-group">
                    <label>Username</label>
                    <!-- required attribute provides HTML5 validation -->
                    <input type="text" id="loginUsername" required>
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="loginPassword" required>
                </div>
                <!-- type="submit" makes Enter key submit form -->
                <button type="submit">Login</button>
            </form>
            <p style="margin-top: 20px;">
                Don't have an account? <a href="#" id="showRegister">Register here</a>
            </p>
        </div>

        <!-- Register Section -->
        <!-- Initially hidden - shown when user clicks "Register here" -->
        <div id="registerSection" class="login-card hidden">
            <h2>Register</h2>
            <div id="registerAlert"></div>
            <form id="registerForm">
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" id="regUsername" required>
                </div>
                <div class="form-group">
                    <label>Email</label>
                    <!-- type="email" provides email format validation -->
                    <input type="email" id="regEmail" required>
                </div>
                <div class="form-group">
                    <label>Password (min 8 characters)</label>
                    <!-- minlength attribute enforces minimum length -->
                    <input type="password" id="regPassword" minlength="8" required>
                </div>
                <button type="submit">Register</button>
            </form>
            <p style="margin-top: 20px;">
                Already have an account? <a href="#" id="showLogin">Login here</a>
            </p>
        </div>

        <!-- Dashboard Section -->
        <!-- Initially hidden - shown after successful login -->
        <div id="dashboardSection" class="dashboard-card hidden">
            <!-- Dashboard header with logout button -->
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h2>Telescope Dashboard</h2>
                <!-- onclick calls JavaScript function directly -->
                <button class="logout-btn" onclick="logout()">Logout</button>
            </div>
            
            <!-- Alert container for dashboard messages -->
            <div id="dashboardAlert"></div>

            <!-- Status Display Grid -->
            <!-- Shows real-time telescope status -->
            <!-- Updated every 3 seconds by JavaScript -->
            <div class="status-grid" id="statusGrid">
                <!-- Connection status card -->
                <div class="status-card">
                    <h3>Connection</h3>
                    <!-- This div will be updated by JavaScript -->
                    <div class="status-value" id="statusConnected">-</div>
                </div>
                <!-- Tracking status card -->
                <div class="status-card">
                    <h3>Tracking</h3>
                    <div class="status-value" id="statusTracking">-</div>
                </div>
                <!-- Right Ascension display -->
                <div class="status-card">
                    <h3>Right Ascension</h3>
                    <div class="status-value" id="statusRA">-</div>
                </div>
                <!-- Declination display -->
                <div class="status-card">
                    <h3>Declination</h3>
                    <div class="status-value" id="statusDec">-</div>
                </div>
            </div>

            <!-- Connection Control Section -->
            <div class="control-section">
                <h2>Connection</h2>
                <div class="button-group">
                    <!-- onclick attributes call JavaScript functions -->
                    <button onclick="connectTelescope()">Connect</button>
                    <button onclick="disconnectTelescope()">Disconnect</button>
                </div>
            </div>

            <!-- Slew Control Section -->
            <div class="control-section">
                <h2>Slew to Coordinates</h2>
                <!-- Two-column grid for RA and Dec inputs -->
                <div class="coordinate-input">
                    <div class="form-group">
                        <label>Right Ascension (0-24 hours)</label>
                        <!-- type="number" provides numeric keyboard on mobile -->
                        <!-- step="0.1" allows decimal values -->
                        <!-- min/max enforce valid ranges -->
                        <input type="number" id="slewRA" step="0.1" min="0" max="24" value="12.0">
                    </div>
                    <div class="form-group">
                        <label>Declination (-90 to 90 degrees)</label>
                        <input type="number" id="slewDec" step="0.1" min="-90" max="90" value="45.0">
                    </div>
                </div>
                <div class="button-group">
                    <button onclick="slewTelescope()">Slew</button>
                    <button onclick="abortSlew()">Abort</button>
                </div>
            </div>

            <!-- Park Control Section -->
            <div class="control-section">
                <h2>Park/Unpark</h2>
                <div class="button-group">
                    <button onclick="parkTelescope()">Park</button>
                    <button onclick="unparkTelescope()">Unpark</button>
                </div>
            </div>
        </div>
    </div>

    <!-- JavaScript code -->
    <!-- This is where all the interactive functionality lives -->
    <script>
        // Global variables
        // authToken stores the JWT token for authentication
        // Try to load it from localStorage (persists across page reloads)
        // If not found, will be null
        let authToken = localStorage.getItem('authToken');
        
        // statusUpdateInterval stores the interval ID for status updates
        // We'll use this to stop updates when user logs out
        let statusUpdateInterval;

        // Initialize on page load
        // Check if we already have a token (user was logged in before)
        if (authToken) {
            // User has token - show dashboard immediately
            // This provides seamless experience - user stays logged in
            showDashboard();
        }
        // If no token, login form is shown by default (not hidden)

        // Toggle between login and register forms
        // These event listeners respond to "Register here" and "Login here" links
        
        // "Register here" link clicked
        document.getElementById('showRegister').addEventListener('click', (e) => {
            // e.preventDefault() stops the default link behavior
            // Without this, clicking the link would navigate to "#"
            e.preventDefault();
            
            // Hide login section by adding 'hidden' class
            document.getElementById('loginSection').classList.add('hidden');
            
            // Show register section by removing 'hidden' class
            document.getElementById('registerSection').classList.remove('hidden');
        });

        // "Login here" link clicked
        document.getElementById('showLogin').addEventListener('click', (e) => {
            e.preventDefault();
            // Hide register, show login
            document.getElementById('registerSection').classList.add('hidden');
            document.getElementById('loginSection').classList.remove('hidden');
        });

        // Login form submission handler
        // addEventListener is better than onclick - allows multiple handlers
        document.getElementById('loginForm').addEventListener('submit', async (e) => {
            // Prevent default form submission
            // Default would reload page, but we want to handle it with JavaScript
            e.preventDefault();
            
            // Get values from form inputs
            // .value gets the current text in the input field
            const username = document.getElementById('loginUsername').value;
            const password = document.getElementById('loginPassword').value;

            // Try to login
            // async/await makes asynchronous code look synchronous
            // Much easier to read than callbacks or .then() chains
            try {
                // fetch() makes HTTP request
                // Returns a Promise that resolves to the Response
                const response = await fetch('/api/auth/login', {
                    method: 'POST',  // HTTP POST method
                    headers: { 
                        'Content-Type': 'application/json'  // Tell server we're sending JSON
                    },
                    // JSON.stringify converts JavaScript object to JSON string
                    body: JSON.stringify({ username, password })
                });

                // Parse response body as JSON
                // .json() also returns a Promise
                const data = await response.json();

                // Check if request succeeded
                // response.ok is true for status codes 200-299
                if (response.ok) {
                    // Login succeeded!
                    
                    // Store token in memory
                    authToken = data.token;
                    
                    // Store token in localStorage so it persists across page reloads
                    // localStorage is a browser API for storing key-value pairs
                    // Data persists even when browser is closed
                    localStorage.setItem('authToken', authToken);
                    
                    // Show success message
                    showAlert('loginAlert', 'Login successful!', 'success');
                    
                    // After 1 second, show dashboard
                    // setTimeout delays execution
                    // Gives user time to see success message
                    setTimeout(showDashboard, 1000);
                } else {
                    // Login failed - show error
                    // data.error contains error message from server
                    showAlert('loginAlert', data.error || 'Login failed', 'error');
                }
            } catch (error) {
                // Network error or other exception
                // This happens if:
                //   - Server is down
                //   - No internet connection
                //   - CORS issue
                //   - JavaScript error in fetch
                showAlert('loginAlert', 'Network error: ' + error.message, 'error');
            }
        });

        // Register form submission handler
        // Similar structure to login handler
        document.getElementById('registerForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            // Get form values
            const username = document.getElementById('regUsername').value;
            const email = document.getElementById('regEmail').value;
            const password = document.getElementById('regPassword').value;

            try {
                // Send registration request
                const response = await fetch('/api/auth/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, email, password })
                });

                const data = await response.json();

                if (response.ok) {
                    // Registration succeeded
                    showAlert('registerAlert', 'Registration successful! Please login.', 'success');
                    
                    // After 2 seconds, switch to login form
                    setTimeout(() => {
                        document.getElementById('registerSection').classList.add('hidden');
                        document.getElementById('loginSection').classList.remove('hidden');
                    }, 2000);
                } else {
                    // Registration failed
                    showAlert('registerAlert', data.error || 'Registration failed', 'error');
                }
            } catch (error) {
                showAlert('registerAlert', 'Network error: ' + error.message, 'error');
            }
        });

        // showDashboard switches from login view to dashboard view
        // Called after successful login or on page load if token exists
        function showDashboard() {
            // Hide both login and register sections
            document.getElementById('loginSection').classList.add('hidden');
            document.getElementById('registerSection').classList.add('hidden');
            
            // Show dashboard
            document.getElementById('dashboardSection').classList.remove('hidden');
            
            // Start updating telescope status
            // Call updateStatus immediately (don't wait for first interval)
            updateStatus();
            
            // Set up interval to update status every 3 seconds
            // setInterval calls function repeatedly at specified interval
            // Returns interval ID that we can use to stop it later
            statusUpdateInterval = setInterval(updateStatus, 3000);
        }

        // logout clears authentication and returns to login screen
        function logout() {
            // Clear token from memory
            authToken = null;
            
            // Remove token from localStorage
            // Without this, user would be logged back in on page reload
            localStorage.removeItem('authToken');
            
            // Stop status updates
            // clearInterval stops the interval created by setInterval
            clearInterval(statusUpdateInterval);
            
            // Hide dashboard
            document.getElementById('dashboardSection').classList.add('hidden');
            
            // Show login form
            document.getElementById('loginSection').classList.remove('hidden');
        }

        // apiCall is a helper function for making authenticated API requests
        // It adds the JWT token to every request automatically
        // Parameters:
        //   url: API endpoint to call
        //   method: HTTP method (GET, POST, etc.)
        //   body: request body (optional, for POST/PUT)
        // Returns: Promise that resolves to parsed JSON response
        async function apiCall(url, method = 'GET', body = null) {
            // Build request options
            const options = {
                method,  // HTTP method
                headers: {
                    // Authorization header with Bearer token
                    // This is what authMiddleware checks
                    'Authorization': `Bearer ${authToken}`,
                    'Content-Type': 'application/json'
                }
            };

            // If body provided, add it to request
            // Only for POST/PUT requests
            if (body) {
                options.body = JSON.stringify(body);
            }

            // Make the request
            const response = await fetch(url, options);
            
            // Check if request was unauthorized (401)
            // This means token is invalid or expired
            if (response.status === 401) {
                // Token is bad - log user out
                logout();
                // Throw error to stop execution
                throw new Error('Unauthorized');
            }

            // Parse and return response
            return response.json();
        }

        // updateStatus queries telescope status and updates display
        // Called every 3 seconds to show real-time status
        async function updateStatus() {
            try {
                // Query telescope status
                const data = await apiCall('/api/telescope/status');
                
                // Update connection status
                // Uses emoji and text to show clearly
                document.getElementById('statusConnected').textContent = 
                    data.connected ? '✅ Connected' : '❌ Disconnected';
                
                // Update tracking status
                document.getElementById('statusTracking').textContent = 
                    data.tracking ? '✅ Tracking' : '❌ Not Tracking';
                
                // Update Right Ascension
                // .toFixed(4) formats number with 4 decimal places
                // ? operator checks if value exists (null checking)
                document.getElementById('statusRA').textContent = 
                    data.right_ascension ? data.right_ascension.toFixed(4) + 'h' : '-';
                
                // Update Declination
                document.getElementById('statusDec').textContent = 
                    data.declination ? data.declination.toFixed(4) + '°' : '-';
            } catch (error) {
                // Status update failed
                // Don't show error to user - just log to console
                // Errors are expected if telescope coordinator is down
                console.error('Failed to update status:', error);
            }
        }

        // connectTelescope establishes connection to telescope
        async function connectTelescope() {
            try {
                // Send connect command
                // body specifies device number (0 = first telescope)
                const data = await apiCall('/api/telescope/connect', 'POST', { device_number: 0 });
                
                // Show success message
                showAlert('dashboardAlert', 'Telescope connected', 'success');
                
                // Update status immediately to show connection
                updateStatus();
            } catch (error) {
                // Connection failed
                showAlert('dashboardAlert', 'Failed to connect: ' + error.message, 'error');
            }
        }

        // disconnectTelescope closes connection to telescope
        async function disconnectTelescope() {
            try {
                await apiCall('/api/telescope/disconnect', 'POST');
                showAlert('dashboardAlert', 'Telescope disconnected', 'success');
                updateStatus();
            } catch (error) {
                showAlert('dashboardAlert', 'Failed to disconnect: ' + error.message, 'error');
            }
        }

        // slewTelescope commands telescope to move to coordinates
        async function slewTelescope() {
            // Get coordinate values from input fields
            // parseFloat converts string to floating-point number
            const ra = parseFloat(document.getElementById('slewRA').value);
            const dec = parseFloat(document.getElementById('slewDec').value);

            // Validate RA range (0-24 hours)
            if (ra < 0 || ra > 24) {
                showAlert('dashboardAlert', 'RA must be between 0 and 24', 'error');
                return;  // Stop execution
            }

            // Validate Dec range (-90 to +90 degrees)
            if (dec < -90 || dec > 90) {
                showAlert('dashboardAlert', 'Dec must be between -90 and 90', 'error');
                return;
            }

            try {
                // Show that slewing is starting
                showAlert('dashboardAlert', 'Slewing telescope...', 'success');
                
                // Send slew command
                // This will take 10-30 seconds to complete
                // await blocks until telescope finishes moving
                await apiCall('/api/telescope/slew', 'POST', { ra, dec });
                
                // Slew completed successfully
                showAlert('dashboardAlert', 'Slew completed', 'success');
                
                // Update status to show new position
                updateStatus();
            } catch (error) {
                showAlert('dashboardAlert', 'Slew failed: ' + error.message, 'error');
            }
        }

        // abortSlew sends emergency stop command
        async function abortSlew() {
            try {
                await apiCall('/api/telescope/abort', 'POST');
                showAlert('dashboardAlert', 'Slew aborted', 'success');
            } catch (error) {
                showAlert('dashboardAlert', 'Failed to abort: ' + error.message, 'error');
            }
        }

        // parkTelescope moves telescope to safe storage position
        async function parkTelescope() {
            try {
                showAlert('dashboardAlert', 'Parking telescope...', 'success');
                
                // Send park command
                // This will take 10-30 seconds
                await apiCall('/api/telescope/park', 'POST');
                
                showAlert('dashboardAlert', 'Telescope parked', 'success');
                updateStatus();
            } catch (error) {
                showAlert('dashboardAlert', 'Park failed: ' + error.message, 'error');
            }
        }

        // unparkTelescope takes telescope out of storage position
        async function unparkTelescope() {
            try {
                await apiCall('/api/telescope/unpark', 'POST');
                showAlert('dashboardAlert', 'Telescope unparked', 'success');
                updateStatus();
            } catch (error) {
                showAlert('dashboardAlert', 'Unpark failed: ' + error.message, 'error');
            }
        }

        // showAlert displays a temporary alert message
        // Parameters:
        //   containerId: ID of element to show alert in
        //   message: text to display
        //   type: 'success' or 'error'
        function showAlert(containerId, message, type) {
            // Get the alert container element
            const container = document.getElementById(containerId);
            
            // Set innerHTML to create the alert div
            // Uses CSS classes defined in <style> section
            container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
            
            // After 5 seconds, clear the alert
            // This prevents alerts from piling up
            setTimeout(() => {
                container.innerHTML = '';  // Clear content
            }, 5000);  // 5000 milliseconds = 5 seconds
        }
    </script>
</body>
</html>
```

### Step 3: Update main.go to Serve Static Files

Add routes to serve the HTML interface:

```go
// In main.go, add these routes BEFORE your API routes:

	// Serve static files (CSS, JavaScript, images)
	// This makes files in the "static" directory available via HTTP
	// Example: static/style.css becomes available at /static/style.css
	router.Static("/static", "./static")
	
	// Serve index.html at root URL
	// When user visits http://localhost:8080/ they get the web interface
	router.GET("/", func(c *gin.Context) {
		// c.File sends a file as the response
		// This serves our complete web application
		c.File("./static/index.html")
	})
```

### Step 4: Run the Complete Application

```bash
# Make sure Big Skies services are running
make docker-up

# Make sure ASCOM simulator is running
make plugin-ascom-up

# Start your web service
cd tutorials/lesson4-web-service
go run .
```

### Step 5: Use the Web Interface

Open your web browser and navigate to: **http://localhost:8080**

**Complete Usage Flow**:

1. **Register Account**
   - Click "Register here"
   - Enter username, email, password (min 8 characters)
   - Click "Register"
   - Wait for "Registration successful" message
   - You'll automatically be switched to login screen

2. **Login**
   - Enter your username and password
   - Click "Login"
   - Dashboard will appear

3. **Connect to Telescope**
   - Click "Connect" button in Connection section
   - Status will update to show "✅ Connected"

4. **Unpark Telescope**
   - Click "Unpark" in Park/Unpark section
   - This readies telescope for movement

5. **Slew to Coordinates**
   - Enter RA (e.g., 10.5 for 10h 30m)
   - Enter Dec (e.g., 45.0 for 45°)
   - Click "Slew"
   - Watch status display update as telescope moves
   - "Slew completed" message appears when done

6. **Monitor Real-Time Status**
   - Status cards update every 3 seconds automatically
   - Shows: Connection, Tracking, RA, Dec

7. **Park and Disconnect**
   - Click "Park" when done observing
   - Wait for telescope to reach park position
   - Click "Disconnect" to release telescope

8. **Logout**
   - Click "Logout" button (red, top-right)
   - Returns to login screen
   - Token is cleared from browser storage

### Understanding the Complete System

**Data Flow**:
```
Browser (JavaScript)
  ↓ HTTP POST with JWT token
Web Service (Go)
  ↓ MQTT message with unique ID
Telescope Coordinator (Go)
  ↓ HTTP request
ASCOM Alpaca Simulator
  ↓ Response
Telescope Coordinator
  ↓ MQTT response with matching ID
Web Service
  ↓ HTTP response (JSON)
Browser (JavaScript updates display)
```

**Security Layers**:
1. **Authentication**: JWT token required for all telescope commands
2. **Authorization**: Token verified on every request
3. **Validation**: Coordinates validated on both client and server
4. **Audit Trail**: All commands logged with username

**User Experience Features**:
1. **Persistent Login**: Token stored in localStorage
2. **Real-Time Updates**: Status polls every 3 seconds
3. **Visual Feedback**: Success/error messages for all operations
4. **Responsive Design**: Works on desktop and mobile
5. **Graceful Errors**: Network errors handled, user stays logged in

**Production Considerations**:
- **HTTPS Required**: Always use HTTPS in production for token security
- **Token Refresh**: Implement token refresh before expiration
- **Error Recovery**: Add retry logic for network failures
- **Accessibility**: Add ARIA labels for screen readers
- **Performance**: Consider WebSocket for real-time updates instead of polling
- **Mobile**: Add PWA manifest for installable app

### Testing the Complete System

**Test Sequence**:

```bash
# 1. Verify all services are running
docker ps | grep bigskies

# Should show:
# - bigskies-mqtt
# - bigskies-postgres  
# - message-coordinator
# - security-coordinator
# - telescope-coordinator
# - ascom-alpaca-simulator

# 2. Verify web service is running
curl http://localhost:8080/health

# Should return: {"status":"healthy",...}

# 3. Verify simulator is accessible
curl http://localhost:32323

# Should return HTML page

# 4. Open browser to test UI
open http://localhost:8080

# Follow the usage flow described above
```

**Common Issues and Solutions**:

| Issue | Cause | Solution |
|-------|-------|----------|
| "Failed to connect to telescope" | Simulator not running | `make plugin-ascom-up` |
| "Unauthorized" after login | Token expired | Logout and login again |
| Status shows "-" for all values | Telescope not connected | Click "Connect" button |
| Slew fails | Telescope is parked | Click "Unpark" first |
| "Network error" | Backend not running | Start web service with `go run .` |

---

## Appendix: Common Patterns and Best Practices

### A. Error Handling Pattern

```go
// Always check errors immediately after operations that can fail
// Don't ignore errors - handle them appropriately
response, err := mqtt.PublishAndWait(topic, payload, timeout)
if err != nil {
	// Log the error for debugging
	// Include context: what operation failed, relevant parameters
	logger.Error("Operation failed", 
		zap.Error(err),              // The error itself
		zap.String("topic", topic),  // What we were trying to do
		zap.Any("payload", payload)) // Relevant data
	
	// Return wrapped error with context
	// %w wraps the error, preserving the error chain
	// This allows callers to unwrap and check the original error
	return fmt.Errorf("operation failed: %w", err)
}
```

### B. Logging Best Practices

```go
// Use appropriate log levels:
// - Debug: Detailed info for debugging (not shown in production)
// - Info: General informational messages
// - Warn: Warning conditions (not errors, but noteworthy)
// - Error: Error conditions that need attention
// - Fatal: Critical errors that require immediate termination

// Include context with structured fields
// This makes logs searchable and parseable
logger.Info("Operation started",
    zap.String("user", username),       // Who
    zap.String("operation", "slew"),    // What
    zap.Float64("ra", ra),              // Parameters
    zap.Duration("timeout", timeout))   // Configuration
```

### C. MQTT Topic Naming Conventions

```go
// Big Skies follows a hierarchical topic structure:
// bigskies/{coordinator}/{category}/{action}/{optional-id}

// Commands - tell a coordinator to do something
"bigskies/telescope/command/slew"
"bigskies/security/command/create_user"

// Queries - ask a coordinator for information
"bigskies/telescope/query/status"
"bigskies/application/query/services"

// Responses - coordinator replies to commands/queries
"bigskies/telescope/response/slew/abc-123"
"bigskies/security/response/authenticate/def-456"

// Events - coordinator broadcasts state changes
"bigskies/telescope/event/position_changed"
"bigskies/security/event/user_logged_in"

// Health - coordinator reports health status
"bigskies/telescope/health"
"bigskies/message/health"
```

### D. Production Checklist

- [ ] **Security**
  - [ ] Change JWT secret to cryptographically random string
  - [ ] Store secrets in environment variables
  - [ ] Enable HTTPS/TLS
  - [ ] Implement rate limiting
  - [ ] Add CORS headers if needed

- [ ] **Database**
  - [ ] Configure connection pooling
  - [ ] Set appropriate timeouts
  - [ ] Enable connection health checks
  - [ ] Set up backups

- [ ] **Logging**
  - [ ] Use production logger (not development)
  - [ ] Configure log rotation
  - [ ] Set appropriate log levels
  - [ ] Don't log sensitive data

- [ ] **Monitoring**
  - [ ] Add health check endpoints
  - [ ] Implement metrics collection
  - [ ] Set up alerts
  - [ ] Monitor resource usage

- [ ] **Deployment**
  - [ ] Implement graceful shutdown
  - [ ] Configure process manager
  - [ ] Set up reverse proxy
  - [ ] Enable automatic restarts

---

## Conclusion

Congratulations! You've completed the heavily commented version of the Big Skies Framework tutorial. You now understand:

✅ **MQTT Communication** - How messages flow through the system  
✅ **Go Web Development** - Building APIs with Gin framework  
✅ **Authentication** - JWT tokens and middleware  
✅ **Asynchronous Patterns** - Goroutines and channels  
✅ **Error Handling** - Proper error checking and wrapping  
✅ **Code Organization** - Structuring a Go application  

### Next Steps

1. Complete Lessons 6-7 (telescope control and web UI)
2. Build your own plugin
3. Explore the coordinator source code
4. Contribute to the project

### Getting Help

- Read the inline comments in existing code
- Check the `/docs` folder for architecture details
- Review the coordinator implementations in `/internal/coordinators`
- Ask questions on GitHub Issues

**Happy Coding!** 🚀
