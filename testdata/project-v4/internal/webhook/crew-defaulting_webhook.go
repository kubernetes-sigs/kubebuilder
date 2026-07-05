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

package webhook

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-crew-defaulting,mutating=true,failurePolicy=fail,sideEffects=None,groups=crew.testproject.org,resources=captains;sailors,verbs=create;update,versions=v1,name=mcrew-defaulting.kb.io,admissionReviewVersions=v1

// CrewDefaultingWebhook mutates intercepted resources.

type CrewDefaultingWebhook struct {
}

// CrewDefaultingWebhook implements admission.Handler.
var _ admission.Handler = &CrewDefaultingWebhook{}

// Handle implements admission.Handler.
func (h *CrewDefaultingWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {

	// TODO(user): fill in your defaulting logic.

	// Use admission.Decoder to decode req.Object.Raw.
	// Use req.Resource.Group, req.Resource.Version, and req.Resource.Resource
	// to identify which resource type triggered this webhook.
	return admission.Allowed("")
}
