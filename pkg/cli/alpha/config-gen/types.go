/*
Copyright 2021 The Kubernetes Authors.

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

package configgen

import (
	"io/ioutil"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// KubebuilderConfigGen implements the API for generating configuration
type KubebuilderConfigGen struct {
	metav1.TypeMeta `json:",inline" yaml:",omitempty"`

	// ObjectMeta has metadata about the object
	ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the configuration spec defining what configuration should be produced.
	Spec KubebuilderConfigGenSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Status is the configuration status defined at runtime.
	Status KubebuilderConfigGenStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// ObjectMeta contains metadata about the resource
type ObjectMeta struct {
	// Name is used to generate the names of resources.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Namespace defines the namespace for the controller resources.
	// Must be a DNS_LABEL.
	// More info: http://kubernetes.io/docs/user-guide/namespaces
	// Defaults to "${name}-system" -- e.g. if name is "foo", then namespace defaults
	// to "foo-system"
	// +optional
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// KubebuilderConfigGenSpec defines the desired configuration to be generated
type KubebuilderConfigGenSpec struct {
	// CRDs configures how CRDs + related RBAC and Webhook resources are generated.
	CRDs CRDs `json:"crds,omitempty" yaml:"crds,omitempty"`

	// ControllerManager configures how the controller-manager Deployment is generated.
	ControllerManager ControllerManager `json:"controllerManager,omitempty" yaml:"controllerManager,omitempty"`

	// Webhooks configures how webhooks and certificates are generated.
	Webhooks Webhooks `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
}

// CRDs configures how controller-gen is run against the project go source code in order to generate CRDs and RBAC.
type CRDs struct {
	// SourceDirectory is the go project directory containing source code marked up with controller-gen tags.
	// Defaults to the directory containing the KubebuilderConfigGen configuration file.
	// +optional
	SourceDirectory string `json:"sourceDirectory,omitempty" yaml:"sourceDirectory,omitempty"`
}

// ControllerManager configures how the controller-manager resources are generated.
type ControllerManager struct {
	// Image is the container image to run as the controller-manager.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Metrics configures how prometheus metrics are exposed.
	Metrics Metrics `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// ComponentConfig configures how the controller-manager is configured.
	// +optional
	ComponentConfig ComponentConfig `json:"componentConfig,omitempty" yaml:"componentConfig,omitempty"`
}

// Metrics configures how prometheus metrics are exposed from the controller.
type Metrics struct {
	// DisableAuthProxy if set to true will disable the auth proxy
	// +optional
	DisableAuthProxy bool `json:"disableAuthProxy,omitempty" yaml:"disableAuthProxy,omitempty"`

	// EnableServiceMonitor if set to true with generate the prometheus ServiceMonitor resource
	// +optional
	EnableServiceMonitor bool `json:"enableServiceMonitor,omitempty" yaml:"enableServiceMonitor,omitempty"`
}

// ComponentConfig configures how to setup the controller-manager to use component config rather
// than flag driven options.
type ComponentConfig struct {
	// Enable if set to true will use component config rather than flags.
	Enable bool `json:"enable,omitempty" yaml:"enable,omitempty"`

	// ConfigFilepath is the relative path to a file containing component config.
	ConfigFilepath string `json:"configFilepath,omitempty" yaml:"configFilepath,omitempty"`
}

// Webhooks configures how webhooks are generated.
type Webhooks struct {
	// Enable if set to true will generate webhook configurations.
	Enable bool `json:"enable,omitempty" yaml:"enable,omitempty"`

	// Conversions configures which resource types to enable conversion webhooks for.
	// Conversion will be set in the CRDs for these resource types.
	// The key is the CRD name.
	// Note: This is a map rather than a list so it can be overridden when patched or merged.
	Conversions map[string]bool `json:"conversions,omitempty" yaml:"conversions,omitempty"`

	// CertificateSource defines where to get the webhook certificates from.
	CertificateSource CertificateSource `json:"certificateSource,omitempty" yaml:"certificateSource,omitempty"`
}

// CertificateSource configures where to get webhook certificates from.
// It is a discriminated union.
type CertificateSource struct {
	// Type is a discriminator for this union.
	// One of: ["certManager", "dev", "manual"].
	// Defaults to "manual".
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// ManualCertificate requires the user to provide a certificate.
	// Requires "manual" as the type.
	ManualCertificate *ManualCertificate `json:"manualCertificate,omitempty" yaml:"manualCertificate,omitempty"`

	// CertManagerCertificate relies on the certificate manager operator installed separately.
	// Requires "certManager" as the type.
	//nolint:lll
	CertManagerCertificate *CertManagerCertificate `json:"certManagerCertificate,omitempty" yaml:"certManagerCertificate,omitempty"`

	// GenerateCert will generate self signed certificate and inject it into the caBundles.
	// For development only, not a production grade solution.
	// Requires "dev" as the type.
	DevCertificate *DevCertificate `json:"devCertificate,omitempty" yaml:"devCertificate,omitempty"`
}

// ManualCertificate will not generate any certificate, and requires the user to manually
// specify and wire one in.
type ManualCertificate struct {
	// Placeholder for future options
	// TODO: Consider allowing users to specify the path to a file containing a certificate
}

// CertManagerCertificate will generate cert-manager.io/v1 Issuer and Certificate resources.
type CertManagerCertificate struct {
	// Placeholder for future options
}

// DevCertificate generates a certificate for development purposes and wires it into the appropriate locations.
type DevCertificate struct {
	// CertDuration sets the duration for the generated cert.  Defaults to 1 hour.
	CertDuration time.Duration `json:"certDuration,omitempty" yaml:"certDuration,omitempty"`
}

// KubebuilderConfigGenStatus is runtime status for the api configuration.
// It is used to pass values generated at runtime (not directly specified by users)
// to templates.
type KubebuilderConfigGenStatus struct {
	// CertCA is the CertCA generated at runtime.
	CertCA string

	// CertKey is the CertKey generated at runtime.
	CertKey string

	// ComponentConfigString is the contents of the component config file read from disk.
	ComponentConfigString string
}

// Default defaults the values
func (kp *KubebuilderConfigGen) Default() error {
	// Validate the input
	if kp.Name == "" {
		return errors.Errorf("must specify metadata.name field")
	}
	if kp.Spec.ControllerManager.Image == "" {
		return errors.Errorf("must specify spec.controllerManager.image field")
	}

	// Perform defaulting
	if kp.Namespace == "" {
		kp.Namespace = kp.Name + "-system"
	}

	if kp.Spec.CRDs.SourceDirectory == "" {
		kp.Spec.CRDs.SourceDirectory = "./..."
	}

	if kp.Spec.ControllerManager.ComponentConfig.ConfigFilepath != "" {
		b, err := ioutil.ReadFile(kp.Spec.ControllerManager.ComponentConfig.ConfigFilepath)
		if err != nil {
			return err
		}
		kp.Status.ComponentConfigString = string(b)
	}

	if kp.Spec.Webhooks.CertificateSource.Type == "dev" {
		if kp.Spec.Webhooks.CertificateSource.DevCertificate == nil {
			kp.Spec.Webhooks.CertificateSource.DevCertificate = &DevCertificate{}
		}
		if kp.Spec.Webhooks.CertificateSource.DevCertificate.CertDuration == 0 {
			kp.Spec.Webhooks.CertificateSource.DevCertificate.CertDuration = time.Hour
		}
	}

	return nil
}
