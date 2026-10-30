# Project lint rules

This directory holds lint rules that are specific to Kubebuilder. They catch
mistakes that the standard linters cannot see, such as a test that ignores a
project convention.

The rules are written in the [ruleguard](https://github.com/quasilyte/go-ruleguard)
DSL, in [`rules.go`](rules.go). golangci-lint runs them through the `ruleguard`
check of the `gocritic` linter, which is configured in [`.golangci.yml`](../../../.golangci.yml).

## The rules

| Rule | What it reports |
|---|---|
| `testBuildsKubebuilder` | A test that builds a binary itself, with `go build`, `go install`, `go run` or `gexec.Build`. |
| `testFindsKubebuilder` | A test that looks up the `kubebuilder` binary with `exec.LookPath`. |
| `testRunsKubebuilderByName` | A test that runs the `kubebuilder` binary by a name of its own, such as `exec.Command("kubebuilder", ...)`. |
| `testRunsKubebuilderDirectly` | A test that runs the `kubebuilder` binary right where it builds the command, such as `exec.Command(util.KubebuilderBinName, "init").Run()`, instead of through the test context's `Run` method. |

### Tests that depend on the Kubebuilder binary use the test context

[`AGENTS.md`](../../../AGENTS.md) says that tests depending on the Kubebuilder
binary use `utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")`
from `test/e2e/utils`. The test context runs the first `kubebuilder` binary on
your `PATH`. `make install` builds it from your checkout and copies it to
`$(go env GOBIN)`, or to `$(go env GOPATH)/bin` when `GOBIN` is not set. Keep that
directory first on your `PATH`, so every test runs the build you just made.
Check which binary the tests will run:

```shell
make install
command -v kubebuilder   # must print the directory make install copies to
```

A test that builds or finds its own binary can test a different build than
the one you installed, and can pass or fail for reasons unrelated to your
change.

For example, these lines are reported:

```go
build := exec.Command("go", "build", "-o", binary, "../..") // testBuildsKubebuilder
binary, err := exec.LookPath("kubebuilder")                  // testFindsKubebuilder
cmd := exec.Command("kubebuilder", "init")                   // testRunsKubebuilderByName
err = exec.Command(util.KubebuilderBinName, "init").Run()    // testRunsKubebuilderDirectly
```

Use the test context instead:

```go
kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
Expect(err).NotTo(HaveOccurred())
Expect(kbc.Prepare()).To(Succeed()) // creates the test directory
Expect(kbc.Init("--domain", "example.com", "--repo", "example.com/project")).To(Succeed())

// For a command the test context has no method for, run its binary through it:
cmd := exec.Command(kbc.BinaryName, "alpha", "generate")
output, err := kbc.Run(cmd)
```

The rules read one expression at a time. They cannot follow a command that a
test stores in a variable and runs later with `cmd.Run()`, so reviewers still
check that tests run the binary through the test context.

## Run the rules

`make lint` runs every linter, these rules included. golangci-lint checks only
the default build, so `make lint` also runs `make lint-project-rules`, which
checks the test files tagged `//go:build integration` with these rules.

```shell
make lint                # every linter, plus the project rules on integration files
make lint-project-rules  # only the project rules, on the integration files
```

golangci-lint caches its results and does not notice a change to `rules.go`.
After you edit the rules, clear the cache before you run them:

```shell
./bin/golangci-lint cache clean
```

## Skip a line

Every rule reports through the `gocritic` linter, and the message starts with
the rule name, for example:

```text
pkg/cli/cli_test.go:42:12: ruleguard: testFindsKubebuilder: tests that depend on the Kubebuilder binary should get it from utils.NewTestContext(...) (gocritic)
```

When a test really needs a different binary, such as a released version,
skip that line with a `nolint` comment that says why:

```go
binary, err := exec.LookPath(releasedBinary) //nolint:gocritic // runs a released kubebuilder to test the upgrade
```

The comment skips every project rule on that line, because golangci-lint
skips findings per linter, not per rule. Keep the reason short and specific.

## Add a rule

1. Add a function to [`rules.go`](rules.go) that takes a `dsl.Matcher`.
2. Describe it with the `//doc:summary`, `//doc:before` and `//doc:after` tags,
   and start its `Report` message with the function name.
3. Add a row to the table in this file.
4. Run `./bin/golangci-lint cache clean` and then `make lint`, and make sure the
   rule reports the cases it is for and nothing else.

You do not need to change the Makefile or `.golangci.yml`: they run every rule
in `rules.go`.
