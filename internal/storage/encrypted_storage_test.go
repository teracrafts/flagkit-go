package storage

import (
	"encoding/json"
	"testing"
)

func TestNewEncryptedStorage(t *testing.T) {
	t.Run("creates storage with valid API key", func(t *testing.T) {
		storage, err := NewEncryptedStorage(&EncryptedStorageConfig{
			APIKey: "sdk_test_api_key_12345",
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if storage == nil {
			t.Error("expected storage to be created")
		}
	})

	t.Run("returns error for empty API key", func(t *testing.T) {
		_, err := NewEncryptedStorage(&EncryptedStorageConfig{
			APIKey: "",
		})
		if err == nil {
			t.Error("expected error for empty API key")
		}
	})
}

func TestEncryptedStorageEncryptDecrypt(t *testing.T) {
	storage, err := NewEncryptedStorage(&EncryptedStorageConfig{
		APIKey: "sdk_test_api_key_12345",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	t.Run("encrypts and decrypts string", func(t *testing.T) {
		plaintext := "Hello, World!"
		encrypted, err := storage.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		// Verify it's actually encrypted (different from plaintext)
		if encrypted == plaintext {
			t.Error("encrypted text should not equal plaintext")
		}

		// Verify it's valid JSON with expected structure
		var encData EncryptedData
		if err := json.Unmarshal([]byte(encrypted), &encData); err != nil {
			t.Errorf("encrypted data is not valid JSON: %v", err)
		}
		if encData.Version != EncryptionVersion {
			t.Errorf("expected version %d, got %d", EncryptionVersion, encData.Version)
		}
		if encData.IV == "" {
			t.Error("expected non-empty IV")
		}
		if encData.Data == "" {
			t.Error("expected non-empty data")
		}

		// Decrypt
		decrypted, err := storage.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("expected '%s', got '%s'", plaintext, decrypted)
		}
	})

	t.Run("encrypts and decrypts JSON data", func(t *testing.T) {
		data := map[string]any{
			"key":   "test-flag",
			"value": true,
			"nested": map[string]any{
				"foo": "bar",
			},
		}
		plaintext, _ := json.Marshal(data)

		encrypted, err := storage.Encrypt(string(plaintext))
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		decrypted, err := storage.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		if decrypted != string(plaintext) {
			t.Errorf("expected '%s', got '%s'", plaintext, decrypted)
		}
	})

	t.Run("produces different ciphertext for same plaintext", func(t *testing.T) {
		plaintext := "test data"

		encrypted1, _ := storage.Encrypt(plaintext)
		encrypted2, _ := storage.Encrypt(plaintext)

		// Due to random IV, encryptions should be different
		if encrypted1 == encrypted2 {
			t.Error("expected different ciphertext for each encryption (random IV)")
		}
	})

	t.Run("fails to decrypt with wrong key", func(t *testing.T) {
		storage1, _ := NewEncryptedStorage(&EncryptedStorageConfig{
			APIKey: "sdk_key_one_12345",
		})
		storage2, _ := NewEncryptedStorage(&EncryptedStorageConfig{
			APIKey: "sdk_key_two_12345",
		})

		plaintext := "secret message"
		encrypted, _ := storage1.Encrypt(plaintext)

		_, err := storage2.Decrypt(encrypted)
		if err == nil {
			t.Error("expected decryption to fail with wrong key")
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		encrypted, err := storage.Encrypt("")
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		decrypted, err := storage.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		if decrypted != "" {
			t.Errorf("expected empty string, got '%s'", decrypted)
		}
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		plaintext := "Hello, World! Привет, мир! " + "\n\t"
		encrypted, err := storage.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		decrypted, err := storage.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("expected '%s', got '%s'", plaintext, decrypted)
		}
	})
}

func TestIsEncrypted(t *testing.T) {
	storage, _ := NewEncryptedStorage(&EncryptedStorageConfig{
		APIKey: "sdk_test_api_key_12345",
	})

	t.Run("returns true for encrypted data", func(t *testing.T) {
		encrypted, _ := storage.Encrypt("test")
		if !IsEncrypted(encrypted) {
			t.Error("expected IsEncrypted to return true")
		}
	})

	t.Run("returns false for plain text", func(t *testing.T) {
		if IsEncrypted("plain text") {
			t.Error("expected IsEncrypted to return false for plain text")
		}
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		if IsEncrypted("{invalid json}") {
			t.Error("expected IsEncrypted to return false for invalid JSON")
		}
	})

	t.Run("returns false for JSON without encryption fields", func(t *testing.T) {
		if IsEncrypted(`{"key": "value"}`) {
			t.Error("expected IsEncrypted to return false for non-encrypted JSON")
		}
	})
}

func TestEncryptedCacheStorage(t *testing.T) {
	t.Run("creates cache storage", func(t *testing.T) {
		cache, err := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}
		if cache == nil {
			t.Error("expected cache to be created")
		}
	})

	t.Run("set and get", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		err := cache.Set("key1", "value1")
		if err != nil {
			t.Errorf("set failed: %v", err)
		}

		value, err := cache.Get("key1")
		if err != nil {
			t.Errorf("get failed: %v", err)
		}
		if value != "value1" {
			t.Errorf("expected 'value1', got '%s'", value)
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		value, err := cache.Get("nonexistent")
		if err != nil {
			t.Errorf("get failed: %v", err)
		}
		if value != "" {
			t.Errorf("expected empty string for non-existent key, got '%s'", value)
		}
	})

	t.Run("delete", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		_ = cache.Set("key1", "value1")
		cache.Delete("key1")

		value, _ := cache.Get("key1")
		if value != "" {
			t.Errorf("expected empty string after delete, got '%s'", value)
		}
	})

	t.Run("clear", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		_ = cache.Set("key1", "value1")
		_ = cache.Set("key2", "value2")
		cache.Clear()

		if cache.Has("key1") || cache.Has("key2") {
			t.Error("expected cache to be cleared")
		}
	})

	t.Run("has", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		if cache.Has("key1") {
			t.Error("expected Has to return false for non-existent key")
		}

		_ = cache.Set("key1", "value1")

		if !cache.Has("key1") {
			t.Error("expected Has to return true after set")
		}
	})

	t.Run("isEncryptionAvailable", func(t *testing.T) {
		cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

		if !cache.IsEncryptionAvailable() {
			t.Error("expected encryption to be available")
		}
	})
}

