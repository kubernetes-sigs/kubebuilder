# License Header

## What is it?

Kubebuilder creates a file at `hack/boilerplate.go.txt` that contains your license header. This header is automatically added to all generated Go files.

The boilerplate file is used by:
- Kubebuilder when creating new files (`create api`, `create webhook`, etc.)
- `controller-gen` when generating code (DeepCopy methods, etc.)
- `make generate` and `make manifests` commands

By default, Kubebuilder uses the Apache 2.0 license.

## How to Customize

### Option 1: Use Built-in Licenses

During project initialization, use the `--license` flag:

```bash
# Use Apache 2.0 (default)
kubebuilder init --domain example.com --license apache2 --owner "Your Name"

# Use copyright only (no license text)
kubebuilder init --domain example.com --license copyright --owner "Your Company"

# Use no license header at all
kubebuilder init --domain example.com --license none
```

Available license values:
- `apache2` - Apache 2.0 License (default)
- `copyright` - Copyright notice only (no license text)
- `none` - No license header

### Option 2: Use a Custom License File

Provide your own license header from a file. Kubebuilder will read the content from your file and copy it to `hack/boilerplate.go.txt`:

```bash
# During initialization
kubebuilder init --domain example.com --license-file ./my-header.txt

# After initialization
kubebuilder edit --license-file ./my-header.txt
```

**How it works:**
1. Kubebuilder reads the content from your custom license file
2. The content is copied to `hack/boilerplate.go.txt` (the standard location)
3. All tools (`controller-gen`, `make generate`, etc.) use `hack/boilerplate.go.txt`
4. Your original license file is not referenced again - `hack/boilerplate.go.txt` is the source of truth

**Important:** The `--license-file` flag **overrides** the `--license` flag, including `--license none`. If you provide `--license-file`, a boilerplate will always be created.

Use this when you need:
- A license not available in the built-in templates
- A specific company license format
- Custom copyright notices

### Option 3: Edit the File Directly

After initialization, you can edit `hack/boilerplate.go.txt` directly:

```bash
vim hack/boilerplate.go.txt
make manifests  # Apply to generated files
```

## Custom License File Format

Your custom license file must include Go comment delimiters (`/*` and `*/`).

**Example** (`my-license-header.txt`):

```go
/*
Copyright 2026 Your Company Name.

Licensed under the MIT License.
See LICENSE file in the project root for full license text.
*/
```

**File Validation:**
- The license file **must exist** before running `kubebuilder init` or `kubebuilder edit`
- If the file doesn't exist, the command will fail immediately **before** creating any files
- This prevents leaving your project in a broken state
- The content is copied exactly as-is (byte-for-byte) - no modifications are made

## Regenerating Your Project

The `kubebuilder alpha generate` command automatically preserves your license header:

```bash
kubebuilder alpha generate
```

What happens:
- **If `hack/boilerplate.go.txt` exists**: Your custom license is preserved and restored after regeneration
- **If it doesn't exist**: Project is regenerated with `--license none`

This ensures your custom license headers remain when upgrading to newer Kubebuilder versions.

## How It's Used

**`hack/boilerplate.go.txt` is the single source of truth** for license headers in your project.

The file is automatically used by:

- `kubebuilder create api` and `create webhook`
- `controller-gen` (called by `make generate` and `make manifests`)
- `make generate` - adds headers to generated Go code
- `make manifests` - adds headers to generated YAML files

**Important Notes:**
- The Makefile already configures `controller-gen` to use `hack/boilerplate.go.txt`
- The Makefile configuration is **never changed** regardless of which option you use
- All tools always reference `hack/boilerplate.go.txt` - no custom file paths are used
- If you used `--license-file`, your original file is **not** referenced again after initialization

## Examples

### Example 1: Apache 2.0 License (Default)

Use the built-in Apache 2.0 license:

```bash
kubebuilder init --domain example.com --license apache2 --owner "Your Company"
```

This generates:

```go
/*
Copyright 2026 Your Company.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
```

### Example 2: Copyright Only

Use the built-in `copyright` license for just a copyright notice:

```bash
kubebuilder init --domain example.com --license copyright --owner "Your Company"
```

This generates:

```go
/*
Copyright 2026 Your Company.
*/
```

### Example 3: Custom License (MIT)

For licenses not built-in, create a custom file `mit-header.txt`:

```go
/*
Copyright 2026 Your Name.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
```

Then initialize your project:

```bash
kubebuilder init --domain example.com --license-file ./mit-header.txt
```

**What happens:**
1. Kubebuilder reads the content from `./mit-header.txt`
2. Creates `hack/boilerplate.go.txt` with that content
3. The file `./mit-header.txt` is no longer referenced - you can delete it or keep it for your records
4. `hack/boilerplate.go.txt` is now the source of truth for all generated files

## Related

- [Good Practices](./good-practices.md) - Best practices for project structure
- [Generating CRDs](./generating-crd.md) - How CRDs are generated with headers
- [controller-gen CLI](./controller-gen.md) - Details on code generation
