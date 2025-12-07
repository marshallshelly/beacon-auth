package plugin

import (
	"context"
	"net/http"

	"github.com/marshallshelly/beacon-auth/core"
)

// Plugin defines the interface for BeaconAuth plugins
type Plugin interface {
	// ID returns the unique plugin identifier
	ID() string

	// Init initializes the plugin with the auth context
	Init(ctx *core.AuthContext) error

	// Endpoints returns HTTP endpoints added by this plugin
	Endpoints() map[string]core.Endpoint

	// Hooks returns lifecycle hooks
	Hooks() *HookConfig

	// Middleware returns middleware to be injected
	Middleware() []MiddlewareConfig
}

// Endpoint represents an HTTP endpoint
type Endpoint = core.Endpoint

// EndpointOptions holds endpoint configuration
type EndpointOptions struct {
	RequireAuth bool
	Middleware  []func(http.Handler) http.Handler
}

// HookConfig defines lifecycle hooks
type HookConfig struct {
	Before []Hook
	After  []Hook
}

// Hook is a function that runs before or after a request
type Hook struct {
	Matcher func(path string, method string) bool
	Handler func(ctx context.Context, data interface{}) error
}

// MiddlewareConfig defines middleware to inject
type MiddlewareConfig struct {
	Path       string
	Middleware func(http.Handler) http.Handler
	Priority   int // Lower numbers run first
}

// BasePlugin provides a base implementation for plugins
type BasePlugin struct {
	id string
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(id string) *BasePlugin {
	return &BasePlugin{id: id}
}

// ID returns the plugin identifier
func (p *BasePlugin) ID() string {
	return p.id
}

// Init provides default initialization (no-op)
func (p *BasePlugin) Init(ctx *core.AuthContext) error {
	return nil
}

// Endpoints returns no endpoints by default
func (p *BasePlugin) Endpoints() map[string]Endpoint {
	return nil
}

// Hooks returns no hooks by default
func (p *BasePlugin) Hooks() *HookConfig {
	return nil
}

// Middleware returns no middleware by default
func (p *BasePlugin) Middleware() []MiddlewareConfig {
	return nil
}
