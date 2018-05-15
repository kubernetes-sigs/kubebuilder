// Api versions allow the api contract for a resource to be changed while keeping
// backward compatibility by support multiple concurrent versions
// of the same resource

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/apis/myapps
// +k8s:defaulter-gen=TypeMeta
// +groupName=myapps.memcached.example.com
package v1alpha1 // import "github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/apis/myapps/v1alpha1"
