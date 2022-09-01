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

const (
	ControllerRuntime        = "sigs.k8s.io/controller-runtime"
	ControllerRuntimeVersion = v3scaffolds.ControllerRuntimeVersion
	GoVersion                = "1.18"
	DefaultRequireVersion    = "v0.0.0"
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
				"create go.mod in "+apiPath, "go", "mod", "init", module); err != nil {
				return err
			}
			if err := util.RunCmd("pin go version to "+GoVersion, "go", "mod", "edit", "-go",
				GoVersion); err != nil {
				return err
			}
			if err := util.RunCmd("require directive for "+ControllerRuntime, "go", "mod", "edit", "-require",
				ControllerRuntime+"@"+ControllerRuntimeVersion); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
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
