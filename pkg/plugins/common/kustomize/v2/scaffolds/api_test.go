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

package scaffolds

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const defaultResources = `resources:
# comment
- service_account.yaml
- role.yaml
`

const resourcesIndentedByTwoSpaces = `resources:
# comment
  - service_account.yaml
  - role.yaml
`

const resourcesIndentedByFourSpaces = `resources:
    - service_account.yaml
`

const withoutResources = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`

const resourcesNotAList = `resources:
namespace: system
`

const nestedResourcesKey = `patches:
  resources:
    - foo.yaml
resources:
- role.yaml
`

const resourcesKeyWithComment = `resources: # RBAC
  - role.yaml
`

const resourcesBareDashItem = `resources:
  -
    role.yaml
`

var _ = Describe("resourcesItemIndent", func() {
	DescribeTable("returns the indentation of the resources list items",
		func(content, expected string) {
			Expect(resourcesItemIndent(content)).To(Equal(expected))
		},
		Entry("with the default scaffolded layout", defaultResources, ""),
		Entry("with items indented by two spaces", resourcesIndentedByTwoSpaces, "  "),
		Entry("with items indented by four spaces", resourcesIndentedByFourSpaces, "    "),
		Entry("when there is no resources list", withoutResources, ""),
		Entry("when resources is not a list", resourcesNotAList, ""),
		Entry("ignoring nested resources keys", nestedResourcesKey, ""),
		Entry("with a trailing comment on the resources key", resourcesKeyWithComment, "  "),
		Entry("with a bare dash list item", resourcesBareDashItem, "  "),
	)
})

const resourcesLastKey = `resources:
# comment
- role.yaml
- role_binding.yaml
`

const componentsOnly = `components:
- ../extra
`

const unindentedResourcesBeforeComponents = `resources:
- role.yaml
- role_binding.yaml
components:
- ../extra
`

const unindentedResourcesList = `resources:
- role.yaml
- role_binding.yaml`

const indentedResourcesBeforeCommentedComponents = `resources:
  - role.yaml

# components
components:
  - ../extra
`

const indentedResourcesList = `resources:
  - role.yaml`

const resourcesKeyWithCommentBeforeComponents = `resources: # RBAC
- role.yaml
components:
- ../extra
`

const resourcesKeyWithCommentList = `resources: # RBAC
- role.yaml`

const emptyResourcesBeforeComponents = `resources:
components:
- ../extra
`

const emptyResourcesList = `resources:`

const nestedResourcesKeyBeforeNamespace = `patches:
  resources:
  - foo.yaml
resources:
- role.yaml
namespace: system
`

const nestedResourcesKeyList = `patches:
  resources:
  - foo.yaml
resources:
- role.yaml`

var _ = Describe("resourcesListEnd", func() {
	It("returns -1 when resources is the last top-level key", func() {
		Expect(resourcesListEnd(resourcesLastKey)).To(Equal(-1))
	})

	It("returns -1 when there is no resources list", func() {
		Expect(resourcesListEnd(componentsOnly)).To(Equal(-1))
	})

	DescribeTable("returns the end of the last resources item when another key follows",
		func(content, before string) {
			end := resourcesListEnd(content)
			Expect(end).To(BeNumerically(">=", 0))
			Expect(content[:end]).To(Equal(before))
		},
		Entry("with an unindented list",
			unindentedResourcesBeforeComponents, unindentedResourcesList),
		Entry("with an indented list and a comment before the next key",
			indentedResourcesBeforeCommentedComponents, indentedResourcesList),
		Entry("with a trailing comment on the resources key",
			resourcesKeyWithCommentBeforeComponents, resourcesKeyWithCommentList),
		Entry("with an empty resources list",
			emptyResourcesBeforeComponents, emptyResourcesList),
		Entry("ignoring nested resources keys",
			nestedResourcesKeyBeforeNamespace, nestedResourcesKeyList),
	)
})
