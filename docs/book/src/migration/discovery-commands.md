# Step 2: Discovery CLI Commands

Use AI to analyze your (now reorganized) Kubebuilder project and generate all CLI commands needed to recreate it with the latest version.

<aside class="note">

<h4>You May Not Need This</h4>

**If you have a PROJECT file** and used Kubebuilder CLI to scaffold **all** resources (APIs, controllers, webhooks), you can use `kubebuilder alpha generate` instead.

The `alpha generate` command re-scaffolds everything tracked in your PROJECT file automatically. See the [alpha generate documentation](../reference/commands/alpha_generate.md) for details.

**Use this AI discovery step if:**
- You don't have a PROJECT file (Kubebuilder < v3.0.0)
- You manually created some APIs, controllers, or webhooks (not tracked in PROJECT file)
- You want to verify all resources are discovered

</aside>

<aside class="note">

<h4>When to Use This</h4>

Use AI discovery if your project has:
- APIs not tracked in the PROJECT file (manually created)
- Controllers for external types (Deployments, Pods, cert-manager resources)
- Multiple versions of the same Kind
- Complex webhook configurations

AI scans your entire codebase to discover everything, ensuring nothing is missed.

</aside>

## Instructions to provide to your AI assistant

<aside class="warning">

<h4>Standard Kubebuilder Layout Only</h4>

These instructions work for projects using **standard Kubebuilder directory layout**:
- API types in `api/` directory (some projects use `apis/`)
- Controllers in `controllers/`, `internal/controller/`, or `pkg/controllers/`
- Standard file naming: `<kind>_types.go`, `<kind>_controller.go`

Projects with heavily customized layouts may require manual analysis.

</aside>

Copy and paste these instructions to your AI assistant (Cursor, Claude, GitHub Copilot, etc.):

