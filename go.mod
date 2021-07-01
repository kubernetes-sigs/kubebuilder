module sigs.k8s.io/kubebuilder/v3

go 1.16

require (
	github.com/cloudflare/cfssl v1.5.0 // for `kubebuilder alpha config-gen`
	github.com/gobuffalo/flect v0.2.2
	github.com/joelanford/go-apidiff v0.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.1.2
	k8s.io/apimachinery v0.21.1 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/controller-tools v0.6.0 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/kustomize/kyaml v0.10.21 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/yaml v1.2.0
)
