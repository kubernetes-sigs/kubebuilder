/*
Copyright 2022 The Kubernetes Authors.

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

package util

import (
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

// Deprecated: go/v4 no longer supports v1beta1 option
// HasDifferentCRDVersion returns true if any other CRD version is tracked in the project configuration.
func HasDifferentCRDVersion(config config.Config, crdVersion string) bool {
	return hasDifferentAPIVersion(config.ListCRDVersions(), crdVersion)
}

// Deprecated: go/v4 no longer supports v1beta1 option
// HasDifferentWebhookVersion returns true if any other webhook version is tracked in the project configuration.
func HasDifferentWebhookVersion(config config.Config, webhookVersion string) bool {
	return hasDifferentAPIVersion(config.ListWebhookVersions(), webhookVersion)
}

// Deprecated: go/v4 no longer supports v1beta1 option
func hasDifferentAPIVersion(versions []string, version string) bool {
	return !(len(versions) == 0 || (len(versions) == 1 && versions[0] == version))
}