```
Analyze this Kubebuilder project and generate all CLI commands to recreate it.

CONTEXT:
Kubebuilder projects have these components:

APIs (Custom Resources):
- Location: api/ or apis/ directory
- Recognition: Look for Go structs with marker: // +kubebuilder:object:root=true
- Pattern: type <Name> struct with metav1.TypeMeta and metav1.ObjectMeta fields
- Example: type Captain struct { metav1.TypeMeta; metav1.ObjectMeta; Spec CaptainSpec; Status CaptainStatus }

Controllers:
- Location: controllers/, internal/controller/, or pkg/controllers/
- Recognition: Look for Reconcile() function signature
- Pattern: func (r *<Name>Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
- Struct embeds: client.Client
- Has function: SetupWithManager(mgr ctrl.Manager) error

Webhooks:
- Location: api/v1/ or internal/webhook/v1/
- Recognition: Look for webhook method signatures
- Defaulting pattern: func Default() or func Default(ctx context.Context, obj *<Type>) error
- Validation pattern: func ValidateCreate() error or func ValidateCreate(ctx context.Context, obj *<Type>) (admission.Warnings, error)
- Conversion pattern: func Hub() or func ConvertTo() or func ConvertFrom()

CLI Command Formats:
- kubebuilder init --domain <domain> --repo <module>
- kubebuilder edit --multigroup=true (if multi-group layout)
- kubebuilder create api --group <group> --version <version> --kind <Kind> --controller=<bool> --resource=<bool>
  * --controller=true: create controller
  * --resource=true: create API definition
  * --resource=false: controller only (for external types like Deployment, Pod)
- kubebuilder create webhook --group <group> --version <version> --kind <Kind> [flags]
  * --defaulting: sets default values
  * --programmatic-validation: validates create/update/delete
  * --conversion --spoke <versions>: for multi-version APIs (hub-spoke pattern)
    - Hub version: Usually oldest stable version (e.g., v1) - command runs on this version
    - Spoke versions: Newer versions that convert to/from hub (e.g., v2, v3) - specified with --spoke
    - Example: --group crew --version v1 --kind Captain --conversion --spoke v2
      (v1 is hub, v2 is spoke)
- External types (k8s.io/api/*): use --resource=false --controller=true

Project structure patterns:
- Single-group: api/v1/, api/v2/ (versions directly under api/)
- Multi-group: api/<group>/v1/, api/<group>/v2/ (group subdirectories)
- Multi-group detection: Check PROJECT file for "multigroup: true" OR check if api/ has group subdirectories

Files to IGNORE:
- zz_generated.*.go (auto-generated code)
- groupversion_info.go (just group registration)
- config/crd/bases/*.yaml (auto-generated from code)
- config/rbac/*.yaml (auto-generated from markers)

References:
- Kubebuilder Book: https://book.kubebuilder.io
- controller-runtime: https://github.com/kubernetes-sigs/controller-runtime
- controller-tools: https://github.com/kubernetes-sigs/controller-tools

ANALYZE PROJECT:

1. Extract module path from go.mod (line 1: "module <path>")
2. Extract domain from PROJECT file (domain: <value>) OR api/*/groupversion_info.go (// +groupName=<group>.<domain>)
3. Detect multi-group: api/ has api/<group>/v1/ structure? (yes/no)

4. Scan api/ or apis/ directory - Find ALL your own APIs:
   - Find all *_types.go files OR types.go (exclude groupversion_info.go, zz_generated.deepcopy.go)
   - For each file, find: type <Kind> struct with // +kubebuilder:object:root=true above it
   - Extract: Kind name, group (from groupversion_info.go +groupName comment), version (from directory)
   - Check controller: look for controllers/<lowercaseKind>_controller.go OR internal/controller/<lowercaseKind>_controller.go OR pkg/controllers/<lowercaseKind>_controller.go
   - Check webhooks: look for api/v1/<lowercaseKind>_webhook.go OR internal/webhook/v1/<lowercaseKind>_webhook.go
   - If webhook file found, scan for methods:
     * "func (r *<Kind>) Default()": has --defaulting
     * "func (r *<Kind>) ValidateCreate()": has --programmatic-validation
     * "func (*<Kind>) Hub()": this version is conversion hub
     * "func (r *<Kind>) ConvertTo(": this version is a spoke

5. Scan internal/controller/, controllers/, or pkg/controllers/ - Find controllers for external types:
   - For each *_controller.go file, check imports NOT from your module
   - Look for: k8s.io/api/apps/v1, k8s.io/api/core/v1, github.com/cert-manager/cert-manager/pkg/apis/*
   - Extract type from: type <Kind>Reconciler struct OR Reconcile signature
   - This is a controller-only resource (use --controller=true --resource=false)

6. Scan internal/webhook/ - Find webhooks for external types:
   - For each *_webhook.go file in internal/webhook/v1/ (or other versions)
   - Check if the Kind type is imported (not defined in your api/)
   - If imported from k8s.io/api/* or external package: external type webhook
   - Scan for Default() and ValidateCreate() methods to determine flags

OUTPUT FORMAT (bash script):

#!/bin/bash
# Module: <module-path>
# Domain: <domain>
# Multi-group: <yes/no>

set -e
kubebuilder init --domain <domain> --repo <module-path>
kubebuilder edit --multigroup=true  # only if multi-group

# External type controllers (--resource=false)
kubebuilder create api --group cert-manager --version v1 --kind Certificate \
  --controller=true --resource=false \
  --external-api-path=<path> --external-api-domain=<domain> --external-api-module=<module>

# Your own APIs (--resource=true)
kubebuilder create api --group crew --version v1 --kind Captain --controller=true --resource=true
kubebuilder create api --group crew --version v2 --kind FirstMate --controller=false --resource=true

# Webhooks for your own APIs
kubebuilder create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
kubebuilder create webhook --group crew --version v1 --kind FirstMate --conversion --spoke v2

# Webhooks for external/core types (no create api needed)
kubebuilder create webhook --group apps --version v1 --kind Deployment --defaulting --programmatic-validation
kubebuilder create webhook --group core --version v1 --kind Pod --defaulting

make manifests && make generate && make build

RULES:
- Combine ALL webhook types in ONE command: --defaulting --programmatic-validation together
- Conversion webhooks: use hub version and list ALL spokes: --conversion --spoke v2,v3
- List EVERY Kind found in source code, not just what's in PROJECT file
- External type controllers: use --controller=true --resource=false
- Webhooks for external/core types: just create webhook (no create api needed)
- Order: external controllers first, then your APIs, then all webhooks
```

## Understanding the Output

The AI will analyze your project and output a bash script. The script will contain commands in this order:

1. `kubebuilder init` - Initialize the project
2. `kubebuilder edit --multigroup=true` - If multi-group detected
3. `kubebuilder create api` - For external type controllers (with `--resource=false`)
4. `kubebuilder create api` - For your own APIs (with `--resource=true`)
5. `kubebuilder create webhook` - For all webhooks
6. `make manifests && make generate && make build` - Verify

## Example Outputs

Here are real examples of what the AI instructions generate:

### Example 1: Simple Multi-Group Project

