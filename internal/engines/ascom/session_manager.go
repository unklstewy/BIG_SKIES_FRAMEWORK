// Package ascom provides ASCOM protocol engines and bridges.
package ascom

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// SessionManager tracks ASCOM client sessions and links them to users and telescope sessions.
// It maintains an in-memory map of active sessions with database persistence for audit trails.
//
// Session tracking flow:
// 1. ASCOM client connects with ClientID
// 2. SessionManager creates or retrieves session
// 3. After authentication, session is linked to user_id
// 4. When device is accessed, session is linked to telescope_session_id
// 5. Session is updated on each API request (last_activity_at)
// 6. Idle sessions are automatically cleaned up
type SessionManager struct {
	// db provides database access for session persistence
	db *pgxpool.Pool

	// logger provides structured logging
	logger *zap.Logger

	// sessions tracks active sessions in memory
	// Key: ClientID, Value: *ASCOMSession
	sessions sync.Map

	// sessionTimeout defines how long sessions remain active without activity
	sessionTimeout time.Duration

	// cleanupInterval defines how often to run cleanup of stale sessions
	cleanupInterval time.Duration

	// stopChan signals cleanup goroutine to stop
	stopChan chan struct{}

	// wg tracks active goroutines
	wg sync.WaitGroup
}

// ASCOMSession represents an active ASCOM client session with user and telescope linkage.
type ASCOMSession struct {
	// SessionID is the unique session identifier (UUID)
	SessionID string

	// ClientID is the ASCOM ClientID from API requests
	ClientID int

	// ClientName is the client software name (e.g., "N.I.N.A.", "PHD2")
	ClientName string

	// ClientVersion is the client software version
	ClientVersion string

	// ClientIPAddress is the client IP address
	ClientIPAddress string

	// DeviceID is the ASCOM device ID being accessed
	DeviceID string

	// UserID is the authenticated user (empty if not authenticated)
	UserID string

	// Username is the authenticated username (for logging)
	Username string

	// TelescopeSessionID links to telescope_sessions table (empty if not linked)
	TelescopeSessionID string

	// StartedAt is when the session was created
	StartedAt time.Time

	// LastActivityAt is when the session was last used
	LastActivityAt time.Time

	// Status is the session status (active, idle, closed)
	Status string

	// TotalCommands counts PUT/POST requests (commands)
	TotalCommands int

	// TotalQueries counts GET requests (queries)
	TotalQueries int

	// mutex protects session fields
	mu sync.RWMutex
}

const (
	// SessionStatusActive indicates an active session
	SessionStatusActive = "active"

	// SessionStatusIdle indicates an idle session (no recent activity)
	SessionStatusIdle = "idle"

	// SessionStatusClosed indicates a closed session
	SessionStatusClosed = "closed"
)

// NewSessionManager creates a new session manager instance.
func NewSessionManager(
	db *pgxpool.Pool,
	sessionTimeout time.Duration,
	logger *zap.Logger,
) (*SessionManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	if sessionTimeout == 0 {
		sessionTimeout = 1 * time.Hour
	}

	manager := &SessionManager{
		db:              db,
		logger:          logger.With(zap.String("component", "ascom_session_manager")),
		sessionTimeout:  sessionTimeout,
		cleanupInterval: 5 * time.Minute,
		stopChan:        make(chan struct{}),
	}

	manager.logger.Info("Session manager initialized",
		zap.Duration("session_timeout", sessionTimeout),
		zap.Duration("cleanup_interval", manager.cleanupInterval))

	return manager, nil
}

// Start begins session manager operations including cleanup goroutine.
func (sm *SessionManager) Start(ctx context.Context) error {
	sm.logger.Info("Starting session manager")

	// Start cleanup goroutine
	sm.wg.Add(1)
	go sm.cleanupStaleSessions()

	sm.logger.Info("Session manager started")
	return nil
}

// Stop shuts down the session manager and cleans up resources.
func (sm *SessionManager) Stop() error {
	sm.logger.Info("Stopping session manager")

	// Signal cleanup goroutine to stop
	close(sm.stopChan)

	// Wait for goroutines to finish
	sm.wg.Wait()

	// Close all active sessions
	sm.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*ASCOMSession); ok {
			sm.endSession(context.Background(), session)
		}
		return true
	})

	sm.logger.Info("Session manager stopped")
	return nil
}

