package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
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
	List(ctx context.Context, namespace string, opts metav1.ListOptions, list runtime.Object) error
}

// NewForConfig returns a Kubernetes client implementation given a
// configuration.
func NewForConfig(config *rest.Config) (Interface, error) {
	if config == nil {
		return nil, fmt.Errorf("missing config")
	}
	config = rest.CopyConfig(config)
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(cs.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	c := &client{
		conf:       config,
		dc:         dc,
		scheme:     scheme.Scheme,
		restMapper: restMapper,
	}
	// c.scheme.AddToScheme(scheme.Scheme)
	c.runBackgroundCacheReset(1 * time.Minute)
	return c, nil
}

// Client is a concrete implementation of Kubernetes Çlient interface.
type client struct {
	conf       *rest.Config
	dc         dynamic.Interface
	scheme     *runtime.Scheme
	restMapper *restmapper.DeferredDiscoveryRESTMapper
}

// runBackgroundCacheReset - Starts the rest mapper cache reseting
// at a duration given.
func (c *client) runBackgroundCacheReset(duration time.Duration) {
	ticker := time.NewTicker(duration)
	go func() {
		for range ticker.C {
			c.restMapper.Reset()
		}
	}()
}

func (c *client) Create(ctx context.Context, obj runtime.Object) error {
	ns, _, err := namespaceAndName(obj)
	if err != nil {
		return err
	}
	gvks, _, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("object is not registered")
	}
	gvk := gvks[0]

	resource, err := apiResource(gvk, c.restMapper)
	if err != nil {
		return fmt.Errorf("failed to get resource type: %v", err)
	}

	result, err := c.dc.Resource(resource).Namespace(ns).Create(unstructuredFromRuntimeObject(obj))
	if err != nil {
		return err
	}
	err = unstructuredToObject(result, obj)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Get(ctx context.Context, key ObjectKey, obj runtime.Object) error {
	// create dynamic client on the fly (worry about efficiency later)

	// TODO(droot): figure out why GetObjectKind() is not working for an empty
	// object ? Shouldn't it use scheme internally. If not then, there isn't
	// much benefit in taking the runtime.Object

	// gvk := obj.GetObjectKind().GroupVersionKind()
	// apiVersion, kind := gvk.ToAPIVersionAndKind()

	// consult scheme to fetch GVK

	gvks, _, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("object is not registered")
	}

	gvk := gvks[0]

	resource, err := apiResource(gvk, c.restMapper)
	if err != nil {
		return fmt.Errorf("failed to get resource type: %v", err)
	}

	r, err := c.dc.Resource(resource).Namespace(key.Namespace).Get(key.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = unstructuredToObject(r, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Update(ctx context.Context, obj runtime.Object) error {
	ns, _, err := namespaceAndName(obj)
	if err != nil {
		return err
	}
	gvks, _, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("object is not registered")
	}
	gvk := gvks[0]

	resource, err := apiResource(gvk, c.restMapper)
	if err != nil {
		return fmt.Errorf("failed to get resource type: %v", err)
	}

	result, err := c.dc.Resource(resource).Namespace(ns).Update(unstructuredFromRuntimeObject(obj))
	if err != nil {
		return err
	}
	err = unstructuredToObject(result, obj)
	if err != nil {
		return err
	}

	return nil
}

// GetNameAndNamespace extracts the name and namespace from the given runtime.Object
// and returns a error if any of those is missing.
func namespaceAndName(object runtime.Object) (string, string, error) {
	accessor := meta.NewAccessor()
	name, err := accessor.Name(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get name for object: %v", err)
	}
	namespace, err := accessor.Namespace(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get namespace for object: %v", err)
	}
	return namespace, name, nil
}

func (c *client) Delete(ctx context.Context, obj runtime.Object) error {
	ns, name, err := namespaceAndName(obj)
	if err != nil {
		return err
	}
	gvks, _, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("object is not registered")
	}

	gvk := gvks[0]

	resource, err := apiResource(gvk, c.restMapper)
	if err != nil {
		return fmt.Errorf("failed to get resource type: %v", err)
	}

	err = c.dc.Resource(resource).Namespace(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *client) List(ctx context.Context, namespace string, opts metav1.ListOptions, list runtime.Object) error {
	gvks, _, err := c.scheme.ObjectKinds(list)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("object is not registered")
	}

	gvk := gvks[0]
	// TODO(droot): Hack to determine the resource for XXXList kinds.
	if strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	}

	resource, err := apiResource(gvk, c.restMapper)
	if err != nil {
		return fmt.Errorf("failed to get resource type: %v", err)
	}
	fmt.Printf("resource: %v \n", resource)

	ulist, err := c.dc.Resource(resource).Namespace(namespace).List(opts)
	if err != nil {
		return err
	}

	err = unstructuredListToObjectList(ulist, list)
	if err != nil {
		return err
	}
	return nil
}

// apiResource consults the REST mapper to translate an <apiVersion, kind, namespace> tuple to a schema.GroupVersionResource struct.
func apiResource(gvk schema.GroupVersionKind, restMapper *restmapper.DeferredDiscoveryRESTMapper) (schema.GroupVersionResource, error) {
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to get the resource REST mapping for GroupVersionKind(%s): %v", gvk.String(), err)
	}
	return mapping.Resource, nil
}

// unstructuredToObject converts given unstructured object in to runtime object.
func unstructuredToObject(in *unstructured.Unstructured, out runtime.Object) error {
	bytes, err := in.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return err
	}
	return nil
}

// UnstructuredFromRuntimeObject converts a runtime object to an unstructured
func unstructuredFromRuntimeObject(ro runtime.Object) *unstructured.Unstructured {
	b, err := json.Marshal(ro)
	if err != nil {
		panic(err)
	}
	var u unstructured.Unstructured
	if err := json.Unmarshal(b, &u.Object); err != nil {
		panic(err)
	}
	return &u
}

func unstructuredListToObjectList(in *unstructured.UnstructuredList, out runtime.Object) error {
	bytes, err := in.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, out)
	if err != nil {
		return err
	}
	return nil
}

// ensure client struct implements the Interface.
var _ Interface = &client{}
