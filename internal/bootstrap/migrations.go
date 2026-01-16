package bootstrap

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// MigrationRunner manages database schema migrations.
type MigrationRunner struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	config *MigrationConfig
}

// Migration represents a single database migration.
type Migration struct {
	// Name is the migration identifier (typically the filename)
	Name string

	// Version is a sequential version number
	Version int

	// Checksum is the SHA256 hash of the migration SQL
	Checksum string

	// SQL is the migration SQL content
	SQL string

	// AppliedAt is when the migration was applied (nil if not applied)
	AppliedAt *time.Time

	// ExecutionTime is how long the migration took to apply
	ExecutionTime time.Duration
}

// NewMigrationRunner creates a new migration runner.
func NewMigrationRunner(pool *pgxpool.Pool, config *MigrationConfig, logger *zap.Logger) *MigrationRunner {
	return &MigrationRunner{
		pool:   pool,
		logger: logger,
		config: config,
	}
}

// Initialize creates the schema_migrations table if it doesn't exist.
func (mr *MigrationRunner) Initialize(ctx context.Context) error {
	mr.logger.Info("Initializing migration tracking table")

	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			execution_time_ms INTEGER NOT NULL,
			applied_by VARCHAR(255),
			CONSTRAINT unique_migration_name UNIQUE(name)
		);

		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
		ON schema_migrations(applied_at);

		CREATE INDEX IF NOT EXISTS idx_schema_migrations_version 
		ON schema_migrations(version);
	`

	_, err := mr.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize schema_migrations table: %w", err)
	}

	mr.logger.Info("Migration tracking table initialized")
	return nil
}

// LoadMigrations loads migration files from the configured schema path.
func (mr *MigrationRunner) LoadMigrations() ([]*Migration, error) {
	if mr.config.SchemaPath == "" {
		return nil, fmt.Errorf("schema path not configured")
	}

	migrations := make([]*Migration, 0)

	// If specific order is defined, use it
	if len(mr.config.Order) > 0 {
		for version, filename := range mr.config.Order {
			migration, err := mr.loadMigrationFile(filename, version+1)
			if err != nil {
				return nil, fmt.Errorf("failed to load migration %s: %w", filename, err)
			}
			migrations = append(migrations, migration)
		}
		return migrations, nil
	}

	// Otherwise, scan directory for .sql files
	files, err := filepath.Glob(filepath.Join(mr.config.SchemaPath, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan migration directory: %w", err)
	}

	for version, filePath := range files {
		filename := filepath.Base(filePath)
		migration, err := mr.loadMigrationFile(filename, version+1)
		if err != nil {
			mr.logger.Warn("Skipping invalid migration file",
				zap.String("file", filename),
				zap.Error(err))
			continue
		}
		migrations = append(migrations, migration)
	}

	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", mr.config.SchemaPath)
	}

	return migrations, nil
}

// loadMigrationFile loads a single migration file.
func (mr *MigrationRunner) loadMigrationFile(filename string, version int) (*Migration, error) {
	filePath := filepath.Join(mr.config.SchemaPath, filename)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate checksum
	hash := sha256.New()
	hash.Write(content)
	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	return &Migration{
		Name:     filename,
		Version:  version,
		Checksum: checksum,
		SQL:      string(content),
	}, nil
}

// GetAppliedMigrations retrieves the list of already applied migrations.
func (mr *MigrationRunner) GetAppliedMigrations(ctx context.Context) (map[string]*Migration, error) {
	query := `
		SELECT version, name, checksum, applied_at, execution_time_ms
		FROM schema_migrations
		ORDER BY version
	`

	rows, err := mr.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]*Migration)
	for rows.Next() {
		var m Migration
		var executionTimeMs int
		err := rows.Scan(&m.Version, &m.Name, &m.Checksum, &m.AppliedAt, &executionTimeMs)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		m.ExecutionTime = time.Duration(executionTimeMs) * time.Millisecond
		applied[m.Name] = &m
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// Run executes all pending migrations in order.
func (mr *MigrationRunner) Run(ctx context.Context) error {
	// Initialize migration tracking table
	if err := mr.Initialize(ctx); err != nil {
		return err
	}

	// Load migrations from files
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return err
	}

	mr.logger.Info("Loaded migrations",
		zap.Int("count", len(migrations)))

	// Get already applied migrations
	applied, err := mr.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	mr.logger.Info("Found applied migrations",
		zap.Int("count", len(applied)))

	// Apply pending migrations
	pendingCount := 0
	for _, migration := range migrations {
		// Check if already applied
		if appliedMigration, exists := applied[migration.Name]; exists {
			// Verify checksum matches
			if appliedMigration.Checksum != migration.Checksum {
				return fmt.Errorf("migration %s has different checksum (applied: %s, current: %s) - manual intervention required",
					migration.Name,
					appliedMigration.Checksum[:8],
					migration.Checksum[:8])
			}
			mr.logger.Debug("Migration already applied",
				zap.String("name", migration.Name),
				zap.Int("version", migration.Version))
			continue
		}

		// Apply migration
		mr.logger.Info("Applying migration",
			zap.String("name", migration.Name),
			zap.Int("version", migration.Version))

		if err := mr.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}

		pendingCount++
		mr.logger.Info("Migration applied successfully",
			zap.String("name", migration.Name),
			zap.Int("version", migration.Version),
			zap.Duration("execution_time", migration.ExecutionTime))
	}

	if pendingCount == 0 {
		mr.logger.Info("No pending migrations, database schema is up to date")
	} else {
		mr.logger.Info("All migrations applied successfully",
			zap.Int("applied", pendingCount))
	}

	return nil
}

// applyMigration executes a single migration and records it.
func (mr *MigrationRunner) applyMigration(ctx context.Context, migration *Migration) error {
	startTime := time.Now()

	// Begin transaction
	tx, err := mr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	mr.logger.Debug("Executing migration SQL",
		zap.String("name", migration.Name),
		zap.Int("sql_length", len(migration.SQL)))

	_, err = tx.Exec(ctx, migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	executionTime := time.Since(startTime)
	executionTimeMs := int(executionTime.Milliseconds())

	// Get the user who applied the migration (from system environment)
	appliedBy := os.Getenv("USER")
	if appliedBy == "" {
		appliedBy = "bootstrap-coordinator"
	}

	recordQuery := `
		INSERT INTO schema_migrations (version, name, checksum, applied_at, execution_time_ms, applied_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, recordQuery,
		migration.Version,
		migration.Name,
		migration.Checksum,
		time.Now(),
		executionTimeMs,
		appliedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	migration.ExecutionTime = executionTime
	migration.AppliedAt = &startTime

	return nil
}

