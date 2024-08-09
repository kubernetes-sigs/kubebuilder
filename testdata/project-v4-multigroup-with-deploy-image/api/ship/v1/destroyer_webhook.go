/*
Copyright 2024 The Kubernetes authors.

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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var destroyerlog = logf.Log.WithName("destroyer-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Destroyer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&DestroyerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-ship-testproject-org-v1-destroyer,mutating=true,failurePolicy=fail,sideEffects=None,groups=ship.testproject.org,resources=destroyers,verbs=create;update,versions=v1,name=mdestroyer.kb.io,admissionReviewVersions=v1

type DestroyerCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &DestroyerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (d *DestroyerCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	destroyerlog.Info("CustomDefaulter for Admiral")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Destroyer" {
		return fmt.Errorf("expected Kind Admiral got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*Destroyer)
	if !ok {
		return fmt.Errorf("expected an Destroyer object but got %T", obj)
	}
	destroyerlog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
