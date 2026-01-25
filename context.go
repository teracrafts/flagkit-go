package flagkit

// EvaluationContext contains user and environment information for flag evaluation.
type EvaluationContext struct {
	UserID            string                 `json:"userId,omitempty"`
	Email             string                 `json:"email,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Anonymous         bool                   `json:"anonymous,omitempty"`
	Country           string                 `json:"country,omitempty"`
	DeviceType        string                 `json:"deviceType,omitempty"`
	OS                string                 `json:"os,omitempty"`
	Browser           string                 `json:"browser,omitempty"`
	Custom            map[string]any `json:"custom,omitempty"`
	PrivateAttributes []string               `json:"privateAttributes,omitempty"`
}

// NewContext creates a new EvaluationContext with the given user ID.
func NewContext(userID string) *EvaluationContext {
	return &EvaluationContext{
		UserID: userID,
		Custom: make(map[string]any),
	}
}

// NewAnonymousContext creates a new anonymous EvaluationContext.
func NewAnonymousContext() *EvaluationContext {
	return &EvaluationContext{
		Anonymous: true,
		Custom:    make(map[string]any),
	}
}

// WithEmail sets the email and returns the context.
func (c *EvaluationContext) WithEmail(email string) *EvaluationContext {
	c.Email = email
	return c
}

// WithName sets the name and returns the context.
func (c *EvaluationContext) WithName(name string) *EvaluationContext {
	c.Name = name
	return c
}

// WithCountry sets the country and returns the context.
func (c *EvaluationContext) WithCountry(country string) *EvaluationContext {
	c.Country = country
	return c
}

// WithDeviceType sets the device type and returns the context.
func (c *EvaluationContext) WithDeviceType(deviceType string) *EvaluationContext {
	c.DeviceType = deviceType
	return c
}

// WithOS sets the OS and returns the context.
func (c *EvaluationContext) WithOS(os string) *EvaluationContext {
	c.OS = os
	return c
}

// WithBrowser sets the browser and returns the context.
func (c *EvaluationContext) WithBrowser(browser string) *EvaluationContext {
	c.Browser = browser
	return c
}

// WithCustom sets a custom attribute and returns the context.
func (c *EvaluationContext) WithCustom(key string, value any) *EvaluationContext {
	if c.Custom == nil {
		c.Custom = make(map[string]any)
	}
	c.Custom[key] = value
	return c
}

// WithPrivateAttribute marks an attribute as private.
func (c *EvaluationContext) WithPrivateAttribute(attr string) *EvaluationContext {
	c.PrivateAttributes = append(c.PrivateAttributes, attr)
	return c
}

// Merge merges another context into this one.
// Values from the other context take precedence.
func (c *EvaluationContext) Merge(other *EvaluationContext) *EvaluationContext {
	if other == nil {
		return c
	}

	merged := &EvaluationContext{
		UserID:            c.UserID,
		Email:             c.Email,
		Name:              c.Name,
		Anonymous:         c.Anonymous,
		Country:           c.Country,
		DeviceType:        c.DeviceType,
		OS:                c.OS,
		Browser:           c.Browser,
		Custom:            make(map[string]any),
		PrivateAttributes: make([]string, 0),
	}

	// Copy custom from base
	for k, v := range c.Custom {
		merged.Custom[k] = v
	}

	// Override with other's values
	if other.UserID != "" {
		merged.UserID = other.UserID
	}
	if other.Email != "" {
		merged.Email = other.Email
	}
	if other.Name != "" {
		merged.Name = other.Name
	}
	if other.Country != "" {
		merged.Country = other.Country
	}
	if other.DeviceType != "" {
		merged.DeviceType = other.DeviceType
	}
	if other.OS != "" {
		merged.OS = other.OS
	}
	if other.Browser != "" {
		merged.Browser = other.Browser
	}
	if other.Anonymous {
		merged.Anonymous = other.Anonymous
	}

	// Merge custom
	for k, v := range other.Custom {
		merged.Custom[k] = v
	}

	// Merge private attributes
	privateSet := make(map[string]bool)
	for _, attr := range c.PrivateAttributes {
		privateSet[attr] = true
	}
	for _, attr := range other.PrivateAttributes {
		privateSet[attr] = true
	}
	for attr := range privateSet {
		merged.PrivateAttributes = append(merged.PrivateAttributes, attr)
	}

	return merged
}

// StripPrivateAttributes returns a copy of the context with private attributes removed.
func (c *EvaluationContext) StripPrivateAttributes() *EvaluationContext {
	stripped := &EvaluationContext{
		UserID:    c.UserID,
		Anonymous: c.Anonymous,
		Custom:    make(map[string]any),
	}

	privateSet := make(map[string]bool)
	for _, attr := range c.PrivateAttributes {
		privateSet[attr] = true
	}

	if !privateSet["email"] {
		stripped.Email = c.Email
	}
	if !privateSet["name"] {
		stripped.Name = c.Name
	}
	if !privateSet["country"] {
		stripped.Country = c.Country
	}
	if !privateSet["deviceType"] {
		stripped.DeviceType = c.DeviceType
	}
	if !privateSet["os"] {
		stripped.OS = c.OS
	}
	if !privateSet["browser"] {
		stripped.Browser = c.Browser
	}

	for k, v := range c.Custom {
		if !privateSet[k] {
			stripped.Custom[k] = v
		}
	}

	return stripped
}

// Copy creates a deep copy of the context.
func (c *EvaluationContext) Copy() *EvaluationContext {
	result := &EvaluationContext{
		UserID:            c.UserID,
		Email:             c.Email,
		Name:              c.Name,
		Anonymous:         c.Anonymous,
		Country:           c.Country,
		DeviceType:        c.DeviceType,
		OS:                c.OS,
		Browser:           c.Browser,
		Custom:            make(map[string]any),
		PrivateAttributes: make([]string, len(c.PrivateAttributes)),
	}

	for k, v := range c.Custom {
		result.Custom[k] = v
	}
	copy(result.PrivateAttributes, c.PrivateAttributes)

	return result
}

// ToMap converts the context to a map for serialization.
func (c *EvaluationContext) ToMap() map[string]any {
	m := make(map[string]any)

	if c.UserID != "" {
		m["userId"] = c.UserID
	}
	if c.Email != "" {
		m["email"] = c.Email
	}
	if c.Name != "" {
		m["name"] = c.Name
	}
	if c.Anonymous {
		m["anonymous"] = c.Anonymous
	}
	if c.Country != "" {
		m["country"] = c.Country
	}
	if len(c.Custom) > 0 {
		m["custom"] = c.Custom
	}

	return m
}
