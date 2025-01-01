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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var memcachedlog = logf.Log.WithName("memcached-resource")

// SetupMemcachedWebhookWithManager registers the webhook for Memcached in the manager.
func SetupMemcachedWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&examplecomv1alpha1.Memcached{}).
		WithValidator(&MemcachedCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-example-com-testproject-org-v1alpha1-memcached,mutating=false,failurePolicy=fail,sideEffects=None,groups=example.com.testproject.org,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=vmemcached-v1alpha1.kb.io,admissionReviewVersions=v1

// MemcachedCustomValidator struct is responsible for validating the Memcached resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type MemcachedCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &MemcachedCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Memcached.
func (v *MemcachedCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	memcached, ok := obj.(*examplecomv1alpha1.Memcached)
	if !ok {
		return nil, fmt.Errorf("expected a Memcached object but got %T", obj)
	}
	memcachedlog.Info("Validation for Memcached upon creation", "name", memcached.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Memcached.
func (v *MemcachedCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	memcached, ok := newObj.(*examplecomv1alpha1.Memcached)
	if !ok {
		return nil, fmt.Errorf("expected a Memcached object for the newObj but got %T", newObj)
	}
	memcachedlog.Info("Validation for Memcached upon update", "name", memcached.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Memcached.
func (v *MemcachedCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	memcached, ok := obj.(*examplecomv1alpha1.Memcached)
	if !ok {
		return nil, fmt.Errorf("expected a Memcached object but got %T", obj)
	}
	memcachedlog.Info("Validation for Memcached upon deletion", "name", memcached.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
