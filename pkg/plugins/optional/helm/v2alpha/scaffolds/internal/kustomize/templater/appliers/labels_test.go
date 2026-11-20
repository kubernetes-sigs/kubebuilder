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

package appliers

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	sigsyaml "sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// countMetadataHeader counts how many times key (for example "labels:") appears as a standalone
// metadata header line, ignoring Helm guards and value references that also mention the word.
func countMetadataHeader(rendered, key string) int {
	count := 0
	for line := range strings.SplitSeq(rendered, "\n") {
		if strings.TrimSpace(line) == key {
			count++
		}
	}
	return count
}

// The ServiceAccount metadata reaching the applier depends on what config/kustomize produced:
// the base always ships block-style labels, and commonAnnotations or edits to service_account.yaml
// can add annotations, drop labels, add empty-value keys, or emit inline empty maps.
const (
	saLabelsOnly = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
  namespace: system`

	saAnnotationsThenLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    example.com/existing-annotation: keep
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`

	saLabelsThenAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  annotations:
    example.com/existing-annotation: keep
  name: controller-manager`

	saAnnotationsOnly = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    example.com/existing-annotation: keep
  name: controller-manager`

	saNoLabelsNoAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: system`

	saEmptyValueKey = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
    example.com/empty:
  name: controller-manager`

	saInlineAnnotations = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations: {}
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`

	saInlineLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels: {}
  name: controller-manager`

	// A labels: header that is not a direct child of metadata: (here nested under another key).
	// Only the metadata block should be templated; the deeper labels: must be left as written.
	saNestedNonMetadataLabels = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
imagePullSecrets:
  - name: my-registry
    labels:
      example.com/team: platform`
)

var _ = Describe("AddServiceAccountLabelsAndAnnotations", func() {
	// Every metadata shape must render exactly one labels block and one annotations block, both
	// wired to the user-value maps, so the manifest stays valid YAML and honors the
	// serviceAccount.labels and serviceAccount.annotations chart options.
	DescribeTable("keeps one labels and one annotations block wired to the chart values",
		func(input string, check func(rendered string)) {
			rendered := AddServiceAccountLabelsAndAnnotations(input)

			Expect(countMetadataHeader(rendered, "labels:")).To(Equal(1),
				"want exactly one labels: header in:\n%s", rendered)
			Expect(countMetadataHeader(rendered, "annotations:")).To(Equal(1),
				"want exactly one annotations: header in:\n%s", rendered)
			Expect(rendered).To(ContainSubstring(valuesServiceAccountLabels))
			Expect(rendered).To(ContainSubstring(valuesServiceAccountAnnotations))

			check(rendered)
		},
		Entry("labels only (scaffold default): the annotations block is injected",
			saLabelsOnly, func(rendered string) {
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountAnnotations + " }}"))
			}),
		Entry("annotations then labels (Kustomize alphabetical order): both merge, none duplicated",
			saAnnotationsThenLabels, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`example.com/existing-annotation: keep`))
			}),
		Entry("labels then annotations (hand-ordered): both merge, none duplicated",
			saLabelsThenAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`example.com/existing-annotation: keep`))
			}),
		Entry("annotations only (labels removed from base): the labels block is injected",
			saAnnotationsOnly, func(rendered string) {
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountLabels + " }}"))
			}),
		Entry("neither block present: both are injected",
			saNoLabelsNoAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountLabels + " }}"))
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountAnnotations + " }}"))
			}),
		Entry("an empty-value label key is kept",
			saEmptyValueKey, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`"example.com/empty"`))
			}),
		Entry("inline empty annotations map does not duplicate the annotations key",
			saInlineAnnotations, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountAnnotations + " }}"))
			}),
		Entry("inline empty labels map does not duplicate the labels key",
			saInlineLabels, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring("{{- with " + valuesServiceAccountLabels + " }}"))
			}),
	)

	It("only templates labels/annotations that are direct children of metadata", func() {
		rendered := AddServiceAccountLabelsAndAnnotations(saNestedNonMetadataLabels)

		// The labels: nested under imagePullSecrets is left exactly as written.
		Expect(rendered).To(ContainSubstring("    labels:\n      example.com/team: platform"))
	})
})

var _ = Describe("extractMapKeys", func() {
	DescribeTable("returns the keys of the block in order",
		func(body string, want []string) {
			Expect(extractMapKeys(strings.Split(strings.TrimSuffix(body, "\n"), "\n"))).To(Equal(want))
		},
		Entry("a quoted key",
			"    \"on\": original\n", []string{"on"}),
		Entry("YAML text inside a multi-line value",
			"    example.com/config: |\n      metadata:\n        labels:\n          foo: bar\n",
			[]string{"example.com/config"}),
		Entry("Helm expressions in values",
			"    foo: {{ .Values.foo }}\n    bar: {{ include \"something\" . }}\n"+
				"    baz: \"{{ .Values.baz }}\"\n    qux: prefix-{{ .Values.qux | quote }}\n",
			[]string{"foo", "bar", "baz", "qux"}),
		Entry("an escaped literal template in a quoted value",
			"    \"on\": \"{{ \"{{ .Spec.Branch }}\" }}\\t\"\n", []string{"on"}),
		Entry("an escaped literal template on the first line of a multi-line value",
			"    note: |\n      {{ \"{{ .Spec.Branch }}\" }}\n        nested text\n      labels:\n",
			[]string{"note"}),
		Entry("a block that is not valid YAML, through the line-based fallback",
			"    example.com/keep: value\n    example.com/broken: [unclosed\n",
			[]string{"example.com/keep", "example.com/broken"}),
		Entry("an empty block", "", []string{}),
	)
})

// A ServiceMonitor's metadata.labels always carries the standard Helm/chart labels added by
// AddStandardHelmLabels before this applier runs. Kustomize commonAnnotations add an annotations
// block, which sorts before labels.
const (
	smStandardLabelsOnly = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ .Release.Service }}
    app.kubernetes.io/name: {{ include "test-project.name" . }}
    helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    control-plane: controller-manager
  name: controller-manager-metrics-monitor
  namespace: system
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "test-project.name" . }}
      control-plane: controller-manager`

	smAnnotationsThenLabels = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  annotations:
    example.com/existing-annotation: keep
  labels:
    app.kubernetes.io/managed-by: {{ .Release.Service }}
    app.kubernetes.io/name: {{ include "test-project.name" . }}
    helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    control-plane: controller-manager
  name: controller-manager-metrics-monitor
  namespace: system`
)

var _ = Describe("AddServiceMonitorLabelsAndAnnotations", func() {
	It("merges .Values.prometheus.labels into metadata.labels", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly)

		Expect(countMetadataHeader(rendered, "labels:")).To(Equal(1))
		Expect(rendered).To(ContainSubstring(valuesPrometheusLabels))
	})

	It("injects a guarded .Values.prometheus.annotations block since none existed", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly)

		Expect(countMetadataHeader(rendered, "annotations:")).To(Equal(1))
		Expect(rendered).To(ContainSubstring(valuesPrometheusAnnotations))
	})

	It("merges into an existing annotations block (commonAnnotations) without duplicating it", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smAnnotationsThenLabels)

		Expect(countMetadataHeader(rendered, "annotations:")).To(Equal(1),
			"want exactly one annotations: header in:\n%s", rendered)
		Expect(countMetadataHeader(rendered, "labels:")).To(Equal(1),
			"want exactly one labels: header in:\n%s", rendered)
		Expect(rendered).To(ContainSubstring(`example.com/existing-annotation: keep`))
		Expect(rendered).To(ContainSubstring(valuesPrometheusAnnotations))
	})

	It("keeps an existing annotation with a quoted key", func() {
		input := strings.Replace(smAnnotationsThenLabels,
			"    example.com/existing-annotation: keep\n", "    \"on\": original\n", 1)

		rendered := AddServiceMonitorLabelsAndAnnotations(input)

		Expect(rendered).To(ContainSubstring("    \"on\": original\n"))
	})

	It("leaves the spec.selector.matchLabels block untouched", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly)

		Expect(rendered).To(ContainSubstring(
			"  selector:\n    matchLabels:\n      app.kubernetes.io/name: " +
				`{{ include "test-project.name" . }}` + "\n      control-plane: controller-manager"))
	})
})

// The Deployment metadata reaching AddCustomLabelsAndAnnotations depends on what config/kustomize
// produced. The scaffold ships block-style labels on both the Deployment and pod template; edits or
// commonAnnotations can add annotations at either scope in either order. These fixtures cover the
// realistic shapes so a refactor of this function has a safety net.
const (
	// Scaffold default: labels on Deployment and pod template, no annotations at either scope.
	depLabelsOnly = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: controller-manager
  namespace: system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

	// commonAnnotations at the Deployment scope: annotations precede labels (Kustomize alphabetical).
	depAnnotationsThenLabels = `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    example.com/managed: keep
  labels:
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  name: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

	// Pod-template annotations set by the scaffold (default-container hint) precede pod labels.
	depPodAnnotationsThenLabels = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

	// Both scopes carry annotations before labels; the merged output must not duplicate any header.
	depBothAnnotationsThenLabels = `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    example.com/managed: keep
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`

	// Hand-ordered: labels precede annotations at the Deployment scope. Kustomize alphabetizes so
	// this shape only appears when a human edits the manifest, but a refactor could regress it
	// silently — the current implementation emits a second annotations: header instead of merging.
	depLabelsThenAnnotations = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/name: test-project
    control-plane: controller-manager
  annotations:
    example.com/managed: keep
  name: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - name: manager
        image: controller:latest`
)

// deploymentMetadataSlice returns the Deployment metadata block: from the first "metadata:" up to
// (but not including) the first "spec:". podTemplateSlice returns the pod-template block: from
// "template:" up to the "containers:" line where the applier stops injecting.
func deploymentMetadataSlice(rendered string) string {
	return rendered[strings.Index(rendered, "metadata:"):strings.Index(rendered, "\nspec:")]
}

func podTemplateSlice(rendered string) string {
	start := strings.Index(rendered, "template:")
	end := strings.Index(rendered[start:], "containers:")
	return rendered[start : start+end]
}

var _ = Describe("AddCustomLabelsAndAnnotations", func() {
	It("merges Deployment labels into the existing block", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsOnly)
		meta := deploymentMetadataSlice(rendered)

		// Existing block preserved.
		Expect(meta).To(ContainSubstring("app.kubernetes.io/name: test-project"))
		Expect(meta).To(ContainSubstring("control-plane: controller-manager"))

		Expect(meta).To(ContainSubstring(".Values.manager.labels"))

		// Single labels: header — the merge must not duplicate the key.
		Expect(countMetadataHeader(meta, "labels:")).To(Equal(1))
	})

	It("injects a Deployment annotations block when Kustomize emitted none", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsOnly)
		meta := deploymentMetadataSlice(rendered)

		Expect(meta).To(ContainSubstring("{{- if .Values.manager.annotations }}"))
		Expect(meta).To(ContainSubstring("annotations:"))
		Expect(meta).To(ContainSubstring("{{- toYaml .Values.manager.annotations | nindent 4 }}"))
		Expect(countMetadataHeader(meta, "annotations:")).To(Equal(1))
	})

	It("merges into an existing Deployment annotations block instead of duplicating it", func() {
		rendered := AddCustomLabelsAndAnnotations(depAnnotationsThenLabels)
		meta := deploymentMetadataSlice(rendered)

		Expect(meta).To(ContainSubstring("example.com/managed: keep"))
		Expect(meta).To(ContainSubstring(".Values.manager.annotations"))
		Expect(countMetadataHeader(meta, "annotations:")).To(Equal(1))
	})

	It("merges pod-template labels via .Values.manager.pod.labels", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsOnly)
		pod := podTemplateSlice(rendered)

		Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
		Expect(pod).To(ContainSubstring("{{- with .labels }}"))
		Expect(pod).To(ContainSubstring("control-plane: controller-manager"))
	})

	It("merges into an existing pod annotations block", func() {
		rendered := AddCustomLabelsAndAnnotations(depPodAnnotationsThenLabels)
		pod := podTemplateSlice(rendered)

		Expect(pod).To(ContainSubstring("kubectl.kubernetes.io/default-container: manager"))
		Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
		Expect(pod).To(ContainSubstring("{{- with .annotations }}"))
		Expect(countMetadataHeader(pod, "annotations:")).To(Equal(1))
	})

	It("handles both scopes carrying annotations before labels without header duplication", func() {
		rendered := AddCustomLabelsAndAnnotations(depBothAnnotationsThenLabels)
		meta := deploymentMetadataSlice(rendered)
		pod := podTemplateSlice(rendered)

		// Each scope keeps a single labels: and a single annotations: header, both merged.
		Expect(countMetadataHeader(meta, "labels:")).To(Equal(1))
		Expect(countMetadataHeader(meta, "annotations:")).To(Equal(1))
		Expect(countMetadataHeader(pod, "labels:")).To(Equal(1))
		Expect(countMetadataHeader(pod, "annotations:")).To(Equal(1))

		// Both existing keys preserved.
		Expect(rendered).To(ContainSubstring("example.com/managed: keep"))
		Expect(rendered).To(ContainSubstring("kubectl.kubernetes.io/default-container: manager"))
	})

	It("pins the current behavior for hand-ordered labels-then-annotations input", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsThenAnnotations)
		meta := deploymentMetadataSlice(rendered)

		// Existing annotation is kept (as bare metadata under the source block).
		Expect(meta).To(ContainSubstring("example.com/managed: keep"))

		Expect(meta).To(ContainSubstring(".Values.manager.labels"))

		// A second annotations: header is injected — current shape, buggy, pinned so a refactor
		// cannot regress silently in the other direction either.
		Expect(countMetadataHeader(meta, "annotations:")).To(Equal(2))
		Expect(meta).To(ContainSubstring("{{- if .Values.manager.annotations }}"))
	})

	It("is a no-op on a raw string that already references .Values.manager.labels", func() {
		once := AddCustomLabelsAndAnnotations(depLabelsOnly)
		twice := AddCustomLabelsAndAnnotations(once)

		Expect(twice).To(Equal(once))
	})
})

var _ = Describe("AddStandardHelmLabels", func() {
	DescribeTable("adds the chart labels to metadata labels only, not to the text of a multi-line value",
		func(input, annotation string) {
			Expect(input).To(ContainSubstring(annotation))
			rendered := AddStandardHelmLabels(input)

			Expect(rendered).To(ContainSubstring(annotation), "got:\n%s", rendered)
			Expect(strings.Count(rendered, "helm.sh/chart:")).To(Equal(1), "got:\n%s", rendered)
		},
		Entry("a ServiceAccount annotation", `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    example.com/description: |
      labels:
        app.kubernetes.io/name: other
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`,
			"    example.com/description: |\n      labels:\n        app.kubernetes.io/name: other\n  labels:\n"),
		Entry("an annotation with a labels: line, followed by a value with app.kubernetes.io/name:",
			`apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    note: |
      labels:
    sel: 'app.kubernetes.io/name: x'
  labels:
    app.kubernetes.io/name: test-project
  name: controller-manager`,
			"    note: |\n      labels:\n    sel: 'app.kubernetes.io/name: x'\n  labels:\n"),
		Entry("a pod template annotation", `apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    metadata:
      annotations:
        note: |
          labels:
            app.kubernetes.io/name: other
      labels:
        app.kubernetes.io/name: test-project`,
			"        note: |\n          labels:\n            app.kubernetes.io/name: other\n      labels:\n"),
	)
})

const (
	labelsField      = "labels"
	annotationsField = "annotations"
	controlPlaneKey  = "control-plane"
	appNameKey       = "app.kubernetes.io/name"
	testChartName    = "test-project"
	appsV1           = "apps/v1"
)

// Labels and annotations Kubernetes accepts, for keys that are already in config/.
var (
	annotationShapes = map[string]string{
		"example.com/plain":            "plain",
		"example.com/empty":            "",
		"example.com/bool":             "true",
		"example.com/number":           "007",
		"example.com/null":             "null",
		"example.com/colon":            "a: b # c",
		"example.com/spaces":           "  padded  ",
		"example.com/json":             `{"a":1}`,
		"example.com/quotes":           `it's "quoted"`,
		"example.com/unicode":          "café ✓",
		"example.com/tab":              "a\tb",
		"example.com/multi-line":       "line one\nline two\n",
		"example.com/no-final-break":   "line one\nline two",
		"example.com/blank-line":       "first\n\n",
		"example.com/blank-lines":      "first\n\n\n",
		"example.com/leading-spaces":   "  leading spaces\n",
		"example.com/key-like-lines":   "labels:\n  app.kubernetes.io/name: other\nspec:\nmetadata:\nendpoint: api\n",
		"example.com/template-text":    "{{ .Spec.Branch }}",
		"example.com/carriage-return":  "a\rb",
		"example.com/line-separator":   "a\u2028b",
		"example.com/paragraph-break":  "a\u2029b",
		"example.com/next-line":        "a\u0085b",
		"example.com/large":            strings.Repeat("x", 64*1024),
		longKey:                        "long key",
		"on":                           "original",
		"123":                          "digits",
		"example.com/Upper_Case-key.1": "upper",
	}
	labelShapes = map[string]string{
		"example.com/tier":   "a",
		"example.com/empty":  "",
		"example.com/bool":   "true",
		"example.com/number": "007",
		"on":                 "x",
		"123":                "digits",
		"Upper_Case-key.1":   "V_1-x.y",
		"example.com/y":      "y",
		"example.com/float":  "1e3",
		"example.com/hex":    "0x1F",
		"example.com/max":    strings.Repeat("v", 63),
		longKey:              "long-key",
	}
	// longKey has the longest prefix (253) and name (63) Kubernetes accepts.
	longKey = strings.Repeat(strings.Repeat("a", 63)+".", 3) + strings.Repeat("a", 61) + "/" + strings.Repeat("n", 63)
	// chartLabels are the labels the chart writes; the values can set them too.
	chartLabels = []string{
		"app.kubernetes.io/managed-by", appNameKey, "app.kubernetes.io/instance", "helm.sh/chart", controlPlaneKey,
	}

	baseLabels     = map[string]string{"app.kubernetes.io/managed-by": "kustomize", appNameKey: testChartName}
	managerLabels  = withShapes(baseLabels, map[string]string{controlPlaneKey: controllerManagerName})
	podLabels      = map[string]string{controlPlaneKey: controllerManagerName}
	podAnnotations = map[string]string{common.DefaultContainerAnnotation: managerVal}

	serviceAccountMaps = []metadataMap{
		{valuesPath: []string{"serviceAccount", labelsField}, get: labelsOf, labels: true},
		{valuesPath: []string{"serviceAccount", annotationsField}, get: annotationsOf},
	}
	serviceMonitorMaps = []metadataMap{
		{valuesPath: []string{"prometheus", labelsField}, get: labelsOf, labels: true},
		{valuesPath: []string{"prometheus", annotationsField}, get: annotationsOf},
	}
	managerMaps = []metadataMap{
		{valuesPath: []string{managerVal, labelsField}, get: labelsOf, labels: true},
		{valuesPath: []string{managerVal, annotationsField}, get: annotationsOf},
		{valuesPath: []string{managerVal, "pod", labelsField}, get: podLabelsOf, labels: true},
		{valuesPath: []string{managerVal, "pod", annotationsField}, get: podAnnotationsOf},
	}
)

