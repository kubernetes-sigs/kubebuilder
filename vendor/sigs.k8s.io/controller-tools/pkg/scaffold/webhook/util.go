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

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/resource"
)

// Use the k8s.io/api package for core resources
var coreGroups = map[string]string{
	"apps":                  "",
	"admissionregistration": "k8s.io",
	"apiextensions":         "k8s.io",
	"authentication":        "k8s.io",
	"autoscaling":           "",
	"batch":                 "",
	"certificates":          "k8s.io",
	"core":                  "",
	"extensions":            "",
	"metrics":               "k8s.io",
	"policy":                "",
	"rbac.authorization":    "k8s.io",
	"storage":               "k8s.io",
}

func builderName(config Config, resource string) string {
	opsStr := strings.Join(config.Operations, "-")
	return fmt.Sprintf("%s-%s-%s", config.Type, opsStr, resource)
}

func getResourceInfo(coreGroups map[string]string, r *resource.Resource, in input.Input) (resourcePackage, groupDomain string) {
	resourcePath := filepath.Join("pkg", "apis", r.Group, r.Version,
		fmt.Sprintf("%s_types.go", strings.ToLower(r.Kind)))
	if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
		if domain, found := coreGroups[r.Group]; found {
			resourcePackage := path.Join("k8s.io", "api")
			groupDomain = r.Group
			if domain != "" {
				groupDomain = r.Group + "." + domain
			}
			return resourcePackage, groupDomain
		}
		// TODO: need to support '--resource-pkg-path' flag for specifying resourcePath
	}
	return path.Join(in.Repo, "pkg", "apis"), r.Group + "." + in.Domain
}