func TestEncryptedCacheStorageWithLegacyData(t *testing.T) {
	cache, _ := NewEncryptedCacheStorage("sdk_test_api_key_12345", nil)

	// Simulate legacy unencrypted data by directly setting in internal cache
	cache.mu.Lock()
	cache.cache["legacy"] = "plain text value"
	cache.mu.Unlock()

	// Should be able to read legacy unencrypted data
	value, err := cache.Get("legacy")
	if err != nil {
		t.Errorf("failed to get legacy data: %v", err)
	}
	if value != "plain text value" {
		t.Errorf("expected 'plain text value', got '%s'", value)
	}
}

func BenchmarkEncryption(b *testing.B) {
	storage, _ := NewEncryptedStorage(&EncryptedStorageConfig{
		APIKey: "sdk_test_api_key_12345",
	})

	data := `{"flags":[{"key":"feature-1","value":true},{"key":"feature-2","value":"enabled"}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Encrypt(data)
	}
}

func BenchmarkDecryption(b *testing.B) {
	storage, _ := NewEncryptedStorage(&EncryptedStorageConfig{
		APIKey: "sdk_test_api_key_12345",
	})

	data := `{"flags":[{"key":"feature-1","value":true},{"key":"feature-2","value":"enabled"}]}`
	encrypted, _ := storage.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Decrypt(encrypted)
	}
}
