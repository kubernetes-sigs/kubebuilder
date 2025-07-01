package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionStringIncludesExpectedFields(t *testing.T) {
	output := versionString()

	assert.Contains(t, output, "KubeBuilder Version:")
	assert.Contains(t, output, "Kubernetes Vendor:")
	assert.Contains(t, output, "Git Commit:")
	assert.Contains(t, output, "Build Date:")
	assert.Contains(t, output, "Go OS/Arch:")
}

func TestVersionJSONFormatAndKeys(t *testing.T) {
	jsonStr := versionJSON()

	var result map[string]string
	err := json.Unmarshal([]byte(jsonStr), &result)
	assert.NoError(t, err)

	assert.Contains(t, result, "kubeBuilderVersion")
	assert.Contains(t, result, "kubernetesVendor")
	assert.Contains(t, result, "gitCommit")
	assert.Contains(t, result, "buildDate")
	assert.Contains(t, result, "goOs")
	assert.Contains(t, result, "goArch")
}

func TestGetVersionInfoFieldsArePopulated(t *testing.T) {
	v := getVersionInfo()

	assert.NotEmpty(t, v.KubeBuilderVersion)
	assert.NotEmpty(t, v.KubernetesVendor)
	assert.NotEmpty(t, v.GitCommit)
	assert.NotEmpty(t, v.BuildDate)
	assert.NotEmpty(t, v.GoOS)
	assert.NotEmpty(t, v.GoArch)

	assert.NotEqual(t, "unknown", v.KubeBuilderVersion, "KubeBuilderVersion should not be 'unknown'")
	assert.NotEqual(t, "unknown", v.GoOS, "GoOS should not be 'unknown'")
	assert.NotEqual(t, "unknown", v.GoArch, "GoArch should not be 'unknown'")
	assert.NotEqual(t, "$Format:%H$", v.GitCommit, "GitCommit should not be default placeholder")
}
