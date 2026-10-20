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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmchart "helm.sh/helm/v3/pkg/chart"
	helmchartutil "helm.sh/helm/v3/pkg/chartutil"
	helmengine "helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	keyServiceAccount = "serviceAccount"
	keyPrometheus     = "prometheus"
	keyAnnotations    = "annotations"
)

type helmValues = map[string]any

const (
	renderedResourceName = "test/templates/resource.yaml"
	renderedHelpers      = `{{- define "test-project.name" -}}test-project{{- end -}}`

	saRenderInput = `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::111111111111:role/a
    example.com/multi: |
      line one
      line two
  labels:
    app.kubernetes.io/managed-by: {{ .Release.Service }}
    app.kubernetes.io/name: test-project
    example.com/tier: a
  name: controller-manager
  namespace: system`
)

// renderResource renders templateText with the real Helm engine and the given user values, then parses
// the result in strict mode. Strict parsing fails on duplicate keys, which is the rule the omit and
// hasKey logic must guarantee.
func renderResource(templateText string, userValues helmValues) *unstructured.Unstructured {
	GinkgoHelper()

	testChart := &helmchart.Chart{
		Metadata: &helmchart.Metadata{Name: "test", Version: "0.1.0", APIVersion: "v2"},
		Templates: []*helmchart.File{
			{Name: "templates/_helpers.tpl", Data: []byte(renderedHelpers)},
			{Name: "templates/resource.yaml", Data: []byte(templateText)},
		},
		Values: helmValues{
			keyServiceAccount: helmValues{"enabled": true},
			keyPrometheus:     helmValues{"enabled": true},
		},
	}

	releaseValues, err := helmchartutil.ToRenderValues(testChart, userValues,
		helmchartutil.ReleaseOptions{Name: "demo", Namespace: "default"}, nil)
	Expect(err).NotTo(HaveOccurred())

	rendered, err := helmengine.Render(testChart, releaseValues)
	Expect(err).NotTo(HaveOccurred())

	var content map[string]any
	Expect(sigsyaml.UnmarshalStrict([]byte(rendered[renderedResourceName]), &content)).To(Succeed(),
		fmt.Sprintf("rendered output must be valid YAML without duplicate keys:\n%s",
			rendered[renderedResourceName]))
	return &unstructured.Unstructured{Object: content}
}

func renderedStringMap(resource *unstructured.Unstructured, fields ...string) map[string]string {
	GinkgoHelper()
	found, _, err := unstructured.NestedStringMap(resource.Object, fields...)
	Expect(err).NotTo(HaveOccurred())
	return found
}

