/*
Copyright 2020 The Kubernetes Authors.

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

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version contains all the information related to the CLI version
type Version struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
	GoOs               string `json:"goOs"`
	GoArch             string `json:"goArch"`
}

// String implements Stringer for Version so we can format the output as needed
func (v Version) String() string {
	return fmt.Sprintf(
		"Kubebuilder:\t%v\n"+
			"Kubernetes:\t%v\n"+
			"Git Commit:\t%v\n"+
			"Build Date:\t%v\n"+
			"OS/Arch:\t%v/%v",
		v.KubeBuilderVersion,
		v.KubernetesVendor,
		v.GitCommit,
		v.BuildDate,
		v.GoOs,
		v.GoArch)
}

func (c CLI) newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   fmt.Sprintf("Print the %s version", c.commandName),
		Long:    fmt.Sprintf("Print the %s version", c.commandName),
		Example: fmt.Sprintf("%s version", c.commandName),
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println(c.version)
			return nil
		},
	}
	return cmd
}
