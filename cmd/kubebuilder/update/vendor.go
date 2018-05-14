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

package update

import (
	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/initproject"
)

var overwriteDepManifest bool

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Update the vendor packages managed by kubebuilder.",
	Long:  `Update the vendor packages managed by kubebuilder.`,
	Example: `# Replace the vendor packages managed by kubebuilder with versions for the current install.
kubebuilder update vendor
`,
	Run: RunUpdateVendor,
}

func AddUpdateVendorCmd(cmd *cobra.Command) {
	cmd.AddCommand(vendorCmd)
	vendorCmd.Flags().BoolVar(&overwriteDepManifest, "overwrite-dep-manifest", false, "if true, overwrites the dep manifest file (Gopkg.toml)")
}

func RunUpdateVendor(cmd *cobra.Command, args []string) {
	initproject.Update = true
	if overwriteDepManifest {
		// suppress the update behavior
		initproject.Update = false
	}
	initproject.RunVendorInstall(cmd, args)
}
