# CLI Command Contracts

**Feature**: 001-fridaforge-cli
**Date**: 2026-05-09

## Command Tree

```
fridaforge                           # Root command
├── device list                      # List connected devices
├── spec validate <file>             # Validate a HookSpec YAML file
└── help [command]                   # Show help (built-in via cobra)
```

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--help`, `-h` | bool | false | Show help for any command |
| `--version`, `-v` | bool | false | Print version information |

Note: M1 does not introduce `--verbose`/`--debug` (per clarification Q3).

---

## Command: `fridaforge device list`

List all connected Frida-capable devices.

**Usage**: `fridaforge device list`

**Flags**: None (M1)

**Exit Codes**:

| Code | Meaning |
|------|---------|
| 0 | Success: devices listed or "no devices" message shown |
| 1 | Error: Frida service unreachable |

**Stdout (success with devices)**:
```
ID              NAME                    TYPE
emulator-5554   Android Emulator 5554   emulator
R5CT1234ABCD    Samsung Galaxy S21      usb
```

**Stdout (success, no devices)**:
```
No connected devices found.
```

**Stderr (Frida unreachable)**:
```
Error: Frida service is not reachable. Please ensure frida-server is running on the target device.
```
Exit code: 1

---

## Command: `fridaforge spec validate <file>`

Validate a HookSpec YAML file for structural correctness.

**Usage**: `fridaforge spec validate <file>`

**Arguments**:

| Argument | Required | Description |
|----------|----------|-------------|
| `<file>` | yes | Path to the YAML spec file |

**Exit Codes**:

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Validation failed (file not found, YAML syntax error, missing fields, invalid hook type) |

**Stdout (success)**:
```
✓ Valid configuration: /path/to/hooks.yaml
  Target app: com.example.app
  Hooks defined: 3
```

**Stdout (validation failure)**:
```
✗ Invalid configuration: /path/to/hooks.yaml
  hooks[0].class_name: must not be empty (line 3)
  hooks[1].hook_type: unsupported value "patch" (line 8)
  Valid hook types: overload, replace
```
Exit code: 1

**Stderr (file not found)**:
```
Error: file not found: /path/to/missing.yaml
```
Exit code: 1

**Stderr (YAML syntax error)**:
```
Error: cannot parse YAML: yaml: line 5: did not find expected key
```
Exit code: 1

**Stderr (IO error)**:
```
Error: cannot read file: permission denied
```
Exit code: 1

---

## Ethical Disclaimer Flow

On first invocation of ANY command, the root command's `PersistentPreRun` executes:

1. Check for marker file `~/.fridaforge/agreed`
2. If absent: print disclaimer and prompt user to type `AGREE`
3. If user types `AGREE`: create marker file, proceed
4. If user declines: exit code 1

**Disclaimer text**:
```
╔══════════════════════════════════════════════════════════╗
║  FridaForge — Ethical Use Disclaimer                     ║
║                                                          ║
║  This tool is intended for AUTHORIZED security testing   ║
║  and educational purposes only.                          ║
║                                                          ║
║  Before using FridaForge, you must have explicit          ║
║  permission from the owner of the target application     ║
║  and device.                                             ║
║                                                          ║
║  Unauthorized use may violate applicable laws.           ║
║                                                          ║
║  Type 'AGREE' to accept and continue:                    ║
╚══════════════════════════════════════════════════════════╝
```
