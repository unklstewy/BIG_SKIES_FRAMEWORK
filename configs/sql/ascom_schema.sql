-- ASCOM Coordinator Database Schema
-- BIG SKIES Framework
-- ASCOM Alpaca device configurations and state management
-- Integrates with existing security and telescope schemas

-- Prerequisites: security_schema.sql and telescope_schema.sql must be applied first

-- Drop existing tables if they exist (for development)
DROP TABLE IF EXISTS ascom_device_state CASCADE;
DROP TABLE IF EXISTS ascom_sessions CASCADE;
DROP TABLE IF EXISTS ascom_devices CASCADE;

-- ASCOM Devices table - device configurations exposed via ASCOM Alpaca API
-- This extends the existing telescope_devices with ASCOM-specific configuration
CREATE TABLE ascom_devices (
    id UUID PRIMARY KEY,
    device_type VARCHAR(50) NOT NULL CHECK (device_type IN (
        'telescope', 'camera', 'dome', 'focuser', 'filterwheel',
        'rotator', 'switch', 'safetymonitor', 'observingconditions', 'covercalibrator'
    )),
    device_number INT NOT NULL,                -- ASCOM device number (unique per type)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unique_id VARCHAR(255) NOT NULL UNIQUE,    -- ASCOM unique device identifier
    
    -- Backend configuration
    backend_mode VARCHAR(50) NOT NULL CHECK (backend_mode IN ('mqtt', 'network', 'hybrid')),
    backend_config JSONB NOT NULL DEFAULT '{}', -- Backend-specific configuration
    
    -- Multi-tenant ownership and access control
    organization_id UUID,                       -- Optional: for multi-organization deployments
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    
    -- Optional association with telescope configuration
    -- Links ASCOM device to a BigSkies telescope config for integrated control
    telescope_config_id UUID REFERENCES telescope_configurations(id) ON DELETE SET NULL,
    
    -- Status and metadata
    enabled BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Ensure unique device_type/device_number combinations
    CONSTRAINT unique_ascom_device_type_number UNIQUE (device_type, device_number)
);

CREATE INDEX idx_ascom_devices_type ON ascom_devices(device_type);
CREATE INDEX idx_ascom_devices_number ON ascom_devices(device_number);
CREATE INDEX idx_ascom_devices_enabled ON ascom_devices(enabled);
CREATE INDEX idx_ascom_devices_organization ON ascom_devices(organization_id);
CREATE INDEX idx_ascom_devices_telescope_config ON ascom_devices(telescope_config_id);
CREATE INDEX idx_ascom_devices_unique_id ON ascom_devices(unique_id);
CREATE INDEX idx_ascom_devices_backend_mode ON ascom_devices(backend_mode);

-- ASCOM Device State table - caches current device state for performance
-- Reduces round-trips to backend devices for frequently accessed properties
CREATE TABLE ascom_device_state (
    device_id UUID PRIMARY KEY REFERENCES ascom_devices(id) ON DELETE CASCADE,
    
    -- Connection state
    connected BOOLEAN DEFAULT false NOT NULL,
    last_connected TIMESTAMP WITH TIME ZONE,
    last_disconnected TIMESTAMP WITH TIME ZONE,
    
    -- Cached device properties (JSONB for flexibility across device types)
    -- Common properties: interfaceversion, driverinfo, driverversion, supportedactions
    cached_properties JSONB NOT NULL DEFAULT '{}',
    
    -- Device-specific state (telescope: coordinates, camera: ccd temp, etc.)
    device_state JSONB NOT NULL DEFAULT '{}',
    
    -- Performance metrics
    total_requests INT DEFAULT 0 NOT NULL,
    failed_requests INT DEFAULT 0 NOT NULL,
    average_latency_ms DOUBLE PRECISION,
    last_error TEXT,
    last_error_time TIMESTAMP WITH TIME ZONE,
    
    -- Cache metadata
    cache_updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    cache_ttl_seconds INT DEFAULT 30 NOT NULL  -- Time-to-live for cached data
);

CREATE INDEX idx_ascom_device_state_connected ON ascom_device_state(connected);
CREATE INDEX idx_ascom_device_state_cache_updated ON ascom_device_state(cache_updated_at);

-- ASCOM Sessions table - tracks ASCOM client connections and usage
-- Complements telescope_sessions with ASCOM-specific connection tracking
CREATE TABLE ascom_sessions (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES ascom_devices(id) ON DELETE CASCADE,
    
    -- Client identification
    client_id INT NOT NULL,                     -- ASCOM ClientID from API requests
    client_name VARCHAR(255),                   -- Client software name (e.g., "N.I.N.A.")
    client_version VARCHAR(100),                -- Client software version
    client_ip_address INET,                     -- Client IP address
    
    -- Session tracking
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'idle', 'closed')) DEFAULT 'active',
    
    -- Link to user if authenticated (may be null for anonymous ASCOM clients)
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Link to telescope session if associated with telescope operation
    telescope_session_id UUID REFERENCES telescope_sessions(id) ON DELETE SET NULL,
    
    -- Session statistics
    total_commands INT DEFAULT 0 NOT NULL,
    total_queries INT DEFAULT 0 NOT NULL,
    
    CONSTRAINT check_ascom_session_times CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_ascom_sessions_device ON ascom_sessions(device_id);
