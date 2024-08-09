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
var lakerslog = logf.Log.WithName("lakers-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Lakers) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&LakersCustomValidator{}).
		WithDefaulter(&LakersCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-testproject-org-v1-lakers,mutating=true,failurePolicy=fail,sideEffects=None,groups=testproject.org,resources=lakers,verbs=create;update,versions=v1,name=mlakers.kb.io,admissionReviewVersions=v1

type LakersCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &LakersCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (d *LakersCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	lakerslog.Info("CustomDefaulter for Admiral")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Lakers" {
		return fmt.Errorf("expected Kind Admiral got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*Lakers)
	if !ok {
		return fmt.Errorf("expected an Lakers object but got %T", obj)
	}
	lakerslog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-testproject-org-v1-lakers,mutating=false,failurePolicy=fail,sideEffects=None,groups=testproject.org,resources=lakers,verbs=create;update,versions=v1,name=vlakers.kb.io,admissionReviewVersions=v1

type LakersCustomValidator struct{}

var _ webhook.CustomValidator = &LakersCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *LakersCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lakerslog.Info("Creation Validation for Lakers")

	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Lakers" {
		return nil, fmt.Errorf("expected Kind Lakers got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", obj)
	}
	lakerslog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *LakersCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	lakerslog.Info("Update Validation for Lakers")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Lakers" {
		return nil, fmt.Errorf("expected Kind Lakers got %q", req.Kind.Kind)
	}
	castedObj, ok := newObj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", newObj)
	}
	lakerslog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *LakersCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lakerslog.Info("Deletion Validation for Lakers")
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("expected admission.Request in ctx: %w", err)
	}
	if req.Kind.Kind != "Lakers" {
		return nil, fmt.Errorf("expected Kind Lakers got %q", req.Kind.Kind)
	}
	castedObj, ok := obj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", obj)
	}
	lakerslog.Info("default", "name", castedObj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
