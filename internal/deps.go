// Package internal contains placeholder imports to ensure dependencies are retained in go.mod
// These imports will be used by actual implementations in subsequent phases.
// This file can be removed once real implementations are in place.
package internal

import (
	// MQTT client for message bus
	_ "github.com/eclipse/paho.mqtt.golang"

	// PostgreSQL driver
	_ "github.com/jackc/pgx/v5"

	// Docker SDK for container management
	_ "github.com/moby/moby/client"

	// HTTP router for REST APIs
	_ "github.com/gin-gonic/gin"

	// Configuration management
	_ "github.com/spf13/viper"

	// Structured logging
	_ "go.uber.org/zap"

	// Testing utilities
	_ "github.com/stretchr/testify/assert"
)
