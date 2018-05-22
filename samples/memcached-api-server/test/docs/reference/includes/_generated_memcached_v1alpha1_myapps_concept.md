

-----------
# Memcached v1alpha1



Group        | Version     | Kind
------------ | ---------- | -----------
`myapps` | `v1alpha1` | `Memcached`







Memcached



Field        | Description
------------ | -----------
`apiVersion`<br /> *string*    | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources
`kind`<br /> *string*    | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
`metadata`<br /> *[ObjectMeta](#objectmeta-v1)*    | 
`spec`<br /> *[MemcachedSpec](#memcachedspec-v1alpha1)*    | 
`status`<br /> *[MemcachedStatus](#memcachedstatus-v1alpha1)*    | 


### MemcachedSpec v1alpha1

<aside class="notice">
Appears In:

<ul>
<li><a href="#memcached-v1alpha1">Memcached v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`size`<br /> *integer*    | INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

### MemcachedStatus v1alpha1

<aside class="notice">
Appears In:

<ul>
<li><a href="#memcached-v1alpha1">Memcached v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`nodes`<br /> *string array*    | INSERT ADDITIONAL STATUS FIELD - define observed state of cluster





