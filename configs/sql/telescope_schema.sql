-- Telescope Coordinator Database Schema
-- BIG SKIES Framework
-- Multi-tenant telescope configuration and session management

-- Drop existing tables if they exist (for development)
DROP TABLE IF EXISTS telescope_sessions CASCADE;
DROP TABLE IF EXISTS telescope_devices CASCADE;
DROP TABLE IF EXISTS telescope_permissions CASCADE;
DROP TABLE IF EXISTS telescope_configurations CASCADE;
DROP TABLE IF EXISTS observatory_sites CASCADE;

-- Observatory sites table - supports multiple observatory locations
CREATE TABLE observatory_sites (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    elevation DOUBLE PRECISION NOT NULL,  -- meters above sea level
    timezone VARCHAR(100) NOT NULL,        -- e.g., 'America/Los_Angeles'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_observatory_sites_name ON observatory_sites(name);

-- Telescope configurations table - main telescope setup with ownership
CREATE TABLE telescope_configurations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_type VARCHAR(50) NOT NULL CHECK (owner_type IN ('user', 'group')),
    site_id UUID REFERENCES observatory_sites(id) ON DELETE SET NULL,
    mount_type VARCHAR(50) NOT NULL CHECK (mount_type IN ('altaz', 'equatorial', 'dobsonian')),
    enabled BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT unique_telescope_name_per_owner UNIQUE (name, owner_id)
);

CREATE INDEX idx_telescope_configs_owner ON telescope_configurations(owner_id, owner_type);
CREATE INDEX idx_telescope_configs_site ON telescope_configurations(site_id);
CREATE INDEX idx_telescope_configs_enabled ON telescope_configurations(enabled);

-- Telescope devices table - ASCOM device associations
CREATE TABLE telescope_devices (
    id UUID PRIMARY KEY,
    telescope_id UUID NOT NULL REFERENCES telescope_configurations(id) ON DELETE CASCADE,
    device_role VARCHAR(50) NOT NULL CHECK (device_role IN (
        'telescope', 'camera', 'dome', 'focuser', 'filterwheel', 
        'rotator', 'switch', 'safety', 'observingconditions', 'covercalibrator'
    )),
    device_id VARCHAR(255) NOT NULL,       -- ASCOM device ID
    device_type VARCHAR(50) NOT NULL,      -- ASCOM device type
    device_number INT NOT NULL,            -- ASCOM device number
    server_url VARCHAR(255) NOT NULL,      -- Alpaca server URL
    device_name VARCHAR(255),
    device_uuid VARCHAR(255),
    enabled BOOLEAN DEFAULT true NOT NULL,
    last_connected TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT unique_device_per_telescope_role UNIQUE (telescope_id, device_role)
);

CREATE INDEX idx_telescope_devices_telescope ON telescope_devices(telescope_id);
CREATE INDEX idx_telescope_devices_role ON telescope_devices(device_role);
CREATE INDEX idx_telescope_devices_enabled ON telescope_devices(enabled);
CREATE INDEX idx_telescope_devices_device_id ON telescope_devices(device_id);

-- Telescope permissions table - fine-grained per-telescope access control
CREATE TABLE telescope_permissions (
    id UUID PRIMARY KEY,
    telescope_id UUID NOT NULL REFERENCES telescope_configurations(id) ON DELETE CASCADE,
    principal_id UUID NOT NULL,            -- user_id or group_id
    principal_type VARCHAR(50) NOT NULL CHECK (principal_type IN ('user', 'group')),
    permission VARCHAR(50) NOT NULL CHECK (permission IN ('read', 'write', 'control')),
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT unique_telescope_permission UNIQUE (telescope_id, principal_id, principal_type, permission)
);

CREATE INDEX idx_telescope_permissions_telescope ON telescope_permissions(telescope_id);
CREATE INDEX idx_telescope_permissions_principal ON telescope_permissions(principal_id, principal_type);
CREATE INDEX idx_telescope_permissions_permission ON telescope_permissions(permission);
CREATE INDEX idx_telescope_permissions_expires ON telescope_permissions(expires_at);

-- Telescope sessions table - tracks active telescope usage
CREATE TABLE telescope_sessions (
    id UUID PRIMARY KEY,
    telescope_id UUID NOT NULL REFERENCES telescope_configurations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    ended_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'completed', 'aborted', 'error')) DEFAULT 'active',
    session_type VARCHAR(50) NOT NULL CHECK (session_type IN ('manual', 'automated', 'maintenance')),
    notes TEXT,
    CONSTRAINT check_session_times CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_telescope_sessions_telescope ON telescope_sessions(telescope_id);
CREATE INDEX idx_telescope_sessions_user ON telescope_sessions(user_id);
CREATE INDEX idx_telescope_sessions_status ON telescope_sessions(status);
CREATE INDEX idx_telescope_sessions_started ON telescope_sessions(started_at);

-- Insert default observatory site
INSERT INTO observatory_sites (id, name, description, latitude, longitude, elevation, timezone)
VALUES (
    '40000000-0000-0000-0000-000000000001',
    'Default Observatory',
    'Default observatory site for testing and development',
    34.0522,    -- Los Angeles latitude
    -118.2437,  -- Los Angeles longitude
    100.0,      -- 100 meters elevation
    'America/Los_Angeles'
);

-- Comments for documentation
COMMENT ON TABLE observatory_sites IS 'Physical observatory locations with geographic coordinates';
COMMENT ON TABLE telescope_configurations IS 'Telescope system configurations with multi-tenant ownership';
COMMENT ON TABLE telescope_devices IS 'ASCOM Alpaca device assignments to telescope configurations';
COMMENT ON TABLE telescope_permissions IS 'Fine-grained access control for telescope configurations';
COMMENT ON TABLE telescope_sessions IS 'Active and historical telescope usage sessions';

-- Grant privileges (adjust based on your database user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO bigskies_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO bigskies_user;
