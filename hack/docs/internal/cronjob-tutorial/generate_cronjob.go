/*
Copyright 2023 The Kubernetes Authors.

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

package cronjob

import (
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	hackutils "sigs.k8s.io/kubebuilder/v4/hack/docs/utils"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Sample define the sample which will be scaffolded
type Sample struct {
	ctx *utils.TestContext
}

// NewSample create a new instance of the cronjob sample and configure the KB CLI that will be used
func NewSample(binaryPath, samplePath string) Sample {
	log.Infof("Generating the sample context of Cronjob...")
	ctx := hackutils.NewSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Infof("destroying directory for cronjob sample project")
	sp.ctx.Destroy()

	log.Infof("refreshing tools and creating directory...")
	err := sp.ctx.Prepare()

	hackutils.CheckError("creating directory for sample project", err)
}

// GenerateSampleProject will generate the sample
func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the cronjob project")

	err := sp.ctx.Init(
		"--domain", "tutorial.kubebuilder.io",
		"--repo", "tutorial.kubebuilder.io/project",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
	)
	hackutils.CheckError("Initializing the cronjob project", err)

	log.Infof("Adding a new config type")
	err = sp.ctx.CreateAPI(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--resource", "--controller",
	)
	hackutils.CheckError("Creating the API", err)

	log.Infof("Implementing admission webhook")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--defaulting", "--programmatic-validation",
	)
	hackutils.CheckError("Implementing admission webhook", err)
}

// UpdateTutorial the cronjob tutorial with the scaffold changes
func (sp *Sample) UpdateTutorial() {
	log.Println("Update tutorial with cronjob code")
	// 1. update specs
	sp.updateSpec()
	// 2. update webhook
	sp.updateWebhook()
	// 3. update webhookTests
	sp.updateWebhookTests()
	// 4. update makefile
	sp.updateMakefile()
	// 5. generate extra files
	cmd := exec.Command("go", "mod", "tidy")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run go mod tidy for cronjob tutorial", err)

	cmd = exec.Command("go", "get", "github.com/robfig/cron")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to get package robfig/cron", err)

	cmd = exec.Command("make", "generate", "manifests")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("run make generate and manifests", err)

	// 6. compensate other intro in API
	sp.updateAPIStuff()
	// 7. update reconciliation and main.go
	// 7.1 update controller
	sp.updateController()
	// 7.2 update main.go
	sp.updateMain()
	// 8. generate extra files
	cmd = exec.Command("make", "generate", "manifests")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("run make generate and manifests", err)
	// 9. update suite_test explanation
	sp.updateSuiteTest()
	// 10. uncomment kustomization
	sp.updateKustomization()
	// 11. add example
	sp.updateExample()
	// 12. add test
	sp.addControllerTest()
}

// CodeGen is a noop for this sample, just to make generation of all samples
// more efficient. We may want to refactor `UpdateTutorial` some day to take
// advantage of a separate call, but it is not necessary.
func (sp *Sample) CodeGen() {
	cmd := exec.Command("make", "all")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make all for cronjob tutorial", err)

	cmd = exec.Command("make", "build-installer")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make build-installer for cronjob tutorial", err)
}

// insert code to fix docs
func (sp *Sample) updateSpec() {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License

/*
 */`)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`package v1`,
		`
/*
 */`)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`import (`,
		`
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"`)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`to be serialized.`, cronjobSpecExplaination)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`type CronJobSpec struct {`, cronjobSpecStruct)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CronJob. Edit cronjob_types.go to remove/update
	Foo string`+" `"+`json:"foo,omitempty"`+"`", "")
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// Important: Run "make" to regenerate code after modifying this file`, cronjobList)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}`, `
// +kubebuilder:docs-gen:collapse=Root Object Definitions`)
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// CronJob is the Schema for the cronjobs API.
type CronJob struct {`, `// CronJob is the Schema for the cronjobs API.
type CronJob struct {`+`
	/*
	 */`)
	hackutils.CheckError("fixing cronjob_types.go", err)

	// fix lint
	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`
	
}`, "")
	hackutils.CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`


}`, "")
	hackutils.CheckError("fixing cronjob_types.go", err)
}

func (sp *Sample) updateAPIStuff() {
	var err error
	// fix groupversion_info
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/groupversion_info.go"),
		`limitations under the License.
