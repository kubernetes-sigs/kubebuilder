# Manual Migration Process

Please ensure you have followed the [installation guide][quick-start]
to install the required components and have the desired version of the
Kubebuilder CLI available in your `PATH`.

This guide outlines the manual steps to migrate your existing Kubebuilder
project to a newer version of the Kubebuilder framework. This process involves
re-scaffolding your project and manually porting over your custom code and configurations.

From Kubebuilder `v3.0.0` onwards, all inputs used by Kubebuilder are tracked in the [PROJECT][project-config] file.
Ensure that you check this file in your current project to verify the recorded configuration and metadata.
Review the [PROJECT file documentation][project-config] for a better understanding.

Also, before starting, it is recommended to check [What's in a basic project?][basic-project-doc]
to better understand the project layouts and structure.


<aside class="warning">
<h1>About Manual Migration</h1>

Manual migration is more complex than automated methods but gives you complete control. Use manual migration when:
- Your project has significant customizations
- Automated tools aren't available for your version yet

**Two-phase approach (recommended for legacy layouts):**
1. **Reorganize layout** - Move files to new structure (controllers → internal/controller, webhooks → internal/webhook, main.go → cmd), update imports, test, commit
2. **Migrate to latest** - Re-scaffold with latest version, port code

This keeps your project working at each step and simplifies porting.
AI migration helpers are provided to automate repetitive tasks.
See [AI Migration Helpers](./ai-helpers.md) for AI instructions that automate both phases.

**For future updates:** Once migrated, use the [AutoUpdate plugin][autoupdate-plugin] or [alpha update][alpha-update] command to automatically update scaffolds with 3-way merge while preserving customizations.

</aside>


## Phase 1: Reorganize to New Layout (Required only for Legacy Layouts)

**Only needed if ANY of these are true:**
- Controllers are NOT in `internal/controller/`
- Webhooks are NOT in `internal/webhook/`
- Main is NOT in `cmd/`

**Skip this phase if** your project already uses `internal/controller/`, `internal/webhook/`, and `cmd/main.go`.

### 1.1 Create a reorganization branch

```bash
git checkout -b reorganize
```

### 1.2 Reorganize file locations

<aside class="note">

<h1>Skip if Using AI Migration Helper</h1>

If you used [Step 1: Reorganize to New Layout](./reorganize-layout.md) AI migration helper, your project is already reorganized. Skip to [Phase 2](#phase-2-migrate-to-latest-version).

</aside>

Move files to new layout:

```bash
# If you have controllers/ directory
mkdir -p internal/controller
mv controllers/* internal/controller/
rmdir controllers

# OR if you have pkg/controllers/ directory
mkdir -p internal/controller
mv pkg/controllers/* internal/controller/

# If you have webhooks in api/v1/ or apis/v1/
mkdir -p internal/webhook/v1
mv api/v1/*_webhook* internal/webhook/v1/ 2>/dev/null || mv apis/v1/*_webhook* internal/webhook/v1/ 2>/dev/null || echo "No webhook files found to move (this is expected if your project has no webhooks)"

# If main.go is in root
mkdir -p cmd
mv main.go cmd/
```

### 1.3 Update package declarations

After moving files, update package declarations:

**Controllers:** Change `package controllers` → `package controller` in all `*_controller.go` and `*_controller_test.go` files.

**Webhooks:** Keep version as package name (e.g., `package v1` stays `package v1` in `internal/webhook/v1/`).

### 1.4 Update import paths

Find and update all imports:

```bash
grep -r "pkg/controllers\|/controllers\"" --include="*.go"
```

In each file found, update:
- Imports: `<module>/controllers` or `<module>/pkg/controllers` → `<module>/internal/controller`
- References: `controllers.TypeName` → `controller.TypeName`

### 1.5 Update Dockerfile (if needed)

If your Dockerfile has explicit COPY statements for moved paths, update them to reflect the new structure, or simplify to `COPY . .` and use `.dockerignore` to exclude unnecessary files.

### 1.6 Verify and commit

Build and test the reorganized project:

```bash
make generate manifests
make build && make test
```

If successful, commit the layout changes. Your project now uses the new layout. Proceed to Phase 2.

## Phase 2: Migrate to Latest Version

### Step 1: Prepare Your Current Project

### 1.1 Create a migration branch

Create a branch from your current codebase:

```bash
git checkout -b migration
```

### 1.2 Create a backup

```bash
mkdir ../migration-backup
cp -r . ../migration-backup/
```

### 1.3 Clean your project directory

Remove all files except `.git`:

```bash
find . -not -path './.git*' -not -name '.' -not -name '..' -delete
```

## Step 2: Initialize the New Project

**About the PROJECT file:** From v3.0.0+, the `PROJECT` file tracks all scaffolding metadata. If you have one and used CLI for all resources, try `kubebuilder alpha generate` first. Otherwise, follow the manual steps below to identify and re-scaffold all resources.

### 2.1 Identify your module and domain

Identify the information you'll need for initialization from your backup.

<aside class="note">

<h1>Skip if Using AI Migration Helper</h1>

If you used [Step 2: Discovery CLI Commands](./discovery-commands.md) AI migration helper, you already have a complete script with all commands. Execute it and skip to [Step 4: Port Your Custom Code](#step-4-port-your-custom-code).

</aside>

**Module path** - Check your backup's `go.mod` file:

```bash
cat ../migration-backup/go.mod
```

Look for the module line:

```go
module tutorial.kubebuilder.io/migration-project
```

**Domain** - Check your backup's `PROJECT` file:

```bash
cat ../migration-backup/PROJECT
```

Look for the domain line:

```yaml
domain: tutorial.kubebuilder.io
```

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

Multi-group projects organize APIs into different groups, with each group in its own directory.
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

You can also check your backup's `PROJECT` file for `multigroup: true`.

**If your project uses multi-group layout**, enable it before creating APIs:

```bash
kubebuilder edit --multigroup=true
```

<aside class="warning">
<h1>Important</h1>

This must be done before creating any APIs to ensure they're scaffolded in the multi-group structure.

</aside>

When following this guide, you'll get the new layout automatically since you're creating a fresh project with the latest version and porting your code into it.

## Step 3: Re-scaffold APIs and Controllers

For each API resource in your original project, re-scaffold them in the new project.

### 3.1 Identify all your APIs

Review your backup project (`../migration-backup/`) to identify all APIs. **It's recommended to check the backup directory
regardless of whether you have a `PROJECT` file**, as not all resources may have been created using the CLI.

**Check the directory structure** in your backup to ensure you don't miss any manually created resources:

- Look in the `api/` directory (or `apis/` for projects generated with older Kubebuilder versions) for `*_types.go` files:
  - Single-group: `api/v1/cronjob_types.go` - extract: version `v1`, kind `CronJob`, group from imports
  - Multi-group: `api/batch/v1/cronjob_types.go` - extract: group `batch`, version `v1`, kind `CronJob`

- Check for controllers in these locations:
  - **Current:** `internal/controller/cronjob_controller.go` or `internal/controller/<group>/cronjob_controller.go`
  - **Legacy:** `controllers/cronjob_controller.go` or `pkg/controllers/cronjob_controller.go`

**If you used the CLI to create all APIs from Kubebuilder `v3.0.0+` you should have them in the `PROJECT` file** under the `resources` section, such as:

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
   - Legacy location (v3 and earlier): `api/v1/<kind>_webhook.go` or `api/<group>/<version>/<kind>_webhook.go`
   - Current location (single-group): `internal/webhook/<version>/<kind>_webhook.go`
   - Current location (multi-group): `internal/webhook/<group>/<version>/<kind>_webhook.go`

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

Create the conversion webhook for the **hub** version, with spoke versions specified using the `--spoke` flag.

**Note:** In the examples below, we use `v1` as the hub for illustration. Choose the version in your project that should be the central conversion point—typically your most feature-complete and stable storage version, not necessarily the oldest or newest.

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion --spoke v2
```

This command:
- Creates conversion webhook for `v1` as the **hub** version
- Configures `v2` as a **spoke** that converts to/from the hub `v1`
- Generates `*_conversion.go` files with conversion method stubs

**For multiple spokes**, specify them as a comma-separated list:

```bash
kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion --spoke v2,v1alpha1
```

This sets up `v1` as the **hub** with both `v2` and `v1alpha1` as **spokes**.

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
make manifests && make build
```

## Step 4: Port Your Custom Code

<aside class="note">

<h1>Using AI Migration Helper</h1>

If you used [Step 3: Port Custom Code](./port-code.md) AI migration helper, your code is already ported.

You may skip to [Step 5](#step-5-test-and-verify). However, it's still recommended to at least review the following steps and do manual validation to ensure all code was properly ported.

</aside>

Manually port your custom business logic and configurations from the backup to the new project.

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

**If your APIs reference a parent package** (e.g., `scheduling.GroupName`), port it:

```bash
mkdir -p api/<group>/
cp ../migration-backup/apis/<group>/groupversion_info.go api/<group>/
```

After porting API definitions, regenerate and verify:

```bash
make manifests  # Generate CRD manifests from your types
make generate   # Generate DeepCopy methods
```

This ensures your API types and CRD manifests are properly generated before moving forward.

### 4.2 Port controller logic

**Files to compare:**

- **Current single-group:** `internal/controller/<kind>_controller.go`
- **Current multi-group:** `internal/controller/<group>/<kind>_controller.go`

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

**Current webhook location:**
- Single-group: `internal/webhook/<version>/<kind>_webhook.go`
- Multi-group: `internal/webhook/<group>/<version>/<kind>_webhook.go`

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

### 4.4 Port main.go customizations (if any)

**File:** `cmd/main.go`

Most projects don't need to customize `main.go` as Kubebuilder handles all the standard setup automatically
(registering APIs, setting up controllers and webhooks, manager initialization, metrics, etc.).

Only port customizations that are not part of the standard scaffold. Compare your backup `main.go` with the
new scaffolded one to identify any custom logic you added.

### 4.5 Configure Kustomize manifests

The `config/` directory contains Kustomize manifests for deploying your operator. Compare with your backup to ensure all configurations are properly set up.

**Review and update these directories:**

1. **`config/default/kustomization.yaml`** - Main kustomization file
   - Ensure webhook configurations are enabled if you have webhooks (uncomment webhook-related patches)
   - Ensure cert-manager is enabled if using webhooks (uncomment certmanager resources)
   - Enable or disable metrics endpoint based on your original configuration
   - Review namespace and name prefix settings

2. **`config/manager/`** - Controller manager deployment
   - Usually no changes are needed unless you have customizations. In that case, compare resource limits and requests with your backup and check environment variables

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

### 4.6 Port additional customizations

Port any additional packages, dependencies, and customizations from your backup:

**Additional packages** (e.g., `pkg/util`):

```bash
cp -r ../migration-backup/pkg/<package-name> pkg/
# Update import paths (works on both macOS and Linux)
find pkg/ -name "*.go" -exec sed -i.bak 's|<module>/apis/|<module>/api/|g' {} \;
find pkg/ -name "*.go.bak" -delete
```

For dependencies, run `go mod tidy` or copy `go.mod`/`go.sum` from backup for complex projects.

Check for additional customizations (Makefile, Dockerfile, test files). Use diff tools to compare with backup and identify missed files.


<aside class="note">
<h1>Using diff tools</h1>

Use your IDE's diff tools to compare the current directory with your backup (`../migration-backup/`) or use git to compare
your current branch with your main branch. This helps identify any files you may have missed.

</aside>

After porting all customizations, verify everything builds:

```bash
make all
```

## Step 5: Test and Verify

Compare against the backup to ensure all customizations were correctly ported, such as:

```bash
diff -r --brief ../migration-backup/ . | grep "Only in ../migration-backup"
```

Run tests and verify functionality:

```bash
make test && make lint-fix
```

Deploy to a test cluster (e.g. [kind][kind-doc]) and verify the changes (i.e. validate expected behavior, run regression checks, confirm the full CI pipeline still passes, and execute the e2e tests).

<aside class="note">

<h1>If You Have a Helm Chart</h1>

If you had a Helm chart to distribute your project, you may want to regenerate it with the [helm/v2-alpha plugin](../plugins/available/helm-v2-alpha.md), then apply your customizations.

```bash
kubebuilder edit --plugins=helm/v2-alpha
```

Compare your backup's `chart/values.yaml` and custom templates with the newly generated chart, and apply your customizations and ensure that all is still working
as before.

</aside>

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
[kind-doc]: ../reference/kind.md
[autoupdate-plugin]: ../plugins/available/autoupdate-v1-alpha.md
[alpha-update]: ../reference/commands/alpha_update.md