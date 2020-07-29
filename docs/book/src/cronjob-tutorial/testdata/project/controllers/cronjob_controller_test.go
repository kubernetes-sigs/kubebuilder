/*
理想情況下，我们应该对于每一个控制器支架有一个 `<kind>_conroller_test.go` ，并在 `test_suite.go` 中调用。
因此，让我们为CronJob控制器编写示例测试(`cronjob_controller_test.go。`)
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
请注意，要创建一个 CronJob，你需要创建一个包含您 CronJob 规范的存根 CronJob 结构体。

请注意，当我们创建一个存根 CronJob ，CronJob 还需要它所需要的下游对象的存根。
没有下面存根的 Job 模板 spec 和 Pod 模板 spec ，Kubernetes API 将不能创建 CronJob 。
*/
var _ = Describe("CronJob controller", func() {

	// 定义对象名称和测试超时/持续时间和间隔的有用的常量。
	const (
		CronjobName      = "test-cronjob"
		CronjobNamespace = "test-cronjob-namespace"
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
							// 为简单起见，我们只填写必需的字段。
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									// 为简单起见，我们只填写必需的字段。
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
				在创建这个 CronJob 之后，让我们检查 CronJob 的 Spec 字段是否与我们传入的匹配。
				请注意，因为 k8s apiserver 在前面的 ‘Create()’ 调用之后可能还没有完成 CronJob 的创建，我将用 Gomega 的 Eventually() 测试函数代替 Expect() 去给 apiserver 一个机会去完成CronJob的创建。

				`Eventually()` 将会每隔一秒重复运行作为一个参数的函数直到
				(a) 函数的输出匹配后续的`Should()`调用期望的，或者
				(b) 尝试的次数 * 间隔周期 的值超过提供的超时的值。

				在下面的示例中，timeout 和 interval 是我们选择的 Go Duration 值。
			*/

			cronjobLookupKey := types.NamespacedName{Name: CronjobName, Namespace: CronjobNamespace}
			createdCronjob := &cronjobv1.CronJob{}

			// 我们需要重新尝试获得这个新创建的 CronJob ，因为创建可能不会立即发生。
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
				让我们测试 CronJob 控制器负责更新活动的正在运行的 Jobs 的 CronJob.Status.Active 的逻辑。
				我们将验证当 CronJob 有一个活动的下游 Job ，它的 CronJob.Status.Active 字段包含对该Job的引用。

				首先，我们应该获取之前创建的测试 CronJob ，并验证它目前没有任何活动的作业。
				我们在这里用 Gomega 的 `Consistently()` 检查，以确保活动的 Job 总数在一段时间内保持为 0 。
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
				下一步，我们实际上创建了一个存根的 Job ，它属于我们的 CronJob，以及它的下游模板 specs 。
				我们设置 Job 的状态的 "Active" 总数为 2，以模拟我们的 Job 运行了二个 pod ，意味着Job是正在活跃地运行。

				然后，我们接受存根的 Job ，并将其所有者引用设置为指向我们的测试 CronJob 。
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
							// 为简单起见，我们只填写必需的字段。
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

			// 请注意，需要您 CronJob 的 GroupVersionKind 来设置此所有者引用。
			kind := reflect.TypeOf(cronjobv1.CronJob{}).Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(createdCronjob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).Should(Succeed())
			/*
				添加这个 Job 到我们的测试 CronJob 中将会触发我们的控制器协调器的逻辑。
				之后，我们可以编写一个测试用例验证我们的控制器是否按照预期最终更新了我们的 CronJob 的状态字段！
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
	编写完这些代码之后，您可以在 `controllers/` 目录下执行 `go test ./...` 去再次运行您的新的测试示例！
*/
