-- Security Coordinator Database Schema
-- BIG SKIES Framework

-- Drop existing tables if they exist (for development)
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS group_permissions CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS user_groups CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tls_certificates CASCADE;

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Index for faster username and email lookups
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_enabled ON users(enabled);

-- Groups table
CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_groups_name ON groups(name);

-- Roles table
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_roles_name ON roles(name);

-- Permissions table
CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    resource VARCHAR(255) NOT NULL,  -- e.g., 'telescope', 'plugin', 'user'
    action VARCHAR(255) NOT NULL,    -- e.g., 'read', 'write', 'delete'
    effect VARCHAR(50) NOT NULL CHECK (effect IN ('allow', 'deny'))
);

-- Unique constraint on resource+action combination
CREATE UNIQUE INDEX idx_permissions_resource_action ON permissions(resource, action, effect);

-- User-Group association table
CREATE TABLE user_groups (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX idx_user_groups_user ON user_groups(user_id);
CREATE INDEX idx_user_groups_group ON user_groups(group_id);

-- User-Role association table
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);

-- Group-Permission association table
CREATE TABLE group_permissions (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (group_id, permission_id)
);

CREATE INDEX idx_group_permissions_group ON group_permissions(group_id);
CREATE INDEX idx_group_permissions_permission ON group_permissions(permission_id);

-- Role-Permission association table
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

-- TLS Certificates table
CREATE TABLE tls_certificates (
    id UUID PRIMARY KEY,
    domain VARCHAR(255) UNIQUE NOT NULL,
    certificate_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    issuer VARCHAR(100) NOT NULL,  -- 'letsencrypt', 'self-signed', etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_tls_certificates_domain ON tls_certificates(domain);
CREATE INDEX idx_tls_certificates_expires_at ON tls_certificates(expires_at);
CREATE INDEX idx_tls_certificates_issuer ON tls_certificates(issuer);

-- Insert default admin user
-- Password: 'bigskies_admin_2024' (change this immediately in production!)
-- Hashed using bcrypt cost 10
INSERT INTO users (id, username, email, password_hash, enabled)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin',
    'admin@bigskies.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',  -- bigskies_admin_2024
    true
);

-- Insert default roles
INSERT INTO roles (id, name, description) VALUES
    ('r0000000-0000-0000-0000-000000000001', 'admin', 'Full system administrator'),
    ('r0000000-0000-0000-0000-000000000002', 'operator', 'Telescope operator'),
    ('r0000000-0000-0000-0000-000000000003', 'observer', 'Read-only observer'),
    ('r0000000-0000-0000-0000-000000000004', 'developer', 'Plugin developer');

-- Insert default groups
INSERT INTO groups (id, name, description) VALUES
    ('g0000000-0000-0000-0000-000000000001', 'administrators', 'System administrators'),
    ('g0000000-0000-0000-0000-000000000002', 'operators', 'Telescope operators'),
    ('g0000000-0000-0000-0000-000000000003', 'observers', 'Read-only users');

-- Insert default permissions
INSERT INTO permissions (id, resource, action, effect) VALUES
    -- User management
    ('p0000000-0000-0000-0000-000000000001', 'user', 'read', 'allow'),
    ('p0000000-0000-0000-0000-000000000002', 'user', 'write', 'allow'),
    ('p0000000-0000-0000-0000-000000000003', 'user', 'delete', 'allow'),
    -- Telescope operations
    ('p0000000-0000-0000-0000-000000000004', 'telescope', 'read', 'allow'),
    ('p0000000-0000-0000-0000-000000000005', 'telescope', 'write', 'allow'),
    ('p0000000-0000-0000-0000-000000000006', 'telescope', 'control', 'allow'),
    -- Plugin management
    ('p0000000-0000-0000-0000-000000000007', 'plugin', 'read', 'allow'),
    ('p0000000-0000-0000-0000-000000000008', 'plugin', 'write', 'allow'),
    ('p0000000-0000-0000-0000-000000000009', 'plugin', 'install', 'allow'),
    ('p0000000-0000-0000-0000-00000000000a', 'plugin', 'delete', 'allow'),
    -- Security management
    ('p0000000-0000-0000-0000-00000000000b', 'security', 'read', 'allow'),
    ('p0000000-0000-0000-0000-00000000000c', 'security', 'write', 'allow'),
    -- Certificate management
    ('p0000000-0000-0000-0000-00000000000d', 'certificate', 'read', 'allow'),
    ('p0000000-0000-0000-0000-00000000000e', 'certificate', 'write', 'allow');

-- Assign admin role all permissions
INSERT INTO role_permissions (role_id, permission_id) 
SELECT 'r0000000-0000-0000-0000-000000000001', id FROM permissions;

-- Assign operator role telescope and plugin permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('r0000000-0000-0000-0000-000000000002', 'p0000000-0000-0000-0000-000000000004'),  -- telescope read
    ('r0000000-0000-0000-0000-000000000002', 'p0000000-0000-0000-0000-000000000005'),  -- telescope write
    ('r0000000-0000-0000-0000-000000000002', 'p0000000-0000-0000-0000-000000000006'),  -- telescope control
    ('r0000000-0000-0000-0000-000000000002', 'p0000000-0000-0000-0000-000000000007'),  -- plugin read
    ('r0000000-0000-0000-0000-000000000002', 'p0000000-0000-0000-0000-000000000008');  -- plugin write

-- Assign observer role read-only permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('r0000000-0000-0000-0000-000000000003', 'p0000000-0000-0000-0000-000000000001'),  -- user read
    ('r0000000-0000-0000-0000-000000000003', 'p0000000-0000-0000-0000-000000000004'),  -- telescope read
    ('r0000000-0000-0000-0000-000000000003', 'p0000000-0000-0000-0000-000000000007');  -- plugin read

-- Assign developer role plugin management permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('r0000000-0000-0000-0000-000000000004', 'p0000000-0000-0000-0000-000000000007'),  -- plugin read
    ('r0000000-0000-0000-0000-000000000004', 'p0000000-0000-0000-0000-000000000008'),  -- plugin write
    ('r0000000-0000-0000-0000-000000000004', 'p0000000-0000-0000-0000-000000000009'),  -- plugin install
    ('r0000000-0000-0000-0000-000000000004', 'p0000000-0000-0000-0000-00000000000a');  -- plugin delete

-- Assign admin user to admin role and administrators group
INSERT INTO user_roles (user_id, role_id) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'r0000000-0000-0000-0000-000000000001');

INSERT INTO user_groups (user_id, group_id) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'g0000000-0000-0000-0000-000000000001');

-- Comments for documentation
COMMENT ON TABLE users IS 'System user accounts with authentication credentials';
COMMENT ON TABLE groups IS 'User groups for organizing permissions';
COMMENT ON TABLE roles IS 'User roles defining sets of permissions';
COMMENT ON TABLE permissions IS 'Individual permissions for resource/action combinations';
COMMENT ON TABLE tls_certificates IS 'TLS/SSL certificates for secure communications';

-- Grant privileges (adjust based on your database user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO bigskies_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO bigskies_user;
