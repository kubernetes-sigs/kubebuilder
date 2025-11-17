# Manual Migration Process

Please ensure you have followed the [installation guide][quick-start]
to install the required components and have the desired version of the
Kubebuilder CLI available in your `PATH`.

This guide outlines the manual steps to migrate your existing Kubebuilder
project to a newer version of the Kubebuilder framework. This process involves
re-scaffolding your project and manually porting over your custom code and configurations.

<aside class="warning">
<h1>About Manual Migration</h1>

Manual migration is more complex and susceptible to errors compared to automated methods.
However, this approach gives you complete control and visibility into every change, which is valuable when:
- Your project has significant customizations
- You want to understand the new project structure thoroughly
- Automated tools are not available for your project version

</aside>

From Kubebuilder `v3.0.0` onwards, all inputs used by Kubebuilder are tracked in the [PROJECT][project-config] file.
Ensure that you check this file in your current project to verify the recorded configuration and metadata.
Review the [PROJECT file documentation][project-config] for a better understanding.

Also, before starting, it is recommended to check [What's in a basic project?][basic-project-doc]
to better understand the project layouts and structure.


## Step 1: Prepare Your Current Project

Before starting the migration, you need to backup your work and identify key project information.

### 1.1 Create a backup branch

Create a branch from your current codebase to preserve your work:

```bash
git checkout -b migration
```

<aside class="note">
<h1>Understanding the PROJECT file</h1>

From Kubebuilder `v3.0.0` onwards, the `PROJECT` file tracks all inputs and scaffolding metadata,
including APIs, controllers, webhooks, plugins, and project configuration. This file is essential
for automated migration tools like `alpha generate` and `alpha update`.

If your project was created before `v3.0.0`:
- You will not have a `PROJECT` file
- You'll need to identify your APIs, controllers, and webhooks manually by examining the directory structure
- Automated migration tools won't work; you must use the manual process

Recommendation: When re-scaffolding, use Kubebuilder CLI for all APIs, controllers, and webhooks
(including those for external types like Kubernetes built-in resources). This creates a clean base state
that works smoothly with future automated migrations and updates.

</aside>

### 1.2 Create a backup

Create a directory to hold all your current project files as a backup:

```bash
mkdir ../migration-backup
cp -r . ../migration-backup/
```

### 1.3 Clean your project directory

Remove all files except `.git` from your current project directory to start fresh:

```bash
find . -not -path './.git*' -not -name '.' -not -name '..' -delete
```

## Step 2: Initialize the New Project

You have two options: use `alpha generate` if your project has a `PROJECT` file, or manually initialize.

<aside class="note">
<h1>Option A: Try alpha generate first</h1>

Recommended for projects created with Kubebuilder `v3.0.0`+.

If your project has a `PROJECT` file, you can try using `alpha generate`:

```shell
kubebuilder alpha generate
```

This will re-scaffold everything that was created using the CLI based on your `PROJECT` file.

Limitations:
- Only works for projects created with Kubebuilder `v3.0.0` or later
- Only re-scaffolds APIs, controllers, and webhooks that were created using the CLI
- May leave a partial initial state if you have manually created resources
- You will need to verify and manually re-scaffold anything that was created outside the CLI

If `alpha generate` completes successfully, you can skip to [Step 3.1](#31-identify-all-your-apis) to verify everything was scaffolded correctly, then proceed to [Step 4](#step-4-port-your-custom-code).

See the [alpha generate command reference](../reference/commands/alpha_generate.md) for details.

</aside>

### Option B: Manual Initialization

If `alpha generate` doesn't work or you prefer manual control, follow these steps:

### 2.1 Identify your module and domain

First, identify the information you'll need for initialization. You can compare with your main branch or check the backup directory.

**Module path** - Check your `go.mod` file:

```bash
# Compare with main branch
git show main:go.mod
```

Look for the module line:

```go
module tutorial.kubebuilder.io/migration-project
```

**Domain** - Check your `PROJECT` file (if it exists):

```bash
# Compare with main branch
git show main:PROJECT
```

Look for the domain line:

```yaml
domain: tutorial.kubebuilder.io
```

<aside class="note">
<h1>Alternative: Use backup directory</h1>

If you prefer, you can view these files from your backup directory:

```bash
cat ../migration-backup/go.mod
cat ../migration-backup/PROJECT
```

</aside>

If you don't have a `PROJECT` file (versions < `v3.0.0`),
check your CRD files under `config/crd/bases/` or examine the API group names.
The domain is the part after the group name in your API groups.

### 2.2 Initialize the Go module

Initialize a new Go module using the same module path from your original project:

```bash
go mod init tutorial.kubebuilder.io/migration-project
```

Replace `tutorial.kubebuilder.io/migration-project` with your actual module path.

### 2.3 Initialize Kubebuilder project

Initialize the project with Kubebuilder:

```bash
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/migration-project
```

Replace with your actual domain and repository (module path).

<aside class="note">
<h1>Understanding init options</h1>

- `--domain`: The domain for your API groups (e.g., `tutorial.kubebuilder.io`). Your full API groups will be `<group>.<domain>`.
- `--repo`: Your Go module path (same as in `go.mod`)

</aside>

### 2.4 Enable multi-group support (if needed)

**Multi-group** projects organize APIs into different groups, with each group in its own directory.
This is useful when you have APIs for different purposes or domains.

**Check if your project uses multi-group layout** by examining your backup's directory structure:

- **Single-group layout:** All APIs in one group
  - `api/v1/cronjob_types.go`
  - `api/v1/job_types.go`
  - `api/v2/cronjob_types.go`

- **Multi-group layout:** APIs organized by group
  - `api/batch/v1/cronjob_types.go`
  - `api/crew/v1/captain_types.go`
  - `api/sea/v1/ship_types.go`

You can also check your backup's `PROJECT` file for:

```yaml
multigroup: true
```

**If your project uses multi-group layout**, enable it before creating APIs:

```bash
kubebuilder edit --multigroup=true
```

<aside class="warning">
<h1>Important</h1>

This must be done before creating any APIs if you want the multi-group layout.

</aside>

When following this guide for any to latest migration, you'll naturally get the new layout since you're creating
a fresh v4 project and porting your code into it.

## Step 3: Re-scaffold APIs and Controllers

For each API resource in your original project, re-scaffold them in the new project.

### 3.1 Identify all your APIs

Review your backup project (`../migration-backup/`) to identify all APIs. **It's recommended to check the backup directory
regardless of whether you have a `PROJECT` file**, as not all resources may have been created using the CLI.

**Check the directory structure** in your backup to ensure you don't miss any manually created resources:

- Look in the `api/` directory for `*_types.go` files:
  - Single-group: `api/v1/cronjob_types.go` → extract: version `v1`, kind `CronJob`, group from imports
  - Multi-group: `api/batch/v1/cronjob_types.go` → extract: group `batch`, version `v1`, kind `CronJob`

- Check for controllers. The location depends on the Kubebuilder version:
  - **Newer versions (v3+):** `internal/controller/cronjob_controller.go`
  - **Older versions:** `controllers/cronjob_controller.go`
  - A file like `cronjob_controller.go` indicates a controller exists for that kind

**If you used the CLI to create all APIs from Kubebuilder `v3.0.0+` you should have then in the `PROJECT` file** under the `resources` section, such as:

```yaml
resources:
  - api:
      crdVersion: v1
      namespaced: true
    controller: true
    group: batch
    kind: CronJob
    version: v1
```

<aside class="note">
<h1>Tip</h1>

Make a list of all APIs with their group, version, kind, and whether they have a controller.
This will help you systematically re-scaffold everything.

</aside>

### 3.2 Create each API and Controller

For each API identified in step 3.1, re-scaffold it:

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

When prompted:
- Answer **yes** to "Create Resource [y/n]" to generate the API types
- Answer **yes** to "Create Controller [y/n]" if your original project has a controller for this API

**After creating each API**, update the generated manifests and code:

```bash
make manifests  # Generate CRD, RBAC, and other config files
make generate   # Generate code (e.g., DeepCopy methods)
```

Then verify everything compiles:

```bash
make build
```

These steps ensure the newly scaffolded API is properly integrated.
See the [Quick Start][quick-start] guide for a detailed walkthrough of the API creation workflow.

Repeat this process for **ALL** APIs in your project.

<aside class="note">
<h1>Using External Types (controllers for types not defined in your project)</h1>

If your project has controllers for Kubernetes built-in types (like `Deployment`, `Pod`) or types from other projects:

```bash
kubebuilder create api --group apps --version v1 --kind Deployment --resource=false --controller=true
```

Or for CRDs from other projects, i.e. `cert-manager`'s `Certificate` type:

```bash
kubebuilder create api --group "cert-manager" --version v1 --kind Certificate --controller=true --resource=false --make=false --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io --external-api-module=github.com/cert-manager/cert-manager@v1.18.2
```

Use `--resource=false` to skip creating the API definition and only scaffold the controller.

Ensure that you check [Using External Types][external-types] for more details.

</aside>

After creating all resources, regenerate manifests:

```bash
make manifests
make generate
```

### 3.3 Re-scaffold webhooks (if applicable)

If your original project has webhooks, you need to re-scaffold them.

**Identify webhooks in your backup project:**

1. **From directory structure**, look for webhook files:
   - Legacy location: `api/v1/<kind>_webhook.go` or `api/<group>/<version>/<kind>_webhook.go`
   - Current location: `internal/webhook/<version>/<kind>_webhook.go`

2. **From `PROJECT` file** (if available), check each resource's webhooks section:

```yaml
resources:
  - api:
      ...
    webhooks:
      defaulting: true
      validation: true
      webhookVersion: v1
```

**Re-scaffold webhooks:**

For each resource with webhooks, run:

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
```

**Webhook options:**
- `--defaulting` - creates a defaulting webhook (sets default values)
- `--programmatic-validation` - creates a validation webhook (validates create/update/delete operations)
- `--conversion` - creates a conversion webhook (for multi-version APIs, see next section)

### 3.4 Re-scaffold conversion webhooks (if applicable)

If your project has multi-version APIs with conversion webhooks, you need to set up the hub-spoke conversion pattern.

<aside class="note">
<h1>Understanding Hub-Spoke Conversion</h1>

In Kubernetes multi-version APIs, the **hub** is the version that all other versions (spokes) convert to and from:
- **Hub version**: Usually the most complete/stable version (often the storage version)
- **Spoke versions**: All other versions that convert through the hub

The hub implements `Hub()` marker interface, while spokes implement `ConvertTo()` and `ConvertFrom()` methods to convert to/from the hub.

</aside>

**Setting up conversion webhooks:**

The conversion webhook is created for the **hub** version, with spoke versions specified using the `--spoke` flag:

```bash
kubebuilder create webhook --group batch --version v2 --kind CronJob --conversion --spoke v1
```

This command:
- Creates the conversion webhook infrastructure for version `v2` (the hub)
- Sets up conversion for version `v1` (the spoke) to convert to/from `v2`
- Generates `cronjob_conversion.go` files with conversion method stubs

**For multiple spokes**, you can specify them as a comma-separated list:

```bash
kubebuilder create webhook --group batch --version v2 --kind CronJob --conversion --spoke v1,v1alpha1
```

This sets up `v2` as the hub with both `v1` and `v1alpha1` as spokes.

**What you need to implement:**

The command generates method stubs that you'll fill in during Step 4:
- **Hub version**: Implement `Hub()` method (usually just a marker)
- **Spoke versions**: Implement `ConvertTo(hub)` and `ConvertFrom(hub)` methods with your conversion logic

See the [Multi-Version Tutorial][multiversion-tutorial] for comprehensive guidance on implementing the conversion logic.

<aside class="note">
<H1> Forget a type of webhook ? </h1>

If you forget a webhook type, use `--force` to re-run the command:

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --force
```
</aside>

<aside class="note">
<h1>Webhook for External Types</h1>

**For external types**, you can also create webhooks:

```bash
kubebuilder create webhook --group apps --version v1 --kind Deployment --defaulting --programmatic-validation
```

More info: [Webhook Overview][webhook-overview], [Admission Webhook][admission-webhook], and [Creating Webhooks for External Types][external-types-webhooks].
</aside>

After scaffolding all webhooks, verify everything compiles:

```bash
make manifests
make generate
make build
```

## Step 4: Port Your Custom Code

Now you need to manually port your custom business logic and configurations from the backup to the new project.

<aside class="note">
<h1>Use diff tools</h1>

Use IDE diff tools or commands like `diff -r ../migration-backup/ .` to compare directories and identify all customizations you need to port.
Most modern IDEs support directory-level comparison which makes this process much easier.

</aside>

### 4.1 Port API definitions

Compare and merge your custom API fields and markers from your backup project.

**Files to compare:**

- **Single-group:** `api/v1/<kind>_types.go`
- **Multi-group:** `api/<group>/<version>/<kind>_types.go`

**What to port:**

1. **Custom fields** in Spec and Status structs
2. **Validation markers** - e.g., `+kubebuilder:validation:Minimum=0`, `+kubebuilder:validation:Pattern=...`
3. **CRD generation markers** - e.g., `+kubebuilder:printcolumn`, `+kubebuilder:resource:scope=Cluster`
4. **SubResources** - e.g., `+kubebuilder:subresource:status`, `+kubebuilder:subresource:scale`
5. **Documentation comments** - Used for CRD descriptions

See [CRD Generation][crd-generation], [CRD Validation][crd-validation], and [Markers][markers] for all available markers.

After porting API definitions, regenerate and verify:

```bash
make manifests  # Generate CRD manifests from your types
make generate   # Generate DeepCopy methods
```

This ensures your API types and CRD manifests are properly generated before moving forward.

### 4.2 Port controller logic

**Files to compare:**

- **File location:** `internal/controller/<kind>_controller.go` (or `controllers/` in older versions)

**What to port:**

1. **Reconcile function implementation** - Your core business logic
2. **Helper functions** - Any additional functions in the controller file
3. **RBAC markers** - `+kubebuilder:rbac:groups=...,resources=...,verbs=...`
4. **Additional watches** - Custom watch configurations in `SetupWithManager`
5. **Imports** - Any additional packages your controller needs
6. **Struct fields** - Custom fields added to the Reconciler struct

See [RBAC Markers][rbac-markers] for details on permission markers.

After porting controller logic, regenerate manifests and verify compilation:

```bash
make generate
make manifests
make build
```

### 4.3 Port webhook implementations

Webhooks have changed location between Kubebuilder versions. Be aware of the path differences:

**Legacy webhook location** (Kubebuilder v3 and earlier):
- `api/v1/<kind>_webhook.go`
- `api/<group>/<version>/<kind>_webhook.go`

**Current webhook location** (Kubebuilder v4+):
- `internal/webhook/v1/<kind>_webhook.go`
- `internal/webhook/<version>/<kind>_webhook.go`

**What to port:**

1. **Defaulting webhook** - `Default()` method implementation
2. **Validation webhook** - `ValidateCreate()`, `ValidateUpdate()`, `ValidateDelete()` methods
3. **Conversion webhook** - `ConvertTo()` and `ConvertFrom()` methods (for multi-version APIs)
4. **Helper functions** - Any validation or defaulting helper functions
5. **Webhook markers** - Usually auto-generated, but verify they match your needs

<aside class="note">
<h1>Important</h1>

Even if the webhook file is in a different location, the implementation logic remains largely the same.
Copy the method implementations, not just the entire file.

</aside>

See [Webhook Overview][webhook-overview], [Admission Webhook][admission-webhook], and the [Multi-Version Tutorial][multiversion-tutorial] for details.

**For conversion webhooks:**

If you have conversion webhooks, ensure you used the `create webhook --conversion --spoke <version>` command in Step 3.4. This sets up the hub-spoke infrastructure automatically. You only need to fill in the conversion logic in the `ConvertTo()` and `ConvertFrom()` methods in your spoke versions, and the `Hub()` method in your hub version.

The command creates all the necessary boilerplate - you just implement the business logic for converting fields between versions.

After porting webhooks, regenerate and verify:

```bash
make generate
make manifests
make build
```

### 4.4 Configure Kustomize manifests

The `config/` directory contains Kustomize manifests for deploying your operator. Compare with your backup to ensure all configurations are properly set up.

**Review and update these directories:**

1. **`config/default/kustomization.yaml`** - Main kustomization file
   - Ensure that webhook configurations is enabled if you have webhooks (`uncomment webhook-related patches`)
   - Ensure that cert-manager is enabled if using webhooks (`uncomment certmanager resources`)
   - Enable or disable metrics endpoint based on your original configuration
   - Review namespace and name prefix settings

2. **`config/manager/`** - Controller manager deployment
   - Usually no changes needed unless you have customizations, such as:
   - If needed: compare resource limits and requests with your backup
   - If needed: check environment variables

3. **`config/rbac/`** - RBAC configurations
   - Usually auto-generated from markers - no manual changes needed
   - Only check if you have custom role bindings or service account configurations not covered by markers

4. **`config/webhook/`** - Webhook configurations (if applicable)
   - Usually auto-generated - no manual changes needed
   - Only check if you have custom webhook service or certificate configurations

5. **`config/samples/`** - Sample CR manifests
   - Copy your sample resources from the backup

After configuring Kustomize, verify the manifests build correctly:

```bash
make all
make build-installer
```

### 4.5 Port main.go customizations (if any)

**File:** `cmd/main.go`

Most projects don't need to customize `main.go` as Kubebuilder handles all the standard setup automatically
(registering APIs, setting up controllers and webhooks, manager initialization, metrics, etc.).

Only port customizations that are not part of the standard scaffold. Compare your backup `main.go` with the
new scaffolded one to identify any custom logic you added.

### 4.6 Port additional customizations

Now try building and testing the project to identify any missing pieces:

```bash
make all
```

If you encounter issues, you may need to port additional customizations from your backup:

**Dependencies** (`go.mod`):
- Compare your backup `go.mod` with the new one
- Add any additional dependencies not part of the standard scaffold:
  ```bash
  go get <package-name>@<version>
  ```
- Then tidy up:
  ```bash
  go mod tidy
  ```

Compare your project structure with your backup to identify any other custom files or directories you may have added, such as:

- Compare the two **Makefiles** carefully using diff tools you may have custom targets: deployment helpers, code generation scripts, etc.
- Compare the **Dockerfile** you may have some custom build configurations:
- Ensure that you ported any testing-related files and configurations if you have tests in your project.

<aside class="note">
<h1>Using diff tools</h1>

Use your IDE's diff tools to compare the current directory with your backup (`../migration-backup/`) or use git to compare
your current branch with your main branch. This helps identify any files you may have missed.

</aside>

After porting all customizations, run the full build and test cycle:

```bash
make all
```

## Step 5: Test and Deploy

Thoroughly test your migrated project to ensure everything works as expected.

## Additional Resources

- [Migration Overview](../migrations.md) - Overview of all migration options
- [PROJECT File Reference][project-config] - Understanding the PROJECT file
- [What's in a basic project?][basic-project-doc] - Understanding project structure
- [Alpha Generate Command](../reference/commands/alpha_generate.md) - Automated re-scaffolding
- [Alpha Update Command](../reference/commands/alpha_update.md) - Automated migration
- [Using External Types][external-types] - Controllers for types not defined in your project
- [CRD Generation][crd-generation] - Generating CRDs from Go types
- [CRD Validation][crd-validation] - Adding validation to your APIs
- [Markers][markers] - All available markers for code generation
- [RBAC Markers][rbac-markers] - Generating RBAC manifests
- [Webhook Overview][webhook-overview] - Understanding webhooks
- [Admission Webhook][admission-webhook] - Implementing admission webhooks
- [Multi-Version Tutorial][multiversion-tutorial] - Handling multiple API versions
- [Deploying cert-manager][cert-manager] - Required for webhooks
- [Configuring EnvTest][envtest] - Testing with EnvTest

[quick-start]: ../quick-start.md
[project-config]: ../reference/project-config.md
[basic-project-doc]: ../cronjob-tutorial/basic-project.md
[external-types]: ../reference/using_an_external_resource.md
[external-types-webhooks]: ../reference/using_an_external_resource.md#creating-a-webhook-to-manage-an-external-type
[crd-generation]: ../reference/generating-crd.md
[crd-validation]: ../reference/markers/crd-validation.md
[markers]: ../reference/markers.md
[rbac-markers]: ../reference/markers/rbac.md
[webhook-overview]: ../reference/webhook-overview.md
[admission-webhook]: ../reference/admission-webhook.md
[multiversion-tutorial]: ../multiversion-tutorial/tutorial.md
[cert-manager]: ../cronjob-tutorial/cert-manager.md
[envtest]: ../reference/envtest.md
[standard-go-project]: https://github.com/golang-standards/project-layout
