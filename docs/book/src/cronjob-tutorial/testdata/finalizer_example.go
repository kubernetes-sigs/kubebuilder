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
First, we start out with some standard imports.
As before, we need the core controller-runtime library, as well as
the client package, and the package for our API types.
*/
package controllers

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The code snippet below shows skeleton code for implementing a finalizer.
*/

func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	var cronJob *batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
	         // 我们将忽略未找到的错误，因为不能通过重新加入队列的方式来修复这些错误
		 //（我们需要等待新的通知），而且我们可以根据删除的请求来获取它们
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 自定义 finalizer 的名字
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"

	// 检查 DeletionTimestamp 以确定对象是否在删除中
	if cronJob.ObjectMeta.DeletionTimestamp.IsZero() {
		// 该对象不会被删除，因为如果没有我们的 finalizer，
		// 然后添加 finalizer 并更新对象，相当于注册我们的 finalizer。
		if !containsString(cronJob.ObjectMeta.Finalizers, myFinalizerName) {
			cronJob.ObjectMeta.Finalizers = append(cronJob.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// 这个对象将要被删除
		if containsString(cronJob.ObjectMeta.Finalizers, myFinalizerName) {
			// 我们的 finalizer 就在这, 接下来就是处理外部依赖
			if err := r.deleteExternalResources(cronJob); err != nil {
				// 如果无法在此处删除外部依赖项，则返回错误
				// 以便可以重试
				return ctrl.Result{}, err
			}

			// 从列表中删除我们的 finalizer 并进行更新。
			cronJob.ObjectMeta.Finalizers = removeString(cronJob.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}

		// 当它们被删除的时候停止 reconciliation
		return ctrl.Result{}, nil
	}

	// Your reconcile logic

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(cronJob *batch.CronJob) error {

	// 删除与 cronJob 相关的任何外部资源
	// 确保删除实现是幂等且可以安全调用同一对象多次。
}

// 辅助函数用于检查并从字符串切片中删除字符串。
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

