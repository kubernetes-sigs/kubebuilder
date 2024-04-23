| Authors       | Creation Date | Status      | Extra |
|---------------|---------------|-------------|-------|
| @droot | 01/30/2019| implementable | -     |

# API Versioning in Kubebuilder

This document describes high level design and workflow for supporting multiple versions in an API built using Kubebuilder. Multi-version support was added as an alpha feature in kubernetes project in 1.13 release. Here are links to some recommended reading material.

* [CRD version Conversion Design Doc](https://github.com/kubernetes/community/blob/3f8bf88a06a114b3984417d6867bb16506c9c71e/contributors/design-proposals/api-machinery/customresource-conversion-webhook.md)

* [CRD Webhook Conversion API changes PR](https://github.com/kubernetes/kubernetes/pull/67795/files)

* [CRD Webhook Conversion PR](https://github.com/kubernetes/kubernetes/pull/67006)

* [Kubecon talk](https://www.youtube.com/watch?v=HsYtMvvzDyI&t=0s&index=100&list=PLj6h78yzYM2PZf9eA7bhWnIh_mK1vyOfU)

* [CRD version conversion POC](https://github.com/droot/crd-conversion-example)

# Design

## Hub and Spoke

The basic concept is that all versions of an object share the storage. So say if you have versions v1, v2 and v3 of a Kind Toy, kubernetes will use one of the versions to persist the object in stable storage i.e. Etcd. User can specify the version to be used for storage in the Custom Resource definition for that API.

One can think storage version as the hub and other versions as spoke to visualize the relationship between storage and other versions (as shown below in the diagram). The key thing to note is that conversion between storage and other version should be lossless (round trippable). As shown in the diagram below, v3 is the storage/hub version and v1, v2 and v4 are spoke version. The document uses storage version and hub interchangeably.

![hub and spoke version diagram][version-diagram]

So if each spoke version (v1, v2 and v4 in this case) defines conversion function from/to the hub version, then conversion function between the spoke versions (v1, v2, v4) can be derived. For example, for converting an object from v1 to v4, we can convert v1 to v3 (the hub version) and v3 to v4.

We will introduce two interfaces in controller-runtime to express the above relationship.

```Go
// Hub defines capability to indicate whether a versioned type is a Hub or not.

type Hub interface {
    runtime.Object
    Hub()
}

// A versioned type is convertible if it can be converted to/from a hub type.

type Convertible interface {
    runtime.Object
    ConvertTo(dst Hub) error
    ConvertFrom(src Hub) error
}
```

A spoke type needs to implement Convertible interface. Kubebuilder can scaffold the skeleton for a type when it is created. An example of Convertible implementation:

```Go
package v1

func (ej *ExternalJob) ConvertTo(dst conversion.Hub) error {
    switch t := dst.(type) {
    case *v3.ExternalJob:
        jobv3 := dst.(*v3.ExternalJob)
        jobv3.ObjectMeta = ej.ObjectMeta
         // conversion implementation
	   //
        return nil
    default:
        return fmt.Errorf("unsupported type %v", t)
    }
}

func (ej *ExternalJob) ConvertFrom(src conversion.Hub) error {
    switch t := src.(type) {
    case *v3.ExternalJob:
        jobv3 := src.(*v3.ExternalJob)
        ej.ObjectMeta = jobv3.ObjectMeta
	   // conversion implementation
        return nil
    default:
        return fmt.Errorf("unsupported type %v", t)
    }
}
```

The storage type v3 needs to implement the Hub interface:

```Go

package v3
func (ej *ExternalJob) Hub() {}

```
## Conversion Webhook Handler

Controller-runtime will implement a default conversion handler that can handle conversion requests for any API type. Code snippets below captures high level implementation details of the handler. This handler will be registered with the webhook server by default.
```Go

type conversionHandler struct {
	// scheme which has Go types for the APIs are registered. This will be injected by controller manager.
	Scheme runtime.Scheme
	// decoder which will be injected by the webhook server
	// decoder knows how to decode a conversion request and API objects.
	Decoder decoder.Decoder
}

// This is the default handler which will be mounted on the webhook server.
func (ch *conversionHandler) Handle(r *http.Request, w http.Response) {
	// decode the request to converReview request object
	convertReq := ch.Decode(r.Body)
	for _, obj := range convertReq.Objects {
	// decode the incoming object
	src, gvk, _ := ch.Decoder.Decode(obj.raw)

	// get target object instance for convertReq.DesiredAPIVersion and gvk.Kind
	dst, _ := getTargetObject(convertReq.DesiredAPIVersion, gvk.Kind)

	// this is where conversion between objects happens

	ch.ConvertObject(src, dst)

	// append dst to converted object list
}

	// create a conversion response with converted objects
}

func (ch *conversionHandler) convertObject(src, dst runtime.Object) error {
    // check if src and dst are of same type, then may be return with error because API server will never invoke this handler for same version.
    srcIsHub, dstIsHub := isHub(src), isHub(dst)
    srcIsConvertible, dstIsConvertible := isConvertible(src), isConvertable(dst)
    if srcIsHub {
        if dstIsConvertible {
            return dst.(conversion.Convertable).ConvertFrom(src.(conversion.Hub))
        } else {
            // this is error case, this can be flagged at setup time ?
            return fmt.Errorf("%T is not convertible to", src)
        }
    }

    if dstIsHub {
        if srcIsConvertible {
            return src.(conversion.Convertable).ConvertTo(dst.(conversion.Hub))
        } else {
            // this is error case.
            return fmt.Errorf("%T is not convertible", src)
        }
    }

    // neither src or dst are Hub, means both of them are spoke, so lets get the hub
    // version type.

    hub, err := getHub(scheme, src)
    if err != nil {
        return err
    }

    // shall we get Hub for dst type as well and ensure hubs are same ?
    // src and dst needs to be convertible for it to work
    if !srcIsConvertable || !dstIsConvertable {
        return fmt.Errorf("%T and %T needs to be both convertible", src, dst)
    }

    err = src.(conversion.Convertible).ConvertTo(hub)
    if err != nil {
        return fmt.Errorf("%T failed to convert to hub version %T : %v", src, hub, err)
    }

    err = dst.(conversion.Convertible).ConvertFrom(hub)
    if err != nil {
        return fmt.Errorf("%T failed to convert from hub version %T : %v", dst, hub, err)
    }
    return nil
}
```

Handler Registration flow will perform following at the startup:

* For APIs with hub defined, it can examine if spoke versions implement convertible or not and can abort with error.

* It will also be nice if we can detect an API with multiple versions but with no hub defined, but that requires distinguishing between APIs defined in the project vs external.

# CRD Generation

The tool that generates the CRD manifests lives under controller-tools repo. Currently it generates the manifests for each <group, version, kind> discovered under ‘pkg/…’ directory in the project by examining the comments (aka annotations) in Go source files. Following annotations will be added to support multi version:

## Storage/Serve annotations:

The resource annotation will be extended to indicate storage/serve attributes as shown below.

```Go
// ...
// +kubebuilder:resource:storage=true,serve=true
// …
type APIName struct {
   ...
}
```

The default value of *serve* attribute is true. The default value of *storage* attribute will be *true* for single version and *false* for multiple versions to ensure backward compatibility.

CRD generation will be extended to support the following:

* If multiple versions are detected for an API:

    * Ensure only one version is marked as storage version. Assume default value of *storage* to be *false* for this case.

    * Ensure version specific fields such as *OpenAPIValidationSchema, SubResources and AdditionalPrinterColumn* are added per version and omitted from the top level CRD definition.

* In case of single version,

    * Do not use version specific field in CRD spec because users are most likely running with k8s version < 1.13 which doesn’t support version specific specs for *OpenAPIValidationSchema, SubResources and AdditionalPrinterColumn. *This is critical to maintain backward compatibility.

    * Assume default value for storage attribute to be *true* for this case.

The above two requirements will require CRD generation logic to be divided in two phases. In first phase, parse and store CRD information in an internal structure for all versions and then generate the CRD manifest on the basis of multi-version/single-version scenario.

## Conversion Webhook annotations:

Webhook annotations will be extended to support conversion webhook fields.

```Go
// ...
// +kubebuilder:webhook:conversion:....
// ...
```

These annotations would be placed just above the API type definition to associate conversion webhook with an API type.

The exact syntax for annotation is yet to be defined, but goal is CRD generation tool to be able to extract information from these annotation to populate the `CustomResourceConversion` struct in CRD definition. The CA bits for webhook configuration will be populated by using annotations on the CRD as per the [design](https://docs.google.com/document/d/1ipTvFBRoe7fuDiz27Csm5Zb6rH0z6LJTuKM8xY3jaUg/edit?ts=5c49094e#heading=h.u7ei2s2van5b).

# Kubebuilder CLI:

kubebuilder create api --group g1 --version v2 --Kind k1 [--storage]

Fields marked in yellow are proposed new fields to the command and reasoning is stated below.

*  *--storage* flag gives an option to mark a version as storage/hub version.

Generally users have one controller per group/kind, we will avoid scaffolding code for controller if we detect that a controller already exists for an API group/kind.

# TODO:

## There is more exploration/work is required in the following areas related to API versioning:

* Making it easy to write the conversion function itself.

* Making it easy to generate tests for conversion functions using fuzzer.

* Best practices around rolling out different versions of the API

Version History

<table>
  <tr>
    <td>Version</td>
    <td>Updated on</td>
    <td>Description</td>
  </tr>
  <tr>
    <td>Draft</td>
    <td>01/30/2019
</td>
    <td>Initial version</td>
  </tr>
  <tr>
    <td>1.0</td>
    <td>02/27/2019</td>
    <td>Updated the design as per POC implementation</td>
  </tr>
</table>


[version-diaiagram]: assets/version_diagram.png
