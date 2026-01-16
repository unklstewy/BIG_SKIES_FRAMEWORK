package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// ConfigSourceDir is the base directory for configuration sources in the container
	ConfigSourceDir = "/app/configs"

	// ConfigTargetDir is the ASCOM configuration directory
	ConfigTargetDir = "/tmp/ascom-config/alpaca/ascom-alpaca-simulator"

	// StateFile stores the current configuration state
	StateFile = "/tmp/ascom-current-config.json"
)

// ConfigLoader handles configuration file operations.
type ConfigLoader struct {
	sourceDir string
	targetDir string
	stateFile string
}

// NewConfigLoader creates a new configuration loader instance.
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		sourceDir: ConfigSourceDir,
		targetDir: ConfigTargetDir,
		stateFile: StateFile,
	}
}

// LoadConfiguration loads a configuration for the specified model and mount type.
// It validates the configuration exists, backs up the current configuration,
// copies the new configuration, and updates the state file.
func (cl *ConfigLoader) LoadConfiguration(model, mountType, loadedBy string) error {
	// Validate parameters
	if !ValidateModel(model) {
		return fmt.Errorf("invalid model: %s", model)
	}
	if !ValidateMountType(mountType) {
		return fmt.Errorf("invalid mount type: %s", mountType)
	}

	// Check if source configuration exists
	sourcePath := filepath.Join(cl.sourceDir, model, mountType)
	if !cl.directoryExists(sourcePath) {
		return fmt.Errorf("configuration not found: %s/%s", model, mountType)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(cl.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Backup existing configuration if it exists
	if cl.directoryExists(cl.targetDir) && cl.directoryHasFiles(cl.targetDir) {
		backupPath := fmt.Sprintf("%s.backup.%s", cl.targetDir, time.Now().Format("20060102_150405"))
		if err := cl.copyDirectory(cl.targetDir, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing configuration: %w", err)
		}
	}

	// Clear target directory
	if err := cl.clearDirectory(cl.targetDir); err != nil {
		return fmt.Errorf("failed to clear target directory: %w", err)
	}

	// Copy new configuration
	if err := cl.copyDirectory(sourcePath, cl.targetDir); err != nil {
		return fmt.Errorf("failed to copy configuration: %w", err)
	}

	// Update state file
	status := ConfigStatus{
		Model:      model,
		MountType:  mountType,
		LoadedAt:   time.Now(),
		LoadedBy:   loadedBy,
		ConfigPath: sourcePath,
	}

	if err := cl.saveState(status); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// GetCurrentStatus reads the current configuration status from the state file.
func (cl *ConfigLoader) GetCurrentStatus() (*ConfigStatus, error) {
	data, err := os.ReadFile(cl.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No configuration loaded yet
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var status ConfigStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &status, nil
}

// saveState writes the configuration status to the state file.
func (cl *ConfigLoader) saveState(status ConfigStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(cl.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// directoryExists checks if a directory exists.
func (cl *ConfigLoader) directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// directoryHasFiles checks if a directory contains any files.
func (cl *ConfigLoader) directoryHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// clearDirectory removes all contents of a directory.
func (cl *ConfigLoader) clearDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return err
		}
	}

	return nil
}

// copyDirectory recursively copies a directory from src to dst.
func (cl *ConfigLoader) copyDirectory(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := cl.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := cl.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func (cl *ConfigLoader) copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

// ValidateConfigurationExists checks if a configuration exists for the given model and mount type.
func (cl *ConfigLoader) ValidateConfigurationExists(model, mountType string) bool {
	sourcePath := filepath.Join(cl.sourceDir, model, mountType)
	return cl.directoryExists(sourcePath) && cl.directoryHasFiles(sourcePath)
}