Analyzed: [kubernetes-sigs/scheduler-plugins](https://github.com/kubernetes-sigs/scheduler-plugins)

```bash
#!/bin/bash
# Module: sigs.k8s.io/scheduler-plugins
# Domain: scheduling.x-k8s.io
# Multi-group: YES

set -e
kubebuilder init --domain scheduling.x-k8s.io --repo sigs.k8s.io/scheduler-plugins
kubebuilder edit --multigroup=true

kubebuilder create api --group scheduling --version v1alpha1 --kind ElasticQuota --controller=true --resource=true
kubebuilder create api --group scheduling --version v1alpha1 --kind PodGroup --controller=true --resource=true

make manifests && make generate && make build
```

**Discovered:** 2 APIs, multi-group, no webhooks

### Example 2: Single-Group with Webhooks (go/v3 Migration)

Analyzed: [project-v3](https://github.com/kubernetes-sigs/kubebuilder/tree/release-3.13/testdata/project-v3)

```bash
#!/bin/bash
# Module: sigs.k8s.io/kubebuilder/testdata/project-v3
# Domain: testproject.org
# Multi-group: NO

set -e
kubebuilder init --domain testproject.org --repo sigs.k8s.io/kubebuilder/testdata/project-v3

kubebuilder create api --group crew --version v1 --kind Captain --controller=true --resource=true
kubebuilder create api --group crew --version v1 --kind FirstMate --controller=true --resource=true
kubebuilder create api --group crew --version v1 --kind Admiral --controller=true --resource=true

kubebuilder create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
kubebuilder create webhook --group crew --version v1 --kind Admiral --defaulting

make manifests && make generate && make build
```

**Discovered:** 3 APIs, single-group, webhooks with defaulting and validation

### Example 3: Complex Multi-Group with External Types

Analyzed: testdata/project-v4-multigroup

```bash
#!/bin/bash
# Module: sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup
# Domain: testproject.org
# Multi-group: YES

set -e
kubebuilder init --domain testproject.org --repo sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup
kubebuilder edit --multigroup=true

# External type controllers
kubebuilder create api --group cert-manager --version v1 --kind Certificate \
  --controller=true --resource=false \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager@v1.19.2
kubebuilder create api --group apps --version v1 --kind Deployment --controller=true --resource=false

# APIs - Group: crew
kubebuilder create api --group crew --version v1 --kind Captain --controller=true --resource=true

# APIs - Group: ship
kubebuilder create api --group ship --version v1beta1 --kind Frigate --controller=true --resource=true
kubebuilder create api --group ship --version v1 --kind Destroyer --controller=true --resource=true
kubebuilder create api --group ship --version v2alpha1 --kind Cruiser --controller=true --resource=true

# APIs - Group: sea-creatures
kubebuilder create api --group sea-creatures --version v1beta1 --kind Kraken --controller=true --resource=true
kubebuilder create api --group sea-creatures --version v1beta2 --kind Leviathan --controller=true --resource=true

# APIs - Group: foo.policy
kubebuilder create api --group foo.policy --version v1 --kind HealthCheckPolicy --controller=true --resource=true

# APIs - Group: foo
kubebuilder create api --group foo --version v1 --kind Bar --controller=true --resource=true

# APIs - Group: fiz
kubebuilder create api --group fiz --version v1 --kind Bar --controller=true --resource=true

# APIs - Group: example.com
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --controller=true --resource=true
kubebuilder create api --group example.com --version v1alpha1 --kind Busybox --controller=true --resource=true
kubebuilder create api --group example.com --version v1 --kind Wordpress --controller=true --resource=true
kubebuilder create api --group example.com --version v2 --kind Wordpress --controller=false --resource=true

# Webhooks for your APIs
kubebuilder create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
kubebuilder create webhook --group ship --version v1 --kind Destroyer --defaulting
kubebuilder create webhook --group ship --version v2alpha1 --kind Cruiser --programmatic-validation
kubebuilder create webhook --group example.com --version v1alpha1 --kind Memcached --programmatic-validation
kubebuilder create webhook --group example.com --version v1 --kind Wordpress --conversion --spoke v2

# Webhooks for external types
kubebuilder create webhook --group cert-manager --version v1 --kind Issuer --defaulting
kubebuilder create webhook --group core --version v1 --kind Pod --programmatic-validation
kubebuilder create webhook --group apps --version v1 --kind Deployment --defaulting --programmatic-validation

make manifests && make generate && make build
```

**Discovered:** 12 APIs across 6 groups, conversion webhook, external controllers, external webhooks

## What to Do Next

1. Review the generated script carefully and ensure it matches your project structure.
2. Save it as `migration-commands.sh` and make it executable: `chmod +x migration-commands.sh`
3. Follow the [Manual Migration Process](./manual-process.md) to:
   - Backup your project in another location
   - Execute the commands of this script in the root of your project when it is empty
   - After you have the fully re-scaffolded project, you will need to add all your code back on top of it
   - Port your custom code