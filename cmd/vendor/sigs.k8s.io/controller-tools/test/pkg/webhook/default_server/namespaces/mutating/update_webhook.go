/*
Copyright 2018 The Kubernetes authors.

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

package mutating

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

func init() {
	builderName := "mutating-update-namespaces"
	Builders[builderName] = builder.
		NewWebhookBuilder().
		Name(builderName + ".testproject.org").
		Path("/" + builderName).
		Mutating().
		Operations(admissionregistrationv1beta1.Update).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		ForType(&corev1.Namespace{})
}
