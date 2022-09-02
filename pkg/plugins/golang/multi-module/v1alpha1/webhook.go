/*
Copyright 2022 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

var _ plugin.CreateWebhookSubcommand = &createWebhookSubcommand{}

type createWebhookSubcommand struct {
	config config.Config

	resource *resource.Resource

	pluginConfig
}

func (p *createWebhookSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	// Track the config and ensure it exists and can be parsed
	cfg := pluginConfig{}
	if err := p.config.DecodePluginConfig(pluginKey, &cfg); errors.As(err, &config.UnsupportedFieldError{}) {
		// Config doesn't support per-plugin configuration, so we can't track them
	} else {
		// Fail unless they key wasn't found, which just means it is the first resource tracked
		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}
	}

	p.pluginConfig = cfg

	return nil
}

func (p *createWebhookSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.pluginConfig.ApiGoModCreated {
		fmt.Println("using existing multi-module layout, updating submodules...")
		return tidyGoModForAPI(p.config.IsMultiGroup())
	}

	if err := createGoModForAPI(fs, p.config); err != nil {
		return err
	}

	if err := tidyGoModForAPI(p.config.IsMultiGroup()); err != nil {
		return err
	}

	p.pluginConfig.ApiGoModCreated = true

	return p.config.EncodePluginConfig(pluginKey, p.pluginConfig)
}
