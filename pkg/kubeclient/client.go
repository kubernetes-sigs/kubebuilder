package kubeclient

import (
	"context"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

// Interface defines capability of the Kubernetes Client.
type Interface interface {
	// Create creates given object in Kubernetes cluster
	Create(ctx context.Context, obj runtime.Object) error

	// Get retrieves given object from Kubernetes cluster
	Get(ctx context.Context, obj runtime.Object) error

	// Update updates given object in Kubernetes cluster
	Update(ctx context.Context, obj runtime.Object) error

	// Delete deletes given object from Kubernetes cluster
	Delete(ctx context.Context, obj runtime.Object, opts *metav1.DeleteOptions) error

	// List retrieves list of object
	List(ctx context.Context, obj runtime.Object, opts *metav1.ListOptions) error

	// TODO: define capabilities for ListDeletion, Patch and Watch ?
}

// NewFromConfigOrCluster returns a Kubernetes client. If configPath is
// provided, it uses the configPath to initialize a client else it assumes to be
// running within a Kubernetes cluster and tries to initialize with cluster
// configuration.
func FromConfigOrCluster(configPath string) (Interface, error) {

}

// NewFromConfig returns a Kubernetes client given a path to the config file to
// connect to Kubernetes cluster.
func FromConfig(configPath string) (Interface, error) {

}

// InCluster() uses default cluster configuration to initialize a Kubernetes
// client. It assumes that client is being invoked within a Kubernetes cluster.
func InCluster() (Interface, error) {

}

// Client is a concrete implementation of Kubernetes Çlient interface.
type Client struct {
}

// Create saves the object obj in to the Kubernetes cluster. obj must have
// TypeMeta and ObjectMeta field populated. obj must be a struct pointer
// because obj is updated with the content returned by the server.
func (c *Client) Create(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Get retrieves the obj from the Kubernetes Cluster. obj must have TypeMeta and
// ObjectMeta field populated. obj must be a struct pointer so that obj can be
// updated with the content returned by the API Server.
func (c *Client) Get(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Update updates the given obj in the Kubernetes cluster. obj must have TypeMeta and
// ObjectMeta field populated.
func (c *Client) Update(ctx context.Context, obj runtime.Object) error {
	return nil
}

// Delete deletes the given obj from Kubernetes cluster. obj must have TypeMeta and
// ObjectMeta field populated.
func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	return nil
}

// List retrieves list of objects for a given namespace, TypeMeta info
// in the obj and list options. obj must have populated TypeMeta field. On a
// successful call, Items field in the obj will be populated with the returned
// contained.
func (c *Client) List(ctx context.Context, namespace string, obj runtime.Object, opts *metav1.ListOptions) error {
	return nil
}
