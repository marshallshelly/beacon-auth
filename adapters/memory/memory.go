package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/marshallshelly/beaconauth/core"
)

// MemoryAdapter is an in-memory adapter for testing
type MemoryAdapter struct {
	mu     sync.RWMutex
	data   map[string][]map[string]interface{} // model -> records
	nextID int
}

// New creates a new memory adapter
func New() *MemoryAdapter {
	return &MemoryAdapter{
		data:   make(map[string][]map[string]interface{}),
		nextID: 1,
	}
}

// ID returns the adapter identifier
func (m *MemoryAdapter) ID() string {
	return "memory"
}

// Create creates a new record
func (m *MemoryAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Copy data to avoid mutations
	record := copyMap(data)

	// Ensure model exists
	if m.data[model] == nil {
		m.data[model] = make([]map[string]interface{}, 0)
	}

	m.data[model] = append(m.data[model], record)

	return copyMap(record), nil
}

// FindOne finds a single record matching the query
func (m *MemoryAdapter) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, ok := m.data[query.Model]
	if !ok {
		return nil, nil
	}

	for _, record := range records {
		if m.matchesWhere(record, query.Where) {
			return copyMap(record), nil
		}
	}

	return nil, nil
}

// FindMany finds all records matching the query
func (m *MemoryAdapter) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, ok := m.data[query.Model]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}
	for _, record := range records {
		if m.matchesWhere(record, query.Where) {
			results = append(results, copyMap(record))
		}
	}

	// Apply ordering
	if len(query.OrderBy) > 0 {
		sort.Slice(results, func(i, j int) bool {
			for _, order := range query.OrderBy {
				valI := results[i][order.Field]
				valJ := results[j][order.Field]

				cmp := compareValues(valI, valJ)
				if cmp == 0 {
					continue
				}

				if order.Desc {
					return cmp > 0
				}
				return cmp < 0
			}
			return false
		})
	}

	// Apply limit and offset
	if query.Offset > 0 {
		if query.Offset >= len(results) {
			results = []map[string]interface{}{}
		} else {
			results = results[query.Offset:]
		}
	}

	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// Update updates a single record matching the query
func (m *MemoryAdapter) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, ok := m.data[query.Model]
	if !ok {
		return nil, nil
	}

	for i, record := range records {
		if m.matchesWhere(record, query.Where) {
			// Update fields
			for k, v := range data {
				record[k] = v
			}
			m.data[query.Model][i] = record
			return copyMap(record), nil
		}
	}

	return nil, nil
}

// UpdateMany updates all records matching the query
func (m *MemoryAdapter) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, ok := m.data[query.Model]
	if !ok {
		return 0, nil
	}

	var count int64
	for i, record := range records {
		if m.matchesWhere(record, query.Where) {
			for k, v := range data {
				record[k] = v
			}
			m.data[query.Model][i] = record
			count++
		}
	}

	return count, nil
}

// Delete deletes a single record matching the query
func (m *MemoryAdapter) Delete(ctx context.Context, query *core.Query) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, ok := m.data[query.Model]
	if !ok {
		return nil
	}

	for i, record := range records {
		if m.matchesWhere(record, query.Where) {
			// Remove by creating new slice without this element
			m.data[query.Model] = append(records[:i], records[i+1:]...)
			return nil
		}
	}

	return nil
}

// DeleteMany deletes all records matching the query
func (m *MemoryAdapter) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	records, ok := m.data[query.Model]
	if !ok {
		return 0, nil
	}

	var newRecords []map[string]interface{}
	var count int64

	for _, record := range records {
		if !m.matchesWhere(record, query.Where) {
			newRecords = append(newRecords, record)
		} else {
			count++
		}
	}

	m.data[query.Model] = newRecords
	return count, nil
}

// Count counts records matching the query
func (m *MemoryAdapter) Count(ctx context.Context, query *core.Query) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records, ok := m.data[query.Model]
	if !ok {
		return 0, nil
	}

	var count int64
	for _, record := range records {
		if m.matchesWhere(record, query.Where) {
			count++
		}
	}

	return count, nil
}

