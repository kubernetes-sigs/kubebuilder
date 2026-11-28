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
	"cmp"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	kindCRD            = "CustomResourceDefinition"
)

// kustomizeOverlay is a customization of the sample project's config/: edits to
// config/default/kustomization.yaml and an overlay on top of it.
type kustomizeOverlay struct {
	prefix  string
	edits   map[string]string
	overlay string
	// namespace is the namespace of the kustomize output, when it is not the scaffolded one.
	namespace string
	// resources are added next to config/default, outside its namePrefix.
	resources string
}

// chartOptions are the values a user passes to helm.
type chartOptions struct {
	args []string
	// collide sets every label and annotation kustomize wrote on the resources the values reach.
	collide bool
	// name is the value of the app.kubernetes.io/name label.
	name string
	// noServiceAccount and noServiceMonitor mean the chart does not create them.
	noServiceAccount, noServiceMonitor bool
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
		PodSelector json.RawMessage `json:"podSelector"`
		Template    struct {
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
	overlay := overlayBase
	if customization.resources != "" {
		Expect(fSys.WriteFile("/config/overlay/extra.yaml", []byte(customization.resources))).To(Succeed())
		overlay += "- extra.yaml\n"
	}
	Expect(fSys.WriteFile("/config/overlay/kustomization.yaml",
		[]byte(overlay+customization.overlay))).To(Succeed())

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

// labelSelector returns the selector an object selects pods with. A Service selector is a map of
// labels. It returns nil when the object has none.
func labelSelector(object renderedObject) labels.Selector {
	raw := object.Spec.Selector
	if object.Kind == "NetworkPolicy" {
		raw = object.Spec.PodSelector
	}
	if len(raw) == 0 {
		return nil
	}
	if object.Kind == "Service" {
		var selector map[string]string
		Expect(json.Unmarshal(raw, &selector)).To(Succeed())
		return labels.SelectorFromSet(selector)
	}
	var selector metav1.LabelSelector
	Expect(json.Unmarshal(raw, &selector)).To(Succeed())
	converted, err := metav1.LabelSelectorAsSelector(&selector)
	Expect(err).NotTo(HaveOccurred())
	return converted
}

// isManager reports whether object is the manager Deployment.
func isManager(object renderedObject) bool {
	return object.Kind == kindDeployment && strings.Contains(object.Metadata.Name, "controller-manager")
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
		namespace := cmp.Or(customization.namespace, sampleProject+"-system")

		args := append([]string{
			setFlag, enableMetrics, setFlag, "prometheus.enabled=true", setFlag, "networkPolicy.enabled=true",
		}, options.args...)

		// Values that set every label and annotation kustomize wrote on the resources they reach,
		// plus a new key. That includes the Helm labels and the pod labels the selectors match.
		type metadataValues struct {
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		}
		collide := func(labels, annotations map[string]string) metadataValues {
			values := metadataValues{
				Labels:      map[string]string{"example.com/added": setByValues},
				Annotations: map[string]string{"example.com/added": setByValues},
			}
			for key := range labels {
				values.Labels[key] = setByValues
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
				switch {
				case object.Kind == kindServiceAccount:
					valuesFor[kindServiceAccount] = collide(object.Metadata.Labels, object.Metadata.Annotations)
					values["serviceAccount"] = valuesFor[kindServiceAccount]
				case object.Kind == kindServiceMonitor:
					valuesFor[kindServiceMonitor] = collide(object.Metadata.Labels, object.Metadata.Annotations)
					values["prometheus"] = valuesFor[kindServiceMonitor]
				case object.Kind == kindCRD:
					// One set of values reaches every CRD.
					crd := collide(object.Metadata.Labels, object.Metadata.Annotations)
					maps.Copy(crd.Labels, valuesFor[kindCRD].Labels)
					maps.Copy(crd.Annotations, valuesFor[kindCRD].Annotations)
					valuesFor[kindCRD] = crd
					values["crd"] = crd
				case isManager(object):
					valuesFor[kindDeployment] = collide(object.Metadata.Labels, object.Metadata.Annotations)
					valuesFor["Pod"] = collide(object.Spec.Template.Metadata.Labels,
						object.Spec.Template.Metadata.Annotations)
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

		// check compares one labels or annotations map with what config/ and the values set. The
		// values replace every key from config/ but the Helm labels and the fixed keys.
		check := func(where string, got, fromConfig, fromValues map[string]string, chartLabels bool, fixed ...string) {
			want := map[string]string{}
			for key, value := range fromConfig {
				if !chartOwned(key, value, customization.prefix, namespace) {
					want[key] = value
				}
			}
			for key, value := range fromValues {
				_, written := fromConfig[key]
				if !written || !slices.Contains(helmLabels, key) && !slices.Contains(fixed, key) {
					want[key] = value
				}
			}
			for key, value := range want {
				Expect(got).To(HaveKeyWithValue(key, value), "%s", where)
			}
			if chartLabels {
				Expect(got).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"), "%s", where)
				Expect(got).To(HaveKeyWithValue("app.kubernetes.io/instance", releaseName), "%s", where)
				if value, ok := fromConfig["app.kubernetes.io/name"]; ok && value == strings.TrimSuffix(customization.prefix, "-") {
					Expect(got).To(HaveKeyWithValue("app.kubernetes.io/name", options.name), "%s", where)
				}
			}
		}

		var sourcePods, renderedPods map[string]string
		for _, object := range source {
			nameSuffix := suffix(object.Metadata.Name, customization.prefix)
			got, ok := findRendered(rendered, object.Kind, nameSuffix)
			if !ok {
				notCreated := object.Kind == kindServiceAccount && options.noServiceAccount ||
					object.Kind == kindServiceMonitor && options.noServiceMonitor
				Expect(notCreated || !slices.Contains([]string{kindServiceAccount, kindServiceMonitor, kindDeployment},
					object.Kind)).To(BeTrue(), "%s %s is not in the chart", object.Kind, nameSuffix)
				continue
			}
			where := object.Kind + " " + nameSuffix
			var values metadataValues
			valuesReach := object.Kind == kindServiceAccount || object.Kind == kindServiceMonitor || isManager(object)
			if valuesReach || object.Kind == kindCRD {
				values = valuesFor[object.Kind]
			}
			check(where+" labels", got.Metadata.Labels, object.Metadata.Labels, values.Labels, valuesReach)
			var fixedAnnotations []string
			if object.Kind == kindCRD {
				fixedAnnotations = crdFixedAnnotations
			}
			check(where+" annotations", got.Metadata.Annotations, object.Metadata.Annotations,
				values.Annotations, false, fixedAnnotations...)
			if isManager(object) {
				sourcePods, renderedPods = object.Spec.Template.Metadata.Labels, got.Spec.Template.Metadata.Labels
				pod := valuesFor["Pod"]
				// Selectors find the pods by their labels, so the pods keep every label from config/.
				check(where+" pod labels", renderedPods, sourcePods, pod.Labels, true,
					slices.Collect(maps.Keys(sourcePods))...)
				check(where+" pod annotations", got.Spec.Template.Metadata.Annotations,
					object.Spec.Template.Metadata.Annotations, pod.Annotations, false,
					"kubectl.kubernetes.io/default-container")
			}
		}
		Expect(renderedPods).NotTo(BeEmpty())

		// Every selector that matches the manager pods in config/ matches them in the chart.
		selectors := 0
		for _, object := range source {
			selector := labelSelector(object)
			if object.Kind == kindServiceMonitor || selector == nil || !selector.Matches(labels.Set(sourcePods)) {
				continue
			}
			got, ok := findRendered(rendered, object.Kind, suffix(object.Metadata.Name, customization.prefix))
			if !ok {
				continue
			}
			selectors++
			Expect(labelSelector(got).Matches(labels.Set(renderedPods))).To(BeTrue(),
				"%s %s does not select the manager pods %v", got.Kind, got.Metadata.Name, renderedPods)
		}
		Expect(selectors).To(BeNumerically(">", 0))

		// So do the NetworkPolicies the chart adds, and the ServiceMonitor selects the metrics Service.
		var metricsService map[string]string
		for _, object := range rendered {
			if object.Kind == "Service" && strings.Contains(object.Metadata.Name, "metrics") {
				metricsService = object.Metadata.Labels
			}
		}
		for _, object := range rendered {
			switch {
			case object.Kind == "NetworkPolicy" && strings.Contains(object.Metadata.Name, "-traffic"):
				Expect(labelSelector(object).Matches(labels.Set(renderedPods))).To(BeTrue(),
					"NetworkPolicy %s does not select the manager pods %v", object.Metadata.Name, renderedPods)
			case object.Kind == kindServiceMonitor:
				Expect(labelSelector(object).Matches(labels.Set(metricsService))).To(BeTrue(),
					"the ServiceMonitor does not select the metrics Service %v", metricsService)
			}
		}
	}

	DescribeTable("keeps every label and annotation from config/, and the values replace them",
		append([]any{keepsAndReplaces}, customizationEntries()...)...)
})

// helmLabels are the labels the chart sets for Helm. The values do not replace them.
var helmLabels = []string{
	"app.kubernetes.io/managed-by", "app.kubernetes.io/name", "app.kubernetes.io/instance", "helm.sh/chart",
}

// crdFixedAnnotations are the annotations of a CRD the values do not replace.
var crdFixedAnnotations = []string{
	"helm.sh/resource-policy", "controller-gen.kubebuilder.io/version", "cert-manager.io/inject-ca-from",
}

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
	nameSuffix = `
nameSuffix: -blue
`
	otherNamespace = `
namespace: other-system
`
	// selectorWithMoreLabels adds a label to the selector of the manager Deployment and to its pods.
	selectorWithMoreLabels = `
patches:
- target:
    kind: Deployment
  patch: |-
    - op: add
      path: /spec/selector/matchLabels/example.com~1shard
      value: a
    - op: add
      path: /spec/template/metadata/labels/example.com~1shard
      value: a
`
	// selectorWithExpressions makes the manager Deployment select its pods with matchExpressions too.
	selectorWithExpressions = `
patches:
- target:
    kind: Deployment
  patch: |-
    - op: add
      path: /spec/selector/matchExpressions
      value:
      - key: example.com/shard
        operator: In
        values: [a, b]
      - key: app.kubernetes.io/name
        operator: In
        values: [project-v4-with-plugins]
    - op: add
      path: /spec/template/metadata/labels/example.com~1shard
      value: a
`
	// labelThatNamesAResource adds a label whose value the chart templates, to the resources, the
	// selectors and the pods.
	labelThatNamesAResource = `
labels:
- pairs:
    appName: project-v4-with-plugins-operator
  includeSelectors: true
`
	// podLabelOnlyAServiceSelects adds a label to the manager pods that the Deployment selector does
	// not use and a Service does.
	podLabelOnlyAServiceSelects = `
patches:
- target:
    kind: Deployment
  patch: |-
    - op: add
      path: /spec/template/metadata/labels/example.com~1tier
      value: a
`
	serviceByTier = `apiVersion: v1
kind: Service
metadata:
  name: by-tier
spec:
  ports:
  - port: 9000
  selector:
    example.com/tier: a
`
	// moreSelectors are resources that select the manager pods, and a second workload that shares a
	// label with them.
	moreSelectors = `apiVersion: v1
kind: Service
metadata:
  name: extra
spec:
  ports:
  - port: 9000
  selector:
    control-plane: controller-manager
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: manager
spec:
  minAvailable: 1
  selector:
    matchLabels:
      control-plane: controller-manager
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: by-expression
spec:
  podSelector:
    matchExpressions:
    - key: control-plane
      operator: In
      values: [controller-manager]
  policyTypes: [Ingress]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helper
  labels:
    app.kubernetes.io/name: project-v4-with-plugins
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: project-v4-with-plugins
      component: helper
  template:
    metadata:
      labels:
        app.kubernetes.io/name: project-v4-with-plugins
        component: helper
    spec:
      containers:
      - name: helper
        image: example.com/helper:v1
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
        example.com/number: "007"
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
        example.com/number: "007"
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
        example.com/number: "007"
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
		{"the ServiceMonitor the chart adds", kustomizeOverlay{prefix: prefix}},
		{"NetworkPolicies from config/", kustomizeOverlay{prefix: prefix, edits: edits(map[string]string{
			"#- ../network-policy": "- ../network-policy",
		})}},
		{"another namePrefix", kustomizeOverlay{prefix: "team-a-", edits: edits(map[string]string{
			"namePrefix: " + prefix: "namePrefix: team-a-",
		})}},
		{"another namePrefix, the ServiceMonitor the chart adds", kustomizeOverlay{
			prefix: "team-a-", edits: map[string]string{"namePrefix: " + prefix: "namePrefix: team-a-"},
		}},
		{"labels on metadata only", overlay(labelsOnMetadata)},
		{"labels with includeSelectors", overlay(labelsWithSelectors)},
		{"labels with includeTemplates", overlay(labelsWithTemplates)},
		{"commonLabels", overlay(commonLabels)},
		{"commonAnnotations", overlay(commonAnnotations)},
		{"patches with labels and annotations of every shape", overlay(everyShapePatches)},
		{"nameSuffix", overlay(nameSuffix)},
		{"another namespace", kustomizeOverlay{
			prefix: prefix, edits: edits(nil), overlay: otherNamespace, namespace: "other-system",
		}},
		{"a selector with more labels", overlay(selectorWithMoreLabels)},
		{"a selector with matchExpressions", overlay(selectorWithExpressions)},
		{"more resources that select the manager pods", kustomizeOverlay{
			prefix: prefix, edits: edits(nil), resources: moreSelectors,
		}},
		{"a Service that selects a pod label the Deployment does not", kustomizeOverlay{
			prefix: prefix, edits: edits(nil), overlay: podLabelOnlyAServiceSelects, resources: serviceByTier,
		}},
		{"a label that names a resource, the resources the chart adds", kustomizeOverlay{
			prefix: prefix, overlay: labelThatNamesAResource,
		}},
	}

	set := func(values ...string) []string {
		args := make([]string, 0, 2*len(values))
		for _, value := range values {
			args = append(args, setFlag, value)
		}
		return args
	}
	optionSets := []struct {
		name string
		chartOptions
	}{
		{"no values", chartOptions{name: sampleProject}},
		{"values that set every key", chartOptions{name: sampleProject, collide: true}},
		{"nameOverride", chartOptions{name: overrideName, args: set("nameOverride=" + overrideName)}},
		{"nameOverride, values that set every key", chartOptions{
			name: overrideName, collide: true, args: set("nameOverride=" + overrideName),
		}},
		{"fullnameOverride, values that set every key", chartOptions{
			name: sampleProject, collide: true, args: set("fullnameOverride=custom-full"),
		}},
		{"an existing ServiceAccount, values that set every key", chartOptions{
			name: sampleProject, collide: true, noServiceAccount: true,
			args: set("serviceAccount.enabled="+"false", "serviceAccount.name=existing"),
		}},
		{"no Prometheus and no NetworkPolicies, values that set every key", chartOptions{
			name: sampleProject, collide: true, noServiceMonitor: true,
			args: set("prometheus.enabled="+"false", "networkPolicy.enabled="+"false"),
		}},
		{"CRDs not kept and no cert-manager, values that set every key", chartOptions{
			name: sampleProject, collide: true, args: set("crd.keep="+"false", "certManager.enabled="+"false"),
		}},
		{"no metrics, values that set every key", chartOptions{
			name: sampleProject, collide: true, noServiceMonitor: true, args: set("metrics.enabled=" + "false"),
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
