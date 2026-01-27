module v1

go 1.25.3

require (
	github.com/spf13/afero v1.15.0
	github.com/spf13/pflag v1.0.10
	sigs.k8s.io/kubebuilder/v4 v4.11.0
)

replace sigs.k8s.io/kubebuilder/v4 => ../../../../../../../

require (
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
