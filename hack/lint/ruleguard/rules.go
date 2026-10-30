//go:build ruleguard

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

// The ruleguard build tag keeps this file out of builds: golangci-lint reads
// it through the gocritic ruleguard check.

package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// Each rule below reports through golangci-lint's gocritic linter, and its
// message starts with the rule name. See README.md in this directory for how
// to run, skip and add rules.

// testBuildsKubebuilder reports a test that builds a binary itself, with
// go build, go install, go run or gexec.Build. A test that really builds
// another program skips the line with a nolint comment that says why.
//
//doc:summary Tests use the kubebuilder binary that make install puts on PATH, not one they build
//doc:tags    kubebuilder test
//doc:before  exec.Command("go", "build", "-o", binary, "../..")
//doc:after   kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
func testBuildsKubebuilder(m dsl.Matcher) {
	// gexec is not in the standard library, so ruleguard needs its path.
	m.Import("github.com/onsi/gomega/gexec")
	m.Match(
		`exec.Command("go", "build", $*_)`,
		`exec.Command("go", "install", $*_)`,
		`exec.Command("go", "run", $*_)`,
		`exec.CommandContext($_, "go", "build", $*_)`,
		`exec.CommandContext($_, "go", "install", $*_)`,
		`exec.CommandContext($_, "go", "run", $*_)`,
		`gexec.Build($*_)`,
	).
		Where(m.File().Name.Matches(`_test\.go$`)).
		Report(`testBuildsKubebuilder: tests that depend on the Kubebuilder binary should use utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on") instead of building a binary; for another program, skip the line with a nolint comment that says why`)
}

// testFindsKubebuilder reports a test that looks up the Kubebuilder binary
// itself instead of getting it from the test context.
//
//doc:summary Tests get the kubebuilder binary from utils.NewTestContext, not from exec.LookPath
//doc:tags    kubebuilder test
//doc:before  binary, err := exec.LookPath("kubebuilder")
//doc:after   kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
func testFindsKubebuilder(m dsl.Matcher) {
	m.Match(`exec.LookPath($name)`).
		Where(m.File().Name.Matches(`_test\.go$`) && m["name"].Text.Matches(`^"kubebuilder"$|^[A-Za-z0-9_.]*[Kk]ubebuilder[A-Za-z0-9_]*$`)).
		Report(`testFindsKubebuilder: tests that depend on the Kubebuilder binary should get it from utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on") instead of exec.LookPath`)
}

// testRunsKubebuilderByName reports a test that runs the Kubebuilder binary
// by a name of its own. Running util.KubebuilderBinName, the name the test
// context uses, through the test context's Run method is fine.
//
//doc:summary Tests run the kubebuilder binary by util.KubebuilderBinName or the test context's BinaryName
//doc:tags    kubebuilder test
//doc:before  cmd := exec.Command("kubebuilder", "init")
//doc:after   cmd := exec.Command(kbc.BinaryName, "init")
func testRunsKubebuilderByName(m dsl.Matcher) {
	m.Match(
		`exec.Command($name, $*_)`,
		`exec.CommandContext($_, $name, $*_)`,
	).
		Where(m.File().Name.Matches(`_test\.go$`) &&
			m["name"].Text.Matches(`^"kubebuilder"$|^[A-Za-z0-9_.]*[Kk]ubebuilder[A-Za-z0-9_]*$`) &&
			!m["name"].Text.Matches(`\.KubebuilderBinName$`)).
		Report(`testRunsKubebuilderByName: tests that depend on the Kubebuilder binary should run it through utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on"), with util.KubebuilderBinName or the test context's BinaryName`)
}

// testRunsKubebuilderDirectly reports a test that runs the Kubebuilder binary
// right where it builds the command, which skips the test context's working
// directory and environment. It cannot follow a command stored in a variable
// and run later with cmd.Run().
//
//doc:summary Tests run the kubebuilder binary through the test context's Run method
//doc:tags    kubebuilder test
//doc:before  err := exec.Command(util.KubebuilderBinName, "init").Run()
//doc:after   _, err := kbc.Run(exec.Command(kbc.BinaryName, "init"))
func testRunsKubebuilderDirectly(m dsl.Matcher) {
	m.Match(
		`exec.Command($name, $*_).$run($*_)`,
		`exec.CommandContext($_, $name, $*_).$run($*_)`,
	).
		Where(m.File().Name.Matches(`_test\.go$`) &&
			m["name"].Text.Matches(`^"kubebuilder"$|^[A-Za-z0-9_.]*[Kk]ubebuilder[A-Za-z0-9_]*$`) &&
			m["run"].Text.Matches(`^(Run|Output|CombinedOutput|Start)$`)).
		Report(`testRunsKubebuilderDirectly: tests that depend on the Kubebuilder binary should run it with the test context's Run method, such as kbc.Run(exec.Command(kbc.BinaryName, ...)), so it gets the test context's directory and environment`)
}
