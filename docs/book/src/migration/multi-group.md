# Single Group to Multi-Group

Kubebuilder scaffolds single-group projects by default to keep things simple, as most projects don't require multiple API groups. However, you can convert an existing single-group project to use multi-group layout when needed. This reorganizes your APIs and controllers into group-specific directories.

See the [design doc][multigroup-design] for the rationale behind this design decision.

<aside class="note">

<h4>What's a Multi-Group Project?</h4>

Multi-group layout is useful when you're building APIs for different purposes or domains. For example, you might have:
- A `batch` group for job-related resources (CronJob, Job)
- An `apps` group for application resources (Deployment, StatefulSet)
- A `crew` group for team management resources (Captain, Sailor)

Each group gets its own directory, keeping things organized as your project grows.

See [Groups and Versions and Kinds, oh my!][gvks] to better understand API groups.

</aside>

<aside class="note">

<h4>AI-Assisted Migration</h4>

This migration involves repetitive file moving and import path updates. If you're using an AI coding assistant, see the [AI-Assisted Migration](#ai-assisted-migration) section for ready-to-use instructions.

</aside>

## Understanding the Layouts

Here's what changes when you go from single-group to multi-group:

**Single-group layout (default):**
```
api/<version>/*_types.go                  All your CRD schemas in one place
internal/controller/*                     All your controllers together
internal/webhook/<version>/*              Webhooks organized by version (if you have any)
```

**Multi-group layout:**
```
api/<group>/<version>/*_types.go          CRD schemas organized by group
internal/controller/<group>/*             Controllers organized by group
internal/webhook/<group>/<version>/*      Webhooks organized by group and version (if you have any)
```

You can tell which layout you're using by checking your `PROJECT` file for `multigroup: true`.

## Migration Steps

The following steps migrate the [CronJob example][cronjob-tutorial] from single-group to multi-group layout.

### Step 1: Enable multi-group mode

First, tell Kubebuilder you want to use multi-group layout:

```bash
kubebuilder edit --multigroup=true
```

This command updates your `PROJECT` file by adding `multigroup: true`. After this change:
- **New APIs** you create will automatically use the multi-group structure (`api/<group>/<version>/`)
- **Existing APIs** remain in their current location and must be migrated manually (steps 3-9 below)

<aside class="note">
<h4>What this command changes</h4>

The command adds or updates this line in your PROJECT file:

```yaml
multigroup: true
```

This setting tells Kubebuilder to use group-based directories for all future scaffolding operations.

</aside>

### Step 2: Identify your group name

Check `api/v1/groupversion_info.go` to find your group name:

```go
// +groupName=batch.tutorial.kubebuilder.io
package v1
```

The group name is the first part before the dot (`batch` in this example).

### Step 3: Move your APIs

Create a directory for your group and move your version directories:

```bash
mkdir -p api/batch
mv api/v1 api/batch/
```

If you have multiple versions (like `v1`, `v2`, etc.), move them all:

```bash
mv api/v2 api/batch/
```

### Step 4: Move your controllers

Create a group directory and move all controller files:

```bash
mkdir -p internal/controller/batch
mv internal/controller/*.go internal/controller/batch/
```

This will move all your controller files, including `suite_test.go`, into the group directory. Each group needs its own test suite.

### Step 5: Move your webhooks (if you have any)

If your project has webhooks (check for an `internal/webhook/` directory), add the group directory:

```bash
mkdir -p internal/webhook/batch
mv internal/webhook/v1 internal/webhook/batch/
mv internal/webhook/v2 internal/webhook/batch/  # if v2 exists
```

If you don't have webhooks, skip this step.

### Step 6: Update import paths

Update all import statements to point to the new locations.

**What used to look like this:**
```go
import (
    batchv1 "tutorial.kubebuilder.io/project/api/v1"
    "tutorial.kubebuilder.io/project/internal/controller"
)
```

**Should now look like this:**
```go
import (
    batchv1 "tutorial.kubebuilder.io/project/api/batch/v1"
    batchcontroller "tutorial.kubebuilder.io/project/internal/controller/batch"
)
```

**If you have webhooks, you'll also need to update those imports:**
```go
// Before
webhookv1 "tutorial.kubebuilder.io/project/internal/webhook/v1"

// After
webhookbatchv1 "tutorial.kubebuilder.io/project/internal/webhook/batch/v1"
```

Files to check and update:
- `cmd/main.go`
- `internal/controller/batch/*.go`
- `internal/webhook/batch/v1/*.go` (if you have webhooks)
- `api/batch/v1/*_test.go`

Tip: Use your IDE's "Find and Replace" feature across the project.

### Step 7: Update the PROJECT file

The `kubebuilder edit --multigroup=true` command sets `multigroup: true` in your PROJECT file but doesn't update paths for existing APIs. You need to manually update the `path` field for each resource.

**Verify your PROJECT file has these changes:**

1. **Check that `multigroup: true` is set** (at the top level):

```yaml
layout:
- go.kubebuilder.io/v4
multigroup: true  # Must be true
projectName: project
```

2. **Update the `path` field for each resource**:

**Before:**
```yaml
resources:
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  group: batch
  kind: CronJob
  path: tutorial.kubebuilder.io/project/api/v1  # Old path
  version: v1
```

**After:**
```yaml
resources:
- api:
    crdVersion: v1
    namespaced: true
  controller: true
  group: batch
  kind: CronJob
  path: tutorial.kubebuilder.io/project/api/batch/v1  # New path with group
  version: v1
```

Repeat this for **all resources** in your PROJECT file.