// Transaction executes a function in a transaction (no-op for memory adapter)
func (m *MemoryAdapter) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	// For memory adapter, just execute the function
	// Real databases would handle rollback on error
	return fn(m)
}

// Ping checks the connection (always succeeds for memory adapter)
func (m *MemoryAdapter) Ping(ctx context.Context) error {
	return nil
}

// Close closes the adapter (no-op for memory adapter)
func (m *MemoryAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string][]map[string]interface{})
	return nil
}

// matchesWhere checks if a record matches the where clauses
func (m *MemoryAdapter) matchesWhere(record map[string]interface{}, where []core.WhereClause) bool {
	if len(where) == 0 {
		return true
	}

	for _, clause := range where {
		value, exists := record[clause.Field]
		if !exists {
			return false
		}

		if !m.matchesClause(value, clause.Operator, clause.Value) {
			return false
		}
	}

	return true
}

// matchesClause checks if a value matches a where clause
func (m *MemoryAdapter) matchesClause(value interface{}, op core.Operator, clauseValue interface{}) bool {
	switch op {
	case core.OpEqual:
		return equal(value, clauseValue)

	case core.OpNotEqual:
		return !equal(value, clauseValue)

	case core.OpGreaterThan:
		return greaterThan(value, clauseValue)

	case core.OpGreaterOrEqual:
		return greaterThan(value, clauseValue) || equal(value, clauseValue)

	case core.OpLessThan:
		return lessThan(value, clauseValue)

	case core.OpLessOrEqual:
		return lessThan(value, clauseValue) || equal(value, clauseValue)

	case core.OpLike:
		str, ok := value.(string)
		if !ok {
			return false
		}
		pattern, ok := clauseValue.(string)
		if !ok {
			return false
		}
		// Simple LIKE implementation: % is wildcard
		pattern = strings.ReplaceAll(pattern, "%", ".*")
		return strings.Contains(str, strings.Trim(pattern, ".*"))

	case core.OpIn:
		list, ok := clauseValue.([]interface{})
		if !ok {
			return false
		}
		for _, item := range list {
			if equal(value, item) {
				return true
			}
		}
		return false

	case core.OpNotIn:
		list, ok := clauseValue.([]interface{})
		if !ok {
			return false
		}
		for _, item := range list {
			if equal(value, item) {
				return false
			}
		}
		return true

	case core.OpIsNull:
		return value == nil

	case core.OpIsNotNull:
		return value != nil

	default:
		return false
	}
}

// Helper comparison functions

func equal(a, b interface{}) bool {
	// Handle time.Time comparison
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			return ta.Equal(tb)
		}
	}

	return fmt.Sprint(a) == fmt.Sprint(b)
}

func greaterThan(a, b interface{}) bool {
	// Handle time.Time comparison
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			return ta.After(tb)
		}
	}

	// Handle numeric comparison
	if na, ok := toFloat64(a); ok {
		if nb, ok := toFloat64(b); ok {
			return na > nb
		}
	}

	// String comparison
	return fmt.Sprint(a) > fmt.Sprint(b)
}

func lessThan(a, b interface{}) bool {
	// Handle time.Time comparison
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			return ta.Before(tb)
		}
	}

	// Handle numeric comparison
	if na, ok := toFloat64(a); ok {
		if nb, ok := toFloat64(b); ok {
			return na < nb
		}
	}

	// String comparison
	return fmt.Sprint(a) < fmt.Sprint(b)
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// compareValues compares two values for sorting
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareValues(a, b interface{}) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Handle time.Time comparison
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			if ta.Before(tb) {
				return -1
			}
			if ta.After(tb) {
				return 1
			}
			return 0
		}
	}

	// Handle numeric comparison
	if na, ok := toFloat64(a); ok {
		if nb, ok := toFloat64(b); ok {
			if na < nb {
				return -1
			}
			if na > nb {
				return 1
			}
			return 0
		}
	}

	// String comparison
	sa := fmt.Sprint(a)
	sb := fmt.Sprint(b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}