// labelValues and annotationValues are the values the specs set: every shape Kubernetes accepts.
var (
	labelValues = slices.Sorted(maps.Values(labelShapes))
	// Helm's toYaml writes U+0085 as is, which YAML reads back as a line break, so a value set with
	// it in the values cannot reach Kubernetes; config/ values with it are covered above.
	annotationValues = slices.DeleteFunc(slices.Sorted(maps.Values(annotationShapes)), func(value string) bool {
		return strings.ContainsRune(value, '\u0085')
	})
	// addedKeys are keys config/ does not have.
	addedKeys = []string{"example.com/added", "y", "1e3", "Upper.Key_1", "added-" + strings.Repeat("k", 57)}
)

// valueForm is how the values set one labels or annotations map.
type valueForm int

const (
	formMissing valueForm = iota
	formNull
	formEmpty
	formEveryKey
	formEveryOtherKey
	formCount
)

// metadataMap is one labels or annotations map of a resource and where the values set it.
type metadataMap struct {
	valuesPath []string
	get        func(renderedResource) map[string]string
	labels     bool
}

type objectMeta struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type renderedResource struct {
	Metadata objectMeta `json:"metadata"`
	Spec     struct {
		Template struct {
			Metadata objectMeta `json:"metadata"`
		} `json:"template"`
	} `json:"spec"`
}

