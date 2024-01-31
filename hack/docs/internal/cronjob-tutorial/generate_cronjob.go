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
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

type Sample struct {
	ctx *utils.TestContext
}

func NewSample(binaryPath, samplePath string) Sample {
	log.Infof("Generating the sample context of Cronjob...")

	ctx := newSampleContext(binaryPath, samplePath, "GO111MODULE=on")

	return Sample{&ctx}
}

func newSampleContext(binaryPath string, samplePath string, env ...string) utils.TestContext {
	cmdContext := &utils.CmdContext{
		Env: env,
		Dir: samplePath,
	}

	testContext := utils.TestContext{
		CmdContext: cmdContext,
		BinaryName: binaryPath,
	}

	return testContext
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Infof("destroying directory for cronjob sample project")
	sp.ctx.Destroy()

	log.Infof("refreshing tools and creating directory...")
	err := sp.ctx.Prepare()

	CheckError("creating directory for sample project", err)
}

func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the cronjob project")

	err := sp.ctx.Init(
		"--plugins", "go/v4",
		"--domain", "tutorial.kubebuilder.io",
		"--repo", "tutorial.kubebuilder.io/project",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
	)
	CheckError("Initializing the cronjob project", err)

	log.Infof("Adding a new config type")
	err = sp.ctx.CreateAPI(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--resource", "--controller",
	)
	CheckError("Creating the API", err)

	log.Infof("Implementing admission webhook")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--defaulting", "--programmatic-validation",
	)
	CheckError("Implementing admission webhook", err)
}

func (sp *Sample) UpdateTutorial() {
	log.Println("TODO: update tutorial")
	// 1. update specs
	updateSpec(sp)
	// 2. update webhook
	updateWebhook(sp)
	// 3. generate extra files
	codeGen(sp)
	// 4. compensate other intro in API
	updateAPIStuff(sp)
	// 5. update reconciliation and main.go
	// 5.1 update controller
	updateController(sp)
	// 5.2 update main.go
	updateMain(sp)
	// 6. generate extra files
	codeGen(sp)
	// 7. update suite_test explanation
	updateSuiteTest(sp)
	// 8. uncomment kustomization
	updateKustomization(sp)
	// 9. add example
	updateExample(sp)
	// 10. add test
	addControllerTest(sp)
}

// CodeGen is a noop for this sample, just to make generation of all samples
// more efficient. We may want to refactor `UpdateTutorial` some day to take
// advantage of a separate call, but it is not necessary.
func (sp *Sample) CodeGen() {}

func codeGen(sp *Sample) {
	cmd := exec.Command("go", "get", "github.com/robfig/cron")
	_, err := sp.ctx.Run(cmd)
	CheckError("Failed to get package robfig/cron", err)

	cmd = exec.Command("make", "manifests")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run make manifests for cronjob tutorial", err)

	cmd = exec.Command("make", "all")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run make all for cronjob tutorial", err)

	cmd = exec.Command("go", "mod", "tidy")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run go mod tidy for cronjob tutorial", err)
}

// insert code to fix docs
func updateSpec(sp *Sample) {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License

/*
 */`)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`package v1`,
		`
/*
 */`)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`import (`,
		`
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"`)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`to be serialized.`, CronjobSpecExplaination)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`type CronJobSpec struct {`, CronjobSpecStruct)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CronJob. Edit cronjob_types.go to remove/update
	Foo string`+" `"+`json:"foo,omitempty"`+"`", "")
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// Important: Run "make" to regenerate code after modifying this file`, CronjobList)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}`, `
//+kubebuilder:docs-gen:collapse=Root Object Definitions`)
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// CronJob is the Schema for the cronjobs API
type CronJob struct {`, `// CronJob is the Schema for the cronjobs API
type CronJob struct {`+`
	/*
	 */`)
	CheckError("fixing cronjob_types.go", err)

	// fix lint
	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`
	
}`, "")
	CheckError("fixing cronjob_types.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`


}`, "")
	CheckError("fixing cronjob_types.go", err)
}

func updateAPIStuff(sp *Sample) {
	var err error
	// fix groupversion_info
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/groupversion_info.go"),
		`limitations under the License.
*/`, GroupversionIntro)
	CheckError("fixing groupversion_info.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/groupversion_info.go"),
		`	"sigs.k8s.io/controller-runtime/pkg/scheme"
)`, GroupversionSchema)
	CheckError("fixing groupversion_info.go", err)
}

func updateController(sp *Sample) {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`limitations under the License.
*/`, ControllerIntro)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)`, ControllerImport)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`Scheme *runtime.Scheme`, `
	Clock`)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`	Clock
}`, ControllerMockClock)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update`, ControllerReconcile)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}`, ControllerReconcileLogic)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`SetupWithManager(mgr ctrl.Manager) error {`, ControllerSetupWithManager)
	CheckError("fixing cronjob_controller.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller.go"),
		`For(&batchv1.CronJob{}).`, `
		Owns(&kbatch.Job{}).`)
	CheckError("fixing cronjob_controller.go", err)
}

