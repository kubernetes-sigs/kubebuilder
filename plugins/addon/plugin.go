package addon

import (
	"sigs.k8s.io/kubebuilder/pkg/model"
)

// Plugin implements model.Plugin
type Plugin struct {
}

// Pipe implements model.Plugin
func (p *Plugin) Pipe(u *model.Universe) error {
	functions := []PluginFunc{
		ExampleManifest,
		ExampleChannel,
		ReplaceController,
		ReplaceTypes,
	}

	for _, fn := range functions {
		if err := fn(u); err != nil {
			return err
		}

	}

	return nil
}
