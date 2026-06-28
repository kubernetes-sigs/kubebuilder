/*
Copyright 2025 The Kubernetes Authors.

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

package external

import (
	"fmt"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var _ plugin.DeleteSubcommand = &deleteSubcommand{}

type deleteSubcommand struct {
	Path           string
	Args           []string
	pluginChain    []string
	config         config.Config
	supportsDelete bool
}

func (p *deleteSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	if c != nil {
		if chain := c.GetPluginChain(); len(chain) > 0 {
			p.pluginChain = append([]string(nil), chain...)
		}
	}
	return nil
}

func (p *deleteSubcommand) SetPluginChain(chain []string) {
	if len(chain) == 0 {
		p.pluginChain = nil
		return
	}
	p.pluginChain = append([]string(nil), chain...)
}

func (p *deleteSubcommand) UpdateMetadata(_ plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	setExternalPluginMetadata("delete", p.Path, subcmdMeta)
}

func (p *deleteSubcommand) BindFlags(fs *pflag.FlagSet) {
	bindExternalPluginFlags(fs, "delete", p.Path, p.Args)
}

// Scaffold sends Command:"delete" to the external plugin binary.
// Returns an error when PSupportsDelete is false so users get a clear message instead of a cryptic
// "unknown subcommand" from the binary.
func (p *deleteSubcommand) Scaffold(fs machinery.Filesystem) error {
	if !p.supportsDelete {
		return fmt.Errorf(
			"external plugin at %q does not declare delete support; "+
				"set PSupportsDelete:true on the Plugin struct and implement Command:\"delete\" in the binary",
			p.Path,
		)
	}

	req := external.PluginRequest{
		APIVersion:  defaultAPIVersion,
		Command:     "delete",
		Args:        p.Args,
		PluginChain: p.pluginChain,
	}
	return handlePluginResponse(fs, req, p.Path, p.config)
}
