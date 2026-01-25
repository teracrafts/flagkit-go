package flagkit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"

	"golang.org/x/crypto/pbkdf2"
	"crypto/sha256"
)

const (
	// EncryptionVersion is the current encryption format version.
	EncryptionVersion = 1

	// ivLength is the initialization vector length for AES-GCM (96 bits).
	ivLength = 12

	// keyLength is the AES-256 key length (256 bits).
	keyLength = 32

	// pbkdf2Iterations is the number of PBKDF2 iterations for key derivation.
	pbkdf2Iterations = 100000

	// encryptionSalt is the static salt for key derivation.
	encryptionSalt = "FlagKit-v1-cache"
)

// EncryptedData represents encrypted data with metadata.
type EncryptedData struct {
	IV      string `json:"iv"`
	Data    string `json:"data"`
	Version int    `json:"version"`
}

// EncryptedStorage provides AES-256-GCM encryption for cache data.
// The encryption key is derived from the API key using PBKDF2.
type EncryptedStorage struct {
	apiKey     string
	derivedKey []byte
	logger     Logger
	mu         sync.RWMutex
}

// EncryptedStorageConfig contains configuration for encrypted storage.
type EncryptedStorageConfig struct {
	// APIKey is used to derive the encryption key.
	APIKey string

	// Logger for debug output.
	Logger Logger
}

// NewEncryptedStorage creates a new encrypted storage instance.
func NewEncryptedStorage(config *EncryptedStorageConfig) (*EncryptedStorage, error) {
	if config.APIKey == "" {
		return nil, NewError(ErrConfigMissingRequired, "API key is required for encrypted storage")
	}

	storage := &EncryptedStorage{
		apiKey: config.APIKey,
		logger: config.Logger,
	}

	// Derive encryption key from API key using PBKDF2
	if err := storage.deriveKey(); err != nil {
		return nil, err
	}

	return storage, nil
}

// deriveKey derives the encryption key from the API key using PBKDF2.
func (s *EncryptedStorage) deriveKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.derivedKey = pbkdf2.Key(
		[]byte(s.apiKey),
		[]byte(encryptionSalt),
		pbkdf2Iterations,
		keyLength,
		sha256.New,
	)

	if s.logger != nil {
		s.logger.Debug("Derived encryption key using PBKDF2")
	}

	return nil
}

// Encrypt encrypts plaintext data using AES-256-GCM.
func (s *EncryptedStorage) Encrypt(plaintext string) (string, error) {
	s.mu.RLock()
	key := s.derivedKey
	s.mu.RUnlock()

	if key == nil {
		return "", SecurityError(ErrSecurityEncryptionFailed, "encryption key not derived")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", SecurityError(ErrSecurityEncryptionFailed, "failed to create cipher: "+err.Error())
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", SecurityError(ErrSecurityEncryptionFailed, "failed to create GCM: "+err.Error())
	}

	// Generate random IV
	iv := make([]byte, ivLength)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", SecurityError(ErrSecurityEncryptionFailed, "failed to generate IV: "+err.Error())
	}

	// Encrypt data (GCM includes authentication tag automatically)
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	// Create encrypted data structure
	encrypted := EncryptedData{
		IV:      base64.StdEncoding.EncodeToString(iv),
		Data:    base64.StdEncoding.EncodeToString(ciphertext),
		Version: EncryptionVersion,
	}

	// Serialize to JSON
	result, err := json.Marshal(encrypted)
	if err != nil {
		return "", SecurityError(ErrSecurityEncryptionFailed, "failed to marshal encrypted data: "+err.Error())
	}

	return string(result), nil
}

// Decrypt decrypts ciphertext data using AES-256-GCM.
func (s *EncryptedStorage) Decrypt(ciphertext string) (string, error) {
	s.mu.RLock()
	key := s.derivedKey
	s.mu.RUnlock()

	if key == nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "encryption key not derived")
	}

	// Parse encrypted data structure
	var encrypted EncryptedData
	if err := json.Unmarshal([]byte(ciphertext), &encrypted); err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "failed to parse encrypted data: "+err.Error())
	}

	// Check version
	if encrypted.Version != EncryptionVersion {
		return "", SecurityError(ErrSecurityDecryptionFailed, "unsupported encryption version")
	}

	// Decode IV
	iv, err := base64.StdEncoding.DecodeString(encrypted.IV)
	if err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "failed to decode IV: "+err.Error())
	}

	// Decode ciphertext
	data, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "failed to decode ciphertext: "+err.Error())
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "failed to create cipher: "+err.Error())
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "failed to create GCM: "+err.Error())
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, iv, data, nil)
	if err != nil {
		return "", SecurityError(ErrSecurityDecryptionFailed, "decryption failed (invalid key or corrupted data): "+err.Error())
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a string appears to be encrypted data.
func IsEncrypted(data string) bool {
	var encrypted EncryptedData
	if err := json.Unmarshal([]byte(data), &encrypted); err != nil {
		return false
	}
	return encrypted.Version > 0 && encrypted.IV != "" && encrypted.Data != ""
}

// EncryptedCacheStorage wraps a cache to provide encrypted storage.
type EncryptedCacheStorage struct {
	storage   *EncryptedStorage
	cache     map[string]string
	mu        sync.RWMutex
	logger    Logger
}

// NewEncryptedCacheStorage creates a new encrypted cache storage.
func NewEncryptedCacheStorage(apiKey string, logger Logger) (*EncryptedCacheStorage, error) {
	storage, err := NewEncryptedStorage(&EncryptedStorageConfig{
		APIKey: apiKey,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	return &EncryptedCacheStorage{
		storage: storage,
		cache:   make(map[string]string),
		logger:  logger,
	}, nil
}

// Set stores a value with encryption.
func (c *EncryptedCacheStorage) Set(key, value string) error {
	encrypted, err := c.storage.Encrypt(value)
	if err != nil {
		// Fall back to unencrypted storage
		if c.logger != nil {
			c.logger.Warn("Encryption failed, storing unencrypted", "key", key, "error", err.Error())
		}
		c.mu.Lock()
		c.cache[key] = value
		c.mu.Unlock()
		return err
	}

	c.mu.Lock()
	c.cache[key] = encrypted
	c.mu.Unlock()

	return nil
}

// Get retrieves and decrypts a value.
func (c *EncryptedCacheStorage) Get(key string) (string, error) {
	c.mu.RLock()
	encrypted, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return "", nil
	}

	// Check if data is encrypted
	if !IsEncrypted(encrypted) {
		// Return as-is (legacy unencrypted data)
		return encrypted, nil
	}

	return c.storage.Decrypt(encrypted)
}

// Delete removes a value.
func (c *EncryptedCacheStorage) Delete(key string) {
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// Clear removes all values.
func (c *EncryptedCacheStorage) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]string)
	c.mu.Unlock()
}

// Has checks if a key exists.
func (c *EncryptedCacheStorage) Has(key string) bool {
	c.mu.RLock()
	_, ok := c.cache[key]
	c.mu.RUnlock()
	return ok
}

// IsEncryptionAvailable returns true if encryption is available.
func (c *EncryptedCacheStorage) IsEncryptionAvailable() bool {
	c.storage.mu.RLock()
	defer c.storage.mu.RUnlock()
	return c.storage.derivedKey != nil
}
