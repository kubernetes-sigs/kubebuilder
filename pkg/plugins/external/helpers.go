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
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/external"
)

var outputGetter ExecOutputGetter = &execOutputGetter{}

const defaultMetadataTemplate = `
%s is an external plugin for scaffolding files to help with your Operator development.

For more information on how to use this external plugin, it is recommended to 
consult the external plugin's documentation.
`

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

func makePluginRequest(req external.PluginRequest, path string) (*external.PluginResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	out, err := outputGetter.GetExecOutput(reqBytes, path)
	if err != nil {
		return nil, err
	}

	res := external.PluginResponse{}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}

	// Error if the plugin failed.
	if res.Error {
		return nil, fmt.Errorf(strings.Join(res.ErrorMsgs, "\n"))
	}

	return &res, nil
}

func handlePluginResponse(fs machinery.Filesystem, req external.PluginRequest, path string) error {
	req.Universe = map[string]string{}

	res, err := makePluginRequest(req, path)
	if err != nil {
		return fmt.Errorf("error making request to external plugin: %w", err)
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

// getExternalPluginFlags is a helper function that is used to get a list of flags from an external plugin.
// It will return []Flag if successful or an error if there is an issue attempting to get the list of flags.
func getExternalPluginFlags(req external.PluginRequest, path string) ([]external.Flag, error) {
	req.Universe = map[string]string{}

	res, err := makePluginRequest(req, path)
	if err != nil {
		return nil, fmt.Errorf("error making request to external plugin: %w", err)
	}

	return res.Flags, nil
}

// isBooleanFlag is a helper function to determine if an argument flag is a boolean flag
func isBooleanFlag(argIndex int, args []string) bool {
	return argIndex+1 < len(args) &&
		strings.Contains(args[argIndex+1], "--") ||
		argIndex+1 >= len(args)
}

// bindAllFlags will bind all flags passed into the subcommand by a user
func bindAllFlags(fs *pflag.FlagSet, args []string) {
	defaultFlagDescription := "Kubebuilder could not validate this flag with the external plugin. " +
		"Consult the external plugin documentation for more information."

	// Bind all flags passed in
	for i := range args {
		if strings.Contains(args[i], "--") {
			flag := strings.Replace(args[i], "--", "", 1)
			// Check if the flag is a boolean flag
			if isBooleanFlag(i, args) {
				// --help is already a defined flag in the kubebuilder commands and has a description so we skip parsing it again
				if flag != "help" {
					_ = fs.Bool(flag, false, defaultFlagDescription)
				}
			} else {
				_ = fs.String(flag, "", defaultFlagDescription)
			}
		}
	}
}

// bindSpecificFlags with bind flags that are specified by an external plugin as an allowed flag
func bindSpecificFlags(fs *pflag.FlagSet, flags []external.Flag) {
	// Only bind flags returned by the external plugin
	for _, flag := range flags {
		switch flag.Type {
		case "bool":
			defaultValue, _ := strconv.ParseBool(flag.Default)
			_ = fs.Bool(flag.Name, defaultValue, flag.Usage)
		case "int":
			defaultValue, _ := strconv.Atoi(flag.Default)
			_ = fs.Int(flag.Name, defaultValue, flag.Usage)
		case "float":
			defaultValue, _ := strconv.ParseFloat(flag.Default, 64)
			_ = fs.Float64(flag.Name, defaultValue, flag.Usage)
		default:
			_ = fs.String(flag.Name, flag.Default, flag.Usage)
		}
	}
}

func bindExternalPluginFlags(fs *pflag.FlagSet, subcommand string, path string, args []string) {
	req := external.PluginRequest{
		APIVersion: defaultAPIVersion,
		Command:    "flags",
		Args:       []string{"--" + subcommand},
	}

	// Get a list of flags for the init subcommand of the external plugin
	// If it returns an error, parse all flags passed by the user and let
	// the external plugin return an unknown flag error.
	flags, err := getExternalPluginFlags(req, path)
	if err != nil {
		bindAllFlags(fs, args)
	} else {
		bindSpecificFlags(fs, flags)
	}
}

// setExternalPluginMetadata is a helper function that sets the subcommand
// metadata that is used when the help text is shown for a subcommand.
// It will attempt to get the Metadata from the external plugin. If the
// external plugin returns no Metadata or an error, a default will be used.
func setExternalPluginMetadata(subcommand, path string, subcmdMeta *plugin.SubcommandMetadata) {
	fileName := filepath.Base(path)
	subcmdMeta.Description = fmt.Sprintf(defaultMetadataTemplate, fileName[:len(fileName)-len(filepath.Ext(fileName))])

	res, _ := getExternalPluginMetadata(subcommand, path)

	if res != nil {
		if res.Description != "" {
			subcmdMeta.Description = res.Description
		}

		if res.Examples != "" {
			subcmdMeta.Examples = res.Examples
		}
	}
}

// fetchExternalPluginMetadata performs the actual request to the
// external plugin to get the metadata. It returns the metadata
// or an error if an error occurs during the fetch process.
func getExternalPluginMetadata(subcommand, path string) (*plugin.SubcommandMetadata, error) {
	req := external.PluginRequest{
		APIVersion: defaultAPIVersion,
		Command:    "metadata",
		Args:       []string{"--" + subcommand},
		Universe:   map[string]string{},
	}

	res, err := makePluginRequest(req, path)
	if err != nil {
		return nil, fmt.Errorf("error making request to external plugin: %w", err)
	}

	return &res.Metadata, nil
}
