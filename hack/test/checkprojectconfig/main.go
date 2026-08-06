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

// PROJECT config comparer checks that `alpha generate` rebuilt a project from the
// same configuration it started with. It loads both PROJECT files through the
// config store and compares the decoded values, so key order and formatting in
// the file cannot make the check pass or fail on their own.
//
// Resources are compared as a set. The scaffold replays them in the order the
// config lists them, which need not match how the original project was built.
//
// Run with:
//   go run ./hack/test/checkprojectconfig <original-project-dir> <regenerated-project-dir>

package main

import (
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	_ "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // registers project version 3 so the store can decode PROJECT
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

func main() {
	if len(os.Args) != 3 {
		log.Error("usage: checkprojectconfig <original-project-dir> <regenerated-project-dir>")
		os.Exit(1)
	}

	original, err := loadConfig(os.Args[1])
	if err != nil {
		log.Error("failed to load the original PROJECT file", "error", err)
		os.Exit(1)
	}

	regenerated, err := loadConfig(os.Args[2])
	if err != nil {
		log.Error("failed to load the regenerated PROJECT file", "error", err)
		os.Exit(1)
	}

	diffs, err := compare(original, regenerated)
	if err != nil {
		log.Error("failed to compare the PROJECT files", "error", err)
		os.Exit(1)
	}

	if len(diffs) > 0 {
		log.Error("alpha generate did not preserve the project configuration")
		for _, diff := range diffs {
			fmt.Fprintf(os.Stderr, "  - %s\n", diff)
		}
		os.Exit(1)
	}

	resources, err := original.GetResources()
	if err != nil {
		log.Error("failed to read resources", "error", err)
		os.Exit(1)
	}

	log.Info("PROJECT configuration preserved by alpha generate", "resources", len(resources))
}

func loadConfig(dir string) (config.Config, error) {
	store := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := store.LoadFrom(filepath.Join(dir, yaml.DefaultPath)); err != nil {
		return nil, fmt.Errorf("failed to load the PROJECT file under %q: %w", dir, err)
	}

	return store.Config(), nil
}

func compare(original, regenerated config.Config) ([]string, error) {
	var diffs []string

	scalar := func(name, want, got string) {
		if want != got {
			diffs = append(diffs, fmt.Sprintf("%s: expected %q, got %q", name, want, got))
		}
	}

	scalar("domain", original.GetDomain(), regenerated.GetDomain())
	scalar("repository", original.GetRepository(), regenerated.GetRepository())
	scalar("projectName", original.GetProjectName(), regenerated.GetProjectName())
	scalar("version", original.GetVersion().String(), regenerated.GetVersion().String())
	scalar("layout", strings.Join(original.GetPluginChain(), ","),
		strings.Join(regenerated.GetPluginChain(), ","))

	if original.IsMultiGroup() != regenerated.IsMultiGroup() {
		diffs = append(diffs, fmt.Sprintf("multigroup: expected %t, got %t",
			original.IsMultiGroup(), regenerated.IsMultiGroup()))
	}

	originalResources, err := original.GetResources()
	if err != nil {
		return nil, fmt.Errorf("read resources from the original PROJECT file: %w", err)
	}

	regeneratedResources, err := regenerated.GetResources()
	if err != nil {
		return nil, fmt.Errorf("read resources from the regenerated PROJECT file: %w", err)
	}

	return append(diffs, compareResources(originalResources, regeneratedResources)...), nil
}

func compareResources(original, regenerated []resource.Resource) []string {
	want := resourceKeys(original)
	got := resourceKeys(regenerated)

	var diffs []string
	for key := range want {
		if _, ok := got[key]; !ok {
			diffs = append(diffs, "resource missing after regeneration: "+key)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			diffs = append(diffs, "unexpected resource after regeneration: "+key)
		}
	}
	slices.Sort(diffs)

	return diffs
}

// resourceKeys renders each resource as a single comparable string so resources
// can be matched regardless of the order they appear in the config.
func resourceKeys(resources []resource.Resource) map[string]struct{} {
	keys := make(map[string]struct{}, len(resources))
	for _, res := range resources {
		key := fmt.Sprintf("%s/%s/%s controller=%t webhooks=%s",
			res.Group, res.Version, res.Kind, res.Controller, webhooks(res))
		keys[key] = struct{}{}
	}

	return keys
}

func webhooks(res resource.Resource) string {
	if res.Webhooks == nil || res.Webhooks.IsEmpty() {
		return "none"
	}

	var enabled []string
	if res.Webhooks.Defaulting {
		enabled = append(enabled, "defaulting")
	}
	if res.Webhooks.Validation {
		enabled = append(enabled, "validation")
	}
	if res.Webhooks.Conversion {
		enabled = append(enabled, "conversion")
	}
	if len(enabled) == 0 {
		return "none"
	}
	slices.Sort(enabled)

	return strings.Join(enabled, "+")
}