func labelsOf(r renderedResource) map[string]string      { return r.Metadata.Labels }
func annotationsOf(r renderedResource) map[string]string { return r.Metadata.Annotations }
func podLabelsOf(r renderedResource) map[string]string   { return r.Spec.Template.Metadata.Labels }
func podAnnotationsOf(r renderedResource) map[string]string {
	return r.Spec.Template.Metadata.Annotations
}

func withShapes(base, shapes map[string]string) map[string]string {
	merged := maps.Clone(base)
	maps.Copy(merged, shapes)
	return merged
}

// pluginYAML marshals resource as the plugin does before templating it.
func pluginYAML(resource map[string]any) string {
	// Round-trip through JSON so the maps hold the types the plugin reads from install.yaml.
	data, err := json.Marshal(resource)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	var object map[string]any
	ExpectWithOffset(1, json.Unmarshal(data, &object)).To(Succeed())
	out, err := MarshalResource(object)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return out
}

// withMetadata adds the labels and annotations that are not nil to metadata.
func withMetadata(metadata map[string]any, labels, annotations map[string]string) map[string]any {
	if labels != nil {
		metadata[labelsField] = labels
	}
	if annotations != nil {
		metadata[annotationsField] = annotations
	}
	return metadata
}

func serviceAccountYAML(labels, annotations map[string]string) string {
	return pluginYAML(map[string]any{
		apiVersionKey: "v1", kindKey: "ServiceAccount",
		metadataKey: withMetadata(map[string]any{nameKey: controllerManagerName}, labels, annotations),
	})
}

