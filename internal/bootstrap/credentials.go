package bootstrap

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CredentialSource indicates where credentials were loaded from.
type CredentialSource string

const (
	// CredentialSourcePgpass indicates credentials from .pgpass file
	CredentialSourcePgpass CredentialSource = "pgpass"
	// CredentialSourceEnvironment indicates credentials from environment variables
	CredentialSourceEnvironment CredentialSource = "environment"
	// CredentialSourceUnknown indicates credentials from unknown source
	CredentialSourceUnknown CredentialSource = "unknown"
)

// Credentials holds database connection credentials.
type Credentials struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	Source   CredentialSource
}

// LoadCredentials attempts to load database credentials using the fallback chain:
// 1. .pgpass file ($PGPASSFILE or ~/.pgpass)
// 2. Environment variables (DATABASE_URL or individual PG* variables)
// 3. Error if no credentials found
func LoadCredentials(dbConfig *DatabaseConfig) (*Credentials, error) {
	return LoadCredentialsFromPath("", dbConfig)
}

// LoadCredentialsFromPath attempts to load database credentials with a custom .pgpass path.
// If pgpassPath is empty, uses the standard .pgpass file locations.
func LoadCredentialsFromPath(pgpassPath string, dbConfig *DatabaseConfig) (*Credentials, error) {
	// Validate and clean pgpassPath to prevent directory traversal attacks
	if pgpassPath != "" {
		// Clean the path to resolve any .. sequences
		cleanPath := filepath.Clean(pgpassPath)

		// Ensure path doesn't contain suspicious patterns after cleaning
		if strings.Contains(cleanPath, "..") || strings.Contains(cleanPath, "/etc") || strings.Contains(cleanPath, "/proc") {
			return nil, fmt.Errorf("invalid pgpass path: %s", pgpassPath)
		}
		// Ensure it's an absolute path or in a reasonable location
		if !filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, "~") && !strings.HasPrefix(cleanPath, ".") {
			return nil, fmt.Errorf("pgpass path must be absolute or start with ~ or . : %s", pgpassPath)
		}
		pgpassPath = cleanPath
	}

	// Try loading from .pgpass first
	creds, err := LoadFromPgpassFile(pgpassPath, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.User)
	if err == nil {
		return creds, nil
	}

	// Fallback to environment variables
	creds, err = LoadFromEnvironment()
	if err == nil {
		// If specific connection parameters were provided, use them, otherwise use env values
		if dbConfig.Host != "" && dbConfig.Host != creds.Host {
			creds.Host = dbConfig.Host
		}
		if dbConfig.Port != 0 && dbConfig.Port != creds.Port {
			creds.Port = dbConfig.Port
		}
		if dbConfig.Name != "" && dbConfig.Name != creds.Database {
			creds.Database = dbConfig.Name
		}
		if dbConfig.User != "" && dbConfig.User != creds.User {
			creds.User = dbConfig.User
		}
		return creds, nil
	}

	return nil, fmt.Errorf("failed to load credentials: tried .pgpass and environment variables")
}

// LoadFromPgpassFile loads credentials from PostgreSQL .pgpass file at a specific path.
//
// The .pgpass file format is: hostname:port:database:username:password
// Lines starting with # are comments. Wildcards (*) are supported for any field.
//
// If pgpassPath is empty, uses standard .pgpass file locations:
// 1. $PGPASSFILE environment variable
// 2. ~/.pgpass (Unix/macOS)
// 3. %APPDATA%\postgresql\pgpass.conf (Windows)
func LoadFromPgpassFile(pgpassPath string, host string, port int, database, user string) (*Credentials, error) {
	// Determine .pgpass file location
	if pgpassPath == "" {
		pgpassPath = os.Getenv("PGPASSFILE")
		if pgpassPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			pgpassPath = filepath.Join(home, ".pgpass")
		}
	}

	// Check if file exists
	fileInfo, err := os.Stat(pgpassPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(".pgpass file not found at %s", pgpassPath)
		}
		return nil, fmt.Errorf("failed to stat .pgpass file: %w", err)
	}

	// Check file permissions (must be 0600 on Unix-like systems)
	// PostgreSQL enforces this for security
	if fileInfo.Mode().Perm() != 0600 {
		return nil, fmt.Errorf(".pgpass file has incorrect permissions %o (must be 0600)", fileInfo.Mode().Perm())
	}

	// Open and parse the file
	file, err := os.Open(pgpassPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .pgpass file: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: hostname:port:database:username:password
		parts := strings.Split(line, ":")
		if len(parts) != 5 {
			continue // Invalid line format, skip
		}

		fileHost := parts[0]
		filePortStr := parts[1]
		fileDB := parts[2]
		fileUser := parts[3]
		filePassword := parts[4]

		// Parse port
		filePort := 0
		if filePortStr != "*" {
			filePort, err = strconv.Atoi(filePortStr)
			if err != nil {
				continue // Invalid port, skip line
			}
		}

		// Check if this entry matches our requirements
		if matchesWildcard(host, fileHost) &&
			matchesWildcardInt(port, filePort, filePortStr) &&
			matchesWildcard(database, fileDB) &&
			matchesWildcard(user, fileUser) {

			// Use the values from the .pgpass line (or requested values if wildcard)
			resultHost := host
			if fileHost != "*" {
				resultHost = fileHost
			}

			resultPort := port
			if filePortStr != "*" && filePort != 0 {
				resultPort = filePort
			}

			resultDB := database
			if fileDB != "*" {
				resultDB = fileDB
			}

			resultUser := user
			if fileUser != "*" {
				resultUser = fileUser
			}

			return &Credentials{
				Host:     resultHost,
				Port:     resultPort,
				Database: resultDB,
				User:     resultUser,
				Password: filePassword,
				Source:   CredentialSourcePgpass,
			}, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .pgpass file: %w", err)
	}

	return nil, fmt.Errorf("no matching entry found in .pgpass for %s:%d/%s/%s", host, port, database, user)
}

