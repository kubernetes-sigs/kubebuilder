/*
Copyright 2018 The Kubernetes Authors.

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
	"path/filepath"
	"strings"

	"github.com/gobuffalo/flect"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/util"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

// Controller scaffolds a Controller for a Resource
type Controller struct {
	input.Input

	// Resource is the Resource to make the Controller for
	Resource *resource.Resource

	// ResourcePackage is the package of the Resource
	ResourcePackage string

	// Plural is the plural lowercase of kind
	Plural string

	// Is the Group + "." + Domain for the Resource
	GroupDomain string
}

// GetInput implements input.File
func (a *Controller) GetInput() (input.Input, error) {

	a.ResourcePackage, a.GroupDomain = util.GetResourceInfo(a.Resource, a.Repo, a.Domain)

	if a.Plural == "" {
		a.Plural = flect.Pluralize(strings.ToLower(a.Resource.Kind))
	}

	if a.Path == "" {
		a.Path = filepath.Join("controllers",
			strings.ToLower(a.Resource.Kind)+"_controller.go")
	}

	a.TemplateBody = controllerTemplate

	a.Input.IfExistsAction = input.Error
	return a.Input, nil
}

var controllerTemplate = `{{ .Boilerplate }}

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	{{ .Resource.GroupImportSafe }}{{ .Resource.Version }} "{{ .ResourcePackage }}/{{ .Resource.Version }}"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups={{.GroupDomain}},resources={{ .Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{.GroupDomain}},resources={{ .Plural }}/status,verbs=get;update;patch

func (r *{{ .Resource.Kind }}Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("{{ .Resource.Kind | lower }}", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&{{ .Resource.GroupImportSafe }}{{ .Resource.Version }}.{{ .Resource.Kind }}{}).
		Complete(r)
}
`
