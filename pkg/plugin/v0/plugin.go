/*
Copyright 2020 The Kubernetes Authors.

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

package v2

import (
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	pluginsv0 "sigs.k8s.io/kubebuilder/pkg/plugin/internal/v0"
)

const (
	pluginName    = "go" + plugin.DefaultNameQualifier
	pluginVersion = "v0"
)

var supportedProjectVersions = []string{config.Version2}

var (
	_ plugin.Base                      = Plugin{}
	_ plugin.InitPluginGetter          = Plugin{}
	_ plugin.CreateAPIPluginGetter     = Plugin{}
	_ plugin.CreateWebhookPluginGetter = Plugin{}
)

type Plugin struct {
	pluginsv0.InitPlugin
	pluginsv0.CreateAPIPlugin
	pluginsv0.CreateWebhookPlugin
}

func (Plugin) Name() string                                   { return pluginName }
func (Plugin) Version() string                                { return pluginVersion }
func (Plugin) SupportedProjectVersions() []string             { return supportedProjectVersions }
func (p Plugin) GetInitPlugin() plugin.Init                   { return &p.InitPlugin }
func (p Plugin) GetCreateAPIPlugin() plugin.CreateAPI         { return &p.CreateAPIPlugin }
func (p Plugin) GetCreateWebhookPlugin() plugin.CreateWebhook { return &p.CreateWebhookPlugin }
