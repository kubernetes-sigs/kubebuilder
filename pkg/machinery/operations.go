/*
Copyright 2021 The Kubernetes Authors.

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

package machinery

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

func readFile(fs Filesystem, filename string) (string, os.FileInfo, error) {
	info, err := fs.FS.Stat(filename)
	if err != nil {
		return "", nil, err
	}

	content, err := afero.ReadFile(fs.FS, filename)
	if err != nil {
		return "", nil, err
	}

	return string(content), info, nil
}

func writeFile(fs Filesystem, filename string, content string, mode os.FileMode) error {
	return afero.WriteFile(fs.FS, filename, []byte(content), mode)
}

// InsertBefore inserts the provided insertions in the referenced file.
// Insertions are an even number of strings grouped by pairs, where the first one defines the target that
// needs to be searched while the second one defines the code that will be inserted before this target.
func InsertBefore(fs Filesystem, filename string, insertions ...string) error {
	if len(insertions)%2 != 0 {
		panic(fmt.Errorf("an even number of insertion strings are required"))
	}

	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	for i := 0; i < len(insertions); i += 2 {
		target := insertions[i]
		idx := strings.Index(content, target)
		if idx < 0 {
			return fmt.Errorf("unable to find %q", target)
		}
		content = content[:idx] + insertions[i+1] + content[idx:]
	}

	return writeFile(fs, filename, content, info.Mode())
}

// InsertAfter inserts the provided insertions in the referenced file.
// Insertions are an even number of strings grouped by pairs, where the first one defines the target that
// needs to be searched while the second one defines the code that will be inserted after this target.
func InsertAfter(fs Filesystem, filename string, insertions ...string) error {
	if len(insertions)%2 != 0 {
		panic(fmt.Errorf("an even number of insertion strings are required"))
	}

	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	for i := 0; i < len(insertions); i += 2 {
		target := insertions[i]
		idx := strings.Index(content, target) + len(target)
		if idx < 0 {
			return fmt.Errorf("unable to find %q", target)
		}
		content = content[:idx] + insertions[i+1] + content[idx:]
	}

	return writeFile(fs, filename, content, info.Mode())
}

// Replace replaces the provided replacements in the referenced file.
// Replacements are an even number of strings grouped by pairs, where the first one defines the target that
// needs to be replaced while the second one defines the code that will be inserted instead of this target.
func Replace(fs Filesystem, filename string, replacements ...string) error {
	if len(replacements)%2 != 0 {
		panic(fmt.Errorf("an even number of replacement strings are required"))
	}

	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	for i := 0; i < len(replacements); i += 2 {
		target := replacements[i]
		if !strings.Contains(content, target) {
			return fmt.Errorf("unable to find %q", target)
		}
		content = strings.Replace(content, target, replacements[i+1], -1)
	}

	return writeFile(fs, filename, content, info.Mode())
}

// ReplaceRegexp inserts the provided code replacing the matched target in the referenced file.
// Replacements are an even number of strings grouped by pairs, where the first one defines the regexp of the target
// that needs to be replaced while the second one defines the code that will be inserted instead of this target.
func ReplaceRegexp(fs Filesystem, filename string, replacements ...string) error {
	if len(replacements)%2 != 0 {
		panic(fmt.Errorf("an even number of replacement strings are required"))
	}

	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	for i := 0; i < len(replacements); i += 2 {
		target := replacements[i]
		matcher, err := regexp.Compile(target)
		if err != nil {
			return err
		}
		out := matcher.ReplaceAllString(content, replacements[i+1])
		if out == content {
			return fmt.Errorf("unable to find %q", target)
		}
		content = out
	}

	return writeFile(fs, filename, content, info.Mode())
}

// AddPrefix adds the provided prefix to all lines in the provided code block.
func AddPrefix(fs Filesystem, filename, codeBlock, prefix string) error {
	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	idx := strings.Index(content, codeBlock)
	if idx < 0 {
		return fmt.Errorf("unable to find %q", codeBlock)
	}
	modifiedCodeBlock := strings.Builder{}
	for _, line := range strings.Split(codeBlock, "\n") {
		if _, err := modifiedCodeBlock.WriteString(prefix + line + "\n"); err != nil {
			return err
		}
	}
	content = content[:idx] + strings.TrimSuffix(modifiedCodeBlock.String(), "\n") + content[idx+len(codeBlock):]

	return writeFile(fs, filename, content, info.Mode())
}

// RemovePrefix removes the provided prefix from all lines in the provided code block.
func RemovePrefix(fs Filesystem, filename, prefix, codeBlock string) error {
	content, info, err := readFile(fs, filename)
	if err != nil {
		return err
	}

	idx := strings.Index(content, codeBlock)
	if idx < 0 {
		return fmt.Errorf("unable to find %q", codeBlock)
	}
	modifiedCodeBlock := strings.Builder{}
	for _, line := range strings.Split(codeBlock, "\n") {
		if _, err := modifiedCodeBlock.WriteString(strings.TrimPrefix(line, prefix) + "\n"); err != nil {
			return err
		}
	}
	content = content[:idx] + strings.TrimSuffix(modifiedCodeBlock.String(), "\n") + content[idx+len(codeBlock):]

	return writeFile(fs, filename, content, info.Mode())
}
