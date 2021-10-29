/*
Copyright 2021 The Kubernetes Authors.

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

package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubectl", func() {
	var ver KubernetesVersion
	AfterEach(func() {
		ver = KubernetesVersion{}
	})
	Context("successful 'kubectl version' output", func() {
		It("decodes both client and server versions", func() {
			Expect(ver.decode(clientServerOutput)).To(Succeed())
			Expect(ver.ClientVersion.major).To(BeNumerically("==", 1))
			Expect(ver.ClientVersion.minor).To(BeNumerically("==", 21))
			Expect(ver.ServerVersion.major).To(BeNumerically("==", 1))
			Expect(ver.ServerVersion.minor).To(BeNumerically("==", 21))
		})
		It("decodes only client version", func() {
			Expect(ver.decode(clientOnlyOutput)).To(Succeed())
			Expect(ver.ClientVersion.major).To(BeNumerically("==", 1))
			Expect(ver.ClientVersion.minor).To(BeNumerically("==", 21))
			Expect(ver.ServerVersion.major).To(BeNumerically("==", 0))
			Expect(ver.ServerVersion.minor).To(BeNumerically("==", 0))
		})
	})
	Context("'kubectl version' output with non-JSON text", func() {
		It("handles warning logs", func() {
			Expect(ver.decode(clientServerWithWarningOutput)).To(Succeed())
			Expect(ver.ClientVersion.major).To(BeNumerically("==", 1))
			Expect(ver.ClientVersion.minor).To(BeNumerically("==", 21))
			Expect(ver.ServerVersion.major).To(BeNumerically("==", 1))
			Expect(ver.ServerVersion.minor).To(BeNumerically("==", 21))
		})
	})
	Context("with error text", func() {
		It("returns an error", func() {
			Expect(ver.decode(errorOutput)).NotTo(Succeed())
		})
	})
})

const clientServerOutput = `
{
  "clientVersion": {
    "major": "1",
    "minor": "21",
    "gitVersion": "v0.21.0-beta.1",
    "gitCommit": "0d10c3f72592addce965b9bb34992eb6fc283a3b",
    "gitTreeState": "clean",
    "buildDate": "2021-08-31T22:03:33Z",
    "goVersion": "go1.16.6",
    "compiler": "gc",
    "platform": "linux/amd64"
  },
  "serverVersion": {
    "major": "1",
    "minor": "21",
    "gitVersion": "v1.21.1",
    "gitCommit": "5e58841cce77d4bc13713ad2b91fa0d961e69192",
    "gitTreeState": "clean",
    "buildDate": "2021-05-18T01:10:20Z",
    "goVersion": "go1.16.4",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
`

const clientOnlyOutput = `
{
  "clientVersion": {
    "major": "1",
    "minor": "21",
    "gitVersion": "v0.21.0-beta.1",
    "gitCommit": "0d10c3f72592addce965b9bb34992eb6fc283a3b",
    "gitTreeState": "clean",
    "buildDate": "2021-08-31T22:03:33Z",
    "goVersion": "go1.16.6",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
`

const clientServerWithWarningOutput = `
{
  "clientVersion": {
    "major": "1",
    "minor": "21",
    "gitVersion": "v0.21.0-beta.1",
    "gitCommit": "0d10c3f72592addce965b9bb34992eb6fc283a3b",
    "gitTreeState": "clean",
    "buildDate": "2021-08-31T22:03:33Z",
    "goVersion": "go1.16.6",
    "compiler": "gc",
    "platform": "linux/amd64"
  },
  "serverVersion": {
    "major": "1",
    "minor": "21",
    "gitVersion": "v1.21.1",
    "gitCommit": "5e58841cce77d4bc13713ad2b91fa0d961e69192",
    "gitTreeState": "clean",
    "buildDate": "2021-05-18T01:10:20Z",
    "goVersion": "go1.16.4",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
WARNING: version difference between client (0.21) and server (1.21) exceeds the supported minor version skew of +/-1
`

const errorOutput = `
ERROR: reason blah blah
`
