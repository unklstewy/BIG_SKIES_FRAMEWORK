-- Message Coordinator RBAC Database Schema
-- BIG SKIES Framework

-- Drop existing table if it exists (for development)
DROP TABLE IF EXISTS topic_protection_rules CASCADE;

-- Topic Protection Rules table
CREATE TABLE topic_protection_rules (
    id UUID PRIMARY KEY,
    topic_pattern VARCHAR(500) NOT NULL,  -- e.g., "bigskies/coordinator/telescope/+/slew"
    resource VARCHAR(255) NOT NULL,       -- e.g., "telescope"
    action VARCHAR(255) NOT NULL,         -- e.g., "control"
    enabled BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Indexes for efficient lookups
CREATE INDEX idx_topic_protection_rules_topic_pattern ON topic_protection_rules(topic_pattern);
CREATE INDEX idx_topic_protection_rules_resource_action ON topic_protection_rules(resource, action);
CREATE INDEX idx_topic_protection_rules_enabled ON topic_protection_rules(enabled);

-- Insert default protection rules
INSERT INTO topic_protection_rules (id, topic_pattern, resource, action, enabled) VALUES
    (gen_random_uuid(), 'bigskies/coordinator/telescope/+/slew', 'telescope', 'control', true),
    (gen_random_uuid(), 'bigskies/coordinator/telescope/+/park', 'telescope', 'control', true),
    (gen_random_uuid(), 'bigskies/coordinator/telescope/+/track', 'telescope', 'control', true),
    (gen_random_uuid(), 'bigskies/coordinator/security/+/user/create', 'user', 'manage', true),
    (gen_random_uuid(), 'bigskies/coordinator/security/+/user/delete', 'user', 'manage', true),
    (gen_random_uuid(), 'bigskies/coordinator/plugin/+/install', 'plugin', 'manage', true),
    (gen_random_uuid(), 'bigskies/coordinator/plugin/+/uninstall', 'plugin', 'manage', true);