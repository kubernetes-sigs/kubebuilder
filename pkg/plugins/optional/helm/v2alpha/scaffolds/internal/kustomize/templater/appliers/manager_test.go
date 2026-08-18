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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A blank line is not a structural boundary in YAML: it can sit inside a block scalar, where it is
// content. Ending a block there truncates the value and leaves its tail behind as stray lines at an
// indent nothing owns, which is a corrupt document rather than a mis-rendered one.
var _ = Describe("blockValueEnd", func() {
	// from is always 1 here: line 0 is the field that owns the block, and the scan starts after it.
	const from = 1

	scan := func(indentLen int, doc string) (int, []string) {
		lines := strings.Split(doc, "\n")
		return blockValueEnd(lines, from, indentLen), lines
	}

	DescribeTable("should end the block at the first line that does not belong to it",
		func(indentLen int, doc string, wantLastOwned string) {
			end, lines := scan(indentLen, doc)

			Expect(end).To(BeNumerically(">", from), "the block cannot be empty:\n%s", doc)
			Expect(lines[end-1]).To(Equal(wantLastOwned),
				"block ended at %d, owning up to %q", end, lines[end-1])
		},
		Entry("a sibling field", 8,
			"        env:\n        - name: A\n          value: b\n        image: x",
			"          value: b"),
		Entry("an outdent", 8,
			"        env:\n        - name: A\n          value: b\n      volumes: []",
			"          value: b"),
		Entry("another sequence item at the owning indent stays inside", 8,
			"        env:\n        - name: A\n          value: b\n        - name: C\n          value: d\n        image: x",
			"          value: d"),
		Entry("end of file", 8,
			"        env:\n        - name: A\n          value: b",
			"          value: b"),
	)

	// The four block scalar forms differ only in chomping and folding; all of them may contain a
	// blank line, and in every one of them that line is part of the value.
	DescribeTable("should keep a blank line that sits inside a block scalar",
		func(indicator string) {
			doc := "        env:\n        - name: MESSAGE\n          value: " + indicator +
				"\n            first line\n\n            third line\n        image: x"

			end, lines := scan(8, doc)

			Expect(lines[end-1]).To(Equal("            third line"),
				"the tail of the %s scalar was dropped; block ended at %q", indicator, lines[end-1])
			Expect(lines[end]).To(Equal("        image: x"))
		},
		Entry("literal", "|"),
		Entry("literal, chomped", "|-"),
		Entry("folded", ">"),
		Entry("folded, chomped", ">-"),
	)

	It("should keep several consecutive blank lines inside the value", func() {
		doc := "        env:\n        - name: MESSAGE\n          value: |-\n            first" +
			"\n\n\n\n            last\n        image: x"

		end, lines := scan(8, doc)

		Expect(lines[end-1]).To(Equal("            last"))
		Expect(lines[end]).To(Equal("        image: x"))
	})

	It("should keep a blank line that opens the nested content", func() {
		doc := "        env:\n        - name: A\n\n          value: b\n        image: x"

		end, lines := scan(8, doc)

		Expect(lines[end-1]).To(Equal("          value: b"))
		Expect(lines[end]).To(Equal("        image: x"))
	})

	// Trailing blank lines belong to the document, not to the block: a YAML file split on newlines
	// ends with an empty element, and swallowing it would drop the file's final newline.
	It("should not swallow blank lines that trail the block", func() {
		doc := "        env:\n        - name: A\n          value: b\n\n"

		end, lines := scan(8, doc)

		Expect(lines[end-1]).To(Equal("          value: b"))
		Expect(lines[end:]).To(Equal([]string{"", ""}))
	})

	It("should exclude sibling fields from the range even when they follow a blank line", func() {
		doc := "        env:\n        - name: A\n          value: b\n\n        image: x\n        name: manager"

		end, lines := scan(8, doc)

		Expect(lines[end:]).To(ContainElement("        image: x"))
		Expect(lines[end:]).To(ContainElement("        name: manager"))
		Expect(lines[:end]).NotTo(ContainElement("        image: x"))
	})
})

