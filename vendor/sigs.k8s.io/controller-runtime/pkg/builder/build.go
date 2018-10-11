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

package builder

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// application is a simple Controller for a single API type.  It will create a Manager for itself
// if one is not provided.
type application struct {
	mgr  manager.Manager
	ctrl controller.Controller
}

// Supporting mocking out functions for testing
var getConfig = config.GetConfig
var newController = controller.New
var newManager = manager.New
var getGvk = apiutil.GVKForObject

// Builder builds an Application Controller (e.g. Operator) and returns a manager.Manager to start it.
type Builder struct {
	apiType        runtime.Object
	mgr            manager.Manager
	predicates     []predicate.Predicate
	managedObjects []runtime.Object
	config         *rest.Config
	ctrl           controller.Controller
}

// SimpleController returns a new Builder
func SimpleController() *Builder {
	return &Builder{}
}

// ForType sets the ForType that generates other types
func (b *Builder) ForType(apiType runtime.Object) *Builder {
	b.apiType = apiType
	return b
}

// Owns configures the Application Controller to respond to create / delete / update events for objects it managedObjects
// - e.g. creates.  apiType is an empty instance of an object matching the managed object type.
func (b *Builder) Owns(apiType runtime.Object) *Builder {
	b.managedObjects = append(b.managedObjects, apiType)
	return b
}

// WithConfig sets the Config to use for configuring clients.  Defaults to the in-cluster config or to ~/.kube/config.
func (b *Builder) WithConfig(config *rest.Config) *Builder {
	b.config = config
	return b
}

// WithManager sets the Manager to use for registering the Controller.  Defaults to a new manager.Manager.
func (b *Builder) WithManager(m manager.Manager) *Builder {
	b.mgr = m
	return b
}

// WithEventFilter sets the event filters, to filter which create/update/delete/generic events eventually
// trigger reconciliations.  For example, filtering on whether the resource version has changed.
// Defaults to the empty list.
func (b *Builder) WithEventFilter(p predicate.Predicate) *Builder {
	b.predicates = append(b.predicates, p)
	return b
}

// Build builds the Application Controller and returns the Manager used to start it.
func (b *Builder) Build(r reconcile.Reconciler) (manager.Manager, error) {
	if r == nil {
		return nil, fmt.Errorf("must call WithReconciler to set Reconciler")
	}

	// Set the Config
	if err := b.doConfig(); err != nil {
		return nil, err
	}

	// Set the Manager
	if err := b.doManager(); err != nil {
		return nil, err
	}

	// Set the Controller
	if err := b.doController(r); err != nil {
		return nil, err
	}

	a := &application{mgr: b.mgr, ctrl: b.ctrl}

	// Reconcile type
	s := &source.Kind{Type: b.apiType}
	h := &handler.EnqueueRequestForObject{}
	err := a.ctrl.Watch(s, h, b.predicates...)
	if err != nil {
		return nil, err
	}

	// Watch the managed types
	for _, t := range b.managedObjects {
		s := &source.Kind{Type: t}
		h := &handler.EnqueueRequestForOwner{
			OwnerType:    b.apiType,
			IsController: true,
		}
		if err := a.ctrl.Watch(s, h, b.predicates...); err != nil {
			return nil, err
		}
	}

	return a.mgr, nil
}

func (b *Builder) doConfig() error {
	if b.config != nil {
		return nil
	}
	var err error
	b.config, err = getConfig()
	return err
}

func (b *Builder) doManager() error {
	if b.mgr != nil {
		return nil
	}
	var err error
	b.mgr, err = newManager(b.config, manager.Options{})
	return err
}

func (b *Builder) getControllerName() (string, error) {
	gvk, err := getGvk(b.apiType, b.mgr.GetScheme())
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-application", strings.ToLower(gvk.Kind))
	return name, nil
}

func (b *Builder) doController(r reconcile.Reconciler) error {
	name, err := b.getControllerName()
	if err != nil {
		return err
	}
	b.ctrl, err = newController(name, b.mgr, controller.Options{Reconciler: r})
	return err
}
