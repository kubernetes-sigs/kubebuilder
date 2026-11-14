//go:build integration

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

package test

import (
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v3"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	sigsyaml "sigs.k8s.io/yaml"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds"
)

const (
	sampleProject = "project-v4-with-plugins"
	releaseName   = "release"
	setByValues   = "from-values"
	setFlag       = "--set"
	overrideName  = "custom"
	enableMetrics = "metrics.enabled=" + "true"

	kindServiceAccount = "ServiceAccount"
	kindServiceMonitor = "ServiceMonitor"
	kindDeployment     = "Deployment"
)

// kustomizeOverlay is a customization of the sample project's config/: edits to
// config/default/kustomization.yaml and an overlay on top of it.
type kustomizeOverlay struct {
	prefix  string
	edits   map[string]string
	overlay string
}

// chartOptions are the values a user passes to helm.
type chartOptions struct {
	args []string
	// collide sets every label and annotation kustomize wrote on the resources the values reach.
	collide bool
	// name is the value of the app.kubernetes.io/name label.
	name string
	// noServiceAccount and noServiceMonitor mean the chart does not create them.
	noServiceAccount bool
}

// renderedObject is the part of a Kubernetes object the specs read.
type renderedObject struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name        string            `json:"name"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Selector    json.RawMessage `json:"selector"`
		PodSelector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"podSelector"`
		Template struct {
			Metadata struct {
				Labels      map[string]string `json:"labels"`
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		} `json:"template"`
	} `json:"spec"`
}

// decodeObjects decodes a multi-document YAML stream as Helm and kubectl do. Strict decoding fails on
// a key written twice.
func decodeObjects(stream string) []renderedObject {
	var objects []renderedObject
	for doc := range strings.SplitSeq(stream, "\n---") {
		data, err := sigsyaml.YAMLToJSONStrict([]byte(doc))
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "document:\n%s", doc)
		var object renderedObject
		ExpectWithOffset(1, json.Unmarshal(data, &object)).To(Succeed())
		if object.Kind != "" {
			objects = append(objects, object)
		}
	}
	return objects
}

// repoRoot is the root of the kubebuilder repository.
func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..", "..")
}

// kustomizeBuild runs kustomize on a copy of the sample project's config/ with the customization.
func kustomizeBuild(customization kustomizeOverlay) string {
	fSys := filesys.MakeFsInMemory()
	root := filepath.Join(repoRoot(), "testdata", sampleProject, "config")
	Expect(filepath.WalkDir(root, func(path string, entry iofs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("finding %s under %s: %w", path, root, err)
		}
		return fSys.WriteFile(filepath.Join("/config", rel), data)
	})).To(Succeed())

	const kustomization = "/config/default/kustomization.yaml"
	data, err := fSys.ReadFile(kustomization)
	Expect(err).NotTo(HaveOccurred())
	content := string(data)
	for old, replacement := range customization.edits {
		Expect(content).To(ContainSubstring(old))
		content = strings.Replace(content, old, replacement, 1)
	}
	Expect(fSys.WriteFile(kustomization, []byte(content))).To(Succeed())
	Expect(fSys.WriteFile("/config/overlay/kustomization.yaml",
		[]byte(overlayBase+customization.overlay))).To(Succeed())

	resources, err := krusty.MakeKustomizer(krusty.MakeDefaultOptions()).Run(fSys, "/config/overlay")
	Expect(err).NotTo(HaveOccurred())
	out, err := resources.AsYaml()
	Expect(err).NotTo(HaveOccurred())
	return string(out)
}

// suffix returns the name of a kustomize object without the namePrefix, which the chart replaces.
func suffix(name, prefix string) string {
	return strings.TrimPrefix(name, prefix)
}

// findRendered returns the rendered object the kustomize object became, if the chart renders it.
func findRendered(rendered []renderedObject, kind, nameSuffix string) (renderedObject, bool) {
	for _, object := range rendered {
		if object.Kind == kind && (object.Metadata.Name == nameSuffix ||
			strings.HasSuffix(object.Metadata.Name, "-"+nameSuffix)) {
			return object, true
		}
	}
	return renderedObject{}, false
}

