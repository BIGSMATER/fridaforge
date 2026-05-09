# Quickstart: FridaForge CLI

**Feature**: 001-fridaforge-cli
**Date**: 2026-05-09

## Prerequisites

- Go 1.25 installed
- Git

## Build

```bash
git clone <repo-url>
cd fridaforge
go build -o fridaforge ./cmd/fridaforge/
```

Or install to `$GOPATH/bin`:

```bash
go install ./cmd/fridaforge/
```

## Verify

```bash
./fridaforge --help
./fridaforge --version
```

## Basic Usage

### Validate a Hook Spec File

Create `test.yaml`:

```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload
```

Validate:

```bash
./fridaforge spec validate test.yaml
# ✓ Valid configuration: test.yaml
#   Target app: com.example.app
#   Hooks defined: 1
```

### List Devices

```bash
./fridaforge device list
```

## Running Tests

```bash
go test ./...
go test -cover ./...
go test -v ./pkg/config/
```

## Project Layout

```
cmd/fridaforge/   — CLI entry point (cobra commands)
pkg/config/       — YAML loading and validation
pkg/spec/         — Data types (HookSpec, HookTarget, etc.)
pkg/device/       — Device manager interface
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `fridaforge: command not found` | Ensure `$GOPATH/bin` is in `$PATH` or use `./fridaforge` |
| `go: module not found` | Run `go mod tidy` first |
| Build fails | Ensure Go 1.25 is installed: `go version` |