CREATE INDEX idx_ascom_sessions_status ON ascom_sessions(status);
CREATE INDEX idx_ascom_sessions_started ON ascom_sessions(started_at);
CREATE INDEX idx_ascom_sessions_client_id ON ascom_sessions(client_id);
CREATE INDEX idx_ascom_sessions_user ON ascom_sessions(user_id);
CREATE INDEX idx_ascom_sessions_telescope_session ON ascom_sessions(telescope_session_id);
CREATE INDEX idx_ascom_sessions_last_activity ON ascom_sessions(last_activity_at);

-- Insert default ASCOM device for testing
-- This creates a default telescope device that can be used for development and testing
INSERT INTO ascom_devices (
    id, device_type, device_number, name, description, unique_id,
    backend_mode, backend_config, created_by, enabled
) VALUES (
    '50000000-0000-0000-0000-000000000001',
    'telescope',
    0,
    'Default Telescope',
    'Default ASCOM telescope device for testing',
    '50000000-0000-0000-0000-000000000001',
    'mqtt',
    '{"mqtt_topic_prefix": "telescope/default", "timeout_seconds": 30}',
    '00000000-0000-0000-0000-000000000001',  -- admin user
    true
);

-- Initialize device state for default device
INSERT INTO ascom_device_state (device_id, connected, cached_properties, device_state)
VALUES (
    '50000000-0000-0000-0000-000000000001',
    false,
    '{"interfaceversion": 3, "driverinfo": "BigSkies ASCOM Telescope", "driverversion": "1.0.0"}',
    '{"tracking": false, "slewing": false, "parked": true}'
);

-- Function to automatically update ascom_devices.updated_at timestamp
CREATE OR REPLACE FUNCTION update_ascom_devices_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ascom_devices_update_timestamp
    BEFORE UPDATE ON ascom_devices
    FOR EACH ROW
    EXECUTE FUNCTION update_ascom_devices_timestamp();

-- Function to automatically update ascom_device_state.cache_updated_at timestamp
CREATE OR REPLACE FUNCTION update_ascom_device_state_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.cache_updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ascom_device_state_update_timestamp
    BEFORE UPDATE ON ascom_device_state
    FOR EACH ROW
    EXECUTE FUNCTION update_ascom_device_state_timestamp();

-- Function to clean up old idle ASCOM sessions (can be called by cron job)
CREATE OR REPLACE FUNCTION cleanup_idle_ascom_sessions(idle_threshold_minutes INT DEFAULT 60)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    WITH deleted AS (
        UPDATE ascom_sessions
        SET status = 'closed',
            ended_at = CURRENT_TIMESTAMP
        WHERE status = 'idle'
          AND last_activity_at < CURRENT_TIMESTAMP - (idle_threshold_minutes || ' minutes')::INTERVAL
          AND ended_at IS NULL
        RETURNING *
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- View: Active ASCOM devices with their current state
CREATE OR REPLACE VIEW active_ascom_devices AS
SELECT 
    d.id,
    d.device_type,
    d.device_number,
    d.name,
    d.description,
    d.unique_id,
    d.backend_mode,
    d.telescope_config_id,
    s.connected,
    s.last_connected,
    s.average_latency_ms,
    s.total_requests,
    s.failed_requests,
    s.last_error,
    tc.name AS telescope_name,
    u.username AS created_by_username
FROM ascom_devices d
LEFT JOIN ascom_device_state s ON d.id = s.device_id
LEFT JOIN telescope_configurations tc ON d.telescope_config_id = tc.id
LEFT JOIN users u ON d.created_by = u.id
WHERE d.enabled = true;

-- View: Active ASCOM sessions with device and user information
CREATE OR REPLACE VIEW active_ascom_sessions AS
SELECT 
    s.id,
    s.client_id,
    s.client_name,
    s.client_version,
    s.client_ip_address,
    s.started_at,
    s.last_activity_at,
    s.total_commands,
    s.total_queries,
    d.device_type,
    d.device_number,
    d.name AS device_name,
    u.username,
    ts.id AS telescope_session_id
FROM ascom_sessions s
JOIN ascom_devices d ON s.device_id = d.id
LEFT JOIN users u ON s.user_id = u.id
LEFT JOIN telescope_sessions ts ON s.telescope_session_id = ts.id
WHERE s.status = 'active';

-- Comments for documentation
COMMENT ON TABLE ascom_devices IS 'ASCOM Alpaca device configurations exposed via ASCOM coordinator';
COMMENT ON TABLE ascom_device_state IS 'Cached state for ASCOM devices to reduce backend round-trips';
COMMENT ON TABLE ascom_sessions IS 'ASCOM client connection tracking and session management';
COMMENT ON COLUMN ascom_devices.backend_mode IS 'Backend connection type: mqtt (BigSkies native), network (remote Alpaca), hybrid (both)';
COMMENT ON COLUMN ascom_devices.backend_config IS 'Backend-specific configuration as JSON (URLs, topics, timeouts, etc.)';
COMMENT ON COLUMN ascom_devices.telescope_config_id IS 'Optional link to BigSkies telescope configuration for integrated control';
COMMENT ON COLUMN ascom_device_state.cached_properties IS 'Cached ASCOM common properties (interfaceversion, driverinfo, etc.)';
COMMENT ON COLUMN ascom_device_state.device_state IS 'Device-specific state as JSON (varies by device type)';
COMMENT ON COLUMN ascom_sessions.client_id IS 'ASCOM ClientID from API requests (not related to authentication)';

-- Grant privileges (adjust based on your database user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO bigskies_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO bigskies_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO bigskies_user;
