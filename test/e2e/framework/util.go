/*
Copyright 2018 The Kubernetes Authors.

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

package framework

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"sigs.k8s.io/kubebuilder/test/e2e/framework/ginkgowrapper"

	"k8s.io/client-go/tools/clientcmd"
)

// Code originally copied from kubernetes/kubernetes at
// https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/util.go

// GetKubectlArgs wraps with default kubectl related args.
func GetKubectlArgs(args []string) []string {
	defaultArgs := []string{}

	// Reference a --server option so tests can run anywhere.
	if TestContext.Host != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.FlagAPIServer+"="+TestContext.Host)
	}
	if TestContext.KubeConfig != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.RecommendedConfigPathFlag+"="+TestContext.KubeConfig)

		// Reference the KubeContext
		if TestContext.KubeContext != "" {
			defaultArgs = append(defaultArgs, "--"+clientcmd.FlagContext+"="+TestContext.KubeContext)
		}

	} else {
		if TestContext.CertDir != "" {
			defaultArgs = append(defaultArgs,
				fmt.Sprintf("--certificate-authority=%s", filepath.Join(TestContext.CertDir, "ca.crt")),
				fmt.Sprintf("--client-certificate=%s", filepath.Join(TestContext.CertDir, "kubecfg.crt")),
				fmt.Sprintf("--client-key=%s", filepath.Join(TestContext.CertDir, "kubecfg.key")))
		}
	}
	kubectlArgs := append(defaultArgs, args...)

	return kubectlArgs
}

func NowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, NowStamp()+": "+level+": "+format+"\n", args...)
}

func Logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

func Failf(format string, args ...interface{}) {
	FailfWithOffset(1, format, args...)
}

// FailfWithOffset calls "Fail" and logs the error at "offset" levels above its caller
// (for example, for call chain f -> g -> FailfWithOffset(1, ...) error would be logged for "f").
func FailfWithOffset(offset int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log("INFO", msg)
	ginkgowrapper.Fail(NowStamp()+": "+msg, 1+offset)
}

func Skipf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log("INFO", msg)
	ginkgowrapper.Skip(NowStamp() + ": " + msg)
}

// RandomSuffix provides a random string to append to certain base name.
func RandomSuffix() string {
	source := []rune("abcdefghijklmnopqrstuvwxyz")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	res := make([]rune, 4)
	for i := range res {
		res[i] = source[r.Intn(len(source))]
	}
	return string(res)
}

// ParseCmdOutput converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func ParseCmdOutput(output string) []string {
	res := []string{}
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// ReplaceFileConent tries to replace the source content of given file
// with the target concent, source content can be regex.
func ReplaceFileConent(src, target string, filePath string) error {
	// Check if file exist
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	// Read file content
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Replace the content
	r := regexp.MustCompile(src)
	output := r.ReplaceAllString(string(fileContent), target)

	if err := ioutil.WriteFile(filePath, []byte(output), os.ModePerm); err != nil {
		return err
	}

	return nil
}
