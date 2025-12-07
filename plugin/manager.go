package plugin

import (
	"fmt"
	"sort"

	"github.com/marshallshelly/beacon-auth/core"
)

// Manager manages the lifecycle of plugins
type Manager struct {
	plugins    []Plugin
	endpoints  map[string]Endpoint
	hooks      *HookRegistry
	middleware []MiddlewareConfig
}

// NewManager creates a new plugin manager
func NewManager(plugins []Plugin) *Manager {
	return &Manager{
		plugins:    plugins,
		endpoints:  make(map[string]Endpoint),
		hooks:      NewHookRegistry(),
		middleware: make([]MiddlewareConfig, 0),
	}
}

// Initialize initializes all plugins
func (m *Manager) Initialize(ctx *core.AuthContext) error {
	for _, p := range m.plugins {
		if err := p.Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", p.ID(), err)
		}

		// Register endpoints
		for path, endpoint := range p.Endpoints() {
			fullPath := path
			if _, exists := m.endpoints[fullPath]; exists {
				return fmt.Errorf("endpoint conflict: %s already registered", fullPath)
			}
			m.endpoints[fullPath] = endpoint
		}

		// Register hooks
		if hooks := p.Hooks(); hooks != nil {
			m.hooks.Register(p.ID(), hooks)
		}

		// Register middleware
		middleware := p.Middleware()
		if len(middleware) > 0 {
			m.middleware = append(m.middleware, middleware...)
		}
	}

	// Sort middleware by priority (lower priority runs first)
	sort.Slice(m.middleware, func(i, j int) bool {
		return m.middleware[i].Priority < m.middleware[j].Priority
	})

	return nil
}

// GetEndpoints returns all registered endpoints
func (m *Manager) GetEndpoints() map[string]Endpoint {
	return m.endpoints
}

// GetHooks returns the hook registry
func (m *Manager) GetHooks() *HookRegistry {
	return m.hooks
}

// GetMiddleware returns all registered middleware sorted by priority
func (m *Manager) GetMiddleware() []MiddlewareConfig {
	return m.middleware
}

// GetPlugin returns a plugin by ID
func (m *Manager) GetPlugin(id string) Plugin {
	for _, p := range m.plugins {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// HasPlugin checks if a plugin is registered
func (m *Manager) HasPlugin(id string) bool {
	return m.GetPlugin(id) != nil
}
