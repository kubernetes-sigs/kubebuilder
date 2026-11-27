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

// Package yamlindent lets the scaffolding helpers edit YAML files that a
// formatter such as yamlfmt or prettier has re-indented.
//
// Kubebuilder scaffolds top-level sequence items at column 0 ("- role.yaml"),
// while those formatters indent them ("  - role.yaml"). Code inserted at
// column 0 into an indented list mixes indentation levels and makes the file
// invalid YAML, and text looked up with its original indentation is no longer
// found.
package yamlindent

import (
	"path/filepath"
	"strings"
)

// IsYAML reports whether the path is a YAML file
func IsYAML(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}

// ItemIndent returns the indentation of the top-level sequence around the
// position just before the given line, so items inserted there can match it.
// It looks at the sequence items before and after that position, up to the
// closest top-level key in each direction, and returns the smallest
// indentation found, which ignores the items of nested sequences. It returns
// an empty string when there are no items around the position.
func ItemIndent(lines []string, before int) string {
	indent := ""
	found := false
	visit := func(l string) bool {
		trimmed := strings.TrimLeft(l, " ")
		if strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "#") {
			return true
		}
		if !isItem(trimmed) {
			// Stop at a top-level key, which ends the sequence
			return len(trimmed) != len(l)
		}
		current := l[:len(l)-len(trimmed)]
		if !found || len(current) < len(indent) {
			indent, found = current, true
		}
		return true
	}

	for i := min(before, len(lines)) - 1; i >= 0; i-- {
		if !visit(lines[i]) {
			break
		}
	}
	for i := max(before, 0); i < len(lines); i++ {
		if !visit(lines[i]) {
			break
		}
	}
	return indent
}

