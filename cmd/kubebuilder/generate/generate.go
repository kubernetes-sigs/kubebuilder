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

package generate

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
)

var versionedAPIs []string
var unversionedAPIs []string
var Codegenerators []string
var copyright string
var generators = sets.String{}
var vendorDir string
var Docscopyright string
var Docstitle string

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"generated"},
	Short:   "Run code generators.",
	Long:    `Run code generators`,
	Example: `# Run code generators.
kubebuilder generate`,
	Run: RunGenerate,
}

var extraAPI = strings.Join([]string{
	"k8s.io/apimachinery/pkg/apis/meta/v1",
	"k8s.io/apimachinery/pkg/conversion",
	"k8s.io/apimachinery/pkg/runtime"}, ",")

func AddGenerate(cmd *cobra.Command) {
	cmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&copyright, "copyright", filepath.Join("hack", "boilerplate.go.txt"), "Location of copyright boilerplate file.")
	generateCmd.Flags().StringVar(&vendorDir, "vendor-dir", "", "Location of directory containing vendor files.")
	generateCmd.Flags().StringArrayVar(&versionedAPIs, "api-versions", []string{}, "API version to generate code for.  Can be specified multiple times.  e.g. --api-versions foo/v1beta1 --api-versions bar/v1  defaults to all versions found under directories pkg/apis/<group>/<version>")
	generateCmd.Flags().StringArrayVar(&Codegenerators, "generator", []string{}, "list of generators to run.  e.g. --generator kubebuilder --generator conversion Valid values: [kubebuilder,client,openapi]")
	generateCmd.Flags().StringVar(&Docscopyright, "docs-copyright", "<a href=\"https://github.com/kubernetes/kubernetes\">Copyright 2018 The Kubernetes Authors.</a>", "html for the copyright text on the docs")
	generateCmd.Flags().StringVar(&Docstitle, "docs-title", "API Reference", "title of the docs page")
	generateCmd.AddCommand(generateCleanCmd)
}

func GetGenerate() *cobra.Command {
	return generateCmd
}

var generateCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Removes generated source code",
	Long:  `Removes generated source code`,
	Run:   RunCleanGenerate,
}

func RunCleanGenerate(cmd *cobra.Command, args []string) {
	os.RemoveAll(filepath.Join("pkg", "client", "clientset_generated"))
	os.RemoveAll(filepath.Join("pkg", "client", "informers"))
	os.RemoveAll(filepath.Join("pkg", "client", "listers"))

	filepath.Walk("pkg", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasPrefix(info.Name(), "zz_generated.") {
			return os.Remove(path)
		}
		return nil
	})
}

func doGen(g string) bool {
	g = strings.Replace(g, "-gen", "", -1)
	return generators.Has(g) || generators.Len() == 0
}

