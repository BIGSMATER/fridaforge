# Research Notes: FridaForge CLI 命令行工具

**Feature**: 001-fridaforge-cli
**Date**: 2026-05-09

## Technology Decisions

### Decision 1: CLI Framework — cobra

**Decision**: Use `github.com/spf13/cobra` for CLI command routing.

**Rationale**:
- Cobra is the de facto standard CLI framework in the Go ecosystem (used by kubectl, hugo, GitHub CLI).
- Provides built-in help text generation (`--help`), subcommand nesting, and persistent pre-run hooks.
- The PersistentPreRun hook enables the constitution-required ethical disclaimer check on first run.
- Aligns with the project's learning goal of understanding Go CLI patterns.

**Alternatives considered**:
- `flag` (stdlib): Too primitive for subcommand hierarchy; lacks help generation.
- `urfave/cli`: Viable but less widespread in Go tooling; cobra has richer ecosystem integrations (viper, completion).

### Decision 2: Configuration Management — viper + yaml.v3

**Decision**: Use `github.com/spf13/viper` for config file path resolution and `gopkg.in/yaml.v3` for direct YAML unmarshaling.

**Rationale**:
- Viper handles file path resolution, environment variable binding, and config merging (useful for future global config like `~/.fridaforge.yaml`).
- yaml.v3 provides direct `yaml.Unmarshal` with struct tags — simpler and more explicit than viper's `Unmarshal` for custom validation.
- The combination: viper for "where is the file", yaml.v3 for "what's in the file".

**Alternatives considered**:
- yaml.v2: Legacy; v3 adds round-trip preservation and better error messages.
- viper-only unmarshaling: Less flexible for custom validation logic (need to inspect raw YAML nodes for precise line numbers).

### Decision 3: Device Manager Architecture — Interface + Stub

**Decision**: Define `DeviceManager` interface in `pkg/device/` with a stub implementation for M1. Real Frida-based implementation in M2.

**Rationale**:
- Enables the CLI code to be written and tested without actual Frida dependencies.
- The interface contract is defined early; M2 can simply swap the implementation.
- Stub can return predefined devices for testing and demo purposes.
- Follows Go convention: accept interfaces, return structs.

**Interface design**:
```go
type DeviceManager interface {
    ListDevices(ctx context.Context) ([]Device, error)
}
type Device struct {
    ID          string
    Name        string
    ConnectType string // "usb", "network", "emulator"
}
```

**Alternatives considered**:
- Direct Frida integration in M1: Violates the "no actual Frida connection" constraint.
- No abstraction (hardcoded in CLI): Makes M2 integration harder and prevents testing.

### Decision 4: Testing — Table-Driven with Standard Library

**Decision**: Use Go standard library `testing` with table-driven test pattern. Test files co-located with source (`*_test.go`).

**Rationale**:
- Table-driven tests are the Go community standard and constitution-mandated (2.5).
- No external test framework dependency reduces cognitive load for a Go-learning project.
- Co-located tests improve discoverability and follow Go convention.

**Alternatives considered**:
- testify: Popular but adds dependency; constitution doesn't mandate it; stdlib sufficient for M1's validator and parser tests.

### Decision 5: Error Model — Typed Validation Errors

**Decision**: Define structured validation error types in `pkg/spec/errors.go` rather than returning raw strings.

**Rationale**:
- FR-006 requires errors to include field path and line number — structured errors make this testable.
- Typed errors enable downstream consumers to programmatically distinguish error categories.
- Implements `error` interface with descriptive `Error()` string.

**Error types**:
- `ValidationError`: multiple field-level errors
- `FieldError`: single field error with path, line, message

### Decision 6: Ethical Disclaimer — First-Run Check

**Decision**: Use a marker file (e.g., `~/.fridaforge/agreed`) to track whether the user has accepted the ethical disclaimer. Root command's `PersistentPreRun` checks this and prompts if absent.

**Rationale**:
- Constitution 3.4 requires ALL CLI commands to show disclaimer on first run.
- Marker file approach is standard (used by `git`, `cocoa pods`, `npm` for similar EULA flows).
- Avoids requiring `AGREE` on every single invocation.
- `.fridaforge/` directory also serves as future config home.

### Decision 7: YAML Schema — Single App Per File

**Decision**: Each spec file targets one application. Top-level `app_package` is shared; hooks are listed in a `hooks` array.

**Schema**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload
  - class_name: com.example.Utils
    method_name: encrypt
    hook_type: replace
```

**Rationale**: See Q1 in clarifications (spec.md Clarifications section). Simplifies validation (one app context per file) and maps naturally to Frida's attach model.

## Resolved Clarifications

All technical unknowns resolved by user input:
- Go 1.25, cobra, viper, yaml.v3 ✅
- Project layout: `cmd/fridaforge/`, `pkg/{config,spec,device}/` ✅
- Testing: stdlib + table-driven, co-located `*_test.go` ✅
- M1 scope: no network, no database, no Frida connection ✅

No outstanding NEEDS CLARIFICATION markers.
