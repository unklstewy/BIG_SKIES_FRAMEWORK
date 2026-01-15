// Package security provides security engine implementations.
package security

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/internal/models"
	"github.com/unklstewy/BIG_SKIES_FRAMEWORK/pkg/healthcheck"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AccountSecurityEngine manages users, groups, roles, and permissions with RBAC.
type AccountSecurityEngine struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewAccountSecurityEngine creates a new account security engine.
func NewAccountSecurityEngine(db *pgxpool.Pool, logger *zap.Logger) *AccountSecurityEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AccountSecurityEngine{
		db:     db,
		logger: logger.With(zap.String("engine", "account_security")),
	}
}

// CreateUser creates a new user with hashed password.
func (e *AccountSecurityEngine) CreateUser(ctx context.Context, username, email, password string) (*models.User, error) {
	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = e.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Enabled, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		e.logger.Error("Failed to create user", zap.Error(err), zap.String("username", username))
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	e.logger.Info("Created user", zap.String("username", username), zap.String("user_id", user.ID))
	return user, nil
}

// AuthenticateUser validates username/password and returns the user.
func (e *AccountSecurityEngine) AuthenticateUser(ctx context.Context, username, password string) (*models.User, error) {
	user, err := e.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.Enabled {
		return nil, fmt.Errorf("user is disabled")
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		e.logger.Warn("Authentication failed", zap.String("username", username))
		return nil, fmt.Errorf("invalid credentials")
	}

	e.logger.Info("User authenticated", zap.String("username", username))
	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (e *AccountSecurityEngine) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, enabled, created_at, updated_at
		FROM users WHERE username = $1
	`

	err := e.db.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (e *AccountSecurityEngine) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, enabled, created_at, updated_at
		FROM users WHERE id = $1
	`

	err := e.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information.
func (e *AccountSecurityEngine) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $1, enabled = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := e.db.Exec(ctx, query, user.Email, user.Enabled, user.UpdatedAt, user.ID)
	if err != nil {
		e.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", user.ID))
		return fmt.Errorf("failed to update user: %w", err)
	}

	e.logger.Info("Updated user", zap.String("user_id", user.ID))
	return nil
}

// DeleteUser deletes a user (soft delete by disabling).
func (e *AccountSecurityEngine) DeleteUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET enabled = false, updated_at = $1 WHERE id = $2`

	_, err := e.db.Exec(ctx, query, time.Now(), userID)
	if err != nil {
		e.logger.Error("Failed to delete user", zap.Error(err), zap.String("user_id", userID))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	e.logger.Info("Deleted user", zap.String("user_id", userID))
	return nil
}

// CreateRole creates a new role.
func (e *AccountSecurityEngine) CreateRole(ctx context.Context, name, description string) (*models.Role, error) {
	role := &models.Role{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO roles (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := e.db.Exec(ctx, query, role.ID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	e.logger.Info("Created role", zap.String("role_name", name))
	return role, nil
}

// CreateGroup creates a new group.
func (e *AccountSecurityEngine) CreateGroup(ctx context.Context, name, description string) (*models.Group, error) {
	group := &models.Group{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO groups (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := e.db.Exec(ctx, query, group.ID, group.Name, group.Description, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	e.logger.Info("Created group", zap.String("group_name", name))
	return group, nil
}

// AssignRoleToUser assigns a role to a user.
func (e *AccountSecurityEngine) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err := e.db.Exec(ctx, query, userID, roleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	e.logger.Info("Assigned role to user", zap.String("user_id", userID), zap.String("role_id", roleID))
	return nil
}

// AssignUserToGroup assigns a user to a group.
func (e *AccountSecurityEngine) AssignUserToGroup(ctx context.Context, userID, groupID string) error {
	query := `
		INSERT INTO user_groups (user_id, group_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, group_id) DO NOTHING
	`

	_, err := e.db.Exec(ctx, query, userID, groupID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign user to group: %w", err)
	}

	e.logger.Info("Assigned user to group", zap.String("user_id", userID), zap.String("group_id", groupID))
	return nil
}

// CreatePermission creates a new permission.
func (e *AccountSecurityEngine) CreatePermission(ctx context.Context, resource, action, effect string) (*models.Permission, error) {
	permission := &models.Permission{
		ID:       uuid.New().String(),
		Resource: resource,
		Action:   action,
		Effect:   effect,
	}

	query := `
		INSERT INTO permissions (id, resource, action, effect)
		VALUES ($1, $2, $3, $4)
	`

	_, err := e.db.Exec(ctx, query, permission.ID, permission.Resource, permission.Action, permission.Effect)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	e.logger.Info("Created permission",
		zap.String("resource", resource),
		zap.String("action", action),
		zap.String("effect", effect))

	return permission, nil
}

// CheckPermission evaluates if a user has permission for a resource/action.
func (e *AccountSecurityEngine) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	// Query permissions from user's roles and groups
	query := `
		SELECT p.effect
		FROM permissions p
		WHERE p.resource = $2 AND p.action = $3
		AND (
			-- Permissions from user roles
			p.id IN (
				SELECT rp.permission_id
				FROM role_permissions rp
				JOIN user_roles ur ON ur.role_id = rp.role_id
				WHERE ur.user_id = $1
			)
			OR
			-- Permissions from user groups
			p.id IN (
				SELECT gp.permission_id
				FROM group_permissions gp
				JOIN user_groups ug ON ug.group_id = gp.group_id
				WHERE ug.user_id = $1
			)
		)
		ORDER BY
			CASE WHEN p.effect = 'deny' THEN 1 ELSE 2 END
		LIMIT 1
	`

	var effect string
	err := e.db.QueryRow(ctx, query, userID, resource, action).Scan(&effect)

	if err != nil {
		// No permissions found - default deny
		return false, nil
	}

	allowed := effect == "allow"

	e.logger.Debug("Permission check",
		zap.String("user_id", userID),
		zap.String("resource", resource),
		zap.String("action", action),
		zap.Bool("allowed", allowed))

	return allowed, nil
}

// Check returns the health status of the account security engine.
func (e *AccountSecurityEngine) Check(ctx context.Context) *healthcheck.Result {
	// Check database connectivity
	var userCount int
	err := e.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE enabled = true").Scan(&userCount)

	status := healthcheck.StatusHealthy
	message := "Account security engine is operational"

	if err != nil {
		status = healthcheck.StatusUnhealthy
		message = fmt.Sprintf("Database error: %v", err)
		e.logger.Error("Health check failed", zap.Error(err))
	}

	return &healthcheck.Result{
		ComponentName: "account_security_engine",
		Status:        status,
		Message:       message,
		Timestamp:     time.Now(),
		Details: map[string]interface{}{
			"enabled_users": userCount,
			"database_pool": e.db.Stat().TotalConns(),
		},
	}
}

// Name returns the name of the engine.
func (e *AccountSecurityEngine) Name() string {
	return "account_security_engine"
}