// Reindent adds the indentation to every line of a code fragment made of
// top-level sequence items, such as "- role.yaml\n", including the comments
// between them. Any other fragment is returned unchanged, so comments on their
// own and indented code are never moved.
func Reindent(code, indent string) string {
	if indent == "" || !isTopLevelSequence(code) {
		return code
	}
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// ReindentAt re-indents code that will be inserted at the offset of content
// with the indentation of the sequence around it, as Reindent does. The code is
// returned unchanged when its first line would not start at column 0.
func ReindentAt(content string, offset int, code string) string {
	if offset > 0 && content[offset-1] != '\n' && !strings.HasPrefix(code, "\n") {
		return code
	}
	before := strings.Count(content[:offset], "\n")
	if offset > 0 && content[offset-1] != '\n' {
		// The code goes at the end of the line, so before the next one
		before++
	}
	return Reindent(code, ItemIndent(strings.Split(content, "\n"), before))
}

// Index returns the start and end of the first occurrence of text in content,
// or -1, -1 when there is none. Lines may be indented differently than in text,
// as long as they keep the same structure:
//   - text at column 0 matches top-level content, or the items of a top-level
//     sequence indented by a formatter
//   - nested lines stay nested, and lines at the same level stay at the same level
//   - a sequence written at the same column as its key ("key:\n- item") may be
//     indented under it ("key:\n  - item")
//
// Comments may be indented in any way.
func Index(content, text string) (start, end int) {
	if idx := strings.Index(content, text); idx != -1 {
		return idx, idx + len(text)
	}
	textLines := strings.Split(text, "\n")
	if len(textLines) == 1 {
		trimmed := strings.TrimLeft(text, " ")
		if idx := strings.Index(content, trimmed); idx != -1 {
			return idx, idx + len(trimmed)
		}
		return -1, -1
	}

	lines := strings.Split(content, "\n")
	offset := 0
	for i := 0; i+len(textLines) <= len(lines); i++ {
		if start, end, ok := matchLines(lines, i, textLines); ok {
			return offset + start, offset + end
		}
		offset += len(lines[i]) + 1
	}
	return -1, -1
}

// matchLines matches the text lines against the lines of the content starting
// at the given one, and returns the start and end of the match relative to the
// start of that line
func matchLines(lines []string, first int, textLines []string) (start, end int, ok bool) {
	var structure []indented

	offset := 0
	for i, textLine := range textLines {
		line := lines[first+i]
		body := strings.TrimLeft(textLine, " ")
		textIndent := len(textLine) - len(body)
		actual := len(line) - len(strings.TrimLeft(line, " "))
		last := i == len(textLines)-1

		switch {
		case i == 0:
			// The text may start in the middle of the line
			if !strings.HasSuffix(line, body) {
				return 0, 0, false
			}
			start = len(line) - len(body)
			if body == "" || strings.TrimLeft(line[:start], " ") != "" {
				// The indentation of the line is not part of the text
				offset += len(line) + 1
				continue
			}
		case body == "" && last:
			// The text ends with a newline, before the indentation of the next line
			end = offset
		case body == "":
			if strings.TrimSpace(line) != "" {
				return 0, 0, false
			}
		case last:
			if !strings.HasPrefix(line[actual:], body) {
				return 0, 0, false
			}
			end = offset + actual + len(body)
		default:
			if line[actual:] != body {
				return 0, 0, false
			}
		}
		if body != "" && !strings.HasPrefix(body, "#") {
			structure = append(structure, indented{first + i, textIndent, actual, body})
		}
		offset += len(line) + 1
	}

	return start, end, sameStructure(lines, structure)
}

// indented is a line of the text that is not a comment, with its indentation
// in the text and in the content
type indented struct {
	line         int
	text, actual int
	body         string
}

// sameStructure reports whether the lines are nested in the content as they
// are in the text
func sameStructure(lines []string, structure []indented) bool {
	if len(structure) > 0 {
		head := structure[0]
		if head.text == 0 && head.actual != 0 &&
			(!isItem(head.body) || len(ItemIndent(lines, head.line)) != head.actual) {
			return false
		}
	}

	// levels holds the indentation of the enclosing lines, from the outermost
	type level struct {
		text, actual int
		sequence     bool
	}
	var levels []level
	for i, l := range structure {
		for len(levels) > 0 {
			top := levels[len(levels)-1]
			if l.text < top.text || (l.text == top.text && top.sequence && !isItem(l.body)) {
				levels = levels[:len(levels)-1]
				continue
			}
			break
		}
		switch {
		case len(levels) == 0 || l.text > levels[len(levels)-1].text:
			if len(levels) > 0 && l.actual <= levels[len(levels)-1].actual {
				return false
			}
			levels = append(levels, level{text: l.text, actual: l.actual})
		case l.actual == levels[len(levels)-1].actual:
		case i > 0 && isItem(l.body) && opensBlock(structure[i-1].body) && structure[i-1].text == l.text &&
			!levels[len(levels)-1].sequence && l.actual > levels[len(levels)-1].actual:
			// A sequence that a formatter indented under its key
			levels = append(levels, level{text: l.text, actual: l.actual, sequence: true})
		default:
			return false
		}
	}
	return true
}

// isTopLevelSequence reports whether the code is made of sequence items that
// start at column 0, optionally with comments between them and indented lines
// that continue an item
func isTopLevelSequence(code string) bool {
	hasItem := false
	for line := range strings.SplitSeq(code, "\n") {
		switch {
		case line == "", strings.HasPrefix(line, "#"):
		case isItem(line):
			hasItem = true
		case hasItem && strings.HasPrefix(line, " "):
		default:
			return false
		}
	}
	return hasItem
}

// isItem reports whether the line, without its indentation, starts a sequence
// item, including a bare "-" whose value is on the following lines
func isItem(trimmed string) bool {
	return trimmed == "-" || strings.HasPrefix(trimmed, "- ")
}

// keyOf returns the key of a line such as "key:" or "key: # comment", or an
// empty string when the line does not open a block
func keyOf(line string) string {
	line, _, _ = strings.Cut(line, " #")
	line = strings.TrimSpace(line)
	if !strings.HasSuffix(line, ":") {
		return ""
	}
	return strings.TrimSuffix(line, ":")
}

// opensBlock reports whether the line, without its indentation, is a key whose
// value is on the following lines
func opensBlock(trimmed string) bool {
	return keyOf(trimmed) != ""
}
