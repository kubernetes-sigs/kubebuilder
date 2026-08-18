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

package common

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	managerControlPlaneLabel = "control-plane"
	managerControlPlaneValue = "controller-manager"
	managerNameSubstring     = "controller-manager"
)

// IsManagerDeployment reports whether resource is the controller-manager Deployment.
// The default-container annotation is not checked — extra Deployments may carry it,
// causing false positives. Use GetDefaultContainerName for container resolution instead.
func IsManagerDeployment(resource *unstructured.Unstructured) bool {
	if resource == nil {
		return false
	}
	if resource.GetLabels()[managerControlPlaneLabel] == managerControlPlaneValue {
		return true
	}
	if hasContainerNamed(resource, DefaultManagerContainerName) {
		return true
	}
	return strings.Contains(resource.GetName(), managerNameSubstring)
}

// SelectManagerDeployment returns the controller-manager Deployment from the slice using
// ordered signals across all deployments: label, manager container name, then name suffix.
func SelectManagerDeployment(deployments []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, d := range deployments {
		if d.GetLabels()[managerControlPlaneLabel] == managerControlPlaneValue {
			return d
		}
	}
	for _, d := range deployments {
		if hasContainerNamed(d, DefaultManagerContainerName) {
			return d
		}
	}
	for _, d := range deployments {
		if strings.Contains(d.GetName(), managerNameSubstring) {
			return d
		}
	}
	return nil
}

func hasContainerNamed(resource *unstructured.Unstructured, name string) bool {
	containers, _, _ := unstructured.NestedFieldNoCopy(
		resource.Object, "spec", "template", "spec", "containers")
	containersList, ok := containers.([]any)
	if !ok {
		return false
	}
	for _, c := range containersList {
		container, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if n, ok := container["name"].(string); ok && n == name {
			return true
		}
	}
	return false
}
