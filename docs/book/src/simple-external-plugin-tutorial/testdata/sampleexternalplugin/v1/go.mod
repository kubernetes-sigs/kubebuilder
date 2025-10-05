module v1

go 1.24.5

require (
	github.com/spf13/pflag v1.0.10
	sigs.k8s.io/kubebuilder/v4 v4.9.0
	sigs.k8s.io/yaml v1.6.0
)

replace sigs.k8s.io/kubebuilder/v4 => ../../../../../../../

require (
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
)
