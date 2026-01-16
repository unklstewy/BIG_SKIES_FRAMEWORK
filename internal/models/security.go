// Package models provides data structures for the BIG SKIES Framework.
package models

import (
	"time"
)

// User represents a system user account.
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose in JSON
	Enabled      bool      `json:"enabled" db:"enabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Group represents a collection of users with shared permissions.
type Group struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Role represents a named collection of permissions.
type Role struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Permission defines access control for a specific resource and action.
type Permission struct {
	ID       string `json:"id" db:"id"`
	Resource string `json:"resource" db:"resource"` // e.g., "telescope", "plugin", "user"
	Action   string `json:"action" db:"action"`     // e.g., "read", "write", "delete"
	Effect   string `json:"effect" db:"effect"`     // "allow" or "deny"
}

// UserGroup links users to groups.
type UserGroup struct {
	UserID    string    `json:"user_id" db:"user_id"`
	GroupID   string    `json:"group_id" db:"group_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserRole links users to roles.
type UserRole struct {
	UserID    string    `json:"user_id" db:"user_id"`
	RoleID    string    `json:"role_id" db:"role_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GroupPermission links groups to permissions.
type GroupPermission struct {
	GroupID      string    `json:"group_id" db:"group_id"`
	PermissionID string    `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// RolePermission links roles to permissions.
type RolePermission struct {
	RoleID       string    `json:"role_id" db:"role_id"`
	PermissionID string    `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// TLSCertificate represents an SSL/TLS certificate.
type TLSCertificate struct {
	ID             string    `json:"id" db:"id"`
	Domain         string    `json:"domain" db:"domain"`
	CertificatePEM string    `json:"-" db:"certificate_pem"` // Base64 encoded certificate
	PrivateKeyPEM  string    `json:"-" db:"private_key_pem"` // Never expose in JSON
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	Issuer         string    `json:"issuer" db:"issuer"` // "letsencrypt", "self-signed", etc.
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// AuthRequest represents an authentication request.
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response with JWT token.
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// TokenValidationRequest represents a token validation request.
type TokenValidationRequest struct {
	Token string `json:"token"`
}

// TokenValidationResponse represents a token validation response.
type TokenValidationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PermissionCheckRequest represents a permission check request.
type PermissionCheckRequest struct {
	UserID   string `json:"user_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// PermissionCheckResponse represents a permission check response.
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// CertificateRequest represents a certificate generation/renewal request.
type CertificateRequest struct {
	Domain string   `json:"domain"`
	Email  string   `json:"email"`          // For Let's Encrypt notifications
	Type   string   `json:"type"`           // "letsencrypt" or "self-signed"
	SANs   []string `json:"sans,omitempty"` // Subject Alternative Names
}

// CertificateResponse represents a certificate operation response.
type CertificateResponse struct {
	Success   bool      `json:"success"`
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}
