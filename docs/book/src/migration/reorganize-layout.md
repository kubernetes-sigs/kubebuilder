# Step 1: Reorganize to New Layout (Required only for Legacy Layouts)

**If your project was built with Kubebuilder prior to v3.0.0**, you will probably need this step.

Reorganize files to match the new directory layout.

**Check if you need this step (if ANY are true, you need this):**
- Controllers are NOT in `internal/controller/`
- Webhooks are NOT in `internal/webhook/`
- Main is NOT in `cmd/`

**If ALL are already in the new layout**, skip to [Step 2](./discovery-commands.md)

<aside class="warning">

<h4>Important: Best Effort Only</h4>

These AI instructions work for projects using **standard Kubebuilder directory layout**. Projects with heavily customized structures may require manual reorganization.

Due to the variety of customizations, we **cannot guarantee** it will work perfectly for all projects.

Always validate changes carefully.

</aside>

## Instructions to provide to your AI assistant

Copy and paste these instructions to your AI assistant:

```
Reorganize Kubebuilder project files to match new directory layout.

CONTEXT:
- Project location: . (current directory - your existing project)
- Goal: Move files to new layout WITHOUT changing code or versions
- Keep project functional after reorganization

STEP 1 - Check which files need to move:
- Controllers in controllers/ or pkg/controllers/: needs move
- Controllers in internal/controller/ or internal/controller/<group>/ (multi-group): already correct
- Webhooks in api/v1/ or apis/<group>/v1/: needs move
- Webhooks in internal/webhook/v1/ or internal/webhook/<group>/v1/ (multi-group): already correct
- Main in root (main.go): needs move
- Main in cmd/ (cmd/main.go or cmd/*/main.go): already correct

STEP 2 - Reorganize file locations:

a. Move controllers if needed:
   - If controllers/ directory exists:
     mkdir -p internal/controller
     mv controllers/* internal/controller/
     rmdir controllers
   - If pkg/controllers/ directory exists:
     mkdir -p internal/controller
     mv pkg/controllers/* internal/controller/

b. Move webhooks if needed:
   - If api/v1/ or apis/v1/ contains *_webhook.go files:
     mkdir -p internal/webhook/v1
     mv api/v1/*_webhook* internal/webhook/v1/ 2>/dev/null || mv apis/v1/*_webhook* internal/webhook/v1/ 2>/dev/null || true
   - If api/<group>/v1/ or apis/<group>/v1/ contains webhooks (multi-group):
     mkdir -p internal/webhook/<group>/v1
     mv api/<group>/v1/*_webhook* internal/webhook/<group>/v1/ 2>/dev/null || mv apis/<group>/v1/*_webhook* internal/webhook/<group>/v1/ 2>/dev/null || true

c. Move main.go if needed:
   - If main.go exists in root:
     mkdir -p cmd
     mv main.go cmd/

STEP 3 - Update import paths in ALL files:

After moving files, imports will break. Fix them systematically:

a. In cmd/main.go (or cmd/*/main.go, cmd/*/*.go):
   - Find: import "your-module/controllers"
   - Replace with: import "your-module/internal/controller"
   - Find: import "your-module/pkg/controllers"
   - Replace with: import "your-module/internal/controller"
   - Find: &controllers.SomeReconciler or controllers.NewController
   - Replace with: &controller.SomeReconciler or controller.NewController
   - API imports (api/v1, apis/v1alpha1) - NO CHANGE needed

b. In internal/controller/*.go files:
   - Check package declaration is still: package controller (not controllers)
   - API imports stay same - NO CHANGE needed
   - If you had controller-to-controller imports, update paths

c. In internal/webhook/v1/*.go files:
   - Check package declaration: should be package v1
   - API imports stay same - NO CHANGE needed
   - Webhook imports in main.go may need updating

STEP 4 - Update Dockerfile (if using explicit COPY):

Check Dockerfile for explicit COPY statements. If found, update:

Old pattern:
    COPY cmd/main.go cmd/main.go
    COPY api/ api/
    COPY internal/controller/ internal/controller/

Option 1 - Simplify (recommended):
    COPY . .

Ensure .dockerignore has:
    **
    !**/*.go
    **/*_test.go
    !go.mod
    !go.sum

Option 2 - Update explicit paths:
    COPY cmd/ cmd/
    COPY api/ api/
    COPY internal/ internal/

STEP 5 - Verify reorganization:

- Run: go mod tidy
- Run: make generate
- Run: make manifests
- Run: make build
- Run: make test

If errors, fix import paths.

Success: new layout, make build succeeds, make test passes, project functional
```

## What This Does

The AI will:

1. **Move files** to new layout (controllers/ to internal/controller/, webhooks to internal/webhook/, main.go to cmd/)
2. **Fix import paths** in all files after moves
3. **Verify** the reorganized project builds and tests pass

After this step, your project uses the new layout (same code, new locations), making migration much simpler!

## Next Steps

After AI reorganizes:

1. Verify: `make build && make test` (in current project)
2. If successful, backup and proceed to [Step 2: Discovery CLI Commands](./discovery-commands.md)
3. If errors, review and fix before proceeding


