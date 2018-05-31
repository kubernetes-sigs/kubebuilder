package args

import (
	"time"

	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"
	"k8s.io/client-go/rest"

	clientset "github.com/kubernetes-sigs/kubebuilder/test/projects/validations/pkg/client/clientset/versioned"
	informer "github.com/kubernetes-sigs/kubebuilder/test/projects/validations/pkg/client/informers/externalversions"
)

// InjectArgs are the arguments need to initialize controllers
type InjectArgs struct {
	args.InjectArgs

	Clientset *clientset.Clientset
	Informers informer.SharedInformerFactory
}

// CreateInjectArgs returns new controller args
func CreateInjectArgs(config *rest.Config) InjectArgs {
	cs := clientset.NewForConfigOrDie(config)
	return InjectArgs{
		InjectArgs: args.CreateInjectArgs(config),

		Clientset: cs,
		Informers: informer.NewSharedInformerFactory(cs, 2*time.Minute),
	}
}
