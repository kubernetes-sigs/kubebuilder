# go-apidiff
Check API compatibility between different revisions of a Go project

> **NOTE**: `go-apidiff` is an early proof-of-concept and has not been tested and
> verified on a wide variety of packages. Trust its results at your own risk.
>
> `go-apidiff` is dependent on Go's built-in package listing and type parsing,
> and can therefore be unreliable in certain situations. For example, if you're
> working outside of `$GOPATH` on a project that supports Go modules at the new
> commit, but not the old commit, you may not get accurate results.

## GitHub Action

### Inputs

#### `base-ref`

Base reference for API compatibility comparison (default: `origin/${GITHUB_BASE_REF}`)

#### `version`

Version of go-apidiff to use (default: `latest`)

#### `compare-imports`

Compare exported API differences in the imports of the repo (default: `false`)

#### `print-compatible`

Print compatible API changes (default: `true`)

#### `repo-path`

Path to root of git repository to compare (default: current working directory)

### Outputs

_(none)_

### Example usage

```yaml
name: go-apidiff
on: [ pull_request ]
jobs:
  go-apidiff:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
      with:
        fetch-depth: 0
    - uses: actions/setup-go@v2
      with:
        go-version: 1.15
    - uses: joelanford/go-apidiff
```


## Local Installation

To install into $GOPATH/bin, run the following:
```console
$ go get github.com/joelanford/go-apidiff
```

## Usage
```console
$ go-apidiff --help
go-apidiff compares API compatibility of different commits of a Go repository.

By default, it compares just the module itself and prints only incompatible
changes. However it can be configured to print compatible changes and to search
for API incompatibilities in the dependency changes of the repository.

When used with just one argument, the passed argument is used for oldCommit,
and HEAD is used for newCommit."

Usage:
  go-apidiff <oldCommit> [newCommit] [flags]

Flags:
      --compare-imports    Compare exported API differences of the imports in the repo.
  -h, --help               help for go-apidiff
      --print-compatible   Print compatible API changes
      --repo-path string   Path to root of git repository to compare (default "/home/myuser/myproject")
```

## Example output
```console
$ GO111MODULE=off go get golang.org/x/exp
$ cd $GOPATH/src/golang.org/x/exp/apidiff
$ go-apidiff --repo-path=../ 81c71964d733d2e3e2375a315c0e2fd4d162adc4 --print-compatible

golang.org/x/exp/apidiff
  Incompatible changes:
  - Report.Compatible: removed
  - Report.Incompatible: removed
  Compatible changes:
  - Change: added
  - Report.Changes: added
```
