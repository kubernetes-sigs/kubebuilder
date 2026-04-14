/*
Copyright 2026 The Kubernetes Authors.

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

// Package common provides shared constants for the Helm v2-alpha plugin.
package common

// DefaultOutputDir is the default output directory for Helm charts.
const DefaultOutputDir = "dist"

// Resource kind constants
const (
	KindNamespace          = "Namespace"
	KindCertificate        = "Certificate"
	KindService            = "Service"
	KindServiceAccount     = "ServiceAccount"
	KindRole               = "Role"
	KindClusterRole        = "ClusterRole"
	KindRoleBinding        = "RoleBinding"
	KindClusterRoleBinding = "ClusterRoleBinding"
	KindServiceMonitor     = "ServiceMonitor"
	KindIssuer             = "Issuer"
	KindValidatingWebhook  = "ValidatingWebhookConfiguration"
	KindMutatingWebhook    = "MutatingWebhookConfiguration"
	KindDeployment         = "Deployment"
	KindCRD                = "CustomResourceDefinition"
)

// API versions
const (
	APIVersionCertManager = "cert-manager.io/v1"
	APIVersionMonitoring  = "monitoring.coreos.com/v1"
)

// YAML keys
const (
	YamlKeyAnnotations = "annotations:"
	YamlKeyLabels      = "labels:"
	YamlKeyMetadata    = "metadata:"
	YamlKeySpec        = "spec:"
	YamlKeyTemplate    = "template:"
)

// Standard Kubernetes/Helm label keys
const (
	LabelKeyAppName      = "app.kubernetes.io/name:"
	LabelKeyAppInstance  = "app.kubernetes.io/instance:"
	LabelKeyAppManagedBy = "app.kubernetes.io/managed-by:"
	LabelKeyHelmChart    = "helm.sh/chart:"
)