func serviceMonitorYAML(annotations map[string]string) string {
	return pluginYAML(map[string]any{
		apiVersionKey: "monitoring.coreos.com/v1", kindKey: "ServiceMonitor",
		metadataKey:        withMetadata(map[string]any{nameKey: "metrics-monitor"}, managerLabels, annotations),
		k8sObjectSpecField: map[string]any{"selector": map[string]any{"matchLabels": podLabels}},
	})
}

func deploymentYAML(labels, annotations, podLabels, podAnnotations map[string]string) string {
	return pluginYAML(map[string]any{
		apiVersionKey: appsV1, kindKey: common.KindDeployment,
		metadataKey: withMetadata(map[string]any{nameKey: controllerManagerName}, labels, annotations),
		k8sObjectSpecField: map[string]any{
			"selector": map[string]any{"matchLabels": map[string]string{controlPlaneKey: controllerManagerName}},
			k8sObjectTemplateField: map[string]any{
				metadataKey:        withMetadata(map[string]any{}, podLabels, podAnnotations),
				k8sObjectSpecField: map[string]any{"containers": []any{map[string]any{nameKey: managerVal}}},
			},
		},
	})
}

// renderTemplate renders tmpl with the Helm engine, as a template of a chart named test-project.
func renderTemplate(tmpl string, values map[string]any) (string, error) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{APIVersion: chart.APIVersionV2, Name: testChartName, Version: "0.1.0"},
		Templates: []*chart.File{
			{Name: "templates/_helpers.tpl", Data: []byte(`{{- define "test-project.name" -}}test-project{{- end -}}`)},
			{Name: "templates/resource.yaml", Data: []byte(tmpl)},
		},
	}
	renderValues, err := chartutil.ToRenderValues(c, values, chartutil.ReleaseOptions{Name: "r", Namespace: "ns"}, nil)
	if err != nil {
		return "", fmt.Errorf("building the render values: %w", err)
	}
	out, err := engine.Render(c, renderValues)
	if err != nil {
		return "", fmt.Errorf("rendering the template: %w", err)
	}
	return out["test-project/templates/resource.yaml"], nil
}

