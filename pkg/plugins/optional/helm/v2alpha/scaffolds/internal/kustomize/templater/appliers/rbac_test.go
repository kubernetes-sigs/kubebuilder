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

var _ = Describe("TemplateServiceAccountNameInDeployment", func() {
	const helper = `{{ include "test-project.serviceAccountName" . }}`
	deployment := func(selectorLabel, podLabel, serviceAccountName string) string {
		return `apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
` + selectorLabel + `  template:
    metadata:
      labels:
        control-plane: controller-manager
` + podLabel + `    spec:
      {{- if .Values.manager.enabled }}
      serviceAccountName: ` + serviceAccountName + `
      containers:
      - name: manager
        serviceAccountName: controller-manager
`
	}

	DescribeTable("templates the pod serviceAccountName only when it names the manager",
		func(input, expected string) {
			Expect(TemplateServiceAccountNameInDeployment("test-project", "test-project", input)).To(Equal(expected))
		},
		Entry("the plain manager name",
			deployment("", "", "controller-manager"), deployment("", "", helper)),
		Entry("the prefixed manager name",
			deployment("", "", "test-project-controller-manager"), deployment("", "", helper)),
		Entry("the templated manager name",
			deployment("", "", ResourceNameTemplate("test-project", "controller-manager")), deployment("", "", helper)),
		Entry("another ServiceAccount",
			deployment("", "", "custom-sa"), deployment("", "", "custom-sa")),
		Entry("a label and a selector with the same key",
			deployment("      serviceAccountName: controller-manager\n", "        serviceAccountName: controller-manager\n",
				"controller-manager"),
			deployment("      serviceAccountName: controller-manager\n", "        serviceAccountName: controller-manager\n",
				helper)),
		Entry("no pod serviceAccountName",
			"apiVersion: apps/v1\nkind: Deployment\nspec:\n  template:\n    spec:\n      containers: []\n",
			"apiVersion: apps/v1\nkind: Deployment\nspec:\n  template:\n    spec:\n      containers: []\n"),
	)
})

