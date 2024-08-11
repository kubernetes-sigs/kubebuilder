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
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"
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
			return "", err
		}
		res[i] = source[r.Int64()]
	}
	return string(res), nil
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// InsertCode searches target content in the file and insert `toInsert` after the target.
func InsertCode(filename, target, code string) error {
	// false positive
	// nolint:gosec
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	idx := strings.Index(string(contents), target)
	if idx == -1 {
		return fmt.Errorf("string %s not found in %s", target, string(contents))
	}
	out := string(contents[:idx+len(target)]) + code + string(contents[idx+len(target):])
	// false positive
	// nolint:gosec
	return os.WriteFile(filename, []byte(out), 0644)
}

// InsertCodeIfNotExist insert code if it does not already exists
func InsertCodeIfNotExist(filename, target, code string) error {
	// false positive
	// nolint:gosec
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	idx := strings.Index(string(contents), code)
	if idx != -1 {
		return nil
	}

	return InsertCode(filename, target, code)
}

// AppendCodeIfNotExist checks if the code does not already exist in the file, and if not, appends it to the end.
func AppendCodeIfNotExist(filename, code string) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if strings.Contains(string(contents), code) {
		return nil // Code already exists, no need to append.
	}

	return AppendCodeAtTheEnd(filename, code)
}

// AppendCodeAtTheEnd appends the given code at the end of the file.
func AppendCodeAtTheEnd(filename, code string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			return
		}
	}()

	_, err = f.WriteString(code)
	return err
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
func UncommentCode(filename, target, prefix string) error {
	// false positive
	// nolint:gosec
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %s to be uncomment", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	if !scanner.Scan() {
		return nil
	}
	for {
		_, err := out.WriteString(strings.TrimPrefix(scanner.Text(), prefix))
		if err != nil {
			return err
		}
		// Avoid writing a newline in case the previous line was the last in target.
		if !scanner.Scan() {
			break
		}
		if _, err := out.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = out.Write(content[idx+len(target):])
	if err != nil {
		return err
	}
	// false positive
	// nolint:gosec
	return os.WriteFile(filename, out.Bytes(), 0644)
}

// CommentCode searches for target in the file and adds the comment prefix
// to the target content. The target content may span multiple lines.
func CommentCode(filename, target, prefix string) error {
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	strContent := string(content)

	// Find the target code to be commented
	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %s to be commented", target)
	}

	// Create a buffer to hold the modified content
	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return err
	}

	// Add the comment prefix to each line of the target code
	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	for scanner.Scan() {
		_, err := out.WriteString(prefix + scanner.Text() + "\n")
		if err != nil {
			return err
		}
	}

	// Write the rest of the file content
	_, err = out.Write(content[idx+len(target):])
	if err != nil {
		return err
	}

	// Write the modified content back to the file
	return os.WriteFile(filename, out.Bytes(), 0644)
}

// EnsureExistAndReplace check if the content exists and then do the replace
func EnsureExistAndReplace(input, match, replace string) (string, error) {
	if !strings.Contains(input, match) {
		return "", fmt.Errorf("can't find %q", match)
	}
	return strings.Replace(input, match, replace, -1), nil
}

func HasFragment(path, target string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	// false positive
	// nolint:gosec
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	if !strings.Contains(string(b), target) {
		return false, nil
	}
	return true, nil
}

// ReplaceInFile replaces all instances of old with new in the file at path.
func ReplaceInFile(path, old, new string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	// false positive
	// nolint:gosec
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), old) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.Replace(string(b), old, new, -1)
	err = os.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

// ReplaceRegexInFile finds all strings that match `match` and replaces them
// with `replace` in the file at path.
func ReplaceRegexInFile(path, match, replace string) error {
	matcher, err := regexp.Compile(match)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	// false positive
	// nolint:gosec
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := matcher.ReplaceAllString(string(b), replace)
	if s == string(b) {
		return errors.New("unable to find the content to be replaced")
	}
	err = os.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

// HasFileContentWith check if given `text` can be found in file
func HasFileContentWith(path, text string) (bool, error) {
	// nolint:gosec
	contents, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(contents), text), nil
}
