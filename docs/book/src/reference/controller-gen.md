# controller-gen CLI

KubeBuilder makes use of a tool called
[controller-gen](https://sigs.k8s.io/controller-tools/cmd/controller-gen)
for generating utility code and Kubernetes YAML.  This code and config
generation is controlled by the presence of special ["marker
comments"](/reference/markers.md) in Go code.

controller-gen is built out of different "generators" (which specify what
to generate) and "output rules" (which specify how and where to write the
results).

Both are configured through command line options specified in [marker
format](/reference/markers.md).

For instance,

```shell
controller-gen paths=./... crd:trivialVersions=true rbac:roleName=controller-perms output:crd:artifacts:config=config/crd/bases
```

generates CRDs and RBAC, and specifically stores the generated CRD YAML in
`config/crd/bases`.  For the RBAC, it uses the default output rules
(`config/rbac`).  It considers every package in the current directory tree
(as per the normal rules of the go `...` wildcard).

## Generators

Each different generator is configured through a CLI option.  Multiple
generators may be used in a single invocation of `controller-gen`.

{{#markerdocs CLI: generators}}

## Output Rules

Output rules configure how a given generator outputs its results. There is
always one global "fallback" output rule (specified as `output:<rule>`),
plus per-generator overrides (specified as `output:<generator>:<rule>`).

<aside class="note">

<h1>Default Rules</h1>

When no fallback rule is specified manually, a set of default
per-generator rules are used which result in YAML going to
`config/<generator>`, and code staying where it belongs.

The default rules are equivalent to
`output:<generator>:artifacts:config=config/<generator>` for each
generator.

When a "fallback" rule is specified, that'll be used instead of the
default rules.

For example, if you specify `crd rbac:roleName=controller-perms
output:crd:stdout`, you'll get CRDs on standard out, and rbac in a file in
`config/rbac`. If you were to add in a global rule instead, like `crd
rbac:roleName=controller-perms output:crd:stdout output:none`, you'd get
CRDs to standard out, and everything else to /dev/null, because we've
explicitly specified a fallback.

</aside>

For brevity, the per-generator output rules (`output:<generator>:<rule>`)
are omitted below.  They are equivalent to the global fallback options
listed here.

{{#markerdocs CLI: output rules (optionally as output:<generator>:...)}}

## Other Options

{{#markerdocs CLI: generic}}
