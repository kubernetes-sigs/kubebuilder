/*
Copyright 2017 The Kubernetes Authors.

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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/tools/go/packages"
)

// module and goMod arg just enough of the output of `go mod edit -json` for our purposes
type goMod struct {
	Module module
}
type module struct {
	Path string
}

// findGoModulePath finds the path of the current module, if present.
func findGoModulePath(forceModules bool) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-json")
	cmd.Env = append(cmd.Env, os.Environ()...)
	if forceModules {
		cmd.Env = append(cmd.Env, "GO111MODULE=on" /* turn on modules just for these commands */)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, isExitErr := err.(*exec.ExitError); isExitErr {
			err = fmt.Errorf("%s", string(exitErr.Stderr))
		}
		return "", err
	}
	mod := goMod{}
	if err := json.Unmarshal(out, &mod); err != nil {
		return "", err
	}
	return mod.Module.Path, nil
}

// FindCurrentRepo attempts to determine the current repository
// though a combination of go/packages and `go mod` commands/tricks.
func FindCurrentRepo() (string, error) {
	// easiest case: existing go module
	path, err := findGoModulePath(false)
	if err == nil {
		return path, nil
	}

	// next, check if we've got a package in the current directory
	pkgCfg := &packages.Config{
		Mode: packages.NeedName, // name gives us path as well
	}
	pkgs, err := packages.Load(pkgCfg, ".")
	// NB(directxman12): when go modules are off and we're outside GOPATH and
	// we don't otherwise have a good guess packages.Load will fabricate a path
	// that consists of `_/absolute/path/to/current/directory`.  We shouldn't
	// use that when it happens.
	if err == nil && len(pkgs) > 0 && len(pkgs[0].PkgPath) > 0 && pkgs[0].PkgPath[0] != '_' {
		return pkgs[0].PkgPath, nil
	}

	// otherwise, try to get `go mod init` to guess for us -- it's pretty good
	cmd := exec.Command("go", "mod", "init")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "GO111MODULE=on" /* turn on modules just for these commands */)
	if _, err := cmd.Output(); err != nil {
		if exitErr, isExitErr := err.(*exec.ExitError); isExitErr {
			err = fmt.Errorf("%s", string(exitErr.Stderr))
		}
		// give up, let the user figure it out
		return "", fmt.Errorf("could not determine repository path from module data, "+
			"package data, or by initializing a module: %v", err)
	}
	defer os.Remove("go.mod") // clean up after ourselves
	return findGoModulePath(true)
}
