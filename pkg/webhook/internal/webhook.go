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

package internal

import (
	"time"

	"github.com/kubernetes-sigs/kubebuilder/pkg/webhook/internal/certprovider"

	admission "k8s.io/api/admission/v1beta1"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AdmissionWebhookOptions covers all of the possible setting of the admission webhook
type AdmissionWebhookInstallConfig struct {
	CertProvider certprovider.CertProvider

	// List of configuration for each webhook in the webhook server.
	ServerConfig []*HandlerConfig

	// Configuration of how the webhook can be accessed by the client
	ClientConfig *AdmissionWebhookClientConfig

	// Port where the webhook is served. Per k8s admission
	// registration requirements this should be 443 unless there is
	// only a single port for the service.
	Port int

	// RegistrationDelay controls how long admission registration
	// occurs after the webhook is started. This is used to avoid
	// potential races where registration completes and k8s apiserver
	// invokes the webhook before the HTTP server is started.
	RegistrationDelay time.Duration
}

type HandlerConfig struct {
	// Mutating or validating webhook
	WebhookType AdmissionWebhookConfigType

	// Name of the webhook
	Name string

	// The GroupVersionResources that the admission webhook will manage.
	// Required
	GroupVersionResources []metav1.GroupVersionResource

	// Operations is the operations the admission webhook cares about.
	// Default to CREATE and UPDATE
	Operations []admission.Operation

	// Default to all namespace except kube-system.
	// This will apply to all webhooks.
	NamespaceSelector *metav1.LabelSelector

	// FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
	// allowed values are Ignore or Fail. Defaults to Ignore by the API server.
	FailurePolicy *admissionregistration.FailurePolicyType

	// An path that will be used for both server handler registration and
	// webhook configuration registration.
	Path string
}

type AdmissionWebhookConfigType string

const (
	MutatingType   AdmissionWebhookConfigType = "Mutating"
	ValidatingType AdmissionWebhookConfigType = "Validating"
)

type AdmissionWebhookClientConfig struct {
	// clientConfigType is the discriminator of AdmissionWebhookClientConfig which is an union.
	ClientConfigType WebhookClientConfigType

	// the URL of the webhook server
	URL string

	// the name of the k8s service that fronts the webhook server
	ServiceName string
	// the namespace of the k8s service that fronts the webhook server
	ServiceNamespace string
}

type WebhookClientConfigType string

const (
	URLType     WebhookClientConfigType = "URL"
	ServiceType WebhookClientConfigType = "Service"
)
