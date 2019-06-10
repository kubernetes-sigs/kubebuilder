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

package e2e

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
)

// randomSuffix returns a 4-letter string.
func randomSuffix() (string, error) {
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

// getNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func getNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// insertCode searches target content in the file and insert `toInsert` after the target.
func insertCode(filename, target, code string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	idx := strings.Index(string(contents), target)
	out := string(contents[:idx+len(target)]) + code + string(contents[idx+len(target):])
	return ioutil.WriteFile(filename, []byte(out), 0644)
}

// uncommentCode searches for target in the file and remove the prefix of the target content.
func uncommentCode(filename, target, prefix string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	out := new(bytes.Buffer)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// uncomment the target line
		if strings.Contains(line, target) {
			line = strings.ReplaceAll(line, target, strings.TrimSpace(strings.TrimPrefix(target, prefix)))
		}
		_, err = out.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filename, out.Bytes(), 0644)
}
