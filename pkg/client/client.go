package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// ObjectKey identifies a Kubernetes Object.
type ObjectKey types.NamespacedName

// Client uses methods defined in runtime.Object to determine the REST endpoint
// for Kubernetes resources.

// Interface defines the capability for a Kubernetes client.
type Interface interface {
	// Create saves the object obj in the Kubernetes cluster.
	Create(ctx context.Context, obj runtime.Object) error

	// Get retrieves an obj for the given object key from the Kubernetes Cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.
	Get(ctx context.Context, key ObjectKey, obj runtime.Object) error

	// Update updates the given obj in the Kubernetes cluster. obj must be a
	// struct pointer so that obj can be updated with the content returned by the Server.
	Update(ctx context.Context, obj runtime.Object) error

	// Delete deletes the given obj from Kubernetes cluster.
	Delete(ctx context.Context, obj runtime.Object) error

	// List retrieves list of objects for a given namespace and list options. On a
	// successful call, Items field in the list will be populated with the
	// result returned from the server.
	List(ctx context.Context, namespace string, opts *metav1.ListOptions, list runtime.Object) error
}

// NewForConfig returns a Kubernetes client implementation given a
// configuration.
func NewForConfig(config *rest.Config) (Interface, error) {

}

// Client is a concrete implementation of Kubernetes Çlient interface.
type client struct {
}

func (c *Client) Create(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (c *Client) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error {
	return nil
}

func (c *Client) Update(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (c *Client) Delete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (c *Client) List(ctx context.Context, namespace string, opts *metav1.ListOptions, list runtime.Object) error {
	return nil
}

// ensure client struct implements the Interface.
var _ Interface = &client{}
