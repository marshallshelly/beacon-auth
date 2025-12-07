package plugin

import (
	"context"
	"net/http"
	"testing"

	"github.com/marshallshelly/beacon-auth/core"
)

// mockPlugin is a test plugin implementation
type mockPlugin struct {
	*BasePlugin
	initCalled     bool
	endpoints      map[string]Endpoint
	hooks          *HookConfig
	middlewareList []MiddlewareConfig
}

func newMockPlugin(id string) *mockPlugin {
	return &mockPlugin{
		BasePlugin: NewBasePlugin(id),
		endpoints:  make(map[string]Endpoint),
	}
}

func (p *mockPlugin) Init(ctx *core.AuthContext) error {
	p.initCalled = true
	return nil
}

func (p *mockPlugin) Endpoints() map[string]Endpoint {
	return p.endpoints
}

func (p *mockPlugin) Hooks() *HookConfig {
	return p.hooks
}

func (p *mockPlugin) Middleware() []MiddlewareConfig {
	return p.middlewareList
}

func TestBasePlugin(t *testing.T) {
	plugin := NewBasePlugin("test-plugin")

	if plugin.ID() != "test-plugin" {
		t.Errorf("Expected ID 'test-plugin', got '%s'", plugin.ID())
	}

	if err := plugin.Init(nil); err != nil {
		t.Errorf("Expected Init to return nil, got %v", err)
	}

	if endpoints := plugin.Endpoints(); endpoints != nil {
		t.Errorf("Expected no endpoints, got %v", endpoints)
	}

	if hooks := plugin.Hooks(); hooks != nil {
		t.Errorf("Expected no hooks, got %v", hooks)
	}

	if middleware := plugin.Middleware(); middleware != nil {
		t.Errorf("Expected no middleware, got %v", middleware)
	}
}

func TestManager_Initialize(t *testing.T) {
	plugin1 := newMockPlugin("plugin1")
	plugin1.endpoints["/test1"] = Endpoint{
		Path:   "/test1",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	plugin2 := newMockPlugin("plugin2")
	plugin2.endpoints["/test2"] = Endpoint{
		Path:   "/test2",
		Method: "POST",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	manager := NewManager([]Plugin{plugin1, plugin2})

	authCtx := &core.AuthContext{}
	if err := manager.Initialize(authCtx); err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	if !plugin1.initCalled {
		t.Error("Expected plugin1 Init to be called")
	}

	if !plugin2.initCalled {
		t.Error("Expected plugin2 Init to be called")
	}

	endpoints := manager.GetEndpoints()
	if len(endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(endpoints))
	}

	if _, exists := endpoints["/test1"]; !exists {
		t.Error("Expected endpoint /test1 to be registered")
	}

	if _, exists := endpoints["/test2"]; !exists {
		t.Error("Expected endpoint /test2 to be registered")
	}
}

func TestManager_EndpointConflict(t *testing.T) {
	plugin1 := newMockPlugin("plugin1")
	plugin1.endpoints["/conflict"] = Endpoint{
		Path:    "/conflict",
		Method:  "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {},
	}

	plugin2 := newMockPlugin("plugin2")
	plugin2.endpoints["/conflict"] = Endpoint{
		Path:    "/conflict",
		Method:  "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {},
	}

	manager := NewManager([]Plugin{plugin1, plugin2})

	authCtx := &core.AuthContext{}
	err := manager.Initialize(authCtx)
	if err == nil {
		t.Error("Expected error for endpoint conflict")
	}

	if err != nil && err.Error() != "endpoint conflict: /conflict already registered" {
		t.Errorf("Expected endpoint conflict error, got: %v", err)
	}
}

func TestManager_MiddlewarePriority(t *testing.T) {
	plugin1 := newMockPlugin("plugin1")
	plugin1.middlewareList = []MiddlewareConfig{
		{
			Path:     "/",
			Priority: 10,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r)
				})
			},
		},
	}

	plugin2 := newMockPlugin("plugin2")
	plugin2.middlewareList = []MiddlewareConfig{
		{
			Path:     "/",
			Priority: 5,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r)
				})
			},
		},
	}

	manager := NewManager([]Plugin{plugin1, plugin2})

	authCtx := &core.AuthContext{}
	if err := manager.Initialize(authCtx); err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	middleware := manager.GetMiddleware()
	if len(middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(middleware))
	}

	// Middleware should be sorted by priority (5 before 10)
	if middleware[0].Priority != 5 {
		t.Errorf("Expected first middleware priority 5, got %d", middleware[0].Priority)
	}

	if middleware[1].Priority != 10 {
		t.Errorf("Expected second middleware priority 10, got %d", middleware[1].Priority)
	}
}

