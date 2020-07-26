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

/*

我们的 main 文件最开始是 import 一些基本库，尤其是：

- 核心的 [控制器运行时](https://pkg.go.dev/sigs.k8s.io/controller-runtime?tab=doc) 库
- 默认的控制器运行时日志库-- Zap (后续会对它作更多的介绍)

*/

package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

/*
每一组控制器都需要一个 [*Scheme*](https://book.kubebuilder.io/cronjob-tutorial/gvks.html#err-but-whats-that-scheme-thing)，它提供了 Kinds 和相应的 Go 类型之间的映射。我们将在编写 API 定义的时候再谈一谈 Kinds，所以现在只需要记住它就好。
*/
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	// +kubebuilder:scaffold:scheme
}

/*
这段代码的核心逻辑比较简单: 

- 我们通过 flag 库解析入参
- 我们实例化了一个[*manager*](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Manager)，它记录着我们所有控制器的运行情况，以及设置共享缓存和API服务器的客户端（注意，我们把我们的 Scheme 的信息告诉了 manager）。
- 运行 manager，它反过来运行我们所有的控制器和 webhooks。manager 状态被设置为运行中，直到它收到一个优雅停机 (graceful shutdown) 信号。这样一来，当我们在 Kubernetes 上运行时，我们就可以优雅地停止 pod。

虽然我们现在还没有任何业务代码可供执行，但请记住 `+kubebuilder:scaffold:builder` 的注释 --- 事情很快就会变得有趣起来。

*/

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme, MetricsBindAddress: metricsAddr})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	/*
		注意：Manager 可以通过以下方式限制控制器可以监听资源的命名空间。
	*/

	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		Namespace:          namespace,
		MetricsBindAddress: metricsAddr,
	})

	/*
		上面的例子将把你的项目改成只监听单一的命名空间。在这种情况下，建议通过将默认的 ClusterRole 和 ClusterRoleBinding 分别替换为 Role 和RoleBinding 来限制所提供给这个命名空间的授权。

		另外，也可以使用 [MultiNamespacedCacheBuilder](https://pkg.go.dev/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder) 来监听特定的命名空间。
	*/

	var namespaces []string // List of Namespaces

	mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		NewCache:           cache.MultiNamespacedCacheBuilder(namespaces),
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})

	/*
		更多信息请参见 [MultiNamespacedCacheBuilder](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache?tab=doc#MultiNamespacedCacheBuilder)
	*/

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
