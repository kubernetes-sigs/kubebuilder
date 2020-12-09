/*
理想情况下，每个控制器都应该存在对应的测试文件 `<kind>_conroller_test.go`，并在 `test_suite.go` 中调用。
接下来，让我们为CronJob控制器编写示例测试(`cronjob_controller_test.go。`)
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

/*
像往常一样，我们从必要的导入开始。我们还定义了一些有用的变量。
*/
package controllers

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
编写一个简单的集成测试的第一步是真实的创建一个您可以运行测试的 CronJob 的实例。
请注意，要创建一个 CronJob，你需要先创建一个包含 CronJob 定义的 stub 结构体。

请注意，当我们创建一个存根 CronJob ，CronJob 还需要它所需要的下游对象的存根。
没有下面存根的 Job 模板 spec 和 Pod 模板 spec ，Kubernetes API 将不能创建 CronJob 。
*/
var _ = Describe("CronJob controller", func() {

	// 定义对象名称、测试超时时间、持续时间以及测试间隔等常量。
	const (
		CronjobName      = "test-cronjob"
		CronjobNamespace = "default"
		JobName          = "test-job"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating CronJob Status", func() {
		It("Should increase CronJob Status.Active count when new Jobs are created", func() {
			By("By creating a new CronJob")
			ctx := context.Background()
			cronJob := &cronjobv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch.tutorial.kubebuilder.io/v1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CronjobName,
					Namespace: CronjobNamespace,
				},
				Spec: cronjobv1.CronJobSpec{
					Schedule: "1 * * * *",
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							// 简单起见，我们只填写必需的字段。
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									// 简单起见，我们只填写必需的字段。
									Containers: []v1.Container{
										{
											Name:  "test-container",
											Image: "test-image",
										},
									},
									RestartPolicy: v1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cronJob)).Should(Succeed())

			/*
				在创建这个 CronJob 之后，让我们检查 CronJob 的 Spec 字段与我们传入的字段是否匹配。
				请注意，因为 k8s apiserver 在前面的 ‘Create()’ 调用之后可能还没有完成 CronJob 的创建，我将用 Gomega 的 Eventually() 测试函数代替 Expect() 去给 apiserver 一个机会去完成CronJob的创建。

				`Eventually()` 方法每隔一个时间间隔执行一次参数中指定的函数，直到满足下列两个条件之一才会退出方法。
				(a) 函数的输出与`Should()`调用的期望输出匹配
				(b) 重试时间（重试次数 * 间隔周期）大于指定的超时时间

				在下面的示例中，timeout 和 interval 是我们选择的 Go Duration 值。
			*/

			cronjobLookupKey := types.NamespacedName{Name: CronjobName, Namespace: CronjobNamespace}
			createdCronjob := &cronjobv1.CronJob{}

			// 创建操作可能不会立马完成，因此我们需要多次重试去获取这个新建的 CronJob。
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// 让我们确保我们的Schedule 字符串值被正确地转换/处理。
			Expect(createdCronjob.Spec.Schedule).Should(Equal("1 * * * *"))
			/*
				现在，我们已经在我们的测试集群中创建了一个CronJob，下一步是写一个测试用例去真正的测试我们 CronJob 控制器的行为。
				让我们测试一下 CronJob 控制器根据正在运行的 Jobs 更新 CronJob.Status.Active 的逻辑。
				我们将验证当 CronJob 有一个活动的下游 Job ，它的 CronJob.Status.Active 字段包含对该Job的引用。

				首先，我们应该获取之前创建的测试 CronJob ，并验证它目前没有任何正在运行的 Job。
				在这里我们使用 Gomega 的 `Consistently()` 检查，以确保正在运行的 Job 总数在一段时间内保持为 0 。
			*/
			By("By checking the CronJob has zero active Jobs")
			Consistently(func() (int, error) {
				err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
				if err != nil {
					return -1, err
				}
				return len(createdCronjob.Status.Active), nil
			}, duration, interval).Should(Equal(0))
			/*
				下一步，为我们的 CronJob 创建一个 Job 的 stub 对象以及它的下游模板 specs 。
				我们将 Job 的状态 "Active" 设置为 2，来模拟当前 Job 运行了 2 个 pod ，这表示我们的 Job 正在运行。

				然后，我们获取 Job 的 stub 对象 ，并将其所有者引用指向我们的测试 CronJob 。
				这确保了测试 Job 属于我们的测试 CronJob ，并被它跟踪。
				一旦完成，我们就创建新的 Job 实例。
			*/
			By("By creating a new Job")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: CronjobNamespace,
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							// 简单起见，我们只填写必需的字段。
							Containers: []v1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
				Status: batchv1.JobStatus{
					Active: 2,
				},
			}

			// 请注意，所有者引用需要配置 CronJob 的 GroupVersionKind。
			kind := reflect.TypeOf(cronjobv1.CronJob{}).Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(createdCronjob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).Should(Succeed())
			/*
				添加这个 Job 到我们的测试 CronJob 中将会触发我们的控制器的协调逻辑。
				之后，我们可以编写一个测试用例验证我们的控制器最终是否按照预期更新了我们的 CronJob 的状态字段！
			*/
			By("By checking that the CronJob has one active Job")
			Eventually(func() ([]string, error) {
				err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
				if err != nil {
					return nil, err
				}

				names := []string{}
				for _, job := range createdCronjob.Status.Active {
					names = append(names, job.Name)
				}
				return names, nil
			}, timeout, interval).Should(ConsistOf(JobName), "should list our active job %s in the active jobs list in status", JobName)
		})
	})

})

/*
	完成这些代码后，您可以在 `controllers/` 目录下执行 `go test ./...` ，运行新的测试代码！
*/
