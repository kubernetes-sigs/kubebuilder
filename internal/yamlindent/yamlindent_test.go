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

package yamlindent

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const marker = "# +kubebuilder:scaffold:x"

var _ = Describe("yamlindent", func() {
	DescribeTable("IsYAML",
		func(path string, expected bool) {
			Expect(IsYAML(path)).To(Equal(expected))
		},
		Entry("a .yaml file", "config/rbac/kustomization.yaml", true),
		Entry("a .yml file", "config/rbac/kustomization.yml", true),
		Entry("a Go file", "cmd/main.go", false),
		Entry("a file without extension", "Makefile", false),
	)

	DescribeTable("ItemIndent",
		func(content string, expected string) {
			lines := strings.Split(content, "\n")
			line := -1
			for i, l := range lines {
				if strings.TrimSpace(l) == marker {
					line = i
				}
			}
			Expect(line).NotTo(Equal(-1), "the content must contain the marker")
			Expect(ItemIndent(lines, line)).To(Equal(expected))
		},
		Entry("returns no indentation for items at column 0, as scaffolded", `resources:
- a.yaml
`+marker+`
`, ""),
		Entry("returns the indentation of items before a marker left at column 0", `resources:
  - a.yaml
  - b.yaml
`+marker+`
`, "  "),
		Entry("returns the indentation of items indented by four spaces", `resources:
    - a.yaml
`+marker+`
`, "    "),
		Entry("returns the indentation of items after the line", `resources:
`+marker+`
  - a.yaml
`, "  "),
		Entry("skips comments between items", `resources:
  - a.yaml
  # a comment
`+marker+`
`, "  "),
		Entry("ignores the items of nested sequences", `patches:
  - path: patch.yaml
    target:
      - a
`+marker+`
`, "  "),
		Entry("does not look past a top-level key", `resources:
  - a.yaml
patches:
`+marker+`
`, ""),
		Entry("returns no indentation when there are no items", `resources:
`+marker+`
`, ""),
	)

	DescribeTable("Reindent",
		func(code, indent, expected string) {
			Expect(Reindent(code, indent)).To(Equal(expected))
		},
		Entry("indents sequence items",
			"- a.yaml\n", "  ", "  - a.yaml\n"),
		Entry("indents several items and keeps blank lines",
			itemsAfterNewline, "  ", indentedItemsAfterNewline),
		Entry("indents the lines that continue an item",
			itemWithFields, "  ", indentedItemWithFields),
		Entry("does not change a fragment without indentation to add",
			"- a.yaml\n", "", "- a.yaml\n"),
		Entry("indents the comments between items",
			itemsWithComment, "  ", indentedItemsWithComment),
		Entry("does not change comments",
			commentedItem, "  ", commentedItem),
		Entry("does not change a fragment that is already indented",
			"  - a.yaml\n", "  ", "  - a.yaml\n"),
		Entry("does not change a fragment that mixes items with other content",
			itemAndKey, "  ", itemAndKey),
	)

	DescribeTable("Index",
		func(content, text string, expected string) {
			start, end := Index(content, text)
			if expected == "" {
				Expect(start).To(Equal(-1))
				Expect(end).To(Equal(-1))
				return
			}
			Expect(content[start:end]).To(Equal(expected))
		},
		Entry("finds text with the same indentation",
			unindentedResources, "- a.yaml", "- a.yaml"),
		Entry("finds lines indented differently",
			resourcesWithIndentedComments, unindentedComments, indentedCommentsMatch),
		Entry("finds items indented differently",
			indentedResources, itemsAfterNewline, indentedItemsAfterNewline),
		Entry("does not find text that is not there",
			indentedResources, "- c.yaml", ""),
		Entry("does not ignore differences other than indentation",
			indentedResources, itemsAC, ""),
		Entry("finds a sequence indented under its key",
			indentedConfigurations, unindentedConfigurations, indentedConfigurations),
		Entry("finds the key after a sequence indented under the previous key",
			indentedResourcesBeforeNamespace, unindentedResourcesBeforeNamespace, indentedResourcesBeforeNamespace),
		Entry("finds commented lines that were indented",
			patchesWithIndentedCommentedItem, commentedPatch, indentedCommentedPatchMatch),
		Entry("does not find top-level items in a nested sequence",
			patchesWithNestedItems, itemsAfterNewline, ""),
		Entry("does not find a top-level key that is nested",
			nestedResourcesKey, unindentedResourcesA, ""),
		Entry("does not find nested lines at the level of their parent",
			unnestedKey, nestedKey, ""),
		Entry("does not find lines at the same level that were nested",
			itemsAtDifferentLevels, itemsAfterNewline, ""),
	)

	DescribeTable("ReindentAt",
		func(content string, offset int, code, expected string) {
			Expect(ReindentAt(content, offset, code)).To(Equal(expected))
		},
		Entry("indents code inserted at the start of a line",
			indentedResourcesA, len(indentedResourcesA), "- b.yaml\n", "  - b.yaml\n"),
		Entry("indents code that starts with a newline",
			indentedResourcesAWithoutNewline, len(indentedResourcesAWithoutNewline), "\n- b.yaml", "\n  - b.yaml"),
		Entry("indents code inserted before a top-level key that follows the sequence",
			indentedResourcesBeforeNamespace, strings.Index(indentedResourcesBeforeNamespace, "namespace:"),
			"- b.yaml\n", "  - b.yaml\n"),
		Entry("does not indent code inserted after the indentation of a line",
			indentedResourcesWithCommentedItem, strings.Index(indentedResourcesWithCommentedItem, "#- b.yaml"),
			"- b.yaml\n", "- b.yaml\n"),
	)
})

