/*
Copyright 2019 The Kubernetes authors.

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
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	handler "sigs.k8s.io/kubebuilder/test/project/pkg/webhook/default_server/firstmate/mutating/handler"
)

// +kubebuilder:webhook:groups=crew,resources=firstmates
// +kubebuilder:webhook:versions=v1
// +kubebuilder:webhook:verbs=Create;Update
// +kubebuilder:webhook:name=mutating-create-update-firstmate.testproject.org,path=/mutating-create-update-firstmate
// +kubebuilder:webhook:type=mutating
// +kubebuilder:webhook:failure-policy=Fail
func init() {
	var wh webhook.Webhook
	builderName := "mutating-create-update-firstmate"
	wh = builder.
		NewWebhookBuilder().
		Name(builderName + ".testproject.org").
		Path("/" + builderName).
		Mutating().
		Handlers(handler.CreateUpdateHandlers...).
		Build()
	FirstMateWebhooks = append(FirstMateWebhooks, wh)
}
