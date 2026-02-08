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

Provide your own license header from a file:

```bash
# During initialization
kubebuilder init --domain example.com --license-file ./my-header.txt

# After initialization
kubebuilder edit --license-file ./my-header.txt
```

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

The `hack/boilerplate.go.txt` file is automatically used by:

- `kubebuilder create api` and `create webhook`
- `controller-gen` (called by `make generate` and `make manifests`)
- `make generate` - adds headers to generated Go code
- `make manifests` - adds headers to generated YAML files

The Makefile already configures `controller-gen` to use the boilerplate file, so you don't need to do anything manually.

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

## Related

- [Good Practices](./good-practices.md) - Best practices for project structure
- [Generating CRDs](./generating-crd.md) - How CRDs are generated with headers
- [controller-gen CLI](./controller-gen.md) - Details on code generation
