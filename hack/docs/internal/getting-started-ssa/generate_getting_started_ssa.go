/*
Copyright 2024 The Kubernetes Authors.

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

package gettingstartedssa

import (
	"log/slog"
	"os/exec"
	"path/filepath"

	hackutils "sigs.k8s.io/kubebuilder/v4/hack/docs/internal/utils"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Sample defines the getting started SSA sample
type Sample struct {
	ctx *utils.TestContext
}

// NewSample creates a new getting started SSA sample instance
func NewSample(binaryPath, samplePath string) Sample {
	slog.Info("Generating the sample context of getting-started-ssa...")
	ctx := hackutils.NewSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	slog.Info("Destroying directory for getting-started-ssa sample project")
	sp.ctx.Destroy()

	slog.Info("Refreshing tools and creating directory...")
	err := sp.ctx.Prepare()
	hackutils.CheckError("Creating directory for sample project", err)
}

// GenerateSampleProject generates the sample project
func (sp *Sample) GenerateSampleProject() {
	slog.Info("Initializing the getting started SSA project")
	err := sp.ctx.Init(
		"--domain", "example.com",
		"--repo", "example.com/memcached-ssa",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
	)
	hackutils.CheckError("Initializing the getting started SSA project", err)

	slog.Info("Adding a new config type with Server-Side Apply plugin")
	err = sp.ctx.CreateAPI(
		"--group", "cache",
		"--version", "v1alpha1",
		"--kind", "Memcached",
		"--resource", "--controller",
		"--plugins", "ssa/v1-alpha",
	)
	hackutils.CheckError("Creating the API with SSA plugin", err)
}

// UpdateTutorial updates the tutorial files with SSA implementation
func (sp *Sample) UpdateTutorial() {
	sp.updateAPI()
	sp.updateSample()
	sp.updateController()
	sp.updateControllerTest()
}

func (sp *Sample) updateAPI() {
	var err error
	path := "api/v1alpha1/memcached_types.go"

	// Add license collapse marker
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License

`)
	hackutils.CheckError("collapse license in memcached api", err)

	// Add imports collapse marker
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`Any new fields you add must have json tags for the fields to be serialized.
`,
		`
// +kubebuilder:docs-gen:collapse=Imports
`)
	hackutils.CheckError("collapse imports in memcached api", err)

	// Replace Spec with actual fields
	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path), oldSpecAPI, newSpecAPI)
	hackutils.CheckError("replace spec api", err)

	// Add Status fields
	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path), oldStatusAPI, newStatusAPI)
	hackutils.CheckError("replace status api", err)
}

func (sp *Sample) updateSample() {
	file := filepath.Join(sp.ctx.Dir, "config/samples/cache_v1alpha1_memcached.yaml")
	err := pluginutil.ReplaceInFile(file, "# TODO(user): Add fields here", sampleFragment)
	hackutils.CheckError("update sample to add fields", err)
}

func (sp *Sample) updateController() {
	pathFile := "internal/controller/memcached_controller.go"

	// Replace entire import block with full imports including SSA dependencies
	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		`import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "example.com/memcached-ssa/api/v1alpha1"
)`,
		`import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "example.com/memcached-ssa/api/v1alpha1"
)`,
	)
	hackutils.CheckError("replace imports with SSA dependencies", err)

	// Add status type constants
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		`cachev1alpha1 "example.com/memcached-ssa/api/v1alpha1"
)`,
		controllerStatusTypes,
	)
	hackutils.CheckError("add status types", err)

	// Add RBAC markers
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		"// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update",
		`
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch`,
	)
	hackutils.CheckError("add RBAC markers", err)

	// Replace reconcile implementation
	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		reconcileTODOMarker,
		controllerReconcileImplementation,
	)
	hackutils.CheckError("add reconcile implementation", err)

	// Add helper functions
	err = pluginutil.AppendCodeIfNotExist(
		filepath.Join(sp.ctx.Dir, pathFile),
		controllerHelperFunctions,
	)
	hackutils.CheckError("add helper functions", err)

	// Add Owns relationship
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		"For(&cachev1alpha1.Memcached{}).",
		`
		Owns(&appsv1.Deployment{}).`,
	)
	hackutils.CheckError("add Owns relationship", err)
}

func (sp *Sample) updateControllerTest() {
	file := "internal/controller/memcached_controller_test.go"

	// Add imports
	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		`. "github.com/onsi/gomega"`,
		`. "github.com/onsi/gomega"

	"k8s.io/utils/ptr"
	appsv1 "k8s.io/api/apps/v1"`,
	)
	hackutils.CheckError("add test imports", err)

	// Add spec to test
	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		"// TODO(user): Specify other spec details if needed.",
		`Spec: cachev1alpha1.MemcachedSpec{
					Size:          ptr.To(int32(1)),
					ContainerPort: ptr.To(int32(11211)),
				},`,
	)
	hackutils.CheckError("add spec to test", err)

	// Add test assertions
	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		`// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.`,
		testAssertions,
	)
	hackutils.CheckError("add test assertions", err)
}

// CodeGen runs code generation targets
func (sp *Sample) CodeGen() {
	// Run make generate first to create applyconfiguration package
	cmd := exec.Command("make", "generate")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make generate", err)

	cmd = exec.Command("go", "mod", "tidy")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run go mod tidy", err)

	cmd = exec.Command("make", "manifests")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make manifests", err)

	cmd = exec.Command("make", "all")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make all", err)

	cmd = exec.Command("make", "build-installer")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make build-installer", err)

	err = sp.ctx.EditHelmPlugin()
	hackutils.CheckError("Failed to enable helm plugin", err)
}

const (
	oldSpecAPI = `// foo is an example field of Memcached. Edit memcached_types.go to remove/update
	// +optional
	Foo *string ` + "`json:\"foo,omitempty\"`"

	newSpecAPI = `	/*
		Server-Side Apply (SSA) is used in this operator to manage Deployments.
		The Size and ContainerPort fields control the Deployment configuration.

		SSA allows the operator to manage only specific fields while preserving
		user customizations to other fields (labels, annotations, resources, etc.).
	*/

	// Size defines the number of Memcached instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	// +optional
	Size *int32 ` + "`json:\"size,omitempty\"`" + `

	// ContainerPort defines the port exposed by the Memcached container
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=11211
	// +optional
	ContainerPort *int32 ` + "`json:\"containerPort,omitempty\"`"

	oldStatusAPI = `// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Memcached resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition ` + "`json:\"conditions,omitempty\"`"

	newStatusAPI = `	/*
		The status tracks the state of the Memcached Deployment using standard
		Kubernetes Condition types (Available, Degraded).
	*/

	// Conditions represent the current state of the Memcached deployment
	// +optional
	Conditions []metav1.Condition ` + "`json:\"conditions,omitempty\"`"

	sampleFragment = `size: 3
  containerPort: 11211`

	controllerStatusTypes = `

/*
Status condition types for Memcached.
These follow Kubernetes API conventions for condition types.
*/
const (
	// typeAvailableMemcached represents the Deployment is available
	typeAvailableMemcached = "Available"
	// typeDegradedMemcached represents failures in reconciliation
	typeDegradedMemcached = "Degraded"
)`

	reconcileTODOMarker = `log := logf.FromContext(ctx)

	// Fetch the Memcached instance
	var memcached cachev1alpha1.Memcached
	if err := r.Get(ctx, req.NamespacedName, &memcached); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Memcached resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil`

	controllerReconcileImplementation = `	/*
		Fetch the Memcached instance.
	*/
	logger := log.FromContext(ctx)
	var memcached cachev1alpha1.Memcached
	if err := r.Get(ctx, req.NamespacedName, &memcached); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	/*
		Build the desired Deployment state using Server-Side Apply.
		Only the fields we specify will be managed by this controller.
		User customizations to other fields (labels, annotations, resources) are preserved.
	*/
	labels := labelsForMemcached(memcached.Name)

	// Set default values if not specified
	size := int32(1)
	if memcached.Spec.Size != nil {
		size = *memcached.Spec.Size
	}
	containerPort := int32(11211)
	if memcached.Spec.ContainerPort != nil {
		containerPort = *memcached.Spec.ContainerPort
	}

	/*
		Build the Deployment using apply configurations.
		Server-Side Apply will manage only the fields we set here.
	*/
	deployment := appsv1apply.Deployment(memcached.Name, memcached.Namespace).
		WithLabels(labels).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(size).
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name":     "memcached",
					"app.kubernetes.io/instance": memcached.Name,
				})).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithSecurityContext(corev1apply.PodSecurityContext().
						WithRunAsNonRoot(true).
						WithSeccompProfile(corev1apply.SeccompProfile().
							WithType(corev1.SeccompProfileTypeRuntimeDefault))).
					WithContainers(corev1apply.Container().
						WithName("memcached").
						WithImage("memcached:1.6.15-alpine").
						WithCommand("memcached", "-m=64", "-o", "modern", "-v").
						WithPorts(corev1apply.ContainerPort().
							WithName("memcached").
							WithContainerPort(containerPort).
							WithProtocol(corev1.ProtocolTCP))))))

	// Set owner reference using a temporary deployment object
	tempDeploy := &appsv1.Deployment{}
	tempDeploy.Name = memcached.Name
	tempDeploy.Namespace = memcached.Namespace
	if err := ctrl.SetControllerReference(&memcached, tempDeploy, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Add owner references to apply configuration
	for _, ref := range tempDeploy.OwnerReferences {
		deployment.WithOwnerReferences(metav1apply.OwnerReference().
			WithAPIVersion(ref.APIVersion).
			WithKind(ref.Kind).
			WithName(ref.Name).
			WithUID(ref.UID).
			WithController(true).
			WithBlockOwnerDeletion(true))
	}

	/*
		Apply the Deployment using Server-Side Apply.
		- client.Apply() uses the new Server-Side Apply API
		- ForceOwnership resolves conflicts by taking ownership of fields
		- FieldOwner("memcached-controller") identifies this controller
	*/
	if err := r.Apply(ctx, deployment, client.ForceOwnership,
		client.FieldOwner("memcached-controller")); err != nil {
		logger.Error(err, "Failed to apply Deployment")

		// Update status to Degraded
		meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{
			Type:    typeDegradedMemcached,
			Status:  metav1.ConditionTrue,
			Reason:  "DeploymentFailed",
			Message: fmt.Sprintf("Failed to apply Deployment: %v", err),
		})

		if err := r.Status().Update(ctx, &memcached); err != nil {
			logger.Error(err, "Failed to update Memcached status")
		}

		return ctrl.Result{}, err
	}

	/*
		Update status to Available.
		Note: We use traditional Update() for status, but you could also use SSA here.
	*/
	meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{
		Type:    typeAvailableMemcached,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: fmt.Sprintf("Deployment for Memcached (%s) with %d replicas created successfully", memcached.Name, size),
	})

	if err := r.Status().Update(ctx, &memcached); err != nil {
		logger.Error(err, "Failed to update Memcached status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Memcached", "name", memcached.Name)
	return ctrl.Result{RequeueAfter: time.Minute}, nil`

	controllerHelperFunctions = `
// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "memcached",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "memcached-controller",
	}
}
`

	testAssertions = `By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func(g Gomega) {
				found := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the latest Status Condition added to the Memcached instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, memcached)).To(Succeed())
			var conditions []metav1.Condition
			Expect(memcached.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal(typeAvailableMemcached)), &conditions))
			Expect(conditions).To(HaveLen(1), "Multiple conditions of type %s", typeAvailableMemcached)
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "condition %s", typeAvailableMemcached)
			Expect(conditions[0].Reason).To(Equal("Reconciling"), "condition %s", typeAvailableMemcached)`
)
