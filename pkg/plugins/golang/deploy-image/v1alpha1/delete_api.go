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
}

func (p *deleteAPISubcommand) UpdateMetadata(_ plugin.CLIMetadata, _ *plugin.SubcommandMetadata) {
	// This plugin works in chain with go/v4 - go/v4 provides the user-facing metadata
}

func (p *deleteAPISubcommand) BindFlags(_ *pflag.FlagSet) {
	// No additional flags - go/v4 provides --skip-confirmation and resource flags
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
	// This plugin ONLY cleans up deploy-image-specific metadata from PROJECT file.
	// File deletion (types.go, controller.go, etc.) is handled by go/v4 plugin.
	// Both plugins run when user specifies: --plugins=deploy-image/v1-alpha
	// (go/v4 is automatically included from the layout)

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

	cfg.Resources = newResources

	// Update the plugin config in PROJECT file
	if err := p.config.EncodePluginConfig(canonicalKey, cfg); err != nil {
		return fmt.Errorf("error encoding plugin configuration: %w", err)
	}

	return nil
}
