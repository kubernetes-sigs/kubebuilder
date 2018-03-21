package inject

import (
	samplecontrollerv1alpha1 "samplecontroller/pkg/apis/samplecontroller/v1alpha1"
	rscheme "samplecontroller/pkg/client/clientset/versioned/scheme"
	"samplecontroller/pkg/controller/foo"
	"samplecontroller/pkg/inject/args"

	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	rscheme.AddToScheme(scheme.Scheme)

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		if err := arguments.ControllerManager.AddInformerProvider(&samplecontrollerv1alpha1.Foo{}, arguments.Informers.Samplecontroller().V1alpha1().Foos()); err != nil {
			return err
		}

		// Add Kubernetes informers
		if err := arguments.ControllerManager.AddInformerProvider(&appsv1.Deployment{}, arguments.KubernetesInformers.Apps().V1().Deployments()); err != nil {
			return err
		}

		if c, err := foo.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	Injector.CRDs = append(Injector.CRDs, &samplecontrollerv1alpha1.FooCRD)
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"samplecontroller.k8s.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	// Inject GroupVersions
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "samplecontroller.k8s.io",
		Version: "v1alpha1",
	})
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
