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

package initproject

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
)

const (
	depManifestFile = "Gopkg.toml"
)

var vendorInstallCmd = &cobra.Command{
	Use:   "dep",
	Short: "Install Gopkg.toml and update vendor dependencies.",
	Long:  `Install Gopkg.toml and update vendor dependencies.`,
	Example: `Update the vendor dependencies:
kubebuilder update vendor
`,
	Run: RunVendorInstall,
}

var builderCommit string
var Update bool

func RunVendorInstall(cmd *cobra.Command, args []string) {
	if !depExists() {
		log.Fatalf("Dep is not installed. Follow steps at: https://golang.github.io/dep/docs/installation.html")
	}
	if Update {
		if err := updateDepManifest(); err != nil {
			log.Fatalf("error upgrading the dep manifest (Gopkg.toml): %v", err)
		}
	} else {
		createNewDepManifest()
	}
	if err := runDepEnsure(); err != nil {
		fmt.Printf("Error running 'dep ensure': %v\n", err)
		return
	}
}

func runDepEnsure() error {
	fmt.Printf("Updating vendor dependencies. Running 'dep ensure'....\n")
	cmd := exec.Command("dep", "ensure")
	o, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to run 'dep ensure': %s\n", string(o))
		return err
	}
	fmt.Printf("Updated vendor dependencies successfully.\n")
	return nil
}

func depExists() bool {
	_, err := exec.LookPath("dep")
	return err == nil
}

func createNewDepManifest() {
	depTmplArgs := map[string]string{
		"Version": version.GetVersion().KubeBuilderVersion,
	}
	depManifestTmpl := fmt.Sprintf("%s\n%s\n%s", depManifestHeader, depManifestKBMarker, depManifestOverride)
	util.Write(depManifestFile, "dep-manifest-file", depManifestTmpl, depTmplArgs)
}

// updateDepManifest updates the existing dep manifest with newer dependencies.
// dep manifest update workflow:
// Try to read user managed dep manifest section. If success, then append the
// user managed dep with KB managed section and update the dep Manifest.
func updateDepManifest() error {
	// open the existing dep manifest.
	f, err := os.Open(depManifestFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// try to read content till the dep marker
	userDeps, foundKBMarker, err := tryReadingUserDeps(f)
	if err != nil {
		return err
	}

	if !foundKBMarker {
		// depManifest file or abort the operation here.
		// for now, aborting.
		log.Fatalf(`
Failed to upgrade the dep manifest (Gopkg.toml) file. It seems that the dep manifest
is not being managed by Kubebuilder. You can run the command with --overwrite-dep-manifest 
flag if you want to re-initialize the dep manifest file. 
`)
	}

	b := bytes.NewBufferString(userDeps)
	err = addKubeBuilderDeps(b)
	if err != nil {
		return err
	}

	tmpfile, err := ioutil.TempFile(".", "dep")
	if err != nil {
		return err
	}

	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.Write(b.Bytes())
	if err != nil {
		return err
	}
	err = tmpfile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpfile.Name(), depManifestFile)
	if err != nil {
		return err
	}
	return nil
}

func tryReadingUserDeps(r io.Reader) (userDeps string, foundMarker bool, err error) {
	b := &bytes.Buffer{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		b.WriteString(line)
		b.WriteString("\n")
		if strings.HasPrefix(line, depManifestKBMarker) {
			foundMarker = true
			userDeps = b.String()
			return
		}
	}

	err = scanner.Err()
	return
}

func addKubeBuilderDeps(w io.Writer) error {
	depTmplArgs := map[string]string{
		"Version": version.GetVersion().KubeBuilderVersion,
	}
	t := template.Must(template.New("dep-manifest-template").Parse(depManifestOverride))
	return t.Execute(w, depTmplArgs)
}