// Reindent

const itemsAfterNewline = `
- a.yaml
- b.yaml`

const indentedItemsAfterNewline = `
  - a.yaml
  - b.yaml`

const itemWithFields = `- path: patch.yaml
  target:
    kind: A
`

const indentedItemWithFields = `  - path: patch.yaml
    target:
      kind: A
`

const itemsWithComment = `- a.yaml
# note
- b.yaml
`

const indentedItemsWithComment = `  - a.yaml
  # note
  - b.yaml
`

const commentedItem = `#     - select:
#         kind: A
`

const itemAndKey = `- a.yaml
key: value
`

// Index

const unindentedResources = `resources:
- a.yaml
`

const indentedResources = `resources:
  - a.yaml
  - b.yaml
`

const itemsAC = `- a.yaml
- c.yaml`

const resourcesWithIndentedComments = `resources:
  # first
  # second
  - a.yaml
`

const unindentedComments = `# first
# second
`

const indentedCommentsMatch = `# first
  # second
`

const indentedConfigurations = `configurations:
  - kustomizeconfig.yaml`

const unindentedConfigurations = `configurations:
- kustomizeconfig.yaml`

const indentedResourcesBeforeNamespace = `resources:
  - a.yaml
namespace: system`

const unindentedResourcesBeforeNamespace = `resources:
- a.yaml
namespace: system`

const patchesWithIndentedCommentedItem = `patches:
  - path: a.yaml
  #- path: b.yaml
  #  target:
`

const commentedPatch = `#- path: b.yaml
#  target:
`

const indentedCommentedPatchMatch = `#- path: b.yaml
  #  target:
`

const patchesWithNestedItems = `patches:
  - path: p.yaml
    target:
      - a.yaml
      - b.yaml
`

const nestedResourcesKey = `patches:
  resources:
    - a.yaml
`

const unindentedResourcesA = `resources:
- a.yaml`

const unnestedKey = `foo:
bar: 1
`

const nestedKey = `foo:
  bar: 1
`

const itemsAtDifferentLevels = `resources:
  - a.yaml
    - b.yaml
`

// ReindentAt

const indentedResourcesA = `resources:
  - a.yaml
`

const indentedResourcesAWithoutNewline = `resources:
  - a.yaml`

const indentedResourcesWithCommentedItem = `resources:
  - a.yaml
  #- b.yaml
`
