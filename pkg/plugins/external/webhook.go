/*
Copyright 2021 The Kubernetes Authors.

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
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

var _ plugin.CreateWebhookSubcommand = &createWebhookSubcommand{}

type createWebhookSubcommand struct {
	Path string
	Args []string
}

func (p *createWebhookSubcommand) InjectResource(*resource.Resource) error {
	// Do nothing since resource flags are passed to the external plugin directly.
	return nil
}

func (p *createWebhookSubcommand) UpdateMetadata(_ plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	setExternalPluginMetadata("webhook", p.Path, subcmdMeta)
}

func (p *createWebhookSubcommand) BindFlags(fs *pflag.FlagSet) {
	bindExternalPluginFlags(fs, "webhook", p.Path, p.Args)
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	req := external.PluginRequest{
		APIVersion: defaultAPIVersion,
		Command:    "create webhook",
		Args:       p.Args,
	}

	err := handlePluginResponse(fs, req, p.Path)
	if err != nil {
		return err
	}

	return nil
}
