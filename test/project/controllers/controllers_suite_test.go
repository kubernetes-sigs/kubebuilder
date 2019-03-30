/*
Copyright 2019 The Kubernetes Authors.

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

package controllers_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"k8s.io/apimachinery/pkg/api/meta"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Controller Integration Suite", []Reporter{envtest.NewlineReporter{}})
}

var testenv *envtest.Environment
var cfg *rest.Config
var cl client.Client
var mapper meta.RESTMapper
var mapperProvider func(c *rest.Config) (meta.RESTMapper, error)

var _ = BeforeSuite(func(done Done) {
	ctrl.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	testenv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../config/crds"},
	}

	var err error
	cfg, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())

	mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
	Expect(err).NotTo(HaveOccurred())
	mapperProvider = func(c *rest.Config) (meta.RESTMapper, error) {
		return mapper, nil
	}

	cl, err = client.New(cfg, client.Options{Mapper: mapper})
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	testenv.Stop()
})