var _ = Describe("Helm-rendered ServiceAccount and ServiceMonitor metadata", func() {
	const (
		multiDefault = "line one\nline two\n"
		roleA        = "arn:aws:iam::111111111111:role/a"
		roleB        = "arn:aws:iam::111111111111:role/b"
	)

	DescribeTable("ServiceAccount: a value set in the values wins and no key is duplicated",
		func(userValues helmValues, check func(labels, annotations map[string]string)) {
			resource := renderResource(AddServiceAccountLabelsAndAnnotations(saRenderInput), userValues)
			check(
				renderedStringMap(resource, "metadata", "labels"),
				renderedStringMap(resource, "metadata", keyAnnotations))
		},
		Entry("no overrides: the values from config/ are rendered",
			helmValues{},
			func(labels, annotations map[string]string) {
				Expect(labels).To(HaveLen(3))
				Expect(labels).To(HaveKeyWithValue("example.com/tier", "a"))
				Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
				Expect(annotations).To(HaveLen(2))
				Expect(annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleA))
				Expect(annotations).To(HaveKeyWithValue("example.com/multi", multiDefault))
			}),
		Entry("empty label and annotation maps behave like no overrides",
			helmValues{keyServiceAccount: helmValues{"labels": helmValues{}, keyAnnotations: helmValues{}}},
			func(labels, annotations map[string]string) {
				Expect(labels).To(HaveKeyWithValue("example.com/tier", "a"))
				Expect(annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleA))
			}),
		Entry("a label and an annotation set in the values replace the ones from config/",
			helmValues{keyServiceAccount: helmValues{
				"labels":       helmValues{"example.com/tier": "b"},
				keyAnnotations: helmValues{"eks.amazonaws.com/role-arn": roleB},
			}},
			func(labels, annotations map[string]string) {
				Expect(labels).To(HaveLen(3))
				Expect(labels).To(HaveKeyWithValue("example.com/tier", "b"))
				Expect(annotations).To(HaveLen(2))
				Expect(annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleB))
				Expect(annotations).To(HaveKeyWithValue("example.com/multi", multiDefault))
			}),
		Entry("a multi-line annotation from config/ can be replaced",
			helmValues{keyServiceAccount: helmValues{
				keyAnnotations: helmValues{"example.com/multi": "replaced"},
			}},
			func(_, annotations map[string]string) {
				Expect(annotations).To(HaveLen(2))
				Expect(annotations).To(HaveKeyWithValue("example.com/multi", "replaced"))
				Expect(annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleA))
			}),
		Entry("the chart's own Helm labels keep the chart's value",
			helmValues{keyServiceAccount: helmValues{
				"labels": helmValues{
					"app.kubernetes.io/managed-by": "foo",
					"app.kubernetes.io/name":       "foo",
				},
			}},
			func(labels, _ map[string]string) {
				Expect(labels).To(HaveLen(3))
				Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
				Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", "test-project"))
				Expect(labels).To(HaveKeyWithValue("example.com/tier", "a"))
			}),
		Entry("every key overridden plus new keys: each key appears once with the value set",
			helmValues{keyServiceAccount: helmValues{
				"labels": helmValues{"example.com/tier": "b", "example.com/new": "n"},
				keyAnnotations: helmValues{
					"eks.amazonaws.com/role-arn": roleB,
					"example.com/multi":          "x",
					"example.com/extra":          "y",
				},
			}},
			func(labels, annotations map[string]string) {
				Expect(labels).To(HaveLen(4))
				Expect(labels).To(HaveKeyWithValue("example.com/tier", "b"))
				Expect(labels).To(HaveKeyWithValue("example.com/new", "n"))
				Expect(annotations).To(HaveLen(3))
				Expect(annotations).To(HaveKeyWithValue("eks.amazonaws.com/role-arn", roleB))
				Expect(annotations).To(HaveKeyWithValue("example.com/multi", "x"))
				Expect(annotations).To(HaveKeyWithValue("example.com/extra", "y"))
			}),
	)

	Describe("ServiceMonitor", func() {
		It("keeps the labels from config/ when the values set nothing", func() {
			resource := renderResource(AddServiceMonitorLabelsAndAnnotations(smStandardLabelsOnly), helmValues{})

			labels := renderedStringMap(resource, "metadata", "labels")
			Expect(labels).To(HaveKeyWithValue("control-plane", "controller-manager"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))
		})

		It("lets the values override an annotation from config/ and keeps control-plane owned by the chart", func() {
			resource := renderResource(AddServiceMonitorLabelsAndAnnotations(smAnnotationsThenLabels), helmValues{
				keyPrometheus: helmValues{
					"labels": helmValues{labelKeyControlPlane: "custom", "release": "prometheus"},
					keyAnnotations: helmValues{
						"example.com/existing-annotation": "replaced",
						"example.com/owner":               "sre",
					},
				},
			})

			labels := renderedStringMap(resource, "metadata", "labels")
			Expect(labels).To(HaveKeyWithValue("control-plane", "controller-manager"))
			Expect(labels).To(HaveKeyWithValue("release", "prometheus"))
			Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "Helm"))

			annotations := renderedStringMap(resource, "metadata", "annotations")
			Expect(annotations).To(HaveLen(2))
			Expect(annotations).To(HaveKeyWithValue("example.com/existing-annotation", "replaced"))
			Expect(annotations).To(HaveKeyWithValue("example.com/owner", "sre"))
		})
	})
})