// LoadFromEnvironment loads credentials from environment variables.
//
// Supports two formats:
// 1. DATABASE_URL: postgresql://user:password@host:port/database?sslmode=...
// 2. Individual variables: PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD
//
// DATABASE_URL takes precedence over individual PG* variables.
func LoadFromEnvironment() (*Credentials, error) {
	// Try DATABASE_URL first
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		return parseConnectionURL(databaseURL)
	}

	// Try individual environment variables
	host := os.Getenv("PGHOST")
	portStr := os.Getenv("PGPORT")
	database := os.Getenv("PGDATABASE")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")

	// Check if at least some variables are set
	if host == "" && portStr == "" && database == "" && user == "" && password == "" {
		return nil, fmt.Errorf("no environment variables found (DATABASE_URL, PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD)")
	}

	// Apply defaults for missing values
	if host == "" {
		host = "localhost"
	}
	if database == "" {
		database = "postgres"
	}
	if user == "" {
		user = os.Getenv("USER") // Use system username as fallback
		if user == "" {
			user = "postgres"
		}
	}

	port := 5432 // Default PostgreSQL port
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PGPORT value %q: %w", portStr, err)
		}
	}

	if password == "" {
		return nil, fmt.Errorf("PGPASSWORD environment variable not set")
	}

	return &Credentials{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		Source:   CredentialSourceEnvironment,
	}, nil
}

// parseConnectionURL parses a PostgreSQL connection URL.
// Format: postgresql://user:password@host:port/database?param=value
func parseConnectionURL(connURL string) (*Credentials, error) {
	u, err := url.Parse(connURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return nil, fmt.Errorf("invalid DATABASE_URL scheme: %s (expected postgresql or postgres)", u.Scheme)
	}

	// Extract user and password
	user := u.User.Username()
	password, _ := u.User.Password()

	if user == "" {
		return nil, fmt.Errorf("DATABASE_URL missing username")
	}
	if password == "" {
		return nil, fmt.Errorf("DATABASE_URL missing password")
	}

	// Extract host and port
	host := u.Hostname()
	portStr := u.Port()
	port := 5432 // Default
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port in DATABASE_URL: %w", err)
		}
	}

	// Extract database name
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return nil, fmt.Errorf("DATABASE_URL missing database name")
	}

	return &Credentials{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		Source:   CredentialSourceEnvironment,
	}, nil
}

// matchesWildcard checks if a value matches a pattern (supporting * wildcard).
func matchesWildcard(value, pattern string) bool {
	if pattern == "*" {
		return true
	}
	return value == pattern
}

// matchesWildcardInt checks if an int value matches a pattern.
func matchesWildcardInt(value, patternValue int, patternStr string) bool {
	if patternStr == "*" {
		return true
	}
	return value == patternValue
}

// ConnectionString returns a PostgreSQL connection string from credentials.
func (c *Credentials) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
		c.Host, c.Port, c.Database, c.User, c.Password)
}

// ConnectionURL returns a PostgreSQL connection URL from credentials.
func (c *Credentials) ConnectionURL(sslmode string) string {
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, sslmode)
}
