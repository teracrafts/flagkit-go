package flagkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext("user-123")

	assert.Equal(t, "user-123", ctx.UserID)
	assert.False(t, ctx.Anonymous)
	assert.NotNil(t, ctx.Custom)
}

func TestNewAnonymousContext(t *testing.T) {
	ctx := NewAnonymousContext()

	assert.True(t, ctx.Anonymous)
	assert.NotNil(t, ctx.Custom)
}

func TestContextBuilderMethods(t *testing.T) {
	ctx := NewContext("user-123").
		WithEmail("user@example.com").
		WithName("Test User").
		WithCountry("US").
		WithDeviceType("mobile").
		WithOS("iOS").
		WithBrowser("Safari").
		WithCustom("plan", "premium").
		WithPrivateAttribute("email")

	assert.Equal(t, "user@example.com", ctx.Email)
	assert.Equal(t, "Test User", ctx.Name)
	assert.Equal(t, "US", ctx.Country)
	assert.Equal(t, "mobile", ctx.DeviceType)
	assert.Equal(t, "iOS", ctx.OS)
	assert.Equal(t, "Safari", ctx.Browser)
	assert.Equal(t, "premium", ctx.Custom["plan"])
	assert.Contains(t, ctx.PrivateAttributes, "email")
}

func TestContextMerge(t *testing.T) {
	ctx1 := NewContext("user-123").
		WithEmail("old@example.com").
		WithCustom("key1", "value1")

	ctx2 := NewContext("user-456").
		WithEmail("new@example.com").
		WithName("New Name").
		WithCustom("key2", "value2")

	merged := ctx1.Merge(ctx2)

	assert.Equal(t, "user-456", merged.UserID)
	assert.Equal(t, "new@example.com", merged.Email)
	assert.Equal(t, "New Name", merged.Name)
	assert.Equal(t, "value1", merged.Custom["key1"])
	assert.Equal(t, "value2", merged.Custom["key2"])
}

func TestContextMergeNil(t *testing.T) {
	ctx := NewContext("user-123").WithEmail("user@example.com")
	merged := ctx.Merge(nil)

	assert.Equal(t, ctx, merged)
}

func TestContextStripPrivateAttributes(t *testing.T) {
	ctx := NewContext("user-123").
		WithEmail("user@example.com").
		WithName("Test User").
		WithCustom("secret", "value").
		WithPrivateAttribute("email").
		WithPrivateAttribute("secret")

	stripped := ctx.StripPrivateAttributes()

	assert.Equal(t, "user-123", stripped.UserID)
	assert.Empty(t, stripped.Email)
	assert.Equal(t, "Test User", stripped.Name)
	_, hasSecret := stripped.Custom["secret"]
	assert.False(t, hasSecret)
}

func TestContextToMap(t *testing.T) {
	ctx := NewContext("user-123").
		WithEmail("user@example.com").
		WithCustom("plan", "premium")

	m := ctx.ToMap()

	assert.Equal(t, "user-123", m["userId"])
	assert.Equal(t, "user@example.com", m["email"])

	custom := m["custom"].(map[string]interface{})
	assert.Equal(t, "premium", custom["plan"])
}

func TestContextToMapOmitsEmpty(t *testing.T) {
	ctx := NewContext("user-123")

	m := ctx.ToMap()

	_, hasEmail := m["email"]
	assert.False(t, hasEmail)

	_, hasAnonymous := m["anonymous"]
	assert.False(t, hasAnonymous)
}

func TestContextCopy(t *testing.T) {
	ctx := NewContext("user-123").
		WithEmail("user@example.com").
		WithCustom("key", "value")

	copied := ctx.Copy()

	assert.Equal(t, ctx.UserID, copied.UserID)
	assert.Equal(t, ctx.Email, copied.Email)
	assert.Equal(t, ctx.Custom["key"], copied.Custom["key"])

	// Ensure it's a deep copy
	copied.Custom["key"] = "modified"
	assert.NotEqual(t, ctx.Custom["key"], copied.Custom["key"])
}