// selectorLabels returns the labels a Service selector, or a matchLabels selector, matches.
func selectorLabels(object renderedObject) map[string]string {
	if len(object.Spec.Selector) == 0 {
		return nil
	}
	var deployment struct {
		MatchLabels map[string]string `json:"matchLabels"`
	}
	if object.Kind == "Service" {
		var service map[string]string
		Expect(json.Unmarshal(object.Spec.Selector, &service)).To(Succeed())
		return service
	}
	Expect(json.Unmarshal(object.Spec.Selector, &deployment)).To(Succeed())
	return deployment.MatchLabels
}

// chartOwned reports whether the chart rewrites a label or annotation from config/: the Helm labels,
// and values that name an object or namespace kustomize prefixed, which the chart templates.
func chartOwned(key, value, prefix, namespace string) bool {
	switch key {
	case "app.kubernetes.io/managed-by", "app.kubernetes.io/instance", "helm.sh/chart":
		return true
	case "app.kubernetes.io/name":
		return value == strings.TrimSuffix(prefix, "-")
	}
	return strings.Contains(value, prefix) || strings.Contains(value, namespace)
}

var _ = Describe("Labels and annotations from kustomize customizations (rendered)", func() {
	var tmpDir string

	BeforeEach(func() {
		if _, err := exec.LookPath("helm"); err != nil {
			Skip("helm binary not found on PATH; skipping render-based test")
		}
		var err error
		tmpDir, err = os.MkdirTemp("", "helm-labels-test-*")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(tmpDir)).To(Succeed())
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	// render scaffolds the chart from the kustomize output and runs helm template with args.
	render := func(manifests string, args ...string) string {
		manifestsFile := filepath.Join(tmpDir, "dist", "install.yaml")
		Expect(os.MkdirAll(filepath.Dir(manifestsFile), 0o755)).To(Succeed())
		Expect(os.WriteFile(manifestsFile, []byte(manifests), 0o644)).To(Succeed())

		projectConfig := cfgv3.New()
		Expect(projectConfig.SetProjectName(sampleProject)).To(Succeed())
		Expect(projectConfig.SetDomain("testproject.org")).To(Succeed())
		scaffolder := scaffolds.NewChartScaffolder(projectConfig, true, manifestsFile, "dist")
		scaffolder.InjectFS(machinery.Filesystem{FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir)})
		Expect(scaffolder.Scaffold()).To(Succeed())

		helmArgs := append([]string{
			"template", releaseName, filepath.Join(tmpDir, "dist", "chart"),
			"--namespace", "release-system",
		}, args...)
		out, err := exec.Command("helm", helmArgs...).CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "helm template failed: %s", out)
		return string(out)
	}

	// keepsAndReplaces renders the chart for one customization and one set of options.
	keepsAndReplaces := func(customization kustomizeOverlay, options chartOptions) {
		manifests := kustomizeBuild(customization)
		source := decodeObjects(manifests)
		namespace := "project-v4-with-plugins-system"

		args := append([]string{
			setFlag, enableMetrics, setFlag, "prometheus.enabled=true",
		}, options.args...)

		// Values that set every label and annotation kustomize wrote on the resources they reach,
		// plus a new key. Pod labels a selector matches stay as they are, so the pods stay selected.
		type metadataValues struct {
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		}
		collide := func(labels, annotations, keep map[string]string) metadataValues {
			values := metadataValues{
				Labels:      map[string]string{"example.com/added": setByValues},
				Annotations: map[string]string{"example.com/added": setByValues},
			}
			for key := range labels {
				if _, ok := keep[key]; !ok {
					values.Labels[key] = setByValues
				}
			}
			for key := range annotations {
				values.Annotations[key] = setByValues
			}
			return values
		}
		valuesFor := map[string]metadataValues{}
		if options.collide {
			values := map[string]any{}
			for _, object := range source {
				switch object.Kind {
				case kindServiceAccount:
					valuesFor[kindServiceAccount] = collide(object.Metadata.Labels, object.Metadata.Annotations, nil)
					values["serviceAccount"] = valuesFor[kindServiceAccount]
				case kindServiceMonitor:
					valuesFor[kindServiceMonitor] = collide(object.Metadata.Labels, object.Metadata.Annotations, nil)
					values["prometheus"] = valuesFor[kindServiceMonitor]
				case kindDeployment:
					valuesFor[kindDeployment] = collide(object.Metadata.Labels, object.Metadata.Annotations, nil)
					valuesFor["Pod"] = collide(object.Spec.Template.Metadata.Labels,
						object.Spec.Template.Metadata.Annotations, selectorLabels(object))
					values["manager"] = map[string]any{
						"labels":      valuesFor[kindDeployment].Labels,
						"annotations": valuesFor[kindDeployment].Annotations,
						"pod":         valuesFor["Pod"],
					}
				}
			}
			data, err := yaml.Marshal(values)
			Expect(err).NotTo(HaveOccurred())
			valuesFile := filepath.Join(tmpDir, "values.yaml")
			Expect(os.WriteFile(valuesFile, data, 0o644)).To(Succeed())
			args = append(args, "-f", valuesFile)
		}

		rendered := decodeObjects(render(manifests, args...))

		// check compares one labels or annotations map with what config/ and the values set.
		check := func(where string, got, fromConfig, fromValues map[string]string, chartLabels bool) {
			want := map[string]string{}
			for key, value := range fromConfig {
				if !chartOwned(key, value, customization.prefix, namespace) {
					want[key] = value
				}
			}
			maps.Copy(want, fromValues)
			for key, value := range want {
				Expect(got).To(HaveKeyWithValue(key, value), "%s", where)
			}
			if chartLabels {
				if _, set := fromValues["app.kubernetes.io/managed-by"]; !set {
					Expect(got).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"), "%s", where)
				}
				if _, set := fromValues["app.kubernetes.io/instance"]; !set {
					Expect(got).To(HaveKeyWithValue("app.kubernetes.io/instance", releaseName), "%s", where)
				}
				if value, ok := fromConfig["app.kubernetes.io/name"]; ok && value == strings.TrimSuffix(customization.prefix, "-") {
					if _, set := fromValues["app.kubernetes.io/name"]; !set {
						Expect(got).To(HaveKeyWithValue("app.kubernetes.io/name", options.name), "%s", where)
					}
				}
			}
		}

		for _, object := range source {
			nameSuffix := suffix(object.Metadata.Name, customization.prefix)
			got, ok := findRendered(rendered, object.Kind, nameSuffix)
			if !ok {
				if object.Kind == kindServiceAccount && options.noServiceAccount {
					continue
				}
				Expect(slices.Contains([]string{kindServiceAccount, kindServiceMonitor, kindDeployment}, object.Kind)).
					To(BeFalse(), "%s %s is not in the chart", object.Kind, nameSuffix)
				continue
			}
			where := object.Kind + " " + nameSuffix
			values := valuesFor[object.Kind]
			valuesReach := slices.Contains([]string{kindServiceAccount, kindServiceMonitor, kindDeployment}, object.Kind)
			check(where+" labels", got.Metadata.Labels, object.Metadata.Labels, values.Labels, valuesReach)
			check(where+" annotations", got.Metadata.Annotations, object.Metadata.Annotations,
				values.Annotations, false)
			if object.Kind == kindDeployment {
				pod := valuesFor["Pod"]
				check(where+" pod labels", got.Spec.Template.Metadata.Labels,
					object.Spec.Template.Metadata.Labels, pod.Labels, true)
				check(where+" pod annotations", got.Spec.Template.Metadata.Annotations,
					object.Spec.Template.Metadata.Annotations, pod.Annotations, false)
			}
		}

		// The manager pods stay selected by the Deployment, the Services and the NetworkPolicies.
		var podLabels map[string]string
		for _, object := range rendered {
			if object.Kind == kindDeployment {
				podLabels = object.Spec.Template.Metadata.Labels
			}
		}
		Expect(podLabels).NotTo(BeEmpty())
		for _, object := range rendered {
			selector := selectorLabels(object)
			if object.Kind == "NetworkPolicy" {
				selector = object.Spec.PodSelector.MatchLabels
			}
			for key, value := range selector {
				if object.Kind == "Service" && !strings.Contains(object.Metadata.Name, "controller-manager") &&
					!strings.Contains(object.Metadata.Name, "webhook") {
					continue
				}
				Expect(podLabels).To(HaveKeyWithValue(key, value), "%s %s selector", object.Kind, object.Metadata.Name)
			}
		}
	}

	DescribeTable("keeps every label and annotation from config/, and the values replace them",
		append([]any{keepsAndReplaces}, customizationEntries()...)...)
})