func TestManager_GetPlugin(t *testing.T) {
	plugin1 := newMockPlugin("plugin1")
	plugin2 := newMockPlugin("plugin2")

	manager := NewManager([]Plugin{plugin1, plugin2})

	if found := manager.GetPlugin("plugin1"); found != plugin1 {
		t.Error("Expected to find plugin1")
	}

	if found := manager.GetPlugin("plugin2"); found != plugin2 {
		t.Error("Expected to find plugin2")
	}

	if found := manager.GetPlugin("nonexistent"); found != nil {
		t.Error("Expected nil for nonexistent plugin")
	}
}

func TestManager_HasPlugin(t *testing.T) {
	plugin1 := newMockPlugin("plugin1")
	manager := NewManager([]Plugin{plugin1})

	if !manager.HasPlugin("plugin1") {
		t.Error("Expected HasPlugin to return true for plugin1")
	}

	if manager.HasPlugin("nonexistent") {
		t.Error("Expected HasPlugin to return false for nonexistent plugin")
	}
}

func TestHookRegistry(t *testing.T) {
	registry := NewHookRegistry()

	beforeCalled := false
	afterCalled := false

	config := &HookConfig{
		Before: []Hook{
			{
				Matcher: MatchPath("/test"),
				Handler: func(ctx context.Context, data interface{}) error {
					beforeCalled = true
					return nil
				},
			},
		},
		After: []Hook{
			{
				Matcher: MatchPath("/test"),
				Handler: func(ctx context.Context, data interface{}) error {
					afterCalled = true
					return nil
				},
			},
		},
	}

	registry.Register("test-plugin", config)

	ctx := context.Background()
	if err := registry.ExecuteBefore(ctx, "/test", "GET", nil); err != nil {
		t.Errorf("ExecuteBefore failed: %v", err)
	}

	if !beforeCalled {
		t.Error("Expected before hook to be called")
	}

	if err := registry.ExecuteAfter(ctx, "/test", "GET", nil); err != nil {
		t.Errorf("ExecuteAfter failed: %v", err)
	}

	if !afterCalled {
		t.Error("Expected after hook to be called")
	}
}

func TestHookRegistry_NoMatch(t *testing.T) {
	registry := NewHookRegistry()

	called := false

	config := &HookConfig{
		Before: []Hook{
			{
				Matcher: MatchPath("/test"),
				Handler: func(ctx context.Context, data interface{}) error {
					called = true
					return nil
				},
			},
		},
	}

	registry.Register("test-plugin", config)

	ctx := context.Background()
	if err := registry.ExecuteBefore(ctx, "/other", "GET", nil); err != nil {
		t.Errorf("ExecuteBefore failed: %v", err)
	}

	if called {
		t.Error("Expected hook not to be called for non-matching path")
	}
}

func TestHookMatchers(t *testing.T) {
	tests := []struct {
		name    string
		matcher func(path, method string) bool
		path    string
		method  string
		match   bool
	}{
		{
			name:    "MatchAll matches everything",
			matcher: MatchAll(),
			path:    "/any",
			method:  "GET",
			match:   true,
		},
		{
			name:    "MatchPath matches path",
			matcher: MatchPath("/test"),
			path:    "/test",
			method:  "GET",
			match:   true,
		},
		{
			name:    "MatchPath doesn't match different path",
			matcher: MatchPath("/test"),
			path:    "/other",
			method:  "GET",
			match:   false,
		},
		{
			name:    "MatchMethod matches method",
			matcher: MatchMethod("POST"),
			path:    "/any",
			method:  "POST",
			match:   true,
		},
		{
			name:    "MatchMethod doesn't match different method",
			matcher: MatchMethod("POST"),
			path:    "/any",
			method:  "GET",
			match:   false,
		},
		{
			name:    "MatchPathAndMethod matches both",
			matcher: MatchPathAndMethod("/test", "POST"),
			path:    "/test",
			method:  "POST",
			match:   true,
		},
		{
			name:    "MatchPathAndMethod doesn't match different path",
			matcher: MatchPathAndMethod("/test", "POST"),
			path:    "/other",
			method:  "POST",
			match:   false,
		},
		{
			name:    "MatchPathAndMethod doesn't match different method",
			matcher: MatchPathAndMethod("/test", "POST"),
			path:    "/test",
			method:  "GET",
			match:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.matcher(tt.path, tt.method)
			if result != tt.match {
				t.Errorf("Expected match=%v, got %v", tt.match, result)
			}
		})
	}
}

func TestHookRegistry_HasHooks(t *testing.T) {
	registry := NewHookRegistry()

	if registry.HasBeforeHooks() {
		t.Error("Expected no before hooks initially")
	}

	if registry.HasAfterHooks() {
		t.Error("Expected no after hooks initially")
	}

	config := &HookConfig{
		Before: []Hook{
			{
				Matcher: MatchAll(),
				Handler: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
	}

	registry.Register("test", config)

	if !registry.HasBeforeHooks() {
		t.Error("Expected to have before hooks")
	}

	if registry.HasAfterHooks() {
		t.Error("Expected no after hooks")
	}
}
