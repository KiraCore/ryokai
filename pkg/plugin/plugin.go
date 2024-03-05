package plugin

import (
	"context"
	"log"

	"github.com/KiraCore/ryokai/internal/core/orchestration"
)

type Plugin interface {
	Initialize(orchestrator orchestration.Orchestrator) error
	Execute(ctx context.Context) error
	Cleanup(ctx context.Context) error
}

type PluginRegistry struct {
	plugins map[string]Plugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (r *PluginRegistry) RegisterPlugin(name string, plugin Plugin) {
	r.plugins[name] = plugin
}

func (r *PluginRegistry) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := r.plugins[name]
	return plugin, exists
}

func initializePlugins(orchestrator orchestration.Orchestrator, registry *PluginRegistry) {
	for name, plugin := range registry.plugins {
		err := plugin.Initialize(orchestrator)
		if err != nil {
			log.Printf("Failed to initialize plugin %s: %v", name, err)
		}
	}
}
