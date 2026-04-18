# Controller-gen CLI

Kubebuilder makes use of a tool called
[controller-gen](https://sigs.k8s.io/controller-tools/cmd/controller-gen)
for generating utility code and Kubernetes YAML.  The presence of special ["marker
comments"](markers.md) in Go code controls this code and config
generation.

controller-gen builds out of different "generators" (which specify what
to generate) and "output rules" (which specify how and where to write the
results).

You configure both through command line options specified in [marker
format](/reference/markers.md).

For instance, the following command:

```shell
controller-gen paths=./... crd:trivialVersions=true rbac:roleName=controller-perms output:crd:artifacts:config=config/crd/bases
```

generates CRDs and RBAC, and specifically stores the generated CRD YAML in
`config/crd/bases`.  For the RBAC, it uses the default output rules
(`config/rbac`).  It considers every package in the current directory tree
(as per the normal rules of the go `...` wildcard).

## Generators

You configure each different generator through a CLI option. You can use multiple
generators in a single invocation of `controller-gen`.

{{#markerdocs CLI: generators}}

## Output rules

Output rules configure how a given generator outputs its results. There is
always one global "fallback" output rule (specified as `output:<rule>`),
plus per-generator overrides (specified as `output:<generator>:<rule>`).

<aside class="note" role="note">

<p class="note-title">Default rules</p>

When you do not manually specify a fallback rule, controller-gen uses a set of default
per-generator rules which result in YAML going to
`config/<generator>`, and code staying where it belongs.

The default rules are equivalent to
`output:<generator>:artifacts:config=config/<generator>` for each
generator.

When a "fallback" rule is specified, that is used instead of the
default rules.

For example, if you specify `crd rbac:roleName=controller-perms
output:crd:stdout`, you get CRDs on standard out, and rbac in a file in
`config/rbac`. If you were to add in a global rule instead, like `crd
rbac:roleName=controller-perms output:crd:stdout output:none`, you would get
CRDs to standard out, and everything else to /dev/null, because we have
explicitly specified a fallback.

</aside>

For brevity, the per-generator output rules (`output:<generator>:<rule>`)
are omitted below.  They are equivalent to the global fallback options
listed here.

{{#markerdocs CLI: output rules (optionally as output:<generator>:...)}}

## Other options

{{#markerdocs CLI: generic}}
