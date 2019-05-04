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
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"go/build"

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
	gopath := os.Getenv("GOPATH")
	return path.Join(gopath, "src", "sigs.k8s.io", "kubebuilder")
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
	r := &TestResult{}
	// Setup scaffold
	s := &scaffold.Scaffold{
		GetWriter: func(path string) (io.Writer, error) {
			defer ginkgo.GinkgoRecover()
			gomega.Expect(path).To(gomega.Equal(writeToPath))
			return &r.Actual, nil
		},
		ProjectPath: filepath.Join(getProjectRoot(), "testdata", "gopath", "src", "project"),
	}
	build.Default.GOPATH = filepath.Join(getProjectRoot(), "testdata", "gopath")

	if len(goldenPath) > 0 {
		b, err := ioutil.ReadFile(filepath.Join(getProjectRoot(), "testdata", "gopath", "src", "project", goldenPath))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		r.Golden = string(b)
	}
	return s, r
}
