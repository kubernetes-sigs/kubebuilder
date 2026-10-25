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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("annotations then labels (Kustomize alphabetical order): both merge, none duplicated",
			saAnnotationsThenLabels, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`example.com/existing-annotation: keep`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
			}),
		Entry("labels then annotations (hand-ordered): both merge, none duplicated",
			saLabelsThenAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))
			}),
		Entry("annotations only (labels removed from base): the labels block is injected",
			saAnnotationsOnly, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
				Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
			}),
		Entry("neither block present: both are injected",
			saNoLabelsNoAnnotations, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("empty-value label key is still omitted so a user override cannot duplicate it",
			saEmptyValueKey, func(rendered string) {
				Expect(rendered).To(ContainSubstring(`"example.com/empty"`))
			}),
		Entry("inline empty annotations map does not duplicate the annotations key",
			saInlineAnnotations, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.annotations }}`))
			}),
		Entry("inline empty labels map does not duplicate the labels key",
			saInlineLabels, func(rendered string) {
				Expect(rendered).NotTo(ContainSubstring("{}"))
				Expect(rendered).To(ContainSubstring(`{{- with .Values.serviceAccount.labels }}`))
			}),
	)

	It("only templates labels/annotations that are direct children of metadata", func() {
		rendered := AddServiceAccountLabelsAndAnnotations(saNestedNonMetadataLabels)

		// The metadata labels block is merged with the chart values once.
		Expect(strings.Count(rendered, valuesServiceAccountLabels)).To(Equal(1))
		Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" }}`))

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
	It("merges .Values.prometheus.labels into metadata.labels, omitting the standard keys", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly)

		Expect(countMetadataHeader(rendered, "labels:")).To(Equal(1))
		Expect(rendered).To(ContainSubstring(valuesPrometheusLabels))
		Expect(rendered).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/managed-by" ` +
			`"app.kubernetes.io/name" "helm.sh/chart" "app.kubernetes.io/instance" "control-plane" }}`))
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
		Expect(rendered).To(ContainSubstring(`{{- with omit . "example.com/existing-annotation" }}`))
	})

	It("keeps an existing annotation with a quoted key", func() {
		input := strings.Replace(smAnnotationsThenLabels,
			"    example.com/existing-annotation: keep\n", "    \"on\": original\n", 1)

		rendered := AddServiceMonitorLabelsAndAnnotations(input)

		Expect(rendered).To(ContainSubstring("  annotations:\n    \"on\": original\n"))
		Expect(rendered).To(ContainSubstring(`{{- with omit . "on" }}`))
	})

	It("leaves the spec.selector.matchLabels block untouched", func() {
		rendered := AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly)

		Expect(rendered).To(ContainSubstring(
			"  selector:\n    matchLabels:\n      app.kubernetes.io/name: " +
				`{{ include "test-project.name" . }}` + "\n      control-plane: controller-manager"))
		Expect(strings.Count(rendered, valuesPrometheusLabels)).To(Equal(1))
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
	It("merges Deployment labels into the existing block and omits every existing key", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsOnly)
		meta := deploymentMetadataSlice(rendered)

		// Existing block preserved.
		Expect(meta).To(ContainSubstring("app.kubernetes.io/name: test-project"))
		Expect(meta).To(ContainSubstring("control-plane: controller-manager"))

		// Values.manager.labels merged inline with omit() for every key already in the block.
		Expect(meta).To(ContainSubstring("{{- with .Values.manager.labels }}"))
		Expect(meta).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" "control-plane" }}`))

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
		Expect(meta).To(ContainSubstring("{{- with .Values.manager.annotations }}"))
		Expect(meta).To(ContainSubstring(`{{- with omit . "example.com/managed" }}`))
		Expect(countMetadataHeader(meta, "annotations:")).To(Equal(1))
	})

	It("merges pod-template labels via .Values.manager.pod.labels and omits control-plane", func() {
		rendered := AddCustomLabelsAndAnnotations(depLabelsOnly)
		pod := podTemplateSlice(rendered)

		Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
		Expect(pod).To(ContainSubstring("{{- with .labels }}"))
		Expect(pod).To(ContainSubstring(`{{- with omit . "control-plane" }}`))
		Expect(pod).To(ContainSubstring("control-plane: controller-manager"))
	})

	It("merges into an existing pod annotations block, omitting the default-container hint", func() {
		rendered := AddCustomLabelsAndAnnotations(depPodAnnotationsThenLabels)
		pod := podTemplateSlice(rendered)

		Expect(pod).To(ContainSubstring("kubectl.kubernetes.io/default-container: manager"))
		Expect(pod).To(ContainSubstring("{{- with .Values.manager.pod }}"))
		Expect(pod).To(ContainSubstring("{{- with .annotations }}"))
		Expect(pod).To(ContainSubstring(`{{- with omit . "kubectl.kubernetes.io/default-container" }}`))
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

		// Labels block gets merged with omit() over both existing keys.
		Expect(meta).To(ContainSubstring("{{- with .Values.manager.labels }}"))
		Expect(meta).To(ContainSubstring(`{{- with omit . "app.kubernetes.io/name" "control-plane" }}`))

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

	DescribeTable("reads the keys of each block from the block itself",
		func(base, existing, replacement, want string) {
			Expect(base).To(ContainSubstring(existing))
			rendered := AddCustomLabelsAndAnnotations(strings.Replace(base, existing, replacement, 1))

			Expect(rendered).To(ContainSubstring(want), "got:\n%s", rendered)
		},
		Entry("a Deployment annotation whose text ends with a line like labels:",
			depAnnotationsThenLabels, "    example.com/managed: keep\n",
			"    example.com/description: |\n      Runs the manager.\n      labels:\n",
			"  annotations:\n    example.com/description: |\n      Runs the manager.\n      labels:\n"+
				"    {{- with .Values.manager.annotations }}\n"+
				"    {{- with omit . \"example.com/description\" }}\n"+
				"    {{- toYaml . | nindent 4 }}\n"),
		Entry("a Deployment annotation with a line like spec: inside its text",
			depAnnotationsThenLabels, "    example.com/managed: keep\n",
			"    example.com/a-first: keep\n    example.com/description: |\n      spec:\n      text\n"+
				"    example.com/z-last: keep\n",
			`{{- with omit . "example.com/a-first" "example.com/description" "example.com/z-last" }}`),
		Entry("a Deployment annotation with lines like template: and metadata: inside its text",
			depAnnotationsThenLabels, "    example.com/managed: keep\n",
			"    example.com/description: |\n      template:\n      metadata:\n",
			"  labels:\n    app.kubernetes.io/name: test-project\n    control-plane: controller-manager\n"+
				"    {{- with .Values.manager.labels }}\n"),
		Entry("a pod annotation with a key-like line inside its text",
			depPodAnnotationsThenLabels, "        kubectl.kubernetes.io/default-container: manager\n",
			"        kubectl.kubernetes.io/default-container: manager\n"+
				"        note: |\n          endpoint: api\n          second line\n",
			"          second line\n"+
				"        {{- with .Values.manager.pod }}\n"+
				"        {{- with .annotations }}\n"+
				"        {{- with omit . \"kubectl.kubernetes.io/default-container\" \"note\" }}\n"+
				"        {{- toYaml . | nindent 8 }}\n"),
		Entry("a pod annotation with a line like labels: inside its text",
			depPodAnnotationsThenLabels, "        kubectl.kubernetes.io/default-container: manager\n",
			"        kubectl.kubernetes.io/default-container: manager\n"+
				"        note: |\n          Runs the manager.\n          labels:\n",
			"          labels:\n"+
				"        {{- with .Values.manager.pod }}\n"+
				"        {{- with .annotations }}\n"+
				"        {{- with omit . \"kubectl.kubernetes.io/default-container\" \"note\" }}\n"+
				"        {{- toYaml . | nindent 8 }}\n"+
				"        {{- end }}\n        {{- end }}\n        {{- end }}\n"+
				"      labels:\n        control-plane: controller-manager\n"+
				"        {{- with .Values.manager.pod }}\n"+
				"        {{- with .labels }}\n"),
		Entry("a pod annotation with a line like spec: inside its text",
			depPodAnnotationsThenLabels, "        kubectl.kubernetes.io/default-container: manager\n",
			"        kubectl.kubernetes.io/default-container: manager\n        note: |\n          spec:\n",
			"      labels:\n        control-plane: controller-manager\n"+
				"        {{- with .Values.manager.pod }}\n"+
				"        {{- with .labels }}\n"),
	)
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
