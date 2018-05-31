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
	"flag"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/config"

	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultHost        = "http://127.0.0.1:8080"
	defaultBinariesDir = "/usr/local/kubebuilder/bin/"
)

var TestContext TestContextType

type TestContextType struct {
	BinariesDir string
	ProjectDir  string
	// kubectl related items
	KubectlPath        string
	KubeConfig         string
	KubeContext        string
	KubeAPIContentType string
	CertDir            string
	Host               string
}

// Register flags common to all e2e test suites.
func RegisterFlags() {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true

	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	// Randomize specs as well as suites
	config.GinkgoConfig.RandomizeAllSpecs = true

	flag.StringVar(&TestContext.KubeConfig, clientcmd.RecommendedConfigPathFlag, os.Getenv(clientcmd.RecommendedConfigPathEnvVar), "Path to kubeconfig containing embedded authinfo.")
	flag.StringVar(&TestContext.KubeContext, clientcmd.FlagContext, "", "kubeconfig context to use/override. If unset, will use value from 'current-context'")
	flag.StringVar(&TestContext.KubeAPIContentType, "kube-api-content-type", "application/vnd.kubernetes.protobuf", "ContentType used to communicate with apiserver")
	flag.StringVar(&TestContext.CertDir, "cert-dir", "", "Path to the directory containing the certs. Default is empty, which doesn't use certs.")
	flag.StringVar(&TestContext.Host, "host", "", fmt.Sprintf("The host, or apiserver, to connect to. Will default to %s if this argument and --kubeconfig are not set", defaultHost))

	flag.StringVar(&TestContext.BinariesDir, "binaries-dir", defaultBinariesDir, "The path of binaries.")
	flag.StringVar(&TestContext.ProjectDir, "project-dir", "", "Project root path, must under $GOPATH/src/.")
}
