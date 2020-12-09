/*
当我们在[之前章节](/cronjob-tutorial/new-api.md)用 `kubebuilder create api` 创建 CronJob API，Kubebuilder 已经为您创建了一些测试工作。
Kubebuilder 生成了一个用于配置基本测试环境框架的文件 `controllers/suite_test.go` 。

首先，它将包含必要的 imports 。
*/

/*

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
// +kubebuilder:docs-gen:collapse=Apache License

package controllers

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
	// +kubebuilder:scaffold:imports
)

// 这些测试示例使用了 Ginkgo (BDD-style Go 测试框架)。学习更多关于 Ginkgo 请参考 http://onsi.github.io/ginkgo/。

// +kubebuilder:docs-gen:collapse=Imports

/*
现在，让我们看看生成的代码。
*/

var cfg *rest.Config
var k8sClient client.Client //  您将在你的测试代码中使用这个 client。
var testEnv *envtest.Environment

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	/*
		首先， envtest 集群将从 Kubebuilder 为您生成的 CRD 目录下读取 CRD 信息。
	*/
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	/*
		然后，我们启动 envtest 集群。
	*/
	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	/*
		自动生成的测试代码将把 CronJob Kind schema 添加到默认的 client-go k8s scheme 中。
		这保证了 CronJob 的 API/Kind 可以在我们控制器测试代码中正常使用。
	*/
	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	/*
		schemas 之后，你将看到下面标记。
		当一个新的 API 添加到项目中，这个标记允许新的 schemas 自动的添加到这里。
	*/

	// +kubebuilder:scaffold:scheme

	/*
		为我们测试 CRUD 操作创建一个客户端。
	*/
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	/*
		然而，这个自动生成的文件缺少了实际启动控制器的方法。
		上面的代码将会建立一个和您自定义的 Kind 交互的客户端，但是无法测试您的控制器的行为。
		如果你想要测试自定义的控制器逻辑，您需要添加一些相似的管理逻辑到 BeforeSuite() 函数，
		这样就可以将你的自定义控制器运行在这个测试集群上。

		您可能注意到了，下面运行在控制器中的逻辑代码几乎和您的 CronJob 项目中的 main.go 中是相同的！
		唯一不同的是 manager 启动在一个独立的 goroutine 中，因此，当您运行完测试后，它不会阻止 envtest 的清理工作。

		一旦添加了下面的代码，你将可以删除掉上面的 k8sClient，因为你可以从 manager 中获取到 k8sClient (如下所示)。
	*/

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&CronJobReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("CronJob"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

/*
Kubebuilder 还为清除 envtest 和在 controller/ 目录中实际运行测试的文件生成样板函数。
你不需要更改这部分代码
*/

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

/*
现在，您已经在测试集群上运行了控制器，并且客户端已经准备好对 CronJob 执行操作，我们可以开始编写集成测试了!
*/
