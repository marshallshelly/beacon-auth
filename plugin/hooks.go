package plugin

import (
	"context"
	"fmt"
)

// HookRegistry manages lifecycle hooks from plugins
type HookRegistry struct {
	beforeHooks map[string][]Hook // plugin ID -> hooks
	afterHooks  map[string][]Hook // plugin ID -> hooks
}

// NewHookRegistry creates a new hook registry
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		beforeHooks: make(map[string][]Hook),
		afterHooks:  make(map[string][]Hook),
	}
}

// Register registers hooks from a plugin
func (r *HookRegistry) Register(pluginID string, config *HookConfig) {
	if config == nil {
		return
	}

	if len(config.Before) > 0 {
		r.beforeHooks[pluginID] = append(r.beforeHooks[pluginID], config.Before...)
	}

	if len(config.After) > 0 {
		r.afterHooks[pluginID] = append(r.afterHooks[pluginID], config.After...)
	}
}

// ExecuteBefore executes all matching before hooks
func (r *HookRegistry) ExecuteBefore(ctx context.Context, path, method string, data interface{}) error {
	for pluginID, hooks := range r.beforeHooks {
		for i, hook := range hooks {
			if hook.Matcher(path, method) {
				if err := hook.Handler(ctx, data); err != nil {
					return fmt.Errorf("before hook %s[%d] failed: %w", pluginID, i, err)
				}
			}
		}
	}
	return nil
}

// ExecuteAfter executes all matching after hooks
func (r *HookRegistry) ExecuteAfter(ctx context.Context, path, method string, data interface{}) error {
	for pluginID, hooks := range r.afterHooks {
		for i, hook := range hooks {
			if hook.Matcher(path, method) {
				if err := hook.Handler(ctx, data); err != nil {
					return fmt.Errorf("after hook %s[%d] failed: %w", pluginID, i, err)
				}
			}
		}
	}
	return nil
}

// GetBeforeHooks returns all before hooks for a plugin
func (r *HookRegistry) GetBeforeHooks(pluginID string) []Hook {
	return r.beforeHooks[pluginID]
}

// GetAfterHooks returns all after hooks for a plugin
func (r *HookRegistry) GetAfterHooks(pluginID string) []Hook {
	return r.afterHooks[pluginID]
}

// HasBeforeHooks checks if there are any before hooks
func (r *HookRegistry) HasBeforeHooks() bool {
	return len(r.beforeHooks) > 0
}

// HasAfterHooks checks if there are any after hooks
func (r *HookRegistry) HasAfterHooks() bool {
	return len(r.afterHooks) > 0
}

// MatchAll creates a matcher that matches all requests
func MatchAll() func(path, method string) bool {
	return func(path, method string) bool {
		return true
	}
}

// MatchPath creates a matcher that matches a specific path
func MatchPath(targetPath string) func(path, method string) bool {
	return func(path, method string) bool {
		return path == targetPath
	}
}

// MatchMethod creates a matcher that matches a specific HTTP method
func MatchMethod(targetMethod string) func(path, method string) bool {
	return func(path, method string) bool {
		return method == targetMethod
	}
}

// MatchPathAndMethod creates a matcher that matches both path and method
func MatchPathAndMethod(targetPath, targetMethod string) func(path, method string) bool {
	return func(path, method string) bool {
		return path == targetPath && method == targetMethod
	}
}
