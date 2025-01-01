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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// nolint:unused
// log is for logging in this package.
var issuerlog = logf.Log.WithName("issuer-resource")

// SetupIssuerWebhookWithManager registers the webhook for Issuer in the manager.
func SetupIssuerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&certmanagerv1.Issuer{}).
		WithDefaulter(&IssuerCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-cert-manager-io-v1-issuer,mutating=true,failurePolicy=fail,sideEffects=None,groups=cert-manager.io,resources=issuers,verbs=create;update,versions=v1,name=missuer-v1.kb.io,admissionReviewVersions=v1

// IssuerCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Issuer when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type IssuerCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &IssuerCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Issuer.
func (d *IssuerCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	issuer, ok := obj.(*certmanagerv1.Issuer)

	if !ok {
		return fmt.Errorf("expected an Issuer object but got %T", obj)
	}
	issuerlog.Info("Defaulting for Issuer", "name", issuer.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}