*/`, groupVersionIntro)
	hackutils.CheckError("fixing groupversion_info.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/groupversion_info.go"),
		`	"sigs.k8s.io/controller-runtime/pkg/scheme"
)`, groupVersionSchema)
	hackutils.CheckError("fixing groupversion_info.go", err)
}

func (sp *Sample) updateController() {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`limitations under the License.
*/`, controllerIntro)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)`, controllerImport)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`Scheme *runtime.Scheme`, `
	Clock`)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`	Clock
}`, controllerMockClock)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update`, controllerReconcile)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}`, controllerReconcileLogic)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`SetupWithManager(mgr ctrl.Manager) error {`, controllerSetupWithManager)
	hackutils.CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`For(&batchv1.CronJob{}).`, `
		Owns(&kbatch.Job{}).`)
	hackutils.CheckError("fixing cronjob_controller.go", err)
}

func (sp *Sample) updateMain() {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License`)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`// +kubebuilder:scaffold:imports
)`, mainBatch)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`// +kubebuilder:scaffold:scheme
}`, `
/*
The other thing that's changed is that kubebuilder has added a block calling our
CronJob controller's`+" `"+`SetupWithManager`+"`"+` method.
*/`)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`func main() {`, `
	/*
	 */`)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}`, `

	// +kubebuilder:docs-gen:collapse=old stuff`)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}`, mainEnableWebhook)
	hackutils.CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}`, `
	// +kubebuilder:docs-gen:collapse=old stuff`)
	hackutils.CheckError("fixing main.go", err)
}

func (sp *Sample) updateMakefile() {
	const originalManifestTarget = `.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
`
	const changedManifestTarget = `.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	# Note that the option maxDescLen=0 was added in the default scaffold in order to sort out the issue
	# Too long: must have at most 262144 bytes. By using kubectl apply to create / update resources an annotation
	# is created by K8s API to store the latest version of the resource ( kubectl.kubernetes.io/last-applied-configuration).
	# However, it has a size limit and if the CRD is too big with so many long descriptions as this one it will cause the failure.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:maxDescLen=0 webhook paths="./..." output:crd:artifacts:config=config/crd/bases
`
	err := pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, "Makefile"), originalManifestTarget, changedManifestTarget)
	hackutils.CheckError("updating makefile to use maxDescLen=0 in make manifest target", err)

}

func (sp *Sample) updateWebhookTests() {
	file := filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook_test.go")

	err := pluginutil.ReplaceInFile(file,
		webhookTestCreateDefaultingFragment,
		webhookTestCreateDefaultingReplaceFragment)
	hackutils.CheckError("replace create defaulting test", err)

	err = pluginutil.ReplaceInFile(file,
		webhookTestingValidatingTodoFragment,
		webhookTestingValidatingExampleFragment)
	hackutils.CheckError("replace validating defaulting test", err)

	err = pluginutil.ReplaceInFile(file,
		webhookTestsBeforeEachOriginal,
		webhookTestsBeforeEachChanged)
	hackutils.CheckError("replace before each webhook test ", err)
}