// Overlays on top of the sample project's config/default, one per kustomize customization.
const (
	overlayBase = `resources:
- ../default
`
	labelsOnMetadata = `
labels:
- pairs:
    example.com/team: a
`
	labelsWithSelectors = `
labels:
- pairs:
    example.com/team: a
  includeSelectors: true
`
	labelsWithTemplates = `
labels:
- pairs:
    example.com/team: a
  includeTemplates: true
`
	commonLabels = `
commonLabels:
  example.com/team: a
`
	commonAnnotations = `
commonAnnotations:
  example.com/note: |
    line one
    line two
  "on": original
`
	// everyShapePatches adds labels and annotations of hard shapes to the ServiceAccount, the
	// ServiceMonitor, the manager Deployment and its pods.
	everyShapePatches = `
patches:
- target:
    kind: ServiceAccount
  patch: |-
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: any
      labels:
        example.com/tier: a
        "123": "007"
        Upper_Case-key.1: "true"
      annotations:
        example.com/note: |
          line one
          labels:
        "on": original
        example.com/empty: ""
- target:
    kind: ServiceMonitor
  patch: |-
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: any
      labels:
        example.com/tier: a
        "123": "007"
        Upper_Case-key.1: "true"
      annotations:
        example.com/note: |
          line one
          labels:
        "on": original
        example.com/empty: ""
- target:
    kind: Deployment
  patch: |-
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: any
      labels:
        example.com/tier: a
        "123": "007"
        Upper_Case-key.1: "true"
      annotations:
        example.com/note: |
          line one
          labels:
        "on": original
        example.com/empty: ""
    spec:
      template:
        metadata:
          labels:
            example.com/tier: a
          annotations:
            example.com/note: |
              line one
              spec:
`
)

