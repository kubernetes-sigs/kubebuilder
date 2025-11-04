/*
Copyright 2025 The Kubernetes authors.

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

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4/api/v1"
)

// nolint:unused
// log is for logging in this package.
var sailorlog = logf.Log.WithName("sailor-resource")

// SetupSailorWebhookWithManager registers the webhook for Sailor in the manager.
func SetupSailorWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&crewv1.Sailor{}).
		WithValidator(&SailorCustomValidator{}).
		WithValidatorCustomPath("/custom-validate-sailor").
		WithDefaulter(&SailorCustomDefaulter{}).
		WithDefaulterCustomPath("/custom-mutate-sailor").
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/custom-mutate-sailor,mutating=true,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=sailors,verbs=create;update,versions=v1,name=msailor-v1.kb.io,admissionReviewVersions=v1

// SailorCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Sailor when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type SailorCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &SailorCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Sailor.
func (d *SailorCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	sailor, ok := obj.(*crewv1.Sailor)

	if !ok {
		return fmt.Errorf("expected an Sailor object but got %T", obj)
	}
	sailorlog.Info("Defaulting for Sailor", "name", sailor.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/custom-validate-sailor,mutating=false,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=sailors,verbs=create;update,versions=v1,name=vsailor-v1.kb.io,admissionReviewVersions=v1

// SailorCustomValidator struct is responsible for validating the Sailor resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SailorCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &SailorCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Sailor.
func (v *SailorCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	sailor, ok := obj.(*crewv1.Sailor)
	if !ok {
		return nil, fmt.Errorf("expected a Sailor object but got %T", obj)
	}
	sailorlog.Info("Validation for Sailor upon creation", "name", sailor.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Sailor.
func (v *SailorCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	sailor, ok := newObj.(*crewv1.Sailor)
	if !ok {
		return nil, fmt.Errorf("expected a Sailor object for the newObj but got %T", newObj)
	}
	sailorlog.Info("Validation for Sailor upon update", "name", sailor.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Sailor.
func (v *SailorCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	sailor, ok := obj.(*crewv1.Sailor)
	if !ok {
		return nil, fmt.Errorf("expected a Sailor object but got %T", obj)
	}
	sailorlog.Info("Validation for Sailor upon deletion", "name", sailor.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
