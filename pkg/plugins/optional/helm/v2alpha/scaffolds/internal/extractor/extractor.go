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

// Package extractor extracts metadata, features, and configuration from parsed Kubernetes resources.
//
// The extractor performs pure data extraction with no input/output operations, making it easy to test.
// It coordinates three sub-extractors:
//   - MetadataExtractor: Extracts chart name, prefix, namespace, and manager version
//   - FeaturesExtractor: Detects enabled features (CRDs, webhooks, metrics, Prometheus, cert-manager, RBAC)
//   - DeploymentExtractor: Extracts deployment configuration for values.yaml
//
// The extractor avoids circular dependencies by using its own ResourceSet type instead of
// importing the kustomize package's ParsedResources type.
package extractor

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Extractor orchestrates metadata extraction, feature detection, and deployment config extraction.
type Extractor struct {
	metadataExtractor   *MetadataExtractor
	featuresExtractor   *FeaturesExtractor
	deploymentExtractor *DeploymentExtractor
}

// NewExtractor creates a new extractor with all sub-extractors.
func NewExtractor() *Extractor {
	return &Extractor{
		metadataExtractor:   &MetadataExtractor{},
		featuresExtractor:   &FeaturesExtractor{},
		deploymentExtractor: &DeploymentExtractor{},
	}
}

// Extraction contains all extracted information from parsed resources.
// It includes chart metadata, detected features, and manager configuration for values.yaml.
type Extraction struct {
	Metadata ChartMetadata
	Features FeatureSet
	Values   ValuesConfig
}

// ResourceSet contains all parsed resources needed for analysis.
// This avoids importing the kustomize package and creating a circular dependency.
type ResourceSet struct {
	Namespace                 *unstructured.Unstructured
	Deployment                *unstructured.Unstructured
	Services                  []*unstructured.Unstructured
	CustomResourceDefinitions []*unstructured.Unstructured
	ServiceAccount            *unstructured.Unstructured
	Roles                     []*unstructured.Unstructured
	ClusterRoles              []*unstructured.Unstructured
	RoleBindings              []*unstructured.Unstructured
	ClusterRoleBindings       []*unstructured.Unstructured
	WebhookConfigurations     []*unstructured.Unstructured
	Certificates              []*unstructured.Unstructured
	Issuer                    *unstructured.Unstructured
	ServiceMonitors           []*unstructured.Unstructured
	Other                     []*unstructured.Unstructured
}

// Extract performs complete extraction of parsed resources.
// It extracts metadata first, then detects features, and finally extracts deployment configuration.
// projectName is the configured project name which becomes the chart name.
func (e *Extractor) Extract(resources *ResourceSet, projectName string) *Extraction {
	metadata := e.metadataExtractor.ExtractMetadata(resources, projectName)
	features := e.featuresExtractor.DetectFeatures(resources, metadata.DetectedPrefix, metadata.ManagerNamespace)
	values := e.deploymentExtractor.ExtractDeploymentConfig(resources.Deployment)

	return &Extraction{
		Metadata: metadata,
		Features: features,
		Values:   values,
	}
}
