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
	"path/filepath"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-tools/pkg/scaffold"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/project/projectutil"
)

// TestResult is the result of running the scaffolding.
type TestResult struct {
	// Actual is the bytes written to a scaffolded file.
	Actual bytes.Buffer

	// Golden is the golden file contents read from the controller-tools/test package
	Golden string
}

// ProjectPath is the path to the controller-tools/test project file
func ProjectPath() string {
	root, err := projectutil.GetProjectDir()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return filepath.Join(root, "test", "PROJECT")
}

// BoilerplatePath is the path to the controller-tools/test boilerplate file
func BoilerplatePath() string {
	root, err := projectutil.GetProjectDir()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return filepath.Join(root, "test", "hack", "boilerplate.go.txt")
}

// Options are the options for scaffolding in the controller-tools/test directory
func Options() input.Options {
	return input.Options{
		BoilerplatePath: BoilerplatePath(),
		ProjectPath:     ProjectPath(),
	}
}

// NewTestScaffold returns a new Scaffold and TestResult instance for testing
func NewTestScaffold(writeToPath, goldenPath string) (*scaffold.Scaffold, *TestResult) {
	r := &TestResult{}

	root, err := projectutil.GetProjectDir()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Setup scaffold
	s := &scaffold.Scaffold{
		GetWriter: func(path string) (io.Writer, error) {
			defer ginkgo.GinkgoRecover()
			gomega.Expect(path).To(gomega.Equal(writeToPath))
			return &r.Actual, nil
		},
		ProjectPath: filepath.Join(root, "test"),
	}

	if len(goldenPath) > 0 {
		b, err := ioutil.ReadFile(filepath.Join(root, "test", goldenPath))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		r.Golden = string(b)
	}
	return s, r
}
