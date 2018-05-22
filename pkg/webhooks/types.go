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

package webhooks

import (
	"time"

	"github.com/mattbaird/jsonpatch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// ControllerOptions contains the configuration for the webhook
type ControllerOptions struct {
	// WebhookName is the name of the webhook we create to handle mutations
	// before they get stored in the storage.
	WebhookName string

	// ServiceName is the service name of the webhook.
	ServiceName string

	// ServiceNamespace is the namespace of the webhook service.
	ServiceNamespace string

	// APIGroupName is the group name of the custom resource being managed.
	APIGroupName string

	// APIVersion is the version of the custom resource being managed.
	APIVersion string

	// Organization is the organization to use for cert creation.
	Organization string

	// Resources is the plural version of all resources the webhook will be managing.
	Resources []string

	// SecretName is the name of k8s secret that contains the webhook
	// server key/cert and corresponding CA cert that signed them. The
	// server key/cert are used to serve the webhook and the CA cert
	// is provided to k8s apiserver during admission controller
	// registration.
	SecretName string

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

// ResourceCallback defines the signature that resource specific handlers that can validate
// and mutate an object. If non-nil error is returned, object creation is denied. Any
// mutations are to be appended to the patches operations.
type ResourceCallback func(patches *[]jsonpatch.JsonPatchOperation, old GenericCRD, new GenericCRD) error

// GenericCRDHandler defines the factory object to use for unmarshaling incoming objects
type GenericCRDHandler struct {
	Factory   runtime.Object
	Defaulter ResourceCallback
	Validator ResourceCallback
}

// AdmissionController implements the external admission webhook for validation and defaulting of resources.
type AdmissionController struct {
	// Client is the kubernetes client to be used.
	Client kubernetes.Interface
	// Options are the webhook configurations.
	Options ControllerOptions
	// Handlers is the map of Resource to Generic CRD handler mapping.
	Handlers map[string]GenericCRDHandler
}

// GenericCRD is the interface definition that allows us to perform the generic CRD actions.
type GenericCRD interface {
	// GetObjectMeta return the object metadata
	GetObjectMeta() metav1.Object
	// GetSpecJSON returns the Spec part of the resource marshalled into JSON
	GetSpecJSON() ([]byte, error)
}
