# Multiple controllers per resource

Kubebuilder supports multiple named controllers for the same API resource. This allows different reconciliation logic for the same resource type.

## Usage

### Creating the first controller

```bash
kubebuilder create api --group crew --version v1 --kind Captain \
  --resource=true \
  --controller=true \
  --controller-name=captain
```

Creates:
- API types in `api/v1/captain_types.go`
- Controller in `internal/controller/captain_controller.go` with struct `CaptainReconciler`
- Registration in `cmd/main.go`

### Adding additional controllers

```bash
kubebuilder create api --group crew --version v1 --kind Captain \
  --resource=false \
  --controller=true \
  --controller-name=captain-backup
```

Creates:
- Controller in `internal/controller/captain_backup_controller.go` with struct `CaptainBackupReconciler`
- Additional registration in `cmd/main.go`

The API is only created once. Additional controllers reference the existing API.

## Project file format

### Legacy format (still supported)

```yaml
resources:
- api:
    crdVersion: v1
  controller: true
  group: crew
  kind: Captain
  version: v1
```

### Multiple controllers format

```yaml
resources:
- api:
    crdVersion: v1
  controllers:
  - name: captain
  - name: captain-backup
  group: crew
  kind: Captain
  version: v1
```

When both formats exist on the same resource (due to manual editing), the `controllers:` array takes precedence and `controller: true` is automatically cleared.

## Controller naming

### Storage
Controller names are stored in the PROJECT file exactly as provided by the user.

### Code generation
Names are normalized for Go code:

- **File name**: Replace hyphens with underscores: `captain-backup` → `captain_backup_controller.go`
- **Struct name**: Convert to PascalCase and append Reconciler: `captain-backup` → `CaptainBackupReconciler`
- **Runtime name**: Use exact name from PROJECT: `Named("captain-backup")`
- **Multigroup**: Prefix with group name: `Named("crew-captain-backup")`

### Validation rules

1. Names must be unique within a resource
2. Names must be valid DNS labels: lowercase, alphanumeric, and hyphens only, max 63 characters
3. Different names that normalize to the same identifier are rejected (e.g., `captain-backup` and `captainbackup`)

## Controller coordination

Multiple controllers for the same resource require coordination to avoid conflicts:

- **Field ownership**: Each controller should manage different fields
- **Finalizers**: Use unique names: `{controller-name}.example.com/finalizer`
- **Status updates**: Assign different status subfields to each controller
- **Conditional logic**: Use labels or annotations to route resources to specific controllers

Kubebuilder scaffolds the controllers but does not manage coordination between them.

## Migration from legacy format

Existing projects with `controller: true` continue to work unchanged. When adding named controllers to a resource with `controller: true`, Kubebuilder automatically migrates the format:

**Before migration:**
```yaml
resources:
- api:
    crdVersion: v1
  controller: true
  group: crew
  kind: Captain
  version: v1
```

**After adding a named controller:**
```bash
kubebuilder create api --group crew --version v1 --kind Captain \
  --resource=false \
  --controller=true \
  --controller-name=captain-backup
```

**Result:**
```yaml
resources:
- api:
    crdVersion: v1
  controllers:
  - name: captain
  - name: captain-backup
  group: crew
  kind: Captain
  version: v1
```

The original unnamed controller is assigned the name `captain` (lowercase kind) and the `controller: true` flag is automatically cleared. This maintains backward compatibility while enabling multiple controllers.

## Common errors

**"duplicate controller name"**: Two controllers have the same name. Use unique names.

**"conflicts with ... both normalize to"**: Different names generate the same struct name. Choose distinct names.

**"controller with name ... already exists"**: Controller already exists for this resource. Use a different name.