// decodeResource decodes a render as Helm and kubectl do, with sigs.k8s.io/yaml. Strict decoding
// fails on a key rendered twice.
func decodeResource(rendered string) renderedResource {
	data, err := sigsyaml.YAMLToJSONStrict([]byte(rendered))
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "got:\n%s", rendered)
	var resource renderedResource
	ExpectWithOffset(1, json.Unmarshal(data, &resource)).To(Succeed())
	return resource
}

// differentKeys lists the keys whose rendered value is not the one wanted, with both values.
func differentKeys(got, want map[string]string) []string {
	var diffs []string
	for _, key := range slices.Sorted(maps.Keys(maps.Collect(func(yield func(string, string) bool) {
		for k, v := range got {
			yield(k, v)
		}
		for k, v := range want {
			yield(k, v)
		}
	}))) {
		gotValue, inGot := got[key]
		wantValue, inWant := want[key]
		if inGot != inWant || gotValue != wantValue {
			diffs = append(diffs, fmt.Sprintf("%s: got %.80q (%t), want %.80q (%t)", key, gotValue, inGot, wantValue, inWant))
		}
	}
	return diffs
}

// setValue sets value at path in values, creating the maps on the way.
func setValue(values map[string]any, path []string, value any) {
	for _, key := range path[:len(path)-1] {
		next, ok := values[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			values[key] = next
		}
		values = next
	}
	values[path[len(path)-1]] = value
}

