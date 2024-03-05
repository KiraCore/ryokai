package plugin

import (
	"context"
	"fmt"

	"github.com/KiraCore/ryokai/internal/core/orchestration"
)

type (
	Plugin interface {
		Initialize(ctx context.Context, orchestrator orchestration.Orchestrator) error
		Execute(ctx context.Context, orchestrator orchestration.Orchestrator) error
		Cleanup(ctx context.Context, orchestrator orchestration.Orchestrator) error
	}
	PluginRegistry struct {
		Plugins map[string]Plugin
	}
)

func NewPluginRegistry() *PluginRegistry {
	// TODO: read from file?
	return &PluginRegistry{
		Plugins: make(map[string]Plugin),
	}
}

func (r *PluginRegistry) RegisterPlugin(name string, plugin Plugin) {
	r.Plugins[name] = plugin
}

func (r *PluginRegistry) GetPlugin(name string) (Plugin, bool) {
	// TODO: From where we will get plugins? Read from state file?
	plugin, exists := r.Plugins[name]

	return plugin, exists
}

func initializePlugins(ctx context.Context, orchestrator orchestration.Orchestrator, registry *PluginRegistry) error {
	for name, plugin := range registry.Plugins {
		err := plugin.Initialize(ctx, orchestrator)
		if err != nil {
			return fmt.Errorf("error when initializing <%s> plugin: %w", name, err)
		}
	}

	return nil
}
