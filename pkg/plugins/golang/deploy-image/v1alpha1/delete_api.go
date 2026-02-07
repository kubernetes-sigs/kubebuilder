/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ plugin.DeleteAPISubcommand = &deleteAPISubcommand{}

type deleteAPISubcommand struct {
	config   config.Config
	resource *resource.Resource

	// assumeYes skips confirmation - bound for consistency when used in chain
	assumeYes bool
}

func (p *deleteAPISubcommand) UpdateMetadata(_ plugin.CLIMetadata, _ *plugin.SubcommandMetadata) {
	// This plugin works in chain with go/v4 - go/v4 provides the user-facing metadata
}

func (p *deleteAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	// Check if flag already exists (when used in chain with go/v4)
	if fs.Lookup("yes") == nil {
		fs.BoolVarP(&p.assumeYes, "yes", "y", false,
			"proceed without prompting for confirmation")
	}
}

func (p *deleteAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *deleteAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	return nil
}

func (p *deleteAPISubcommand) Scaffold(_ machinery.Filesystem) error {
	canonicalKey := plugin.KeyFor(Plugin{})
	cfg := PluginConfig{}

	if err := p.config.DecodePluginConfig(canonicalKey, &cfg); err != nil {
		// Plugin config doesn't exist or unsupported - nothing to clean up
		var notFoundErr config.PluginKeyNotFoundError
		var unsupportedErr config.UnsupportedFieldError
		if errors.As(err, &notFoundErr) || errors.As(err, &unsupportedErr) {
			return nil
		}
		return fmt.Errorf("error decoding plugin configuration: %w", err)
	}

	// Remove the resource from the plugin config
	newResources := []ResourceData{}
	for _, res := range cfg.Resources {
		if res.Group == p.resource.Group && res.Version == p.resource.Version && res.Kind == p.resource.Kind {
			continue // Skip the resource being deleted
		}
		newResources = append(newResources, res)
	}

	// If no resources remain, remove the entire plugin config
	// Otherwise update with the remaining resources
	if len(newResources) == 0 {
		// Remove plugin config entirely by encoding empty struct
		if err := p.config.EncodePluginConfig(canonicalKey, struct{}{}); err != nil {
			return fmt.Errorf("error removing plugin configuration: %w", err)
		}
	} else {
		cfg.Resources = newResources
		if err := p.config.EncodePluginConfig(canonicalKey, cfg); err != nil {
			return fmt.Errorf("error encoding plugin configuration: %w", err)
		}
	}

	return nil
}
