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

package controller

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &Test{}

// Test scaffolds a Controller Test
type Test struct {
	file.TemplateMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *Test) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "controller", "%[kind]", "%[kind]_controller_test.go")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	fmt.Println(f.Path)

	f.TemplateBody = controllerTestTemplate

	f.IfExistsAction = file.Error

	return nil
}

//nolint:lll
const controllerTestTemplate = `{{ .Boilerplate }}

package {{ lower .Resource.Kind }}

import (
	"testing"
	"time"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	{{ if .Resource.CreateExampleReconcileBody -}}
	appsv1 "k8s.io/api/apps/v1"
	{{ end -}}
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo"{{ if .Resource.Namespaced }}, Namespace: "default"{{end}}}}
{{ if .Resource.CreateExampleReconcileBody }}var depKey = types.NamespacedName{Name: "foo-deployment", Namespace: "default"}
{{ end }}
const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{ObjectMeta: metav1.ObjectMeta{Name: "foo"{{ if .Resource.Namespaced }}, Namespace: "default"{{end}}}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).To(gomega.Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create the {{ .Resource.Kind }} object and expect the Reconcile{{ if .Resource.CreateExampleReconcileBody }} and Deployment to be created{{ end }}
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
{{ if .Resource.CreateExampleReconcileBody }}
	deploy := &appsv1.Deployment{}
	g.Eventually(func() error { return c.Get(context.TODO(), depKey, deploy) }, timeout).
		Should(gomega.Succeed())

	// Delete the Deployment and expect Reconcile to be called for Deployment deletion
	g.Expect(c.Delete(context.TODO(), deploy)).To(gomega.Succeed())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), depKey, deploy) }, timeout).
		Should(gomega.Succeed())

	// Manually delete Deployment since GC isn't enabled in the test control plane
	g.Eventually(func() error { return c.Delete(context.TODO(), deploy) }, timeout).
	    Should(gomega.MatchError("deployments.apps \"foo-deployment\" not found"))
{{ end }}
}
`
