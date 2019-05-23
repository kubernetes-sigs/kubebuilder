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
package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// insertStrings reads content from given reader and insert string below the
// line line containing marker string. So for ex. in insertStrings(r, m1, v1, m2, v2)
// v1 will be inserted below the lines containing m1 string and v2 will be inserted
// below line containing m2 string.
func insertStrings(r io.Reader, markerAndValues ...string) (io.Reader, error) {
	if len(markerAndValues)%2 != 0 {
		return nil, fmt.Errorf("invalid marker and value pairs")
	}

	mvPairs := map[string]string{}
	for i, s := range markerAndValues {
		if i%2 == 0 {
			mvPairs[s] = markerAndValues[i+1]
		}
	}

	buf := new(bytes.Buffer)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		for m, v := range mvPairs {
			if strings.TrimSpace(line) == strings.TrimSpace(v) {
				// since value already exist, so avoid duplication
				delete(mvPairs, m)
			}
			if strings.Contains(line, m) {
				_, err := buf.WriteString(v)
				if err != nil {
					return nil, err
				}
			}
		}
		_, err := buf.WriteString(line + "\n")
		if err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf, nil
}

func InsertStringsInFile(path string, markerAndValues ...string) error {
	isGoFile := false
	if ext := filepath.Ext(path); ext == ".go" {
		isGoFile = true
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	r, err := insertStrings(f, markerAndValues...)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	formattedContent := content
	if isGoFile {
		formattedContent, err = imports.Process(path, content, nil)
		if err != nil {
			return err
		}
	}

	// use Go import process to format the content
	err = ioutil.WriteFile(path, formattedContent, os.ModePerm)
	if err != nil {
		return err
	}

	return err
}
