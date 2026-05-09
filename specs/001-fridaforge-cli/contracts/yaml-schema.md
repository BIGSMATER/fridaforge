# YAML Schema Contract

**Feature**: 001-fridaforge-cli
**Date**: 2026-05-09

## HookSpec File Schema

### Version

Schema version 1 (M1). No explicit version field required; all files are treated as v1.

### Structure

```yaml
# Required: top-level application package name
app_package: <string>

# Required: list of hook targets (at least one)
hooks:
  - class_name: <string>    # Required: fully qualified Dalvik class name
    method_name: <string>   # Required: method name to hook
    hook_type: <enum>       # Required: "overload" | "replace"
```

### Constraints

| Field | Constraint |
|-------|------------|
| `app_package` | Non-empty string. Dot-separated segments. |
| `hooks` | Must be a list with ≥1 entry. |
| `hooks[].class_name` | Non-empty string. Dot-separated fully qualified Java class name. |
| `hooks[].method_name` | Non-empty string. Method name only (no parameter signatures in M1). |
| `hooks[].hook_type` | Must be exactly `"overload"` or `"replace"`. |

### Validation Order

1. File existence and readability
2. UTF-8 encoding check
3. YAML syntax parse
4. Top-level schema: `app_package` and `hooks` fields present
5. `app_package` non-empty
6. `hooks` is a list with ≥1 entries
7. Each hook entry:
   a. Required fields present: `class_name`, `method_name`, `hook_type`
   b. Required fields non-empty
   c. `hook_type` value is valid (`overload` or `replace`)
8. Warnings (non-blocking):
   a. Unknown top-level keys
   b. Unknown keys within hook entries
   c. Duplicate `class_name` + `method_name` combinations

### Complete Valid Example

```yaml
app_package: com.example.bank
hooks:
  - class_name: com.example.bank.MainActivity
    method_name: onCreate
    hook_type: overload

  - class_name: com.example.bank.crypto.AES
    method_name: encrypt
    hook_type: replace

  - class_name: com.example.bank.network.ApiClient
    method_name: sendRequest
    hook_type: overload
```

### Invalid Examples

**Missing required field**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.Foo
    # method_name missing → error
    hook_type: overload
```

**Invalid hook type**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: patch    # → error: must be "overload" or "replace"
```

**Empty hooks list**:
```yaml
app_package: com.example.app
hooks: []    # → error: at least one hook required
```

**Missing top-level field**:
```yaml
# app_package missing → error
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: overload
```

### File Naming Convention

- Extension: `.yaml` or `.yml`
- Encoding: UTF-8

### Extensibility (Future)

- M3 will add optional `params` field to `hooks[]` for parameter signatures.
- M4 may add top-level `config` block for global hook settings.
- Reserved key prefix: `x-` for user extensions (will be silently ignored).
