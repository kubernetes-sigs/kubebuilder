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

package v2alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	shipv2alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v2alpha1"
)

// nolint:unused
// log is for logging in this package.
var cruiserlog = logf.Log.WithName("cruiser-resource")

// SetupCruiserWebhookWithManager registers the webhook for Cruiser in the manager.
func SetupCruiserWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&shipv2alpha1.Cruiser{}).
		WithValidator(&CruiserCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-ship-testproject-org-v2alpha1-cruiser,mutating=false,failurePolicy=fail,sideEffects=None,groups=ship.testproject.org,resources=cruisers,verbs=create;update,versions=v2alpha1,name=vcruiser-v2alpha1.kb.io,admissionReviewVersions=v1

// CruiserCustomValidator struct is responsible for validating the Cruiser resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CruiserCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CruiserCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Cruiser.
func (v *CruiserCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cruiser, ok := obj.(*shipv2alpha1.Cruiser)
	if !ok {
		return nil, fmt.Errorf("expected a Cruiser object but got %T", obj)
	}
	cruiserlog.Info("Validation for Cruiser upon creation", "name", cruiser.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Cruiser.
func (v *CruiserCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cruiser, ok := newObj.(*shipv2alpha1.Cruiser)
	if !ok {
		return nil, fmt.Errorf("expected a Cruiser object for the newObj but got %T", newObj)
	}
	cruiserlog.Info("Validation for Cruiser upon update", "name", cruiser.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Cruiser.
func (v *CruiserCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cruiser, ok := obj.(*shipv2alpha1.Cruiser)
	if !ok {
		return nil, fmt.Errorf("expected a Cruiser object but got %T", obj)
	}
	cruiserlog.Info("Validation for Cruiser upon deletion", "name", cruiser.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
