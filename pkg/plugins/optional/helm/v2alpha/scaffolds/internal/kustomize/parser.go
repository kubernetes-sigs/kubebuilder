/*
Copyright 2025 The Kubernetes Authors.

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

package kustomize

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParsedResources holds Kubernetes resources organized by type for Helm chart generation
type ParsedResources struct {
	// Core Kubernetes resources
	Namespace  *unstructured.Unstructured
	Deployment *unstructured.Unstructured
	Services   []*unstructured.Unstructured

	// RBAC resources
	ServiceAccount      *unstructured.Unstructured
	Roles               []*unstructured.Unstructured
	ClusterRoles        []*unstructured.Unstructured
	RoleBindings        []*unstructured.Unstructured
	ClusterRoleBindings []*unstructured.Unstructured

	// CRD and API resources
	CustomResourceDefinitions []*unstructured.Unstructured
	WebhookConfigurations     []*unstructured.Unstructured

	// Cert-manager resources
	Certificates []*unstructured.Unstructured
	Issuer       *unstructured.Unstructured

	// Monitoring resources
	ServiceMonitors []*unstructured.Unstructured

	// Sample Custom Resources (CR instances from config/samples)
	SampleResources []*unstructured.Unstructured

	// Other resources not fitting above categories
	Other []*unstructured.Unstructured

	// definedCRDTypes tracks GVK (Group/Version/Kind) of CRDs defined in this kustomize output
	// Used to identify which resources are instances of these CRDs (samples)
	definedCRDTypes map[string]bool
}

// Parser parses kustomize output and extracts resources by type
type Parser struct {
	filePath string
}

// NewParser creates a new parser for the given kustomize output file
func NewParser(filePath string) *Parser {
	return &Parser{filePath: filePath}
}

// Parse reads and parses the kustomize output file into organized resource groups
func (p *Parser) Parse() (*ParsedResources, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", p.filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	return p.ParseFromReader(file)
}

// ParseFromReader parses multi-document YAML from a reader and categorizes resources by type
func (p *Parser) ParseFromReader(reader io.Reader) (*ParsedResources, error) {
	decoder := yaml.NewDecoder(reader)
	resources := &ParsedResources{
		CustomResourceDefinitions: make([]*unstructured.Unstructured, 0),
		Roles:                     make([]*unstructured.Unstructured, 0),
		ClusterRoles:              make([]*unstructured.Unstructured, 0),
		RoleBindings:              make([]*unstructured.Unstructured, 0),
		ClusterRoleBindings:       make([]*unstructured.Unstructured, 0),
		Services:                  make([]*unstructured.Unstructured, 0),
		Certificates:              make([]*unstructured.Unstructured, 0),
		WebhookConfigurations:     make([]*unstructured.Unstructured, 0),
		ServiceMonitors:           make([]*unstructured.Unstructured, 0),
		SampleResources:           make([]*unstructured.Unstructured, 0),
		Other:                     make([]*unstructured.Unstructured, 0),
		definedCRDTypes:           make(map[string]bool),
	}

	// First pass: collect all resources
	var allResources []*unstructured.Unstructured

	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML document: %w", err)
		}

		// Skip empty documents
		if doc == nil {
			continue
		}

		obj := &unstructured.Unstructured{Object: doc}
		allResources = append(allResources, obj)
	}

	// Second pass: build CRD type map (extract GVK from all CRDs)
	const kindCRD = "CustomResourceDefinition"
	for _, obj := range allResources {
		if obj.GetKind() == kindCRD {
			p.extractCRDType(obj, resources)
		}
	}

	// Third pass: categorize all resources (now we know which CRDs are defined)
	for _, obj := range allResources {
		p.categorizeResource(obj, resources)
	}

	return resources, nil
}

// extractCRDType extracts the group/version/kind from a CRD and stores it in the definedCRDTypes map
func (p *Parser) extractCRDType(crd *unstructured.Unstructured, resources *ParsedResources) {
	// Extract group from spec.group
	group, found, err := unstructured.NestedString(crd.Object, "spec", "group")
	if !found || err != nil {
		return
	}

	// Extract versions from spec.versions (use NestedFieldNoCopy to avoid deep copy issues)
	versionsField, found, err := unstructured.NestedFieldNoCopy(crd.Object, "spec", "versions")
	if !found || err != nil {
		return
	}

	versions, ok := versionsField.([]any)
	if !ok {
		return
	}

	// Extract kind from spec.names.kind
	kind, found, err := unstructured.NestedString(crd.Object, "spec", "names", "kind")
	if !found || err != nil {
		return
	}

	// Register all versions of this CRD
	for _, v := range versions {
		versionMap, ok := v.(map[string]any)
		if !ok {
			continue
		}
		versionName, ok := versionMap["name"].(string)
		if !ok {
			continue
		}

		// Create GVK key: group/version/Kind
		gvk := fmt.Sprintf("%s/%s/%s", group, versionName, kind)
		resources.definedCRDTypes[gvk] = true
	}
}

// categorizeResource sorts a Kubernetes resource into the appropriate category based on kind and API version
func (p *Parser) categorizeResource(obj *unstructured.Unstructured, resources *ParsedResources) {
	kind := obj.GetKind()
	apiVersion := obj.GetAPIVersion()

	switch {
	case kind == "Namespace":
		resources.Namespace = obj
	case kind == "CustomResourceDefinition":
		resources.CustomResourceDefinitions = append(resources.CustomResourceDefinitions, obj)
	case kind == "ServiceAccount":
		resources.ServiceAccount = obj
	case kind == "Role":
		resources.Roles = append(resources.Roles, obj)
	case kind == "ClusterRole":
		resources.ClusterRoles = append(resources.ClusterRoles, obj)
	case kind == "RoleBinding":
		resources.RoleBindings = append(resources.RoleBindings, obj)
	case kind == "ClusterRoleBinding":
		resources.ClusterRoleBindings = append(resources.ClusterRoleBindings, obj)
	case kind == "Service":
		resources.Services = append(resources.Services, obj)
	case kind == "Deployment":
		resources.Deployment = obj
	case kind == "Certificate" && apiVersion == "cert-manager.io/v1":
		resources.Certificates = append(resources.Certificates, obj)
	case kind == "Issuer" && apiVersion == "cert-manager.io/v1":
		resources.Issuer = obj
	case kind == "ValidatingWebhookConfiguration" || kind == "MutatingWebhookConfiguration":
		resources.WebhookConfigurations = append(resources.WebhookConfigurations, obj)
	case kind == "ServiceMonitor" && apiVersion == "monitoring.coreos.com/v1":
		resources.ServiceMonitors = append(resources.ServiceMonitors, obj)
	case p.isSampleCustomResource(obj, resources):
		// Custom Resource instances (from config/samples) should go to samples group
		// These are resources that match CRDs defined in this kustomize output
		resources.SampleResources = append(resources.SampleResources, obj)
	default:
		resources.Other = append(resources.Other, obj)
	}
}

// isSampleCustomResource determines if a resource is a Custom Resource instance (sample)
// It checks if the resource is an instance of a CRD defined in the kustomize output
func (p *Parser) isSampleCustomResource(obj *unstructured.Unstructured, resources *ParsedResources) bool {
	kind := obj.GetKind()
	apiVersion := obj.GetAPIVersion()

	// Build GVK key: group/version/Kind
	// apiVersion format is either "group/version" or just "version" for core types
	gvk := fmt.Sprintf("%s/%s", apiVersion, kind)

	// Check if this resource type matches any defined CRD
	return resources.definedCRDTypes[gvk]
}

func (pr *ParsedResources) EstimatePrefix(projectName string) string {
	prefix := projectName
	if pr.Deployment != nil {
		if name := pr.Deployment.GetName(); name != "" {
			deploymentPrefix, found := strings.CutSuffix(name, "-controller-manager")
			if found {
				prefix = deploymentPrefix
			}
		}
	}
	// Double check that the prefix is also the prefix for the service names
	for _, svc := range pr.Services {
		if name := svc.GetName(); name != "" {
			if !strings.HasPrefix(name, prefix) {
				// If not, fallback to just project name
				prefix = projectName
				break
			}
		}
	}
	return prefix
}
