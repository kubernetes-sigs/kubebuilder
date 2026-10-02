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

// Package v1alpha implements the multicluster-runtime/v1alpha plugin for Kubebuilder.
//
// This plugin modifies the scaffolded project to use sigs.k8s.io/multicluster-runtime
// instead of the standard single-cluster controller-runtime manager, enabling
// controllers to reconcile objects across multiple Kubernetes clusters.
//
// It is designed to be chained after go/v4:
//
//	kubebuilder init --plugins go/v4,multicluster-runtime/v1-alpha ...
package v1alpha

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

// pluginName is the fully qualified plugin name.
const pluginName = "multicluster-runtime." + plugins.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1, Stage: stage.Alpha}
	supportedProjectVersions = []config.Version{cfgv3.Version}
)

var (
	_ plugin.Init        = Plugin{}
	_ plugin.CreateAPI   = Plugin{}
	_ plugin.Edit        = Plugin{}
	_ plugin.Describable = Plugin{}
)

// Plugin implements plugin.Init, plugin.CreateAPI, and plugin.Edit.
type Plugin struct {
	initSubcommand
	createAPISubcommand
	editSubcommand
}

// Name returns the plugin's qualified name.
func (Plugin) Name() string { return pluginName }

// Version returns the plugin version.
func (Plugin) Version() plugin.Version { return pluginVersion }

// SupportedProjectVersions returns the project config versions supported by this plugin.
func (Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

// GetInitSubcommand returns the init subcommand.
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &p.initSubcommand }

// GetCreateAPISubcommand returns the create api subcommand.
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return &p.createAPISubcommand }

// GetEditSubcommand returns the edit subcommand.
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand { return &p.editSubcommand }

// Description returns a short description of the plugin.
func (Plugin) Description() string {
	return "Rewrites cmd/main.go and controllers to use sigs.k8s.io/multicluster-runtime, " +
		"enabling controllers to reconcile objects across multiple Kubernetes clusters"
}

// DeprecationWarning returns empty — this plugin is not deprecated.
func (Plugin) DeprecationWarning() string { return "" }
