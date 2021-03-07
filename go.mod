module sigs.k8s.io/kubebuilder/v3

go 1.15

require (
	github.com/cloudflare/cfssl v1.5.0 // for `kubebuilder alpha config-gen`
	github.com/gobuffalo/flect v0.2.2
	// TODO: remove this in favor of embed once using 1.16
	github.com/markbates/pkger v0.17.1 // for `kubebuilder alpha config-gen`
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.0.0-20201224043029-2b0845dc783e
	// for `kubebuilder alpha config-gen`
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/apimachinery v0.20.2 // for `kubebuilder alpha config-gen`
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	sigs.k8s.io/controller-tools v0.3.0 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/kustomize/kyaml v0.10.10 // for `kubebuilder alpha config-gen`
	sigs.k8s.io/yaml v1.2.0
)
