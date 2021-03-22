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

package util

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// HasDifferentCRDVersion returns true if any other CRD version is tracked in the project configuration.
func HasDifferentCRDVersion(config config.Config, crdVersion string) bool {
	return hasDifferentAPIVersion(config.ListCRDVersions(), crdVersion)
}

// HasDifferentWebhookVersion returns true if any other webhook version is tracked in the project configuration.
func HasDifferentWebhookVersion(config config.Config, webhookVersion string) bool {
	return hasDifferentAPIVersion(config.ListWebhookVersions(), webhookVersion)
}

func hasDifferentAPIVersion(versions []string, version string) bool {
	return !(len(versions) == 0 || (len(versions) == 1 && versions[0] == version))
}

// CategorizeHubAndSpokes returns the hub and spoke versions present in the config. By design,
// we currently support one hub for the project. If multiple hubs are found in the config, an error
// is returned (there are checks in place which verify that multiple hubs are not added while scaffolding
// the webhook itself).
func CategorizeHubAndSpokes(cfg config.Config, gvk resource.GVK) (hub string, spokes []string, err error) {
	return categorizeHubAndSpokes(cfg.ListResourceswithGK(gvk))
}

func categorizeHubAndSpokes(resources []resource.Resource) (hub string, spokes []string, err error) {
	hub = ""
	spokes = make([]string, 0)

	for _, res := range resources {
		if res.Webhooks != nil && len(res.Webhooks.Spokes) != 0 {
			if hub != "" && res.Version != hub {
				return "", nil, fmt.Errorf("multiples hubs are not allowed, found %s and %s", hub, res.Version)
			}
			hub = res.Version
			for _, s := range res.Webhooks.Spokes {
				spokes = append(spokes, s)
			}
		}
	}
	return hub, spokes, nil
}
