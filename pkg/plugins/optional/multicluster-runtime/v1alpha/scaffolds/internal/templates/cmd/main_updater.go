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

package cmd

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Inserter = &MainUpdater{}

// MainUpdater inserts the multicluster-aware reconciler setup into cmd/main.go.
// It targets the // +kubebuilder:scaffold:multicluster-builder marker so that
// go/v4's standard MainUpdater (which uses mgr.GetClient()) does NOT run at this
// marker and generate incompatible code.
type MainUpdater struct {
	machinery.RepositoryMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin

	// ControllerName is the name of the controller (defaults to Kind).
	ControllerName string

	// WireController indicates that this resource has a controller to wire.
	WireController bool
}

// GetPath implements machinery.Builder.
func (*MainUpdater) GetPath() string { return defaultMainPath }

// GetIfExistsAction implements machinery.Builder.
func (*MainUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// GetIfNotExistsAction implements machinery.HasIfNotExistsAction.
// If cmd/main.go doesn't exist yet (e.g. in unit tests with empty filesystems),
// silently skip rather than error.
func (*MainUpdater) GetIfNotExistsAction() machinery.IfNotExistsAction {
	return machinery.IgnoreFile
}

const mcBuilderMarker = "multicluster-builder"

// GetMarkers implements machinery.Inserter.
func (f *MainUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(defaultMainPath, mcBuilderMarker),
	}
}

const mcReconcilerSetupCodeFragment = `if err := (&controller.%s{
	Client: mgr.GetLocalManager().GetClient(),
	Scheme: mgr.GetLocalManager().GetScheme(),
}).SetupWithManager(mgr); err != nil {
	setupLog.Error(err, "Failed to create controller", "controller", "%s")
	os.Exit(1)
}
`

const mcMultiGroupReconcilerSetupCodeFragment = `if err := (&%scontroller.%s{
	Client: mgr.GetLocalManager().GetClient(),
	Scheme: mgr.GetLocalManager().GetScheme(),
}).SetupWithManager(mgr); err != nil {
	setupLog.Error(err, "Failed to create controller", "controller", "%s")
	os.Exit(1)
}
`

// GetCodeFragments implements machinery.Inserter.
func (f *MainUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	if f.Resource == nil || !f.WireController {
		return fragments
	}

	reconcilerName := resource.NormalizeReconcilerName(f.ControllerName, f.Resource.Kind)
	controllerName := resource.GetControllerName(f.ControllerName, f.Resource.Kind, f.Resource.Group, f.MultiGroup)

	var setup string
	if !f.MultiGroup || f.Resource.Group == "" {
		setup = fmt.Sprintf(mcReconcilerSetupCodeFragment, reconcilerName, controllerName)
	} else {
		setup = fmt.Sprintf(mcMultiGroupReconcilerSetupCodeFragment,
			f.Resource.PackageName(), reconcilerName, controllerName)
	}

	fragments[machinery.NewMarkerFor(defaultMainPath, mcBuilderMarker)] = []string{setup}
	return fragments
}
