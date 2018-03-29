package internal

import "net/url"

var APIServerDefaultArgs = []string{
	"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
	"--cert-dir={{ .CertDir }}",
	"--insecure-port={{ if .URL }}{{ .URL.Port }}{{ end }}",
	"--insecure-bind-address={{ if .URL }}{{ .URL.Hostname }}{{ end }}",
	"--secure-port=0",
}

func DoAPIServerArgDefaulting(args []string) []string {
	if len(args) != 0 {
		return args
	}

	return APIServerDefaultArgs
}

func GetAPIServerStartMessage(u url.URL) string {
	if isSecureScheme(u.Scheme) {
		// https://github.com/kubernetes/kubernetes/blob/5337ff8009d02fad613440912e540bb41e3a88b1/staging/src/k8s.io/apiserver/pkg/server/serve.go#L89
		return "Serving securely on " + u.Host
	}

	// https://github.com/kubernetes/kubernetes/blob/5337ff8009d02fad613440912e540bb41e3a88b1/pkg/kubeapiserver/server/insecure_handler.go#L121
	return "Serving insecurely on " + u.Host
}
