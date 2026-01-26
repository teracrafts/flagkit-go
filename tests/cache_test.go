package tests

import (
	"testing"
	"time"

	. "github.com/flagkit/flagkit-go"
	"github.com/stretchr/testify/assert"
)

func TestCacheSetAndGet(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flag := FlagState{
		Key:     "test-flag",
		Value:   true,
		Enabled: true,
		Version: 1,
	}

	cache.Set("test-flag", flag, 5*time.Minute)

	result := cache.Get("test-flag")
	assert.NotNil(t, result)
	assert.Equal(t, "test-flag", result.Key)
	assert.Equal(t, true, result.Value)
}

func TestCacheGetNonExistent(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	result := cache.Get("non-existent")
	assert.Nil(t, result)
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     50 * time.Millisecond,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flag := FlagState{
		Key:     "test-flag",
		Value:   true,
		Enabled: true,
	}

	cache.Set("test-flag", flag, 50*time.Millisecond)

	// Should exist immediately
	assert.NotNil(t, cache.Get("test-flag"))

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	assert.Nil(t, cache.Get("test-flag"))
}

func TestCacheGetStale(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     50 * time.Millisecond,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flag := FlagState{
		Key:     "test-flag",
		Value:   true,
		Enabled: true,
	}

	cache.Set("test-flag", flag, 50*time.Millisecond)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// GetStale should still return the value
	result := cache.GetStale("test-flag")
	assert.NotNil(t, result)
	assert.Equal(t, true, result.Value)
}

func TestCacheHas(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flag := FlagState{Key: "test-flag", Value: true}
	cache.Set("test-flag", flag, 5*time.Minute)

	assert.True(t, cache.Has("test-flag"))
	assert.False(t, cache.Has("non-existent"))
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flag := FlagState{Key: "test-flag", Value: true}
	cache.Set("test-flag", flag, 5*time.Minute)

	cache.Delete("test-flag")
	assert.Nil(t, cache.Get("test-flag"))
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	cache.Set("flag1", FlagState{Key: "flag1"}, 5*time.Minute)
	cache.Set("flag2", FlagState{Key: "flag2"}, 5*time.Minute)

	cache.Clear()

	assert.Nil(t, cache.Get("flag1"))
	assert.Nil(t, cache.Get("flag2"))
}

func TestCacheSetMany(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	flags := []FlagState{
		{Key: "flag1", Value: true},
		{Key: "flag2", Value: "test"},
		{Key: "flag3", Value: 42.0},
	}

	cache.SetMany(flags, 5*time.Minute)

	assert.NotNil(t, cache.Get("flag1"))
	assert.NotNil(t, cache.Get("flag2"))
	assert.NotNil(t, cache.Get("flag3"))
}

func TestCacheGetAllKeys(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	cache.Set("flag1", FlagState{Key: "flag1"}, 5*time.Minute)
	cache.Set("flag2", FlagState{Key: "flag2"}, 5*time.Minute)

	keys := cache.GetAllKeys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "flag1")
	assert.Contains(t, keys, "flag2")
}

func TestCacheStats(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	cache.Set("flag1", FlagState{Key: "flag1"}, 5*time.Minute)
	cache.Set("flag2", FlagState{Key: "flag2"}, 5*time.Minute)

	stats := cache.Stats()
	assert.Equal(t, 2, stats["size"])
	assert.Equal(t, 2, stats["valid_count"])
	assert.Equal(t, 0, stats["stale_count"])
}

func TestCacheSize(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 100,
		Logger:  &NullLogger{},
	})

	cache.Set("flag1", FlagState{Key: "flag1"}, 5*time.Minute)
	cache.Set("flag2", FlagState{Key: "flag2"}, 5*time.Minute)

	assert.Equal(t, 2, cache.Size())
}

func TestCacheEviction(t *testing.T) {
	cache := NewCache(&CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 2,
		Logger:  &NullLogger{},
	})

	cache.Set("flag1", FlagState{Key: "flag1"}, 5*time.Minute)
	cache.Set("flag2", FlagState{Key: "flag2"}, 5*time.Minute)

	// Add a third item, should evict oldest
	cache.Set("flag3", FlagState{Key: "flag3"}, 5*time.Minute)

	// Should only have 2 entries
	assert.Equal(t, 2, cache.Size())
}
