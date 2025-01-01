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

	shipv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/ship/v1"
)

// nolint:unused
// log is for logging in this package.
var destroyerlog = logf.Log.WithName("destroyer-resource")

// SetupDestroyerWebhookWithManager registers the webhook for Destroyer in the manager.
func SetupDestroyerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&shipv1.Destroyer{}).
		WithDefaulter(&DestroyerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-ship-testproject-org-v1-destroyer,mutating=true,failurePolicy=fail,sideEffects=None,groups=ship.testproject.org,resources=destroyers,verbs=create;update,versions=v1,name=mdestroyer-v1.kb.io,admissionReviewVersions=v1

// DestroyerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Destroyer when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type DestroyerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &DestroyerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Destroyer.
func (d *DestroyerCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	destroyer, ok := obj.(*shipv1.Destroyer)

	if !ok {
		return fmt.Errorf("expected an Destroyer object but got %T", obj)
	}
	destroyerlog.Info("Defaulting for Destroyer", "name", destroyer.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
