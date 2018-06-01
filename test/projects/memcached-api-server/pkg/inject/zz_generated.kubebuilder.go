package inject

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	myappsv1alpha1 "github.com/kubernetes-sigs/kubebuilder/test/projects/memcached-api-server/pkg/apis/myapps/v1alpha1"
	rscheme "github.com/kubernetes-sigs/kubebuilder/test/projects/memcached-api-server/pkg/client/clientset/versioned/scheme"
	"github.com/kubernetes-sigs/kubebuilder/test/projects/memcached-api-server/pkg/controller/memcached"
	"github.com/kubernetes-sigs/kubebuilder/test/projects/memcached-api-server/pkg/inject/args"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	rscheme.AddToScheme(scheme.Scheme)

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		if err := arguments.ControllerManager.AddInformerProvider(&myappsv1alpha1.Memcached{}, arguments.Informers.Myapps().V1alpha1().Memcacheds()); err != nil {
			return err
		}

		// Add Kubernetes informers
		if err := arguments.ControllerManager.AddInformerProvider(&appsv1.ReplicaSet{}, arguments.KubernetesInformers.Apps().V1().ReplicaSets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Pod{}, arguments.KubernetesInformers.Core().V1().Pods()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&appsv1.Deployment{}, arguments.KubernetesInformers.Apps().V1().Deployments()); err != nil {
			return err
		}

		if c, err := memcached.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	Injector.CRDs = append(Injector.CRDs, &myappsv1alpha1.MemcachedCRD)
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"myapps.memcached.example.com"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"pods",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"apps",
		},
		Resources: []string{
			"deployments",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"apps",
		},
		Resources: []string{
			"replicasets",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	// Inject GroupVersions
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "myapps.memcached.example.com",
		Version: "v1alpha1",
	})
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