// Rollback rolls back the last applied migration (if supported).
// Note: This is a simple implementation that drops and recreates tables.
// For production, consider a more sophisticated approach with explicit rollback SQL.
func (mr *MigrationRunner) Rollback(ctx context.Context) error {
	mr.logger.Warn("Rollback requested - this will drop the last applied migration")

	// Get the last applied migration
	query := `
		SELECT version, name, checksum
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1
	`

	var m Migration
	err := mr.pool.QueryRow(ctx, query).Scan(&m.Version, &m.Name, &m.Checksum)
	if err != nil {
		return fmt.Errorf("failed to find last migration: %w", err)
	}

	mr.logger.Info("Rolling back migration",
		zap.String("name", m.Name),
		zap.Int("version", m.Version))

	// Begin transaction
	tx, err := mr.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete migration record
	deleteQuery := `DELETE FROM schema_migrations WHERE version = $1`
	_, err = tx.Exec(ctx, deleteQuery, m.Version)
	if err != nil {
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	mr.logger.Warn("Migration record removed",
		zap.String("name", m.Name),
		zap.String("note", "Database schema not automatically reverted - manual cleanup may be required"))

	return nil
}

// Status returns the current migration status.
func (mr *MigrationRunner) Status(ctx context.Context) ([]MigrationStatus, error) {
	// Initialize if needed
	if err := mr.Initialize(ctx); err != nil {
		return nil, err
	}

	// Load migrations from files
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := mr.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Build status list
	statuses := make([]MigrationStatus, 0, len(migrations))
	for _, migration := range migrations {
		status := MigrationStatus{
			Name:    migration.Name,
			Version: migration.Version,
			Applied: false,
		}

		if appliedMigration, exists := applied[migration.Name]; exists {
			status.Applied = true
			status.AppliedAt = appliedMigration.AppliedAt
			status.ExecutionTime = appliedMigration.ExecutionTime
			status.ChecksumMatch = appliedMigration.Checksum == migration.Checksum
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	Name          string
	Version       int
	Applied       bool
	AppliedAt     *time.Time
	ExecutionTime time.Duration
	ChecksumMatch bool
}

// PrintStatus prints the migration status to the logger.
func (mr *MigrationRunner) PrintStatus(ctx context.Context) error {
	statuses, err := mr.Status(ctx)
	if err != nil {
		return err
	}

	mr.logger.Info("Migration Status",
		zap.Int("total_migrations", len(statuses)))

	for _, status := range statuses {
		if status.Applied {
			mr.logger.Info("Migration",
				zap.String("name", status.Name),
				zap.Int("version", status.Version),
				zap.Bool("applied", status.Applied),
				zap.Time("applied_at", *status.AppliedAt),
				zap.Duration("execution_time", status.ExecutionTime),
				zap.Bool("checksum_match", status.ChecksumMatch))
		} else {
			mr.logger.Info("Migration",
				zap.String("name", status.Name),
				zap.Int("version", status.Version),
				zap.Bool("applied", status.Applied),
				zap.String("status", "PENDING"))
		}
	}

	return nil
}

// ValidateChecksums validates that all applied migrations have matching checksums.
func (mr *MigrationRunner) ValidateChecksums(ctx context.Context) error {
	statuses, err := mr.Status(ctx)
	if err != nil {
		return err
	}

	mismatchCount := 0
	for _, status := range statuses {
		if status.Applied && !status.ChecksumMatch {
			mr.logger.Error("Migration checksum mismatch",
				zap.String("name", status.Name),
				zap.Int("version", status.Version))
			mismatchCount++
		}
	}

	if mismatchCount > 0 {
		return fmt.Errorf("found %d migration(s) with checksum mismatches", mismatchCount)
	}

	mr.logger.Info("All migration checksums are valid")
	return nil
}

// CalculateChecksum calculates the SHA256 checksum of a file.
func CalculateChecksum(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
