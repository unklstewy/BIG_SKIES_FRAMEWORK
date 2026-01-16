-- Coordinator Configuration Database Schema
-- BIG SKIES Framework
-- Runtime coordinator configuration storage and management

-- Drop existing tables if they exist (for development)
DROP TABLE IF EXISTS coordinator_config_history CASCADE;
DROP TABLE IF EXISTS coordinator_config CASCADE;

-- Coordinator configuration table
-- Stores runtime configuration for all coordinators
CREATE TABLE coordinator_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coordinator_name VARCHAR(255) NOT NULL,
    config_key VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL,
    config_type VARCHAR(50) NOT NULL CHECK (config_type IN ('string', 'int', 'bool', 'float', 'duration', 'object')),
    description TEXT,
    is_secret BOOLEAN DEFAULT false NOT NULL,  -- Marks sensitive values (passwords, keys)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by UUID REFERENCES users(id),  -- Track who made changes
    CONSTRAINT unique_coordinator_config UNIQUE (coordinator_name, config_key)
);

-- Indexes for efficient lookup
CREATE INDEX idx_coordinator_config_name ON coordinator_config(coordinator_name);
CREATE INDEX idx_coordinator_config_key ON coordinator_config(config_key);
CREATE INDEX idx_coordinator_config_updated ON coordinator_config(updated_at);

-- Configuration history table - tracks changes over time
CREATE TABLE coordinator_config_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES coordinator_config(id) ON DELETE CASCADE,
    coordinator_name VARCHAR(255) NOT NULL,
    config_key VARCHAR(255) NOT NULL,
    old_value JSONB,
    new_value JSONB NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    changed_by UUID REFERENCES users(id)
);

-- Index for history queries
CREATE INDEX idx_coordinator_config_history_config ON coordinator_config_history(config_id);
CREATE INDEX idx_coordinator_config_history_changed ON coordinator_config_history(changed_at);

-- Trigger to automatically track config changes
CREATE OR REPLACE FUNCTION track_coordinator_config_changes()
RETURNS TRIGGER AS $$
BEGIN
    -- Only insert history if value actually changed
    IF OLD.config_value IS DISTINCT FROM NEW.config_value THEN
        INSERT INTO coordinator_config_history 
            (config_id, coordinator_name, config_key, old_value, new_value, changed_by)
        VALUES 
            (NEW.id, NEW.coordinator_name, NEW.config_key, OLD.config_value, NEW.config_value, NEW.updated_by);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER coordinator_config_change_trigger
    AFTER UPDATE ON coordinator_config
    FOR EACH ROW
    EXECUTE FUNCTION track_coordinator_config_changes();

-- Insert default configurations for all coordinators

-- Message Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description) VALUES
    ('message-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('message-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port'),
    ('message-coordinator', 'monitor_interval', '30', 'int', 'Health monitoring interval in seconds'),
    ('message-coordinator', 'max_reconnect_attempts', '5', 'int', 'Maximum MQTT reconnection attempts before marking unhealthy');

-- Application Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description) VALUES
    ('application-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('application-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port'),
    ('application-coordinator', 'registry_check_interval', '60', 'int', 'Service registry check interval in seconds'),
    ('application-coordinator', 'service_timeout', '180', 'int', 'Service heartbeat timeout in seconds');

-- Security Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description, is_secret) VALUES
    ('security-coordinator', 'database_url', '"postgresql://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable"', 'string', 'PostgreSQL connection string', true),
    ('security-coordinator', 'jwt_secret', '"change-this-secret-in-production"', 'string', 'JWT signing secret', true),
    ('security-coordinator', 'token_duration', '3600', 'int', 'JWT token validity duration in seconds'),
    ('security-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('security-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port');

-- Telescope Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description, is_secret) VALUES
    ('telescope-coordinator', 'database_url', '"postgresql://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable"', 'string', 'PostgreSQL connection string', true),
    ('telescope-coordinator', 'discovery_port', '32227', 'int', 'ASCOM Alpaca discovery port'),
    ('telescope-coordinator', 'health_check_interval', '30', 'int', 'Device health check interval in seconds'),
    ('telescope-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('telescope-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port');

-- Plugin Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description) VALUES
    ('plugin-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('plugin-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port'),
    ('plugin-coordinator', 'plugin_dir', '"/var/lib/bigskies/plugins"', 'string', 'Plugin installation directory'),
    ('plugin-coordinator', 'scan_interval', '300', 'int', 'Plugin verification scan interval in seconds');

-- DataStore Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description, is_secret) VALUES
    ('datastore-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('datastore-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port'),
    ('datastore-coordinator', 'database_url', '"postgresql://bigskies:bigskies@localhost:5432/bigskies?sslmode=disable"', 'string', 'PostgreSQL connection string', true),
    ('datastore-coordinator', 'max_connections', '20', 'int', 'Maximum database connections in pool'),
    ('datastore-coordinator', 'min_connections', '5', 'int', 'Minimum database connections in pool');

-- UIElement Coordinator configuration
INSERT INTO coordinator_config (coordinator_name, config_key, config_value, config_type, description) VALUES
    ('uielement-coordinator', 'broker_url', '"localhost"', 'string', 'MQTT broker hostname or IP address'),
    ('uielement-coordinator', 'broker_port', '1883', 'int', 'MQTT broker port'),
    ('uielement-coordinator', 'scan_interval', '600', 'int', 'Plugin UI scan interval in seconds');

-- Comments for documentation
COMMENT ON TABLE coordinator_config IS 'Runtime configuration for all framework coordinators with type safety and change tracking';
COMMENT ON TABLE coordinator_config_history IS 'Historical record of configuration changes for audit trail';
COMMENT ON COLUMN coordinator_config.is_secret IS 'Marks sensitive configuration values that should not be logged or displayed';
COMMENT ON COLUMN coordinator_config.config_type IS 'Type hint for parsing config_value from JSONB';

-- Grant privileges (adjust based on your database user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO bigskies_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO bigskies_user;