// valuesFor returns what form sets, if anything, and what each key of the map renders.
func valuesFor(form valueForm, keys, setValues []string, config map[string]string) (any, bool, map[string]string) {
	want := maps.Clone(config)
	if want == nil {
		want = map[string]string{}
	}
	switch form {
	case formMissing:
		return nil, false, want
	case formNull:
		return nil, true, want
	case formEmpty:
		return map[string]any{}, true, want
	}
	set := map[string]any{}
	for i, key := range keys {
		if form == formEveryKey || i%2 == 0 {
			value := setValues[i%len(setValues)]
			set[key] = value
			want[key] = value
		}
	}
	return set, true, want
}

var _ = Describe("labels and annotations set in the values", func() {
	DescribeTable("render each key once, with the value set or else the one from config/",
		func(input string, apply func(string) string, metadataMaps []metadataMap, inputs ...map[string]string) {
			// The same steps, in the same order, as Templater.ApplyHelmSubstitutions.
			withHelmLabels := AddHelmLabelsAndAnnotations(testChartName, testChartName,
				EscapeExistingTemplateSyntax(input))
			generated := CollapseBlankLineAfterIf(apply(withHelmLabels))

			// What config/ renders to with no values: the chart labels already templated.
			original, err := renderTemplate(withHelmLabels, nil)
			Expect(err).NotTo(HaveOccurred())
			config := decodeResource(original)

			// Every value from config/ reaches the chart as written.
			for i, entries := range inputs {
				for key, value := range entries {
					Expect(metadataMaps[i].get(config)).To(HaveKeyWithValue(key, value), "%v", metadataMaps[i].valuesPath)
				}
			}

			// Every combination of value forms, one per map.
			combinations := 1
			for range metadataMaps {
				combinations *= int(formCount)
			}
			for combination := range combinations {
				values := map[string]any{}
				for _, m := range metadataMaps {
					values[m.valuesPath[0]] = map[string]any{}
				}
				wants := make([]map[string]string, len(metadataMaps))
				forms := make([]valueForm, len(metadataMaps))
				rest := combination
				for i, m := range metadataMaps {
					forms[i] = valueForm(rest % int(formCount))
					rest /= int(formCount)

					keys := append(slices.Collect(maps.Keys(m.get(config))), addedKeys...)
					setValues := annotationValues
					if m.labels {
						keys = append(keys, chartLabels...)
						setValues = labelValues
					}
					slices.Sort(keys)
					value, set, want := valuesFor(forms[i], slices.Compact(keys), setValues, m.get(config))
					if set {
						setValue(values, m.valuesPath, value)
					}
					wants[i] = want
				}

				rendered, err := renderTemplate(generated, values)
				Expect(err).NotTo(HaveOccurred(), "forms %v, template:\n%s", forms, generated)
				got := decodeResource(rendered)
				for i, m := range metadataMaps {
					gotMap := m.get(got)
					if gotMap == nil {
						gotMap = map[string]string{}
					}
					Expect(differentKeys(gotMap, wants[i])).To(BeEmpty(), "%v with forms %v", m.valuesPath, forms)
				}
			}
		},
		Entry("a ServiceAccount as scaffolded",
			serviceAccountYAML(baseLabels, nil), AddServiceAccountLabelsAndAnnotations, serviceAccountMaps),
		Entry("a ServiceAccount with labels and annotations of every shape",
			serviceAccountYAML(withShapes(baseLabels, labelShapes), annotationShapes),
			AddServiceAccountLabelsAndAnnotations, serviceAccountMaps, labelShapes, annotationShapes),
		Entry("a ServiceAccount with no labels or annotations",
			serviceAccountYAML(nil, nil), AddServiceAccountLabelsAndAnnotations, serviceAccountMaps),
		Entry("a ServiceAccount with empty labels and annotations",
			serviceAccountYAML(map[string]string{}, map[string]string{}),
			AddServiceAccountLabelsAndAnnotations, serviceAccountMaps),
		Entry("a ServiceMonitor as scaffolded",
			serviceMonitorYAML(nil), AddServiceMonitorLabelsAndAnnotations, serviceMonitorMaps),
		Entry("a ServiceMonitor with annotations of every shape",
			serviceMonitorYAML(annotationShapes), AddServiceMonitorLabelsAndAnnotations, serviceMonitorMaps,
			nil, annotationShapes),
		Entry("a ServiceMonitor with empty annotations",
			serviceMonitorYAML(map[string]string{}), AddServiceMonitorLabelsAndAnnotations, serviceMonitorMaps),
		Entry("a manager Deployment as scaffolded",
			deploymentYAML(managerLabels, nil, podLabels, podAnnotations), AddCustomLabelsAndAnnotations, managerMaps),
		Entry("a manager Deployment with labels and annotations of every shape",
			deploymentYAML(withShapes(managerLabels, labelShapes), annotationShapes,
				withShapes(podLabels, labelShapes), withShapes(podAnnotations, annotationShapes)),
			AddCustomLabelsAndAnnotations, managerMaps, labelShapes, annotationShapes, labelShapes, annotationShapes),
		Entry("a manager Deployment with no pod annotations",
			deploymentYAML(managerLabels, nil, podLabels, nil), AddCustomLabelsAndAnnotations, managerMaps),
		Entry("a manager Deployment with empty annotations",
			deploymentYAML(managerLabels, map[string]string{}, podLabels, map[string]string{}),
			AddCustomLabelsAndAnnotations, managerMaps),
	)
})
