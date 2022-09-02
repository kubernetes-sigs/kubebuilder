package v1alpha1

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

func tidyGoModForAPI(apiPath string) error {
	return util.RunInDir(apiPath, func() error {
		if err := util.RunCmd(
			"update dependencies in "+apiPath, "go", "mod", "tidy"); err != nil {
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