// GetOrCreateSession retrieves an existing session or creates a new one.
// This should be called on every ASCOM API request to track session activity.
func (sm *SessionManager) GetOrCreateSession(
	ctx context.Context,
	clientID int,
	clientName string,
	clientVersion string,
	clientIPAddress string,
	deviceID string,
) (*ASCOMSession, error) {
	// Check if session exists in memory
	if val, ok := sm.sessions.Load(clientID); ok {
		session := val.(*ASCOMSession)
		session.mu.Lock()
		session.LastActivityAt = time.Now()
		session.Status = SessionStatusActive
		session.mu.Unlock()

		// Update database asynchronously
		go sm.updateSessionActivity(ctx, session)

		return session, nil
	}

	// Check if session exists in database
	session, err := sm.loadSessionFromDB(ctx, clientID, deviceID)
	if err == nil && session != nil {
		// Session found in database - restore to memory
		sm.sessions.Store(clientID, session)
		sm.logger.Info("Restored session from database",
			zap.Int("client_id", clientID),
			zap.String("session_id", session.SessionID))
		return session, nil
	}

	// Create new session
	session = &ASCOMSession{
		SessionID:       uuid.New().String(),
		ClientID:        clientID,
		ClientName:      clientName,
		ClientVersion:   clientVersion,
		ClientIPAddress: clientIPAddress,
		DeviceID:        deviceID,
		StartedAt:       time.Now(),
		LastActivityAt:  time.Now(),
		Status:          SessionStatusActive,
	}

	// Store in memory
	sm.sessions.Store(clientID, session)

	// Persist to database
	if err := sm.createSessionInDB(ctx, session); err != nil {
		sm.logger.Error("Failed to create session in database",
			zap.Error(err),
			zap.Int("client_id", clientID))
		// Continue - session is still in memory
	}

	sm.logger.Info("Created new ASCOM session",
		zap.Int("client_id", clientID),
		zap.String("session_id", session.SessionID),
		zap.String("client_name", clientName),
		zap.String("client_ip", clientIPAddress))

	return session, nil
}

// LinkSessionToUser associates a session with an authenticated user.
// Should be called after successful JWT authentication.
func (sm *SessionManager) LinkSessionToUser(
	ctx context.Context,
	clientID int,
	userID string,
	username string,
) error {
	val, ok := sm.sessions.Load(clientID)
	if !ok {
		return fmt.Errorf("session not found for ClientID %d", clientID)
	}

	session := val.(*ASCOMSession)
	session.mu.Lock()
	session.UserID = userID
	session.Username = username
	session.mu.Unlock()

	// Update database
	if err := sm.updateSessionUser(ctx, session); err != nil {
		sm.logger.Error("Failed to update session user in database",
			zap.Error(err),
			zap.Int("client_id", clientID),
			zap.String("user_id", userID))
		return err
	}

	sm.logger.Info("Linked session to user",
		zap.Int("client_id", clientID),
		zap.String("session_id", session.SessionID),
		zap.String("user_id", userID),
		zap.String("username", username))

	return nil
}

// LinkSessionToTelescope associates a session with a telescope session.
// Should be called when device access is first established.
func (sm *SessionManager) LinkSessionToTelescope(
	ctx context.Context,
	clientID int,
	telescopeSessionID string,
) error {
	val, ok := sm.sessions.Load(clientID)
	if !ok {
		return fmt.Errorf("session not found for ClientID %d", clientID)
	}

	session := val.(*ASCOMSession)
	session.mu.Lock()
	session.TelescopeSessionID = telescopeSessionID
	session.mu.Unlock()

	// Update database
	if err := sm.updateSessionTelescope(ctx, session); err != nil {
		sm.logger.Error("Failed to update session telescope in database",
			zap.Error(err),
			zap.Int("client_id", clientID),
			zap.String("telescope_session_id", telescopeSessionID))
		return err
	}

	sm.logger.Info("Linked session to telescope",
		zap.Int("client_id", clientID),
		zap.String("session_id", session.SessionID),
		zap.String("telescope_session_id", telescopeSessionID))

	return nil
}