// Only indentation separates a field from a value that happens to be spelled like one. A matcher
// that trims the line first has thrown that away, and will replace an argument or a line inside a
// block scalar as though it were the container's own field.
var _ = Describe("manager field identity", func() {
	const manager = `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
%s        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
`

	It("should not mistake an argument spelled like the env field for the field", func() {
		// env:production is a plain YAML scalar - a colon not followed by a space does not open a
		// mapping - so this really is one argument string, not a field.
		source := fmt.Sprintf(manager, `      - args:
        - --leader-elect
        - env:production
        env:
        - name: FOO
          value: bar
`)

		result := templateEnvironmentVariables(source)

		Expect(result).To(ContainSubstring("- env:production"), "the argument was consumed")
		Expect(result).To(ContainSubstring("- --leader-elect"))
		Expect(result).NotTo(ContainSubstring("value: bar"), "the real env field was not replaced")
		Expect(strings.Count(result, envBlockMarker)).To(Equal(1))
	})

	It("should not mistake a line inside a block scalar for the env field", func() {
		// A command that embeds YAML: the nested env: sits deeper than the container's fields.
		source := fmt.Sprintf(manager, `      - args:
        - --leader-elect
        command:
        - |-
          env:
          - name: NESTED
            value: nested
        env:
        - name: FOO
          value: bar
`)

		result := templateEnvironmentVariables(source)

		By("leaving the command's embedded document alone")
		Expect(result).To(ContainSubstring("- name: NESTED"))
		Expect(result).To(ContainSubstring("value: nested"))

		By("replacing the container's own env field")
		Expect(result).NotTo(ContainSubstring("value: bar"))
		Expect(strings.Count(result, envBlockMarker)).To(Equal(1))
	})

	// The args matcher used to scan from the top of the document, so a sidecar declared first took
	// the match and the manager's own arguments were left untemplated.
	It("should template the manager's args when a sidecar declares args first", func() {
		source := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - args:
        - --sidecar-only
        image: sidecar:v1
        name: sidecar
      - args:
        - --leader-elect
        env: []
        image: controller:latest
        name: manager
`

		result := templateControllerManagerArgs(source)

		Expect(result).To(ContainSubstring(".Values.manager.args"), "the manager's args were not templated")
		Expect(result).To(ContainSubstring("- --sidecar-only"), "the sidecar's args were rewritten")
		Expect(strings.Count(result, ".Values.manager.args")).To(Equal(1))
	})

	DescribeTable("should not mistake a nested lookalike for a manager field",
		func(apply func(string) string, containerFields string, keep ...string) {
			result := apply(fmt.Sprintf(manager, containerFields))

			for _, want := range keep {
				Expect(result).To(ContainSubstring(want), "a nested value was treated as a field")
			}
		},
		Entry("resources inside a command's block scalar", templateResources, `      - command:
        - |-
          resources:
            limits:
              cpu: 1
        env: []
`, "cpu: 1"),
		Entry("securityContext inside a command's block scalar", templateContainerSecurityContext,
			`      - command:
        - |-
          securityContext:
            runAsUser: 1000
        env: []
`, "runAsUser: 1000"),
		Entry("image inside a command's block scalar", templateImageReference, `      - command:
        - |-
          image: nested:v1
        env: []
`, "image: nested:v1"),
	)
})

// The marker that says "this container's env is already generated" is a Helm action. Matching it
// as a substring of the whole document means any user data containing that text - an annotation, a
// value, an argument, a line in a block scalar - claims the block already exists, and the container
// silently keeps its literal env.
var _ = Describe("manager env block marker identity", func() {
	const markerAsUserData = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-project-controller-manager
spec:
  template:
    metadata:
      annotations:
        example.com/note: '{{- $envVars := list }}'
    spec:
      containers:
      - args:
        - '--note={{- $envVars := list }}'
        command:
        - |-
          {{- $envVars := list }}
        env:
        - name: FOO
          value: bar
        image: controller:latest
        name: manager
`

	// A generated marker is a whole action at the container's child indent. Trimming alone is not
	// enough to identify one - a block scalar line can trim to exactly the same text, which is the
	// mistake this whole spec exists to catch - so the count is scoped by indentation too.
	countMarkerActions := func(yamlContent string) int {
		start, end := FindManagerContainerRange(yamlContent)
		Expect(start).To(BeNumerically(">=", 0), "no manager container in:\n%s", yamlContent)

		lines := strings.Split(yamlContent, "\n")
		dashIndent, _ := LeadingWhitespace(lines[start])

		count := 0
		for i := start; i <= end && i < len(lines); i++ {
			indent, _ := LeadingWhitespace(lines[i])
			if indent == dashIndent+"  " && strings.TrimSpace(lines[i]) == envBlockMarker {
				count++
			}
		}
		return count
	}

	It("should still template the env when user data contains the marker text", func() {
		result := templateEnvironmentVariables(markerAsUserData)

		Expect(countMarkerActions(result)).To(Equal(1), "the env was not templated:\n%s", result)
		Expect(result).NotTo(ContainSubstring("value: bar"), "the literal env survived templating")

		By("leaving the user's own copies of the text alone")
		Expect(result).To(ContainSubstring(`example.com/note: '{{- $envVars := list }}'`))
		Expect(result).To(ContainSubstring(`- '--note={{- $envVars := list }}'`))
		Expect(result).To(ContainSubstring("        - |-\n          {{- $envVars := list }}"))
	})

	It("should be idempotent, so a second pass changes nothing", func() {
		once := templateEnvironmentVariables(markerAsUserData)

		Expect(templateEnvironmentVariables(once)).To(Equal(once))
		Expect(countMarkerActions(once)).To(Equal(1))
	})
})

// The env block is replaced line by line, so a value that spans lines has to be identified in full.
// Anything left behind lands after the generated block at an indent nothing owns.
var _ = Describe("templateEnvironmentVariables with multiline values", func() {
	// A value carrying a blank line and lines that look like YAML keys: neither may end the block.
	const multiline = `          value: |-
            first line

            key: not-a-field
            third line`

	container := func(envDeclaration string) string {
		return `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
` + envDeclaration + `        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
`
	}

	assertClean := func(result string) {
		By("removing every line of the original value")
		Expect(result).NotTo(ContainSubstring("first line"))
		Expect(result).NotTo(ContainSubstring("third line"))
		Expect(result).NotTo(ContainSubstring("key: not-a-field"))

		By("generating exactly one env block")
		Expect(strings.Count(result, envBlockMarker)).To(Equal(1))
		envKeys := 0
		for line := range strings.SplitSeq(result, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "env:" || strings.HasPrefix(trimmed, "- env:") {
				envKeys++
			}
		}
		Expect(envKeys).To(Equal(1), "expected one env key:\n%s", result)

		By("leaving the container's other fields intact")
		Expect(result).To(ContainSubstring("image: controller:latest"))
		Expect(result).To(ContainSubstring("name: manager"))
		Expect(result).To(ContainSubstring("cpu: 500m"))
	}

	It("should replace a block-form env whose value spans lines", func() {
		result := templateEnvironmentVariables(container(
			"      - env:\n        - name: MESSAGE\n" + multiline + "\n"))

		assertClean(result)
	})

	It("should replace an env folded onto the sequence dash whose value spans lines", func() {
		// env sorts before image and name, so a container with no earlier field has it folded.
		result := templateEnvironmentVariables(container(
			"      - args:\n        - --leader-elect\n        env:\n        - name: MESSAGE\n" + multiline + "\n"))

		assertClean(result)
	})

	It("should not reach into the container that follows", func() {
		source := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - env:
        - name: MESSAGE
` + multiline + `
        image: controller:latest
        name: manager
        resources:
          limits:
            cpu: 500m
      - image: sidecar:v1
        name: sidecar
        resources:
          limits:
            cpu: 100m
`

		result := templateEnvironmentVariables(source)

		assertClean(result)
		Expect(result).To(ContainSubstring("name: sidecar"), "the sidecar was consumed")
		Expect(result).To(ContainSubstring("cpu: 100m"))
	})
})