### Step 8: Update test suite CRD paths

Update the CRD directory path in test suites. Since files moved one level deeper, add one more `".."` to the path.

**In `internal/controller/batch/suite_test.go`:**

**Before (was at `internal/controller/suite_test.go`):**
```go
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
}
```

**After (now at `internal/controller/batch/suite_test.go`):**
```go
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
}
```

**If you have webhooks, update `internal/webhook/batch/v1/webhook_suite_test.go`:**

**Before (was at `internal/webhook/v1/webhook_suite_test.go`):**
```go
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
}
```

**After (now at `internal/webhook/batch/v1/webhook_suite_test.go`):**
```go
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "..", "config", "crd", "bases")},
}
```

### Step 9: Verify the migration

Run the following commands to verify everything works:

```bash
make manifests      # Regenerate CRDs and RBAC
make generate       # Regenerate code
make test           # Run tests
make build          # Build the project
```

## AI-Assisted Migration

If you're using an AI coding assistant (Cursor, GitHub Copilot, etc.), you can automate most of the migration steps.

<aside class="note">

<h4>AI Migration Instructions</h4>

**Prerequisites:**
1. First, identify the API group name from `api/v1/groupversion_info.go` (look for `+groupName=<group>.<domain>`)
2. Get your module path from `go.mod` (first line: `module <repo>`)

**Instructions to provide to your AI assistant:**

Give your AI assistant these instructions, replacing the values in the first two lines:

```
I need to migrate this Kubebuilder project to multi-group layout.

Project details:
- Group name: batch
- Module path: tutorial.kubebuilder.io/project

Context:
Kubebuilder projects have three main code locations:
- api/<version>/ - Contains CRD type definitions (*_types.go files)
- internal/controller/ - Contains reconcilers (*_controller.go files)
- internal/webhook/<version>/ - Contains webhooks (*_webhook.go files) [if present]

Multi-group layout reorganizes these into group-specific directories:
- api/<group>/<version>/ - Types organized by API group
- internal/controller/<group>/ - Controllers organized by group
- internal/webhook/<group>/<version>/ - Webhooks organized by group

This keeps code organized as projects grow to support multiple API groups.

References:
- Kubebuilder Book: https://book.kubebuilder.io

Steps to execute:

1. Enable multi-group mode:
   Run: kubebuilder edit --multigroup=true

2. Move API files:
   mkdir -p api/batch
   mv api/v1 api/batch/
   mv api/v2 api/batch/  # if v2 exists

3. Move controller files:
   mkdir -p internal/controller/batch
   mv internal/controller/*.go internal/controller/batch/

4. Move webhook version directories (ONLY if internal/webhook/ exists):
   # Skip this step entirely if you don't have an internal/webhook/ directory
   if [ -d "internal/webhook" ]; then
     mkdir -p internal/webhook/batch
     mv internal/webhook/v1 internal/webhook/batch/ 2>/dev/null || true
     mv internal/webhook/v2 internal/webhook/batch/ 2>/dev/null || true
   fi

5. Update all import paths:
   - In cmd/main.go, internal/controller/batch/*.go, api/batch/*/*.go (and webhook files if they exist)
   - Replace: tutorial.kubebuilder.io/project/api/v1 -> tutorial.kubebuilder.io/project/api/batch/v1
   - Replace: tutorial.kubebuilder.io/project/api/v2 -> tutorial.kubebuilder.io/project/api/batch/v2
   - Replace: tutorial.kubebuilder.io/project/internal/controller -> tutorial.kubebuilder.io/project/internal/controller/batch
   - If you have webhooks, also replace:
     tutorial.kubebuilder.io/project/internal/webhook/v1 -> tutorial.kubebuilder.io/project/internal/webhook/batch/v1
     tutorial.kubebuilder.io/project/internal/webhook/v2 -> tutorial.kubebuilder.io/project/internal/webhook/batch/v2

6. Update PROJECT file:
   - Verify multigroup: true is set (should be set by step 1)
   - For each resource entry, update the path field
   - From: tutorial.kubebuilder.io/project/api/v1
   - To: tutorial.kubebuilder.io/project/api/batch/v1
   - Example:
     ```yaml
     layout:
     - go.kubebuilder.io/v4
     multigroup: true  # This must be true
     resources:
     - api:
         crdVersion: v1
         namespaced: true
       controller: true
       domain: tutorial.kubebuilder.io
       group: batch
       kind: CronJob
       path: tutorial.kubebuilder.io/project/api/batch/v1  # Updated path
       version: v1
     ```

7. Fix test suite CRD paths (add one more ".."):
   - In internal/controller/batch/suite_test.go:
     From: filepath.Join("..", "..", "config", "crd", "bases")
     To: filepath.Join("..", "..", "..", "config", "crd", "bases")
   - If you have webhooks, also in internal/webhook/batch/v1/webhook_suite_test.go:
     From: filepath.Join("..", "..", "..", "config", "crd", "bases")
     To: filepath.Join("..", "..", "..", "..", "config", "crd", "bases")

8. Verify:
   Run: make manifests && make generate && make test
```

**After AI completes:**
- Review the changes carefully
- Verify import paths are correct
- Check PROJECT file paths
- Run `make test` to catch any issues

</aside>

[gvks]: /cronjob-tutorial/gvks.md "Groups and Versions and Kinds, oh my!"
[cronjob-tutorial]: /cronjob-tutorial/cronjob-tutorial.md "Tutorial: Building CronJob"
[multigroup-design]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/simplified-scaffolding.md
