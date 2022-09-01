package v1alpha1

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	v3scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

func CreateGoModForAPI(fs machinery.Filesystem, config config.Config) error {
	apiPath := GetAPIPath(config.IsMultiGroup())
	goModPath := filepath.Join(apiPath, "go.mod")
	module := config.GetRepository() + "/" + apiPath
	fmt.Println("resolved module: " + module)

	if err := util.RunInDir(apiPath, func() error {
		if exists, err := afero.Exists(fs.FS, goModPath); err != nil {
			return err
		} else if !exists {
			if err := util.RunCmd(
				"Create go.mod in "+apiPath, "go", "mod", "init", module); err != nil {
				return err
			}
			if err := util.RunCmd("add require directive of sigs.k8s.io/controller-runtime", "go", "mod", "edit", "-require",
				"sigs.k8s.io/controller-runtime"+"@"+v3scaffolds.ControllerRuntimeVersion); err != nil {
				return err
			}
		}
		if err := util.RunCmd(
			"Update dependencies in "+apiPath, "go", "mod", "tidy"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if err := util.RunCmd("Add require directive of API module", "go", "mod", "edit", "-require",
		module+"@v0.0.0"); err != nil {
		return err
	}

	if err := util.RunCmd("Update dependencies", "go", "mod", "edit", "-replace",
		module+"="+"."+string(filepath.Separator)+apiPath); err != nil {
		return err
	}

	// Update Dockerfile
	return insertModUpdatesInDockerfile(apiPath)
}

func CleanUpGoModForAPI(fs machinery.Filesystem, config config.Config) error {
	apiPath := GetAPIPath(config.IsMultiGroup())
	goModPath := filepath.Join(apiPath, "go.mod")
	module := config.GetRepository() + "/" + apiPath
	fmt.Println("resolved module: " + module)

	if err := fs.FS.Remove(goModPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := fs.FS.Remove(filepath.Join(apiPath, "go.sum")); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := util.RunCmd("Remove require directive of API module", "go", "mod", "edit", "-droprequire",
		module); err != nil {
		return err
	}

	if err := util.RunCmd("Update dependencies", "go", "mod", "edit", "-dropreplace",
		module); err != nil {
		return err
	}
	// Update Dockerfile
	if err := removeModUpdatesInDockerfile(apiPath); err != nil {
		return err
	}

	return nil
}
