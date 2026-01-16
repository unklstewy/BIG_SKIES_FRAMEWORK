package main

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ConfigHandler handles configuration-related MQTT messages.
type ConfigHandler struct {
	loader       *ConfigLoader
	mqttClient   mqtt.Client
	pluginID     string
	responseTopic string
	eventTopic    string
}

// NewConfigHandler creates a new configuration handler.
func NewConfigHandler(loader *ConfigLoader, client mqtt.Client, pluginID string) *ConfigHandler {
	return &ConfigHandler{
		loader:        loader,
		mqttClient:    client,
		pluginID:      pluginID,
		responseTopic: "bigskies/plugin/" + pluginID + "/config/response",
		eventTopic:    "bigskies/plugin/" + pluginID + "/config/event",
	}
}

// HandleMessage processes incoming MQTT messages for configuration operations.
func (h *ConfigHandler) HandleMessage(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on topic %s", msg.Topic())

	// Parse the request
	var req ConfigRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		log.Printf("Error parsing request: %v", err)
		h.sendErrorResponse("", "parse_error", "Invalid JSON: "+err.Error())
		return
	}

	// Route to appropriate handler
	switch req.Command {
	case "load_config":
		h.handleLoadConfig(req)
	case "list_configs":
		h.handleListConfigs(req)
	case "get_status":
		h.handleGetStatus(req)
	default:
		log.Printf("Unknown command: %s", req.Command)
		h.sendErrorResponse(req.RequestID, req.Command, "Unknown command: "+req.Command)
	}
}

// handleLoadConfig processes a configuration load request.
func (h *ConfigHandler) handleLoadConfig(req ConfigRequest) {
	log.Printf("Loading configuration: model=%s, mount_type=%s", req.Model, req.MountType)

	// Validate parameters
	if req.Model == "" {
		h.sendErrorResponse(req.RequestID, "load_config", "Missing required parameter: model")
		return
	}
	if req.MountType == "" {
		h.sendErrorResponse(req.RequestID, "load_config", "Missing required parameter: mount_type")
		return
	}

	if !ValidateModel(req.Model) {
		h.sendErrorResponse(req.RequestID, "load_config", "Invalid model: "+req.Model)
		return
	}
	if !ValidateMountType(req.MountType) {
		h.sendErrorResponse(req.RequestID, "load_config", "Invalid mount_type: "+req.MountType)
		return
	}

	// Check if configuration exists
	if !h.loader.ValidateConfigurationExists(req.Model, req.MountType) {
		h.sendErrorResponse(req.RequestID, "load_config",
			"Configuration not found for "+req.Model+"/"+req.MountType)
		return
	}

	// Load the configuration
	loadedBy := "mqtt"
	if req.RequestID != "" {
		loadedBy = "mqtt:" + req.RequestID
	}

	if err := h.loader.LoadConfiguration(req.Model, req.MountType, loadedBy); err != nil {
		log.Printf("Failed to load configuration: %v", err)
		h.sendErrorResponse(req.RequestID, "load_config", "Failed to load configuration: "+err.Error())
		
		// Publish failure event
		h.publishEvent("config_failed", ConfigStatus{}, "Configuration load failed: "+err.Error())
		return
	}

	// Get the loaded configuration status
	status, err := h.loader.GetCurrentStatus()
	if err != nil {
		log.Printf("Warning: Failed to get current status after load: %v", err)
		status = &ConfigStatus{
			Model:     req.Model,
			MountType: req.MountType,
			LoadedAt:  time.Now(),
			LoadedBy:  loadedBy,
		}
	}

	// Send success response
	resp := ConfigResponse{
		RequestID: req.RequestID,
		Command:   "load_config",
		Success:   true,
		Message:   "Configuration loaded successfully",
		Data: map[string]interface{}{
			"model":       req.Model,
			"mount_type":  req.MountType,
			"loaded_at":   status.LoadedAt,
			"loaded_by":   status.LoadedBy,
			"config_path": status.ConfigPath,
		},
		Timestamp: time.Now(),
	}

	h.sendResponse(resp)

	// Publish success event
	h.publishEvent("config_loaded", *status, "Configuration loaded: "+req.Model+"/"+req.MountType)

	log.Printf("Configuration loaded successfully: %s/%s", req.Model, req.MountType)
}

// handleListConfigs returns the list of available configurations.
func (h *ConfigHandler) handleListConfigs(req ConfigRequest) {
	log.Println("Listing available configurations")

	configs := GetAvailableConfigs()
	
	// Get current status
	currentStatus, err := h.loader.GetCurrentStatus()
	if err != nil {
		log.Printf("Warning: Failed to get current status: %v", err)
	}

	data := map[string]interface{}{
		"available_configs": configs,
		"models":            validModels,
		"mount_types":       validMountTypes,
	}

	if currentStatus != nil {
		data["current"] = currentStatus
	}

	resp := ConfigResponse{
		RequestID: req.RequestID,
		Command:   "list_configs",
		Success:   true,
		Message:   "Available configurations",
		Data:      data,
		Timestamp: time.Now(),
	}

	h.sendResponse(resp)
}

// handleGetStatus returns the current configuration status.
func (h *ConfigHandler) handleGetStatus(req ConfigRequest) {
	log.Println("Getting configuration status")

	status, err := h.loader.GetCurrentStatus()
	if err != nil {
		log.Printf("Failed to get status: %v", err)
		h.sendErrorResponse(req.RequestID, "get_status", "Failed to get status: "+err.Error())
		return
	}

	data := make(map[string]interface{})
	if status != nil {
		data["status"] = status
		data["model_description"] = GetModelDescription(status.Model)
		data["mount_type_description"] = GetMountTypeDescription(status.MountType)
	} else {
		data["status"] = nil
		data["message"] = "No configuration loaded"
	}

	resp := ConfigResponse{
		RequestID: req.RequestID,
		Command:   "get_status",
		Success:   true,
		Message:   "Current status",
		Data:      data,
		Timestamp: time.Now(),
	}

	h.sendResponse(resp)
}

// sendResponse publishes a response to the response topic.
func (h *ConfigHandler) sendResponse(resp ConfigResponse) {
	payload, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	token := h.mqttClient.Publish(h.responseTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		log.Printf("Error publishing response: %v", token.Error())
	} else {
		log.Printf("Published response for command: %s", resp.Command)
	}
}

// sendErrorResponse is a helper to send error responses.
func (h *ConfigHandler) sendErrorResponse(requestID, command, message string) {
	resp := ConfigResponse{
		RequestID: requestID,
		Command:   command,
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
	h.sendResponse(resp)
}

// publishEvent publishes a configuration event to the event topic.
func (h *ConfigHandler) publishEvent(eventType string, status ConfigStatus, message string) {
	event := ConfigEvent{
		EventType: eventType,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	token := h.mqttClient.Publish(h.eventTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		log.Printf("Error publishing event: %v", token.Error())
	} else {
		log.Printf("Published event: %s", eventType)
	}
}
