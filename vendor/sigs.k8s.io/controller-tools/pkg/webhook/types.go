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

package webhook

// webhookType defines the type of a webhook
type webhookType int

const (
	_ = iota
	// webhookTypeMutating represents mutating type webhook
	webhookTypeMutating webhookType = iota
	// webhookTypeValidating represents validating type webhook
	webhookTypeValidating
)

// Webhook defines the basics that a webhook should support.
type Webhook interface {
	// GetName returns the name of the webhook.
	GetName() string
	// GetPath returns the path that the webhook registered.
	GetPath() string
	// GetType returns the Type of the webhook.
	// e.g. mutating or validating
	GetType() webhookType
	// Validate validates if the webhook itself is valid.
	// If invalid, a non-nil error will be returned.
	Validate() error
}
