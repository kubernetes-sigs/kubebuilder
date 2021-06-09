module sigs.k8s.io/kubebuilder/v3

go 1.16

require (
	github.com/cloudflare/cfssl v1.5.0 // for `kubebuilder alpha config-gen`
	github.com/gobuffalo/flect v0.2.2
	github.com/joelanford/go-apidiff v0.1.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.0.0-20201224043029-2b0845dc783e
	k8s.io/apimachinery v0.20.2 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/controller-runtime v0.8.3 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/controller-tools v0.5.0 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/kustomize/kyaml v0.10.20 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/yaml v1.2.0
)