// customizationEntries crosses each kustomize customization with each set of chart options.
func customizationEntries() []any {
	prefix := sampleProject + "-"
	withPrometheus := map[string]string{"#- ../prometheus": "- ../prometheus"}
	edits := func(more map[string]string) map[string]string {
		merged := maps.Clone(withPrometheus)
		maps.Copy(merged, more)
		return merged
	}
	overlay := func(content string) kustomizeOverlay {
		return kustomizeOverlay{prefix: prefix, edits: edits(nil), overlay: content}
	}

	customizations := []struct {
		name string
		kustomizeOverlay
	}{
		{"as scaffolded", overlay("")},
		{"another namePrefix", kustomizeOverlay{prefix: "team-a-", edits: edits(map[string]string{
			"namePrefix: " + prefix: "namePrefix: team-a-",
		})}},
		{"labels on metadata only", overlay(labelsOnMetadata)},
		{"labels with includeSelectors", overlay(labelsWithSelectors)},
		{"labels with includeTemplates", overlay(labelsWithTemplates)},
		{"commonLabels", overlay(commonLabels)},
		{"commonAnnotations", overlay(commonAnnotations)},
		{"patches with labels and annotations of every shape", overlay(everyShapePatches)},
	}

	optionSets := []struct {
		name string
		chartOptions
	}{
		{"no values", chartOptions{name: sampleProject}},
		{"values that set every key", chartOptions{name: sampleProject, collide: true}},
		{"nameOverride", chartOptions{name: overrideName, args: []string{setFlag, "nameOverride=" + overrideName}}},
		{"fullnameOverride", chartOptions{name: sampleProject, args: []string{setFlag, "fullnameOverride=custom-full"}}},
		{"an existing ServiceAccount", chartOptions{
			name: sampleProject, noServiceAccount: true,
			args: []string{setFlag, "serviceAccount.enabled=" + "false", setFlag, "serviceAccount.name=existing"},
		}},
	}

	entries := make([]any, 0, len(customizations)*len(optionSets))
	for _, customization := range customizations {
		for _, options := range optionSets {
			entries = append(entries, Entry(customization.name+", "+options.name,
				customization.kustomizeOverlay, options.chartOptions))
		}
	}
	return entries
}
