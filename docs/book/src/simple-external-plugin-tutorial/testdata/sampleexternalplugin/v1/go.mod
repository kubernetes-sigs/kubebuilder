module v1

go 1.24.6

require (
	github.com/spf13/afero v1.15.0
	github.com/spf13/pflag v1.0.10
	sigs.k8s.io/kubebuilder/v4 v4.10.1
)

replace sigs.k8s.io/kubebuilder/v4 => ../../../../../../../

require (
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
