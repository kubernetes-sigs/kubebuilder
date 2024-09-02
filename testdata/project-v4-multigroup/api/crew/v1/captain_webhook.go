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
var captainlog = logf.Log.WithName("captain-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Captain) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&CaptainCustomValidator{}).
		WithDefaulter(&CaptainCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-crew-testproject-org-v1-captain,mutating=true,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=captains,verbs=create;update,versions=v1,name=mcaptain.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// CaptainCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Captain when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type CaptainCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &CaptainCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Captain
func (d *CaptainCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	captain, ok := obj.(*Captain)
	if !ok {
		return fmt.Errorf("expected an Captain object but got %T", obj)
	}
	captainlog.Info("Defaulting for Captain", "name", captain.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-crew-testproject-org-v1-captain,mutating=false,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=captains,verbs=create;update,versions=v1,name=vcaptain.kb.io,admissionReviewVersions=v1

// +kubebuilder:object:generate=false
// CaptainCustomValidator struct is responsible for validating the Captain resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CaptainCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CaptainCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Captain
func (v *CaptainCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	captain, ok := obj.(*Captain)
	if !ok {
		return nil, fmt.Errorf("expected a Captain object but got %T", obj)
	}
	captainlog.Info("Validation for Captain upon creation", "name", captain.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Captain
func (v *CaptainCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	captain, ok := newObj.(*Captain)
	if !ok {
		return nil, fmt.Errorf("expected a Captain object but got %T", newObj)
	}
	captainlog.Info("Validation for Captain upon update", "name", captain.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Captain
func (v *CaptainCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	captain, ok := obj.(*Captain)
	if !ok {
		return nil, fmt.Errorf("expected a Captain object but got %T", obj)
	}
	captainlog.Info("Validation for Captain upon deletion", "name", captain.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
