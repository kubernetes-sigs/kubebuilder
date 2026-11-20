/*
Copyright 2019 The Kubernetes Authors.

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

package util

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/internal/yamlindent"
)

const (
	// KubebuilderBinName define the name of the kubebuilder binary to be used in the tests
	KubebuilderBinName = "kubebuilder"
)

// RandomSuffix returns a 4-letter string.
func RandomSuffix() (string, error) {
	source := []rune("abcdefghijklmnopqrstuvwxyz")
	res := make([]rune, 4)
	for i := range res {
		bi := new(big.Int)
		r, err := rand.Int(rand.Reader, bi.SetInt64(int64(len(source))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		res[i] = source[r.Int64()]
	}
	return string(res), nil
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.SplitSeq(output, "\n")
	for element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// InsertCode searches target content in the file and insert `toInsert` after the target.
// In YAML files, the target is found even when its lines were indented.
func InsertCode(filename, target, code string) error {
	//nolint:gosec // false positive
	contents, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	_, end := index(filename, string(contents), target)
	if end == -1 {
		return fmt.Errorf("string %s not found in %s", target, string(contents))
	}
	if yamlindent.IsYAML(filename) {
		code = yamlindent.ReindentAt(string(contents), end, code)
	}
	out := string(contents[:end]) + code + string(contents[end:])
	//nolint:gosec // false positive
	if errWriteFile := os.WriteFile(filename, []byte(out), 0o644); errWriteFile != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, errWriteFile)
	}

	return nil
}

// InsertCodeIfNotExist insert code if it does not already exist.
// In YAML files, the code is found even when its lines were indented.
func InsertCodeIfNotExist(filename, target, code string) error {
	//nolint:gosec // false positive
	contents, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}

	if start, _ := index(filename, string(contents), code); start != -1 {
		return nil
	}

	return InsertCode(filename, target, code)
}

// AppendCodeIfNotExist checks if the code does not already exist in the file, and if not, appends it to the end.
// In YAML files, the code is found even when its lines were indented.
func AppendCodeIfNotExist(filename, code string) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}

	if start, _ := index(filename, string(contents), code); start != -1 {
		return nil // Code already exists, no need to append.
	}

	return AppendCodeAtTheEnd(filename, code)
}

// index returns the start and end of the first occurrence of text in the content
// of the file, or -1, -1 when there is none. In YAML files it allows the lines to
// be indented differently, as long as they keep the same structure, so it also
// finds text in files reformatted by tools such as yamlfmt or prettier.
func index(filename, content, text string) (start, end int) {
	if yamlindent.IsYAML(filename) {
		return yamlindent.Index(content, text)
	}
	if idx := strings.Index(content, text); idx != -1 {
		return idx, idx + len(text)
	}
	return -1, -1
}

// columnOf returns the column of the offset in the content when only spaces
// precede it on its line, or -1 otherwise
func columnOf(content string, offset int) int {
	lineStart := strings.LastIndex(content[:offset], "\n") + 1
	if strings.TrimLeft(content[lineStart:offset], " ") != "" {
		return -1
	}
	return offset - lineStart
}

// AppendCodeAtTheEnd appends the given code at the end of the file.
func AppendCodeAtTheEnd(filename, code string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filename, err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			return
		}
	}()

	if _, errWriteString := f.WriteString(code); errWriteString != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, errWriteString)
	}

	return nil
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
// In YAML files, the target is found even when its lines were indented.
func UncommentCode(filename, target, prefix string) error {
	//nolint:gosec // false positive
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	start, end := index(filename, strContent, target)
	if start < 0 {
		return fmt.Errorf("unable to find the code %q to be uncommented", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:start])
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}

	// Split on "\n" instead of scanning lines so that a newline at the end
	// of target is kept and the next line of the file is not joined to it.
	isYAML := yamlindent.IsYAML(filename)
	column := -1
	if isYAML {
		column = columnOf(strContent, start)
	}
	lines := strings.Split(strContent[start:end], "\n")
	for i, line := range lines {
		body, indent := line, ""
		if isYAML && i > 0 {
			// Formatters such as yamlfmt or prettier indent comments, not always
			// consistently, so give every line the column of the first one
			body = strings.TrimLeft(line, " ")
			indent = line[:len(line)-len(body)]
			if column >= 0 && body != "" {
				indent = strings.Repeat(" ", column)
			}
		}
		lines[i] = indent + strings.TrimPrefix(body, prefix)
	}
	code := strings.Join(lines, "\n")
	if isYAML {
		// Formatters indent sequence items while leaving the comments after
		// the last item at column 0
		code = yamlindent.ReindentAt(strContent, start, code)
	}
	if _, err = out.WriteString(code); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}

	if _, err = out.Write(content[end:]); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}
	//nolint:gosec // false positive
	if err = os.WriteFile(filename, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}

// CommentCode searches for target in the file and adds the comment prefix
// to the target content. The target content may span multiple lines.
// In YAML files, the target is found even when its lines were indented.
func CommentCode(filename, target, prefix string) error {
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	// Find the target code to be commented
	start, end := index(filename, strContent, target)
	if start < 0 {
		return fmt.Errorf("failed to find the code %q to be commented", target)
	}

	// Create a buffer to hold the modified content
	out := new(bytes.Buffer)
	if _, err = out.Write(content[:start]); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}

	// The column of the prefix on the lines after the first one
	column := 0
	if yamlindent.IsYAML(filename) {
		column = max(columnOf(strContent, start), 0)
	}

	// Add the comment prefix to each line of the target code, keeping its
	// line endings: a newline is only written where target has one.
	lines := strings.Split(strContent[start:end], "\n")
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			break
		}
		at := 0
		if i > 0 {
			at = min(column, len(line)-len(strings.TrimLeft(line, " ")))
		}
		lines[i] = line[:at] + prefix + line[at:]
	}
	if _, err = out.WriteString(strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}

	// Write the rest of the file content
	if _, err = out.Write(content[end:]); err != nil {
		return fmt.Errorf("failed to write to file %q: %w", filename, err)
	}

	// Write the modified content back to the file
	if err = os.WriteFile(filename, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}

// EnsureExistAndReplace check if the content exists and then do the replacement
func EnsureExistAndReplace(input, match, replace string) (string, error) {
	if !strings.Contains(input, match) {
		return "", fmt.Errorf("can't find %q", match)
	}
	return strings.ReplaceAll(input, match, replace), nil
}

// ReplaceInFile replaces all instances of old with new in the file at path.
func ReplaceInFile(path, oldValue, newValue string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	//nolint:gosec // false positive
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}
	if !strings.Contains(string(b), oldValue) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.ReplaceAll(string(b), oldValue, newValue)
	if err = os.WriteFile(path, []byte(s), info.Mode()); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}
	return nil
}

// ReplaceRegexInFile finds all strings that match `match` and replaces them
// with `replace` in the file at path.
//
// This function is currently unused in the Kubebuilder codebase,
// but is used by other projects and may be used in Kubebuilder in the future.
func ReplaceRegexInFile(path, match, replace string) error {
	matcher, err := regexp.Compile(match)
	if err != nil {
		return fmt.Errorf("failed to compile regular expression %q: %w", match, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	//nolint:gosec // false positive
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}
	s := matcher.ReplaceAllString(string(b), replace)
	if s == string(b) {
		return errors.New("unable to find the content to be replaced")
	}

	if err = os.WriteFile(path, []byte(s), info.Mode()); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}

	return nil
}

// HasFileContentWith check if given `text` can be found in file.
// In YAML files, the text is found even when its lines were indented.
func HasFileContentWith(path, text string) (bool, error) {
	//nolint:gosec
	contents, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	start, _ := index(path, string(contents), text)
	return start != -1, nil
}
