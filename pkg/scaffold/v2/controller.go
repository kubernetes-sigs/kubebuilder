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

	// Pattern specifies an alternative style of controller to generate
	Pattern string
}

// GetInput implements input.File
func (a *Controller) GetInput() (input.Input, error) {

	a.ResourcePackage, a.GroupDomain = util.GetResourceInfo(a.Resource, a.Input)

	if a.Plural == "" {
		a.Plural = flect.Pluralize(strings.ToLower(a.Resource.Kind))
	}

	if a.Path == "" {
		a.Path = filepath.Join("controllers",
			strings.ToLower(a.Resource.Kind)+"_controller.go")
	}

	switch a.Pattern {
	case "addon":
		a.TemplateBody = controllerTemplate + controllerTemplateBodyPatternAddon
	default:
		a.TemplateBody = controllerTemplate + controllerTemplateBodyDefault
	}

	a.Input.IfExistsAction = input.Error
	return a.Input, nil
}

// APIImport returns the import alias for the types package
func (a *Controller) APIImport() string {
	switch a.Pattern {
	case "addon":
		return "api"
	default:
		return a.Resource.Group + a.Resource.Version
	}
}

var controllerTemplate = `{{ .Boilerplate }}

package controllers

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/go-logr/logr"
{{- if eq .Pattern "addon" }}
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/status"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative"
{{ end }}

	{{ .APIImport }} "{{ .ResourcePackage }}/{{ .Resource.Version }}"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log logr.Logger

{{ if eq .Pattern "addon" }}
	declarative.Reconciler
{{ end }}
}

// +kubebuilder:rbac:groups={{.GroupDomain}},resources={{ .Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{.GroupDomain}},resources={{ .Plural }}/status,verbs=get;update;patch

`

var controllerTemplateBodyDefault = `
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("{{ .Resource.Kind | lower }}", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&{{ .APIImport }}.{{ .Resource.Kind }}{}).
		Complete(r)
}
`

var controllerTemplateBodyPatternAddon = `
func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	addon.Init()

	labels := map[string]string{
		"k8s-app": "{{ .Resource.Kind | lower }}",
	}

	watchLabels := declarative.SourceLabel(mgr.GetScheme())

	if err := r.Reconciler.Init(mgr, &{{ .APIImport }}.{{ .Resource.Kind }}{},
		declarative.WithObjectTransform(declarative.AddLabels(labels)),
		declarative.WithOwner(declarative.SourceAsOwner),
		declarative.WithLabels(watchLabels),
		declarative.WithStatus(status.NewBasic(mgr.GetClient())),
		// TODO: add an application to your manifest:  declarative.WithObjectTransform(addon.TransformApplicationFromStatus),
		// TODO: add an application to your manifest:  declarative.WithManagedApplication(watchLabels),
		declarative.WithObjectTransform(addon.ApplyPatches),
	); err != nil {
		return err
	}

	c, err := controller.New("{{ .Resource.Kind | lower }}-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to {{ .Resource.Kind }}
	err = c.Watch(&source.Kind{Type: &{{ .APIImport }}.{{ .Resource.Kind }}{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to deployed objects
	_, err = declarative.WatchAll(mgr.GetConfig(), c, r, watchLabels)
	if err != nil {
		return err
	}

	return nil
}
`
