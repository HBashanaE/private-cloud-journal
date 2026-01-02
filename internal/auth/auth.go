package auth

import (
	"encoding/json"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const ConfigFile = "app_config.json"

// AppConfig stores settings that persist across restarts
type AppConfig struct {
	PasswordHash []byte `json:"password_hash"`
}

// Global config variable
var currentConfig *AppConfig

// SavePasswordHash creates a new config with the hashed password
func SavePasswordHash(password string) error {
	// 1. Hash the password using Bcrypt (Cost 14 is secure)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}

	// 2. Create config object
	cfg := &AppConfig{
		PasswordHash: bytes,
	}

	// 3. Convert to JSON
	fileData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// 4. Write to disk
	return os.WriteFile(ConfigFile, fileData, 0644)
}

// VerifyPassword checks if the entered password matches the stored hash
func VerifyPassword(password string) bool {
	if currentConfig == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword(currentConfig.PasswordHash, []byte(password))
	return err == nil
}

// LoadConfig tries to read the config file.
// Returns true if file exists and loads successfully, false otherwise.
func LoadConfig() bool {
	fileData, err := os.ReadFile(ConfigFile)
	if err != nil {
		return false // File likely doesn't exist (First Run)
	}

	var cfg AppConfig
	if err := json.Unmarshal(fileData, &cfg); err != nil {
		return false
	}

	currentConfig = &cfg
	return true
}

// IsFirstRun checks if we need to show the Registration screen
func IsFirstRun() bool {
	return !LoadConfig()
}
