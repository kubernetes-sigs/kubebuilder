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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/markbates/inflect"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
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

	a.ResourcePackage, a.GroupDomain = getResourceInfo(a.Resource, a.Input)

	if a.Plural == "" {
		rs := inflect.NewDefaultRuleset()
		a.Plural = rs.Pluralize(strings.ToLower(a.Resource.Kind))
	}

	if a.Path == "" {
		a.Path = filepath.Join("controllers",
			strings.ToLower(a.Resource.Kind)+"_controller.go")
	}
	a.TemplateBody = controllerTemplate
	a.Input.IfExistsAction = input.Error
	return a.Input, nil
}

func getResourceInfo(r *resource.Resource, in input.Input) (resourcePackage, groupDomain string) {
	// Use the k8s.io/api package for core resources
	coreGroups := map[string]string{
		"apps":                  "",
		"admissionregistration": "k8s.io",
		"apiextensions":         "k8s.io",
		"authentication":        "k8s.io",
		"autoscaling":           "",
		"batch":                 "",
		"certificates":          "k8s.io",
		"core":                  "",
		"extensions":            "",
		"metrics":               "k8s.io",
		"policy":                "",
		"rbac.authorization":    "k8s.io",
		"storage":               "k8s.io",
	}
	resourcePath := filepath.Join("api", r.Version, fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind)))
	if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
		if domain, found := coreGroups[r.Group]; found {
			resourcePackage := path.Join("k8s.io", "api", r.Group)
			groupDomain = r.Group
			if domain != "" {
				groupDomain = r.Group + "." + domain
			}
			return resourcePackage, groupDomain
		}
		// TODO: need to support '--resource-pkg-path' flag for specifying resourcePath
	}
	return path.Join(in.Repo, "api"), r.Group + "." + in.Domain
}

var controllerTemplate = `{{ .Boilerplate }}

package controllers

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/go-logr/logr"

	{{ .Resource.Group}}{{ .Resource.Version }} "{{ .ResourcePackage }}/{{ .Resource.Version }}"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log logr.Logger
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
		For(&{{ .Resource.Group}}{{ .Resource.Version }}.{{ .Resource.Kind }}{}).
		Complete(r)
}
`
