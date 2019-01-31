/*
Copyright 2018 The Kubernetes Authors.

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

package builder

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/types"
)

// WebhookBuilder builds a webhook based on the provided options.
type WebhookBuilder struct {
	// name specifies the name of the webhook. It must be unique among all webhooks.
	name string

	// path is the URL Path to register this webhook. e.g. "/mutate-pods".
	path string

	// handlers handle admission requests.
	// A WebhookBuilder may have multiple handlers.
	// For example, handlers[0] mutates a pod for feature foo.
	// handlers[1] mutates a pod for a different feature bar.
	handlers []admission.Handler

	// t specifies the type of the webhook.
	// Currently, Mutating and Validating are supported.
	t *types.WebhookType
}

// NewWebhookBuilder creates an empty WebhookBuilder.
func NewWebhookBuilder() *WebhookBuilder {
	return &WebhookBuilder{}
}

// Name sets the name of the webhook.
// This is optional
func (b *WebhookBuilder) Name(name string) *WebhookBuilder {
	b.name = name
	return b
}

// Mutating sets the type to mutating admission webhook
// Only one of Mutating and Validating can be invoked.
func (b *WebhookBuilder) Mutating() *WebhookBuilder {
	m := types.WebhookTypeMutating
	b.t = &m
	return b
}

// Validating sets the type to validating admission webhook
// Only one of Mutating and Validating can be invoked.
func (b *WebhookBuilder) Validating() *WebhookBuilder {
	m := types.WebhookTypeValidating
	b.t = &m
	return b
}

// Path sets the path for the webhook.
// Path needs to be unique among different webhooks.
// This is required. If not set, it will be built from the type and resource name.
// For example, a webhook that mutates pods has a default path of "/mutate-pods"
// If the defaulting logic can't find a unique path for it, user need to set it manually.
func (b *WebhookBuilder) Path(path string) *WebhookBuilder {
	b.path = path
	return b
}

// Handlers sets the handlers of the webhook.
func (b *WebhookBuilder) Handlers(handlers ...admission.Handler) *WebhookBuilder {
	b.handlers = handlers
	return b
}

// Build creates the Webhook based on the options provided.
func (b *WebhookBuilder) Build() *admission.Webhook {
	if b.t == nil {
		b.Mutating()
	}

	w := &admission.Webhook{
		Name:     b.name,
		Type:     *b.t,
		Path:     b.path,
		Handlers: b.handlers,
	}

	return w
}