var _ = Describe("TemplateServiceAccountNameInBindings", func() {
	const managerSubject = `- kind: ServiceAccount
  name: controller-manager
  namespace: {{ .Release.Namespace }}`
	const userSubject = `- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: alice`
	const groupSubject = `- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: operators`
	const otherSubject = `- kind: ServiceAccount
  name: observer
  namespace: external`
	const foreignManagerSubject = `- kind: ServiceAccount
  name: controller-manager
  namespace: other-team`
	const helper = `{{ include "test-project.serviceAccountName" . }}`
	const managerName = `{{ include "test-project.resourceName" (dict "suffix" "controller-manager" "context" $) }}`
	const templatedSubject = `- kind: ServiceAccount
  name: ` + helper + `
  namespace: {{ .Release.Namespace }}`
	const annotation = `metadata:
  annotations:
    example.com/subjects: |
      subjects:
      - kind: ServiceAccount
        name: controller-manager
`
	withSubjects := func(subjects string) string {
		return "apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nsubjects:\n" + subjects
	}

	DescribeTable("templates every matching subject while preserving other entries",
		func(input, expected string) {
			Expect(TemplateServiceAccountNameInBindings("test-project", "test-project", input)).To(Equal(expected))
		},
		Entry("manager first",
			withSubjects(managerSubject+"\n"+userSubject),
			withSubjects(templatedSubject+"\n"+userSubject)),
		Entry("manager after a User",
			withSubjects(userSubject+"\n"+managerSubject),
			withSubjects(userSubject+"\n"+templatedSubject)),
		Entry("manager between a Group and another ServiceAccount",
			withSubjects(groupSubject+"\n"+managerSubject+"\n"+otherSubject),
			withSubjects(groupSubject+"\n"+templatedSubject+"\n"+otherSubject)),
		Entry("manager after an unrelated ServiceAccount",
			withSubjects(otherSubject+"\n"+managerSubject),
			withSubjects(otherSubject+"\n"+templatedSubject)),
		Entry("multiple manager subjects",
			withSubjects(managerSubject+"\n"+userSubject+"\n"+managerSubject),
			withSubjects(templatedSubject+"\n"+userSubject+"\n"+templatedSubject)),
		Entry("prefixed manager name",
			withSubjects(userSubject+"\n"+
				strings.Replace(managerSubject, "controller-manager", "test-project-controller-manager", 1)),
			withSubjects(userSubject+"\n"+templatedSubject)),
		Entry("resourceName helper from an earlier substitution",
			withSubjects(userSubject+"\n"+strings.Replace(managerSubject, "controller-manager", managerName, 1)),
			withSubjects(userSubject+"\n"+templatedSubject)),
		Entry("manager after a same-named ServiceAccount in another namespace",
			withSubjects(foreignManagerSubject+"\n"+managerSubject),
			withSubjects(foreignManagerSubject+"\n"+templatedSubject)),
		Entry("same-named ServiceAccount in another namespace after the manager",
			withSubjects(managerSubject+"\n"+foreignManagerSubject),
			withSubjects(templatedSubject+"\n"+foreignManagerSubject)),
		Entry("prefixed manager name in another namespace",
			withSubjects(strings.Replace(foreignManagerSubject, "controller-manager", "test-project-controller-manager", 1)),
			withSubjects(strings.Replace(foreignManagerSubject, "controller-manager", "test-project-controller-manager", 1))),
		Entry("resourceName helper in another namespace",
			withSubjects(strings.Replace(foreignManagerSubject, "controller-manager", managerName, 1)),
			withSubjects(strings.Replace(foreignManagerSubject, "controller-manager", managerName, 1))),
		Entry("foreign namespace before kind",
			withSubjects("- namespace: other-team\n  name: controller-manager\n  kind: ServiceAccount"),
			withSubjects("- namespace: other-team\n  name: controller-manager\n  kind: ServiceAccount")),
		Entry("manager without a namespace",
			withSubjects("- kind: ServiceAccount\n  name: controller-manager"),
			withSubjects("- kind: ServiceAccount\n  name: "+helper)),
		Entry("manager with an empty namespace",
			withSubjects("- kind: ServiceAccount\n  name: controller-manager\n  namespace:"),
			withSubjects("- kind: ServiceAccount\n  name: "+helper+"\n  namespace:")),
		Entry("manager with a serialized empty namespace",
			withSubjects("- kind: ServiceAccount\n  name: controller-manager\n  namespace: \"\""),
			withSubjects("- kind: ServiceAccount\n  name: "+helper+"\n  namespace: \"\"")),
		Entry("manager with a serialized null namespace",
			withSubjects("- kind: ServiceAccount\n  name: controller-manager\n  namespace: null"),
			withSubjects("- kind: ServiceAccount\n  name: "+helper+"\n  namespace: null")),
		Entry("apiGroup before kind",
			withSubjects("- apiGroup: \"\"\n  kind: ServiceAccount\n  name: controller-manager\n"+
				"  namespace: {{ .Release.Namespace }}"),
			withSubjects("- apiGroup: \"\"\n  kind: ServiceAccount\n  name: "+helper+"\n  namespace: {{ .Release.Namespace }}")),
		Entry("name before kind",
			withSubjects("- name: controller-manager\n  namespace: {{ .Release.Namespace }}\n  kind: ServiceAccount"),
			withSubjects("- name: "+helper+"\n  namespace: {{ .Release.Namespace }}\n  kind: ServiceAccount")),
		Entry("indented subject list",
			withSubjects("  - kind: ServiceAccount\n    namespace: {{ .Release.Namespace }}\n    name: controller-manager\n"),
			withSubjects("  - kind: ServiceAccount\n    namespace: {{ .Release.Namespace }}\n    name: "+helper+"\n")),
		Entry("User and Group with manager names",
			withSubjects("- kind: User\n  name: controller-manager\n- kind: Group\n  name: test-project-controller-manager"),
			withSubjects("- kind: User\n  name: controller-manager\n- kind: Group\n  name: test-project-controller-manager")),
		Entry("similarly named ServiceAccounts",
			withSubjects(strings.ReplaceAll(managerSubject, "controller-manager", "controller-manager-extra")+
				"\n"+strings.ReplaceAll(managerSubject, "controller-manager", "test-project-controller-manager-extra")),
			withSubjects(strings.ReplaceAll(managerSubject, "controller-manager", "controller-manager-extra")+
				"\n"+strings.ReplaceAll(managerSubject, "controller-manager", "test-project-controller-manager-extra"))),
		Entry("already resolved helper",
			withSubjects(templatedSubject), withSubjects(templatedSubject)),
		Entry("missing subjects",
			"kind: RoleBinding\nmetadata:\n  name: controller-manager\n",
			"kind: RoleBinding\nmetadata:\n  name: controller-manager\n"),
		Entry("empty subjects", withSubjects(""), withSubjects("")),
		Entry("inline empty subjects", "kind: RoleBinding\nsubjects: []\n", "kind: RoleBinding\nsubjects: []\n"),
		Entry("subject without a name",
			withSubjects("- kind: ServiceAccount\n  namespace: system"),
			withSubjects("- kind: ServiceAccount\n  namespace: system")),
		Entry("subject without a kind",
			withSubjects("- name: controller-manager\n  namespace: system"),
			withSubjects("- name: controller-manager\n  namespace: system")),
		Entry("matching fields outside subjects",
			withSubjects(managerSubject)+"\nother:\n"+managerSubject,
			withSubjects(templatedSubject)+"\nother:\n"+managerSubject),
		Entry("subjects in an annotation before the binding subjects",
			annotation+withSubjects(userSubject+"\n"+managerSubject),
			annotation+withSubjects(userSubject+"\n"+templatedSubject)),
		Entry("subjects only in an annotation", annotation, annotation),
	)
})