func RunGenerate(cmd *cobra.Command, args []string) {
	initApis()

	for _, g := range Codegenerators {
		generators.Insert(strings.Replace(g, "-gen", "", -1))
	}

	util.GetCopyright(copyright)

	root, err := os.Executable()
	if err != nil {
		glog.Fatalf("error: %v", err)
	}
	root = filepath.Dir(root)

	all := []string{}
	versioned := []string{}
	for _, v := range versionedAPIs {
		v = filepath.Join(util.Repo, "pkg", "apis", v)
		versioned = append(versioned, "--input-dirs", v)
		all = append(all, "--input-dirs", v)
	}

	unversioned := []string{}
	for _, u := range unversionedAPIs {
		u = filepath.Join(util.Repo, "pkg", "apis", u)
		unversioned = append(unversioned, "--input-dirs", u)
		all = append(all, "--input-dirs", u)
	}

	if doGen("kubebuilder-gen") {
		c := exec.Command(filepath.Join(root, "kubebuilder-gen"),
			"--go-header-file", copyright,
			"--input-dirs", filepath.Join(util.Repo, "pkg", "apis", "..."),
			"--input-dirs", filepath.Join(util.Repo, "pkg", "controller", "..."),
			"--input-dirs", filepath.Join(util.Repo, "pkg", "inject", "..."),
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run kubebuilder-gen %s %v", out, err)
		}
	}

	if generators.Has("conversion-gen") {
		c := exec.Command(filepath.Join(root, "conversion-gen"),
			append(all,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"-O", "zz_generated.conversion",
				"--extra-peer-dirs", extraAPI)...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run conversion-gen %s %v", out, err)
		}
	}

	if doGen("deepcopy-gen") {
		c := exec.Command(filepath.Join(root, "deepcopy-gen"),
			append(versioned,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"-O", "zz_generated.deepcopy")...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run deepcopy-gen %s %v", out, err)
		}
	}

	if generators.Has("openapi") {
		apis := []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1",
			"k8s.io/apimachinery/pkg/api/resource",
			"k8s.io/apimachinery/pkg/version",
			"k8s.io/apimachinery/pkg/runtime",
			"k8s.io/apimachinery/pkg/util/intstr",
		}

		// Add any vendored apis from core
		apis = append(apis, getVendorApis(filepath.Join("k8s.io", "api"))...)
		apis = append(apis, getVendorApis(filepath.Join("k8s.io", "client-go", "pkg", "apis"))...)

		// Special case 'k8s.io/client-go/pkg/api/v1' because it does not have a group
		if _, err := os.Stat(filepath.Join("vendor", "k8s.io", "client-go", "pkg", "api", "v1", "doc.go")); err == nil {
			apis = append(apis, filepath.Join("k8s.io", "client-go", "pkg", "api", "v1"))
		}

		if _, err := os.Stat(filepath.Join("vendor", "k8s.io", "api", "core", "v1", "doc.go")); err == nil {
			apis = append(apis, filepath.Join("k8s.io", "api", "core", "v1"))
		}

		c := exec.Command(filepath.Join(root, "openapi-gen"),
			append(all,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"-i", strings.Join(apis, ","),
				"--output-package", filepath.Join(util.Repo, "pkg", "generated", "openapi"))...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run openapi-gen %s %v", out, err)
		}
	}

	// Generate reference documentation
	if generators.Has("apidocs") {
		c := exec.Command(filepath.Join(root, "gen-apidocs"),
			"--copyright", Docscopyright,
			"--title", Docstitle,
			"--config-dir", "docs/reference/",
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run gen-apidocs %s %v", out, err)
		}
	}

	if generators.Has("defaulter-gen") {
		c := exec.Command(filepath.Join(root, "defaulter-gen"),
			append(all,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"-O", "zz_generated.defaults",
				"--extra-peer-dirs=", extraAPI)...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run defaulter-gen %s %v", out, err)
		}
	}

	if doGen("client-gen") {
		// Builder the versioned apis client
		clientPkg := filepath.Join(util.Repo, "pkg", "client")
		clientset := filepath.Join(clientPkg, "clientset")
		c := exec.Command(filepath.Join(root, "client-gen"),
			"-o", util.GoSrc,
			"--go-header-file", copyright,
			"--input-base", filepath.Join(util.Repo, "pkg", "apis"),
			"--input", strings.Join(versionedAPIs, ","),
			"--clientset-path", clientset,
			"--clientset-name", "versioned",
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err := c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run client-gen %s %v", out, err)
		}

		listerPkg := filepath.Join(clientPkg, "listers")
		c = exec.Command(filepath.Join(root, "lister-gen"),
			append(versioned,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"--output-package", listerPkg)...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err = c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run lister-gen %s %v", out, err)
		}

		informerPkg := filepath.Join(clientPkg, "informers")
		c = exec.Command(filepath.Join(root, "informer-gen"),
			append(versioned,
				"-o", util.GoSrc,
				"--go-header-file", copyright,
				"--output-package", informerPkg,
				"--listers-package", listerPkg,
				"--versioned-clientset-package", filepath.Join(clientset, "versioned"))...,
		)
		glog.V(4).Infof("%s\n", strings.Join(c.Args, " "))
		out, err = c.CombinedOutput()
		if err != nil {
			glog.Fatalf("failed to run informer-gen %s %v", out, err)
		}
	}
}

func initApis() {
	if len(versionedAPIs) == 0 {
		groups, err := ioutil.ReadDir(filepath.Join("pkg", "apis"))
		if err != nil {
			glog.Fatalf("could not read pkg/apis directory to find api Versions")
		}
		for _, g := range groups {
			if g.IsDir() {
				versionFiles, err := ioutil.ReadDir(filepath.Join("pkg", "apis", g.Name()))
				if err != nil {
					glog.Fatalf("could not read pkg/apis/%s directory to find api Versions", g.Name())
				}
				versionMatch := regexp.MustCompile("^v\\d+(alpha\\d+|beta\\d+)*$")
				for _, v := range versionFiles {
					if v.IsDir() && versionMatch.MatchString(v.Name()) {
						versionedAPIs = append(versionedAPIs, filepath.Join(g.Name(), v.Name()))
					}
				}
			}
		}
	}
	u := map[string]bool{}
	for _, a := range versionedAPIs {
		u[path.Dir(a)] = true
	}
	for a, _ := range u {
		unversionedAPIs = append(unversionedAPIs, a)
	}
}

func getVendorApis(pkg string) []string {
	dir := filepath.Join("vendor", pkg)
	if len(vendorDir) >= 0 {
		dir = filepath.Join(vendorDir, dir)
	}
	apis := []string{}
	if groups, err := ioutil.ReadDir(dir); err == nil {
		for _, g := range groups {
			p := filepath.Join(dir, g.Name())
			if g.IsDir() {
				if versions, err := ioutil.ReadDir(p); err == nil {
					for _, v := range versions {
						versionMatch := regexp.MustCompile("^v\\d+(alpha\\d+|beta\\d+)*$")
						if v.IsDir() && versionMatch.MatchString(v.Name()) {
							apis = append(apis, filepath.Join(pkg, g.Name(), v.Name()))
						}
					}
				}
			}
		}
	}
	return apis
}
