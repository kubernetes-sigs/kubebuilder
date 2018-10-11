/*
Copyright 2018 The Kubernetes Authors.

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

package projectutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// GetProjectDir walks up the tree until it finds a directory with a PROJECT file
func GetProjectDir() (string, error) {
	gopath := os.Getenv("GOPATH")
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	// Walk up until we find PROJECT or are outside the GOPATH
	for strings.Contains(dir, gopath) {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return "", err
		}
		for _, f := range files {
			if f.Name() == "PROJECT" {
				return dir, nil
			}
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("unable to locate PROJECT file")
}