func updateMain(sp *Sample) {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License`)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`//+kubebuilder:scaffold:imports
)`, MainBatch)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`//+kubebuilder:scaffold:scheme
}`, `
/*
The other thing that's changed is that kubebuilder has added a block calling our
CronJob controller's`+" `"+`SetupWithManager`+"`"+` method.
*/`)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`func main() {`, `
	/*
	 */`)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}`, `

	// +kubebuilder:docs-gen:collapse=old stuff`)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}`, MainEnableWebhook)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}`, `
	// +kubebuilder:docs-gen:collapse=old stuff`)
	CheckError("fixing main.go", err)
}

func updateWebhook(sp *Sample) {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License`)
	CheckError("fixing cronjob_webhook.go by adding collapse", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
`, WebhookIntro)
	CheckError("fixing cronjob_webhook.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`var cronjoblog = logf.Log.WithName("cronjob-resource")`,
		`
/*
Then, we set up the webhook with the manager.
*/`)
	CheckError("fixing cronjob_webhook.go by setting webhook with manager comment", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!`, WebhookMarker)
	CheckError("fixing cronjob_webhook.go by replacing TODO", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.`, "")
	CheckError("fixing cronjob_webhook.go by replace TODO to change verbs", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`//+kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io,admissionReviewVersions=v1`, "")
	CheckError("fixing cronjob_webhook.go by replacing marker", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`//+kubebuilder:webhook:path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob.kb.io,admissionReviewVersions=v1`, "")
	CheckError("fixing cronjob_webhook.go validate batch marker", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`cronjoblog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
`, WebhookValidate)
	CheckError("fixing cronjob_webhook.go by adding logic", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`// TODO(user): fill in your validation logic upon object creation.
	return nil, nil`,
		`
	return nil, r.validateCronJob()`)
	CheckError("fixing cronjob_webhook.go by fill in your validation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`// TODO(user): fill in your validation logic upon object update.
	return nil, nil`,
		`
	return nil, r.validateCronJob()`)
	CheckError("fixing cronjob_webhook.go by adding validation logic upon object update", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`func (r *CronJob) ValidateDelete() (admission.Warnings, error) {
	cronjoblog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}`, WebhookValidateSpec)
	CheckError("fixing cronjob_webhook.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_webhook.go"),
		`validate anything on deletion.
*/
}`, `validate anything on deletion.
*/`)
	CheckError("fixing cronjob_webhook.go by adding comments to validate on deletion", err)
}

func updateSuiteTest(sp *Sample) {
	var err error
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`limitations under the License.
*/`, SuiteTestIntro)
	CheckError("updating suite_test.go to add license intro", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`import (`, `
	"context"`)
	CheckError("updating suite_test.go to add context", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	"testing"
`, `
	ctrl "sigs.k8s.io/controller-runtime"
`)
	CheckError("updating suite_test.go to add ctrl import", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
`, SuiteTestEnv)
	CheckError("updating suite_test.go to add more variables", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
`, SuiteTestReadCRD)
	CheckError("updating suite_test.go to add text about CRD", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`, runtime.GOOS, runtime.GOARCH)),
	}
`, `
	/*
		Then, we start the envtest cluster.
	*/`)
	CheckError("updating suite_test.go to add text to show where envtest cluster start", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
`, SuiteTestAddSchema)
	CheckError("updating suite_test.go to add schema", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
`, SuiteTestDescription)
	CheckError("updating suite_test.go for test description", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/controller/suite_test.go"),
		`
var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
`, SuiteTestCleanup)
	CheckError("updating suite_test.go to cleanup tests", err)
}

func updateKustomization(sp *Sample) {
	var err error
	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- ../certmanager`, `#`)
	CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- path: webhookcainjection`, `#`)
	CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- ../prometheus`, `#`)
	CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		DefaultKustomization, `#`)
	CheckError("fixing default/kustomization", err)

	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/crd/kustomization.yaml"),
		`#- path: patches/cainjection_in_cronjobs.yaml`, `#`)
	CheckError("fixing crd/kustomization", err)
}

func updateExample(sp *Sample) {
	var err error

	// samples/batch_v1_cronjob
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "config/samples/batch_v1_cronjob.yaml"),
		`spec:`, CronjobSample)
	CheckError("fixing samples/batch_v1_cronjob.yaml", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "config/samples/batch_v1_cronjob.yaml"),
		`# TODO(user): Add fields here`, "")
	CheckError("fixing samples/batch_v1_cronjob.yaml", err)

	// update default/manager_auth_proxy_patch.yaml
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "config/default/manager_auth_proxy_patch.yaml"),
		` template:
    spec:`, ManagerAuthProxySample)
	CheckError("fixing default/manager_auth_proxy_patch.yaml", err)
}

func addControllerTest(sp *Sample) {
	var fs = afero.NewOsFs()
	err := afero.WriteFile(fs, filepath.Join(sp.ctx.Dir, "internal/controller/cronjob_controller_test.go"), []byte(ControllerTest), 0600)
	CheckError("adding cronjob_controller_test", err)
}

// CheckError will exit with exit code 1 when err is not nil.
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("error %s: %s", msg, err)
		os.Exit(1)
	}
}
