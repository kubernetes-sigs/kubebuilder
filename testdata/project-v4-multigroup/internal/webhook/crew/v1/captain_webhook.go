/*
Copyright 2026 The Kubernetes authors.

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

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/crew/v1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var captainlog = logf.Log.WithName("captain-resource")

// SetupCaptainWebhookWithManager registers the webhook for Captain in the manager.
func SetupCaptainWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &crewv1.Captain{}).
		WithDefaulter(&CaptainDefaulter{}).
		WithValidator(&CaptainValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-crew-testproject-org-v1-captain,mutating=true,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=captains,verbs=create;update,versions=v1,name=mcaptain-v1.kb.io,admissionReviewVersions=v1

// CaptainDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Captain when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type CaptainDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements admission.Defaulter so a webhook will be registered for the Kind Captain.
func (d *CaptainDefaulter) Default(_ context.Context, obj *crewv1.Captain) error {
	captainlog.Info("Defaulting for Captain", "name", obj.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-crew-testproject-org-v1-captain,mutating=false,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=captains,verbs=create;update,versions=v1,name=vcaptain-v1.kb.io,admissionReviewVersions=v1

// CaptainValidator struct is responsible for validating the Captain resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CaptainValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type Captain.
func (v *CaptainValidator) ValidateCreate(_ context.Context, obj *crewv1.Captain) (admission.Warnings, error) {
	captainlog.Info("Validation for Captain upon creation", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type Captain.
func (v *CaptainValidator) ValidateUpdate(_ context.Context, oldObj, newObj *crewv1.Captain) (admission.Warnings, error) {
	captainlog.Info("Validation for Captain upon update", "name", newObj.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type Captain.
func (v *CaptainValidator) ValidateDelete(_ context.Context, obj *crewv1.Captain) (admission.Warnings, error) {
	captainlog.Info("Validation for Captain upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
