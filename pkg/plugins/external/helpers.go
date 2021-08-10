/*
Copyright 2021 The Kubernetes Authors.

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

package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

var outputGetter ExecOutputGetter = &execOutputGetter{}

// ExecOutputGetter is an interface that implements the exec output method.
type ExecOutputGetter interface {
	GetExecOutput(req []byte, path string) ([]byte, error)
}

type execOutputGetter struct{}

func (e *execOutputGetter) GetExecOutput(request []byte, path string) ([]byte, error) {
	cmd := exec.Command(path) //nolint:gosec
	cmd.Stdin = bytes.NewBuffer(request)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return out, nil
}

var currentDirGetter OsWdGetter = &osWdGetter{}

// OsWdGetter is an interface that implements the get current directory method.
type OsWdGetter interface {
	GetCurrentDir() (string, error)
}

type osWdGetter struct{}

func (o *osWdGetter) GetCurrentDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %v", err)
	}

	return currentDir, nil
}

func handlePluginResponse(fs machinery.Filesystem, req external.PluginRequest, path string) error {
	req.Universe = map[string]string{}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	out, err := outputGetter.GetExecOutput(reqBytes, path)
	if err != nil {
		return err
	}

	res := external.PluginResponse{}
	if err := json.Unmarshal(out, &res); err != nil {
		return err
	}

	// Error if the plugin failed.
	if res.Error {
		return fmt.Errorf(strings.Join(res.ErrorMsgs, "\n"))
	}

	currentDir, err := currentDirGetter.GetCurrentDir()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}

	for filename, data := range res.Universe {
		f, err := fs.FS.Create(filepath.Join(currentDir, filename))
		if err != nil {
			return err
		}

		defer func() {
			if err := f.Close(); err != nil {
				return
			}
		}()

		if _, err := f.Write([]byte(data)); err != nil {
			return err
		}
	}

	return nil

}
