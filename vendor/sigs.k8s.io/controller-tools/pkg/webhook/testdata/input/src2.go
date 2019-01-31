package foo

import (
	"fmt"
	"time"
)

// +kubebuilder:webhook:port=7890,cert-dir=/tmp/test-cert
// +kubebuilder:webhook:service=test-system:webhook-service,selector=app:webhook-server
// +kubebuilder:webhook:secret=test-system:webhook-secret
// +kubebuilder:webhook:mutating-webhook-config-name=test-mutating-webhook-cfg,validating-webhook-config-name=test-validating-webhook-cfg
// bar function
// nolint
func foo() {
	fmt.Println(time.Now())
}
