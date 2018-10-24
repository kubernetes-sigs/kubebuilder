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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	crewv1 "sigs.k8s.io/kubebuilder/test/project/pkg/apis/crew/v1"
)

// +kubebuilder:webhook:groups=crew,versions=v1,resources=firstmates,verbs=CREATE;UPDATE
// +kubebuilder:webhook:name=mutating-create-update-firstmate.testproject.org,path=/mutating-create-update-firstmate,type=mutating,failure-policy=Fail
func init() {
	builderName := "mutating-create-update-firstmate"
	Builders[builderName] = builder.
		NewWebhookBuilder().
		Name(builderName+".testproject.org").
		Path("/"+builderName).
		Mutating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		ForType(&crewv1.FirstMate{})
}