// RecordCommand increments the command counter for a session (PUT/POST requests).
func (sm *SessionManager) RecordCommand(clientID int) {
	if val, ok := sm.sessions.Load(clientID); ok {
		session := val.(*ASCOMSession)
		session.mu.Lock()
		session.TotalCommands++
		session.LastActivityAt = time.Now()
		session.mu.Unlock()
	}
}

// RecordQuery increments the query counter for a session (GET requests).
func (sm *SessionManager) RecordQuery(clientID int) {
	if val, ok := sm.sessions.Load(clientID); ok {
		session := val.(*ASCOMSession)
		session.mu.Lock()
		session.TotalQueries++
		session.LastActivityAt = time.Now()
		session.mu.Unlock()
	}
}

// EndSession closes a session and persists final state to database.
func (sm *SessionManager) EndSession(ctx context.Context, clientID int) error {
	val, ok := sm.sessions.Load(clientID)
	if !ok {
		return fmt.Errorf("session not found for ClientID %d", clientID)
	}

	session := val.(*ASCOMSession)
	return sm.endSession(ctx, session)
}

// endSession is the internal implementation of session closure.
func (sm *SessionManager) endSession(ctx context.Context, session *ASCOMSession) error {
	session.mu.Lock()
	session.Status = SessionStatusClosed
	session.mu.Unlock()

	// Remove from memory
	sm.sessions.Delete(session.ClientID)

	// Update database
	query := `
		UPDATE ascom_sessions
		SET status = $1,
		    ended_at = $2,
		    total_commands = $3,
		    total_queries = $4
		WHERE id = $5
	`

	_, err := sm.db.Exec(ctx, query,
		session.Status,
		time.Now(),
		session.TotalCommands,
		session.TotalQueries,
		session.SessionID)

	if err != nil {
		sm.logger.Error("Failed to close session in database",
			zap.Error(err),
			zap.Int("client_id", session.ClientID),
			zap.String("session_id", session.SessionID))
		return err
	}

	sm.logger.Info("Closed ASCOM session",
		zap.Int("client_id", session.ClientID),
		zap.String("session_id", session.SessionID),
		zap.Int("total_commands", session.TotalCommands),
		zap.Int("total_queries", session.TotalQueries))

	return nil
}

// GetSession retrieves an active session by ClientID.
func (sm *SessionManager) GetSession(clientID int) (*ASCOMSession, error) {
	val, ok := sm.sessions.Load(clientID)
	if !ok {
		return nil, fmt.Errorf("session not found for ClientID %d", clientID)
	}
	return val.(*ASCOMSession), nil
}

