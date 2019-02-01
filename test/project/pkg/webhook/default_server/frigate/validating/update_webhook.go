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

package validating

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	handler "sigs.k8s.io/kubebuilder/test/project/pkg/webhook/default_server/frigate/validating/handler"
)

// +kubebuilder:webhook:groups=ship,resources=frigates
// +kubebuilder:webhook:versions=v1beta1
// +kubebuilder:webhook:verbs=Update
// +kubebuilder:webhook:name=validating-update-frigate.testproject.org,path=/validating-update-frigate
// +kubebuilder:webhook:type=validating
// +kubebuilder:webhook:failure-policy=Fail
func init() {
	var wh webhook.Webhook
	builderName := "validating-update-frigate"
	wh = builder.
		NewWebhookBuilder().
		Name(builderName + ".testproject.org").
		Path("/" + builderName).
		Validating().
		Handlers(handler.UpdateHandlers...).
		Build()
	FrigateWebhooks = append(FrigateWebhooks, wh)
}
