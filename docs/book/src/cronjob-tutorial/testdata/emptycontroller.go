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
首先，我们从一些标准的 import 开始。和之前一样，我们需要核心 controller-runtime 运行库，以及 client 包和我们的 API 类型包。
*/
package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

/*
接下来，kubebuilder 为我们搭建了一个基本的 reconciler 结构。几乎每一个调节器都需要记录日志，并且能够获取对象，所以可以直接使用。
*/

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

/*
Most controllers eventually end up running on the cluster, so they need RBAC
permissions, which we specify using controller-tools [RBAC
markers](/reference/markers/rbac.md).  These are the bare minimum permissions
needed to run.  As we add more functionality, we'll need to revisit these.
*/

// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch

/*
`Reconcile` 实际上是对单个对象进行调谐。我们的 Request 只是有一个名字，但我们可以使用 client 从缓存中获取这个对象。

我们返回一个空的结果，没有错误，这就向 controller-runtime 表明我们已经成功地对这个对象进行了调谐，在有一些变化之前不需要再尝试调谐。

大多数控制器需要一个日志句柄和一个上下文，所以我们在 Reconcile 中将他们初始化。

上下文是用来允许取消请求的，也或者是实现 tracing 等功能。它是所有 client 方法的第一个参数。`Background` 上下文只是一个基本的上下文，没有任何额外的数据或超时时间限制。

控制器-runtime通过一个名为logr的库使用结构化的日志记录。正如我们稍后将看到的，日志记录的工作原理是将键值对附加到静态消息中。我们可以在我们的调和方法的顶部预先分配一些对，让这些对附加到这个调和器的所有日志行。

controller-runtime 通过一个名为 [logr](https://github.com/go-logr/logr) 日志库使用结构化的记录日志。正如我们稍后将看到的，日志记录的工作原理是将键值对附加到静态消息中。我们可以在我们的 Reconcile 方法的顶部预先分配一些配对信息，将他们加入这个 Reconcile 的所有日志中。
*/
func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("cronjob", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

/*
最后，我们将 Reconcile 添加到 manager 中，这样当 manager 启动时它就会被启动。

现在，我们只是注意到这个 Reconcile 是在 `CronJob`s 上运行的。以后，我们也会用这个来标记其他的对象。
*/

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Complete(r)
}
