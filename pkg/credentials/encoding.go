package credentials

import (
	"encoding/base64"
	"fmt"
)

// EncodePath encodes a file path using base64 for minor obscurity
// This is NOT cryptographic security, just prevents casual observation
func EncodePath(path string) string {
	return base64.StdEncoding.EncodeToString([]byte(path))
}

// DecodePath decodes a base64-encoded file path
func DecodePath(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode path: %w", err)
	}
	return string(decoded), nil
}

// CredentialMessage represents the MQTT message payload for credential distribution
type CredentialMessage struct {
	// PgpassPath is the base64-encoded path to the .pgpass file
	PgpassPath string `json:"pgpass_path"`

	// Version allows for future protocol changes
	Version string `json:"version"`
}

// NewCredentialMessage creates a credential message with the given path
func NewCredentialMessage(pgpassPath string) *CredentialMessage {
	return &CredentialMessage{
		PgpassPath: EncodePath(pgpassPath),
		Version:    "1.0",
	}
}

// GetDecodedPath returns the decoded .pgpass file path
func (m *CredentialMessage) GetDecodedPath() (string, error) {
	return DecodePath(m.PgpassPath)
}
