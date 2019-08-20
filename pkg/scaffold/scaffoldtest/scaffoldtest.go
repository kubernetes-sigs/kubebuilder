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

package scaffoldtest

import (
	"bytes"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

// TestResult is the result of running the scaffolding.
type TestResult struct {
	// Actual is the bytes written to a scaffolded file.
	Actual bytes.Buffer

	// Golden is the golden file contents read from the controller-tools/testdata package
	Golden string
}

func getProjectRoot() string {
	return path.Join(build.Default.GOPATH, "src", "sigs.k8s.io", "kubebuilder")
}

// ProjectPath is the path to the controller-tools/testdata project file
func ProjectPath() string {
	return filepath.Join(getProjectRoot(), "testdata", "gopath", "src", "project", "PROJECT")
}

// BoilerplatePath is the path to the controller-tools/testdata boilerplate file
func BoilerplatePath() string {
	return filepath.Join(getProjectRoot(), "testdata", "gopath", "src", "project", "hack", "boilerplate.go.txt")
}

// Options are the options for scaffolding in the controller-tools/testdata directory
func Options() input.Options {
	return input.Options{
		BoilerplatePath: BoilerplatePath(),
		ProjectPath:     ProjectPath(),
	}
}

// NewTestScaffold returns a new Scaffold and TestResult instance for testing
func NewTestScaffold(writeToPath, goldenPath string) (*scaffold.Scaffold, *TestResult) {
	projRoot := getProjectRoot()
	r := &TestResult{}
	// Setup scaffold
	s := &scaffold.Scaffold{
		GetWriter: func(path string) (io.Writer, error) {
			defer ginkgo.GinkgoRecover()
			gomega.Expect(path).To(gomega.Equal(writeToPath))
			return &r.Actual, nil
		},
		FileExists: func(path string) bool {
			return path != writeToPath
		},
		ProjectPath: filepath.Join(projRoot, "testdata", "gopath", "src", "project"),
	}
	oldGoPath := build.Default.GOPATH
	build.Default.GOPATH = filepath.Join(projRoot, "testdata", "gopath")
	defer func() { build.Default.GOPATH = oldGoPath }()
	if _, err := os.Stat(build.Default.GOPATH); err != nil {
		panic(err)
	}

	if len(goldenPath) > 0 {
		b, err := ioutil.ReadFile(filepath.Join(projRoot, "testdata", "gopath", "src", "project", goldenPath))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		r.Golden = string(b)
	}
	return s, r
}

// NewGoTestScaffold returns a new Scaffold and TestResult instance for testing
func NewGoTestScaffold(t *testing.T, writeToPath, goldenPath string) (*scaffold.Scaffold, *TestResult) {
	projRoot := getProjectRoot()
	r := &TestResult{}
	// Setup scaffold
	s := &scaffold.Scaffold{
		GetWriter: func(path string) (io.Writer, error) {
			if path != writeToPath {
				t.Errorf("call to GetWriter on unexpected path.  expected=%s, actual=%s", writeToPath, path)
			}
			return &r.Actual, nil
		},
		FileExists: func(path string) bool {
			return path != writeToPath
		},
		ProjectPath: filepath.Join(projRoot, "testdata", "gopath", "src", "project"),
	}
	oldGoPath := build.Default.GOPATH
	build.Default.GOPATH = filepath.Join(projRoot, "testdata", "gopath")
	defer func() { build.Default.GOPATH = oldGoPath }()
	if _, err := os.Stat(build.Default.GOPATH); err != nil {
		t.Fatalf("error from stat on %s: %v", build.Default.GOPATH, err)
	}

	if len(goldenPath) > 0 {
		p := filepath.Join(projRoot, "testdata", "gopath", "src", "project", goldenPath)
		b, err := ioutil.ReadFile(p)
		if err != nil {
			t.Fatalf("unexpected error reading golden file %q: %v", p, err)
		}
		r.Golden = string(b)
	}
	return s, r
}

func (r *TestResult) CheckGoldenOutput(t *testing.T, actual string) {
	if r.Golden == actual {
		return
	}

	t.Errorf("output did not match golden output\nactual=%s\nexpected=%s", actual, r.Golden)
}
