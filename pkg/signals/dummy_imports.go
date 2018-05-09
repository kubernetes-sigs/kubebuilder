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

// The signals package contains libraries for handling signals to shutdown the system.
package signals

// dummy imports to ensure that `dep ensure` can vendor the required
// dependencies after `kubebuilder init` step.
import (
	_ "github.com/emicklei/go-restful"
	_ "github.com/go-openapi/spec"
	_ "github.com/onsi/ginkgo"
	_ "github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "sigs.k8s.io/testing_frameworks/integration"
)