func (sp *Sample) updateWebhook() {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License`)
	hackutils.CheckError("fixing cronjob_webhook.go by adding collapse", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`import (
	"context"
	"fmt"`,
		`
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"`,
	)
	hackutils.CheckError("add extra imports to cronjob_webhook.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

// nolint:unused
// log is for logging in this package.
`, webhookIntro)
	hackutils.CheckError("fixing cronjob_webhook.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`var cronjoblog = logf.Log.WithName("cronjob-resource")`,
		`
/*
Then, we set up the webhook with the manager.
*/`)
	hackutils.CheckError("fixing cronjob_webhook.go by setting webhook with manager comment", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!`, webhooksNoticeMarker)
	hackutils.CheckError("fixing cronjob_webhook.go by replacing note about path attribute", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.`, explanationValidateCRD)
	hackutils.CheckError("fixing cronjob_webhook.go by replacing note about path attribute", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.`, "")
	hackutils.CheckError("fixing cronjob_webhook.go by replace TODO to change verbs", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): Add more fields as needed for defaulting`, fragmentForDefaultFields)
	hackutils.CheckError("fixing cronjob_webhook.go by replacing TODO in Defaulter", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`WithDefaulter(&CronJobCustomDefaulter{}).`,
		`WithDefaulter(&CronJobCustomDefaulter{
        DefaultConcurrencyPolicy:      batchv1.AllowConcurrent,
        DefaultSuspend:                false,
        DefaultSuccessfulJobsHistoryLimit: 3,
        DefaultFailedJobsHistoryLimit: 1,
    }).`)
	hackutils.CheckError("replacing WithDefaulter call in cronjob_webhook.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): fill in your defaulting logic.

	return nil
}`, webhookDefaultingSettings)
	hackutils.CheckError("fixing cronjob_webhook.go by adding logic", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): fill in your validation logic upon object creation.

	return nil, nil`,
		`return nil, validateCronJob(cronjob)`)
	hackutils.CheckError("fixing cronjob_webhook.go by fill in your validation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// TODO(user): fill in your validation logic upon object update.

	return nil, nil`,
		`return nil, validateCronJob(cronjob)`)
	hackutils.CheckError("fixing cronjob_webhook.go by adding validation logic upon object update", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CronJob.`,
		customInterfaceDefaultInfo)
	hackutils.CheckError("fixing cronjob_webhook.go by adding validation logic upon object update", err)

	err = pluginutil.AppendCodeAtTheEnd(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		webhookValidateSpecMethods)
	hackutils.CheckError("adding validation spec methods at the end", err)
}

func (sp *Sample) updateSuiteTest() {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`limitations under the License.
*/`, suiteTestIntro)
	hackutils.CheckError("updating suite_test.go to add license intro", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	"testing"
`, `
	ctrl "sigs.k8s.io/controller-runtime"
`)
	hackutils.CheckError("updating suite_test.go to add ctrl import", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)
`, suiteTestEnv)
	hackutils.CheckError("updating suite_test.go to add more variables", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
`, suiteTestAddSchema)
	hackutils.CheckError("updating suite_test.go to add schema", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}`, `
	/*
		Then, we start the envtest cluster.
	*/`)
	hackutils.CheckError("updating suite_test.go to add text to show where envtest cluster start", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
`, suiteTestClient)
	hackutils.CheckError("updating suite_test.go to add text about test client", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
`, suiteTestDescription)
	hackutils.CheckError("updating suite_test.go for test description", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
`, suiteTestCleanup)
	hackutils.CheckError("updating suite_test.go to cleanup tests", err)
}

func (sp *Sample) updateKustomization() {
	var err error
	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- ../certmanager`, `#`)
	hackutils.CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- ../prometheus`, `#`)
	hackutils.CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- path: cert_metrics_manager_patch.yaml
#  target:
#    kind: Deployment`, `#`)
	hackutils.CheckError("enabling cert_metrics_manager_patch.yaml", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/prometheus/kustomization.yaml"),
		`#patches:
#  - path: monitor_tls_patch.yaml
#    target:
#      kind: ServiceMonitor`, `#`)
	hackutils.CheckError("enabling monitor tls patch", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		certManagerForMetricsAndWebhooks, `#`)
	hackutils.CheckError("fixing default/kustomization", err)
}

func (sp *Sample) updateExample() {
	var err error

	// samples/batch_v1_cronjob
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "config/samples/batch_v1_cronjob.yaml"),
		`spec:`, cronjobSample)
	hackutils.CheckError("fixing samples/batch_v1_cronjob.yaml", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "config/samples/batch_v1_cronjob.yaml"),
		`# TODO(user): Add fields here`, "")
	hackutils.CheckError("fixing samples/batch_v1_cronjob.yaml", err)
}

func (sp *Sample) addControllerTest() {
	var fs = afero.NewOsFs()
	err := afero.WriteFile(fs, filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller_test.go"), []byte(controllerTest), 0600)
	hackutils.CheckError("adding cronjob_controller_test", err)
}
