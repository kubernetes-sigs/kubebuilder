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

// nolint:unused
// log is for logging in this package.
var lakerslog = logf.Log.WithName("lakers-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks.
func (r *Lakers) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&LakersCustomValidator{}).
		WithDefaulter(&LakersCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-testproject-org-v1-lakers,mutating=true,failurePolicy=fail,sideEffects=None,groups=testproject.org,resources=lakers,verbs=create;update,versions=v1,name=mlakers-v1.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// LakersCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Lakers when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type LakersCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &LakersCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Lakers.
func (d *LakersCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	lakers, ok := obj.(*Lakers)
	if !ok {
		return fmt.Errorf("expected an Lakers object but got %T", obj)
	}
	lakerslog.Info("Defaulting for Lakers", "name", lakers.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-testproject-org-v1-lakers,mutating=false,failurePolicy=fail,sideEffects=None,groups=testproject.org,resources=lakers,verbs=create;update,versions=v1,name=vlakers-v1.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// LakersCustomValidator struct is responsible for validating the Lakers resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type LakersCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &LakersCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Lakers.
func (v *LakersCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lakers, ok := obj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", obj)
	}
	lakerslog.Info("Validation for Lakers upon creation", "name", lakers.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Lakers.
func (v *LakersCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	lakers, ok := newObj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", newObj)
	}
	lakerslog.Info("Validation for Lakers upon update", "name", lakers.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Lakers.
func (v *LakersCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	lakers, ok := obj.(*Lakers)
	if !ok {
		return nil, fmt.Errorf("expected a Lakers object but got %T", obj)
	}
	lakerslog.Info("Validation for Lakers upon deletion", "name", lakers.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
