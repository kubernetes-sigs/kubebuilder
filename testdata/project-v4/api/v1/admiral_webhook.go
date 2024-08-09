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
var admirallog = logf.Log.WithName("admiral-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Admiral) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&AdmiralCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-crew-testproject-org-v1-admiral,mutating=true,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=admirales,verbs=create;update,versions=v1,name=madmiral.kb.io,admissionReviewVersions=v1

type AdmiralCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &AdmiralCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (d *AdmiralCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	admirallog.Info("CustomDefaulter for Admiral")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Admiral" {
		return fmt.Errorf("expected Kind Admiral got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*Admiral)
	if !ok {
		return fmt.Errorf("expected an Admiral object but got %T", obj)
	}
	admirallog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