// createSessionInDB inserts a new session into the database.
func (sm *SessionManager) createSessionInDB(ctx context.Context, session *ASCOMSession) error {
	query := `
		INSERT INTO ascom_sessions (
			id, device_id, client_id, client_name, client_version,
			client_ip_address, started_at, last_activity_at, status,
			user_id, telescope_session_id, total_commands, total_queries
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	session.mu.RLock()
	defer session.mu.RUnlock()

	var userID *string
	if session.UserID != "" {
		userID = &session.UserID
	}

	var telescopeSessionID *string
	if session.TelescopeSessionID != "" {
		telescopeSessionID = &session.TelescopeSessionID
	}

	_, err := sm.db.Exec(ctx, query,
		session.SessionID,
		session.DeviceID,
		session.ClientID,
		session.ClientName,
		session.ClientVersion,
		session.ClientIPAddress,
		session.StartedAt,
		session.LastActivityAt,
		session.Status,
		userID,
		telescopeSessionID,
		session.TotalCommands,
		session.TotalQueries)

	return err
}

// loadSessionFromDB retrieves a session from the database.
func (sm *SessionManager) loadSessionFromDB(ctx context.Context, clientID int, deviceID string) (*ASCOMSession, error) {
	query := `
		SELECT id, device_id, client_id, client_name, client_version,
		       client_ip_address, started_at, last_activity_at, status,
		       user_id, telescope_session_id, total_commands, total_queries
		FROM ascom_sessions
		WHERE client_id = $1
		  AND device_id = $2
		  AND status != 'closed'
		ORDER BY started_at DESC
		LIMIT 1
	`

	var session ASCOMSession
	var userID, telescopeSessionID *string

	err := sm.db.QueryRow(ctx, query, clientID, deviceID).Scan(
		&session.SessionID,
		&session.DeviceID,
		&session.ClientID,
		&session.ClientName,
		&session.ClientVersion,
		&session.ClientIPAddress,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.Status,
		&userID,
		&telescopeSessionID,
		&session.TotalCommands,
		&session.TotalQueries)

	if err != nil {
		return nil, err
	}

	if userID != nil {
		session.UserID = *userID
	}
	if telescopeSessionID != nil {
		session.TelescopeSessionID = *telescopeSessionID
	}

	return &session, nil
}

// updateSessionActivity updates the last_activity_at timestamp in the database.
func (sm *SessionManager) updateSessionActivity(ctx context.Context, session *ASCOMSession) {
	query := `
		UPDATE ascom_sessions
		SET last_activity_at = $1,
		    status = $2,
		    total_commands = $3,
		    total_queries = $4
		WHERE id = $5
	`

	session.mu.RLock()
	defer session.mu.RUnlock()

	_, err := sm.db.Exec(ctx, query,
		session.LastActivityAt,
		session.Status,
		session.TotalCommands,
		session.TotalQueries,
		session.SessionID)

	if err != nil {
		sm.logger.Error("Failed to update session activity",
			zap.Error(err),
			zap.String("session_id", session.SessionID))
	}
}

// updateSessionUser updates the user_id in the database.
func (sm *SessionManager) updateSessionUser(ctx context.Context, session *ASCOMSession) error {
	query := `
		UPDATE ascom_sessions
		SET user_id = $1
		WHERE id = $2
	`

	session.mu.RLock()
	defer session.mu.RUnlock()

	_, err := sm.db.Exec(ctx, query, session.UserID, session.SessionID)
	return err
}

// updateSessionTelescope updates the telescope_session_id in the database.
func (sm *SessionManager) updateSessionTelescope(ctx context.Context, session *ASCOMSession) error {
	query := `
		UPDATE ascom_sessions
		SET telescope_session_id = $1
		WHERE id = $2
	`

	session.mu.RLock()
	defer session.mu.RUnlock()

	_, err := sm.db.Exec(ctx, query, session.TelescopeSessionID, session.SessionID)
	return err
}

// cleanupStaleSessions runs periodically to identify and close idle sessions.
func (sm *SessionManager) cleanupStaleSessions() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.performCleanup()
		case <-sm.stopChan:
			return
		}
	}
}

// performCleanup checks all active sessions and marks stale ones as idle or closed.
func (sm *SessionManager) performCleanup() {
	ctx := context.Background()
	now := time.Now()
	idleCount := 0
	closedCount := 0

	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*ASCOMSession)
		session.mu.RLock()
		timeSinceActivity := now.Sub(session.LastActivityAt)
		currentStatus := session.Status
		session.mu.RUnlock()

		// If session is idle for too long, close it
		if timeSinceActivity > sm.sessionTimeout {
			if currentStatus == SessionStatusActive {
				// Mark as idle first
				session.mu.Lock()
				session.Status = SessionStatusIdle
				session.mu.Unlock()
				idleCount++

				// Update database
				go sm.updateSessionActivity(ctx, session)
			} else if currentStatus == SessionStatusIdle && timeSinceActivity > 2*sm.sessionTimeout {
				// Close if idle for twice the timeout
				sm.endSession(ctx, session)
				closedCount++
			}
		}

		return true
	})

	if idleCount > 0 || closedCount > 0 {
		sm.logger.Info("Cleanup completed",
			zap.Int("idle_sessions", idleCount),
			zap.Int("closed_sessions", closedCount))
	}

	// Also cleanup database using the stored procedure
	var dbClosedCount int
	err := sm.db.QueryRow(ctx, "SELECT cleanup_idle_ascom_sessions($1)", int(sm.sessionTimeout.Minutes())).Scan(&dbClosedCount)
	if err != nil {
		sm.logger.Error("Failed to cleanup database sessions", zap.Error(err))
	} else if dbClosedCount > 0 {
		sm.logger.Info("Database cleanup completed", zap.Int("closed_sessions", dbClosedCount))
	}
}

// GetActiveSessions returns a snapshot of all active sessions.
func (sm *SessionManager) GetActiveSessions() []*ASCOMSession {
	sessions := make([]*ASCOMSession, 0)
	sm.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*ASCOMSession); ok {
			sessions = append(sessions, session)
		}
		return true
	})
	return sessions
}
