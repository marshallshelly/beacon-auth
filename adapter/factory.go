package adapter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/marshallshelly/beaconauth/core"
)

// AdapterConfig defines adapter capabilities and transformations
type AdapterConfig struct {
	ID                   string
	SupportsJSON         bool
	SupportsDates        bool
	SupportsBooleans     bool
	SupportsTransaction  bool
	FieldNameMapping     map[string]string // e.g., "id" -> "_id" for MongoDB
	CustomTransformInput func(data map[string]interface{}) map[string]interface{}
	CustomTransformOutput func(data map[string]interface{}) map[string]interface{}
}

// CustomAdapter is implemented by database-specific adapters
type CustomAdapter interface {
	Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error)
	FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error)
	FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error)
	Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error)
	UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error)
	Delete(ctx context.Context, query *core.Query) error
	DeleteMany(ctx context.Context, query *core.Query) (int64, error)
	Count(ctx context.Context, query *core.Query) (int64, error)
	Transaction(ctx context.Context, fn func(core.Adapter) error) error
	Ping(ctx context.Context) error
	Close() error
	ID() string
}

// Factory wraps a custom adapter with transformation logic
type Factory struct {
	config AdapterConfig
	custom CustomAdapter
}

// NewFactory creates a new adapter factory
func NewFactory(config AdapterConfig, custom CustomAdapter) *Factory {
	return &Factory{
		config: config,
		custom: custom,
	}
}

// Create wraps the custom adapter's Create with transformations
func (f *Factory) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	transformed := f.transformInput(data)
	result, err := f.custom.Create(ctx, model, transformed)
	if err != nil {
		return nil, err
	}
	return f.transformOutput(result), nil
}

// FindOne wraps the custom adapter's FindOne with transformations
func (f *Factory) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	result, err := f.custom.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return f.transformOutput(result), nil
}

// FindMany wraps the custom adapter's FindMany with transformations
func (f *Factory) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	results, err := f.custom.FindMany(ctx, query)
	if err != nil {
		return nil, err
	}

	transformed := make([]map[string]interface{}, len(results))
	for i, result := range results {
		transformed[i] = f.transformOutput(result)
	}
	return transformed, nil
}

// Update wraps the custom adapter's Update with transformations
func (f *Factory) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	transformed := f.transformInput(data)
	result, err := f.custom.Update(ctx, query, transformed)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return f.transformOutput(result), nil
}

// UpdateMany wraps the custom adapter's UpdateMany
func (f *Factory) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	transformed := f.transformInput(data)
	return f.custom.UpdateMany(ctx, query, transformed)
}

// Delete wraps the custom adapter's Delete
func (f *Factory) Delete(ctx context.Context, query *core.Query) error {
	return f.custom.Delete(ctx, query)
}

// DeleteMany wraps the custom adapter's DeleteMany
func (f *Factory) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	return f.custom.DeleteMany(ctx, query)
}

// Count wraps the custom adapter's Count
func (f *Factory) Count(ctx context.Context, query *core.Query) (int64, error) {
	return f.custom.Count(ctx, query)
}

// Transaction wraps the custom adapter's Transaction
func (f *Factory) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	if !f.config.SupportsTransaction {
		// Fallback: execute without transaction
		return fn(f)
	}
	return f.custom.Transaction(ctx, fn)
}

// Ping wraps the custom adapter's Ping
func (f *Factory) Ping(ctx context.Context) error {
	return f.custom.Ping(ctx)
}

// Close wraps the custom adapter's Close
func (f *Factory) Close() error {
	return f.custom.Close()
}

// ID wraps the custom adapter's ID
func (f *Factory) ID() string {
	return f.custom.ID()
}

// transformInput applies input transformations to data
func (f *Factory) transformInput(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		// Apply field name mapping
		fieldName := key
		if mapped, ok := f.config.FieldNameMapping[key]; ok {
			fieldName = mapped
		}

		// Apply type transformations
		result[fieldName] = f.transformValue(value, true)
	}

	// Apply custom transform
	if f.config.CustomTransformInput != nil {
		result = f.config.CustomTransformInput(result)
	}

	return result
}

// transformOutput applies output transformations to data
func (f *Factory) transformOutput(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Reverse field name mapping
	reverseMapping := make(map[string]string)
	for k, v := range f.config.FieldNameMapping {
		reverseMapping[v] = k
	}

	for key, value := range data {
		// Apply reverse field name mapping
		fieldName := key
		if mapped, ok := reverseMapping[key]; ok {
			fieldName = mapped
		}

		result[fieldName] = f.transformValue(value, false)
	}

	// Apply custom transform
	if f.config.CustomTransformOutput != nil {
		result = f.config.CustomTransformOutput(result)
	}

	return result
}

// transformValue applies type transformations based on adapter capabilities
func (f *Factory) transformValue(value interface{}, input bool) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		if !f.config.SupportsDates {
			// Convert to string or timestamp
			return v.Format(time.RFC3339)
		}
		return v

	case bool:
		if !f.config.SupportsBooleans {
			// Convert to int
			if v {
				return 1
			}
			return 0
		}
		return v

	case map[string]interface{}:
		if !f.config.SupportsJSON && input {
			// Convert to JSON string
			data, err := json.Marshal(v)
			if err != nil {
				return v
			}
			return string(data)
		}
		return v

	case string:
		// Try to parse JSON back if output and doesn't support JSON natively
		if !input && !f.config.SupportsJSON {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
				return parsed
			}
		}
		return v

	default:
		return v
	}
}
