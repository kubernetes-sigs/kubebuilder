# Kubebuilder Annotation

If you have been using Kubebuilder, you must have seen comments such as `// +kubebuilder:rbac: ....` , `// +kubebuilder:resource:...` in scaffolder Go files. These special comments are used by kubebuilder tools (controller-tools) to generate CRD, RBAC, webhook manifests. In kubebuilder, these special comments are `Kubebuilder Annotation`, a.k.a `annotation`. It is designed for this kind of use case: To use kubebuilder tools, all you have to do is focus on writing your code, and put instructions with parameters as annotations along with your code, so that everything will be handled based on these annotations instructions by kubebuilder. This document illustrates the syntax of these annotations.

## Kubebuilder Annotation Syntax

Kubebuilder Annotation has a series of tokens separated by colons into groups from left to right. Each **Token** is a string identifier in an annotation instance. It has meaning by its position in token slice, in the form of
**+[header]:[module]:[submodule]:[key-value elements]**
Go Annotation starts with `+` (e.g. `// +kubebuilder`) to differentiate from regular go comments.

## Token types

- **header** is the identifier of a group of annotations. It helps user know which project provides this annotation. For example, in Kubernetes project, headers like `kubebuilder`, `k8s`, `genclient`, etc. are all project identifiers. A header is required for all annotations, since you may use multiple annotations from different projects in the same codebase.

- **module** is the identifier of functional module in an annotation. An annotation may have a group of modules, each of which performs a particular function.

- **submodule** (optional) In some cases, the module has a big functional scope, split into fine-grained sub modules, which provide the flexibility of extending module functionality. For example: **module:submodule1:submodule2:submodule3** submodule can be multiple following one by one.

## Levels of symbols

Delimiter symbols are distinguished to work in different levels from top-down for splitting values string in tokens, which provides readability and efficiency.

- **Colon**

  Colon `:` is the 1st level delimiter (to annotation) only for separate tokens. Tokens on different sides of the colon should refer to different token types.

- **Comma**

  Comma `,` is the 2nd level delimiter (to annotation) for splitting key-value pairs in **key-value elements** which is normally the last token in annotation. e.g. `+kubebuilder:printcolumn:name=<name>,type=<type>,description=<desc>,JSONPath:<.spec.Name>,priority=<int32>,format=<format>` It works within token which is the 2nd level of annotation, so it is called "2nd level delimiter"

- **Equal sign**

  Equal sign `=` is the 3rd level delimiter (to annotation) for identifying key and value. Since the `key=value` parts are splitted from single token (2nd level), its inner delimiter `=` works for next level (3rd level)

- **Semicolon sign**

  Semicolon sign `;` is the 4th level delimiter, which works on the `value` part (4th level) of `key=value`(3rd level) for splitting individual values. e.g. `key1=value1;value2;value3`

- **Pipe sign or Vertical bar**

  Pipe sign `|` is the 5th level delimiter, which works inside the single `value` part (4th level) indicating key and value in case of the single value has nested key-value structure. e.g. `outerkey=innerkey1|innervalue1`

### Examples

#### Webhook annotation examples

**[header]** is `kubebuilder`,
**[module]** is `webhook`,
**[submodule]** is `admission` or `serveroption`

```golang
// +kubebuilder:webhook:admission:groups=apps,resources=deployments,verbs=CREATE;UPDATE,name=bar-webhook,path=/bar,type=mutating,failure-policy=Fail

// +kubebuilder:webhook:serveroption:port=7890,cert-dir=/tmp/test-cert,service=test-system|webhook-service,selector=app|webhook-server,secret=test-system|webhook-secret,mutating-webhook-config-name=test-mutating-webhook-cfg,validating-webhook-config-name=test-validating-webhook-cfg
```

**Notes:**

1. Separate two `submodule` (categories) under `webhook`: 1) `admission`and 2) `serveroption`, handling webhookTags and serverTags separately.
2. For each submodule, all key-values should put in the same comment line.
3. using `|` for splitting key-value of `lables`

#### RBAC Annotation examples

**[header]** is `kubebuilder`
**[module]** is `rbac`
No submodule at this moment, support annotations like : `// +rbac`, `// +kubebuilder:rbac`

```golang
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete

// +rbac:groups=apps,resources=deployments,verbs=get;list;watch;delete
```