package v1alpha1

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds"
	v3scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

const (
	ControllerRuntime        = "sigs.k8s.io/controller-runtime"
	ControllerRuntimeVersion = v3scaffolds.ControllerRuntimeVersion
	GoVersion                = "1.18"
	DefaultRequireVersion    = "v0.0.0"
)

func createGoModForAPI(fs machinery.Filesystem, config config.Config) error {
	apiPath := getAPIPath(config.IsMultiGroup())
	goModPath := filepath.Join(apiPath, "go.mod")
	module := config.GetRepository() + "/" + apiPath
	fmt.Println("resolved module: " + module)

	scaffolder := scaffolds.NewAPIScaffolder(config, goModPath)
	scaffolder.InjectFS(fs)
	err := scaffolder.Scaffold()
	if err != nil {
		return err
	}

	if err := util.RunCmd("require directive for new module", "go", "mod", "edit", "-require",
		module+"@"+DefaultRequireVersion); err != nil {
		return err
	}

	if err := util.RunCmd("replace directive for local folder", "go", "mod", "edit", "-replace",
		module+"="+"."+string(filepath.Separator)+apiPath); err != nil {
		return err
	}

	// Update Dockerfile
	return insertModUpdatesInDockerfile(apiPath)
}

func cleanUpGoModForAPI(fs machinery.Filesystem, config config.Config) error {
	apiPath := getAPIPath(config.IsMultiGroup())
	goModPath := filepath.Join(apiPath, "go.mod")
	module := config.GetRepository() + "/" + apiPath
	fmt.Println("resolved module: " + module)

	if err := fs.FS.Remove(goModPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := fs.FS.Remove(filepath.Join(apiPath, "go.sum")); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := util.RunCmd("drop require directive", "go", "mod", "edit", "-droprequire",
		module); err != nil {
		return err
	}

	if err := util.RunCmd("drop replace statement", "go", "mod", "edit", "-dropreplace",
		module); err != nil {
		return err
	}
	// Update Dockerfile
	if err := removeModUpdatesInDockerfile(apiPath); err != nil {
		return err
	}

	return nil
}

func tidyGoModForAPI(isMultiGroup bool) error {
	apiPath := getAPIPath(isMultiGroup)
	return util.RunInDir(apiPath, func() error {
		if err := util.RunCmd(
			"Update dependencies in "+apiPath, "go", "mod", "tidy"); err != nil {
			return err
		}
		return nil
	})
}

func getAPIPath(isMultiGroup bool) string {
	path := ""
	if isMultiGroup {
		path = filepath.Join("apis")
	} else {
		path = filepath.Join("api")
	}
	return path
}
