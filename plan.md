# Refactor Plan (Go-idiomatic Layout)

## Goals
- Align project structure with standard Go layout conventions.
- Keep project CLI-only (no exported public library).
- Rename packages to clearer domain names.
- Isolate internal implementation details under `internal/`.

## Proposed Structure
```
cmd/
  ftb-server-installer/
    main.go
internal/
  app/
  download/
  manifest/
  modloaders/
  model/
  providers/
  support/
  terminal/
  update/
```

## Current → New Package Mapping
- `main.go` → `cmd/ftb-server-installer/main.go` + `internal/app`
- `update.go` → `internal/update`
- `modloaders/` → `internal/modloaders/`
- `repos/` → `internal/providers/`
- `structs/` → `internal/model/`
- `util/` → split into:
  - `internal/download/` (download engine, checksum handling)
  - `internal/manifest/` (manifest read/write helpers)
  - `internal/terminal/` (pterm/logging/prompts)
  - `internal/support/` (generic utilities: path/OS helpers, parsing)

## Implementation Steps
1. Create `cmd/ftb-server-installer/main.go` and wire Cobra root command.
2. Replace `flag` parsing with Cobra commands + persistent flags.
3. Add `internal/app` with `Run(opts)` to own the current installer flow.
4. Move `update.go` into `internal/update` and adjust references from `app`.
5. Relocate `modloaders`, `repos`, and `structs` into their new `internal/*` packages with renamed import paths.
6. Split `util` into `download`, `manifest`, `terminal`, and `support` based on responsibility.
7. Update imports across all packages to use new paths under `internal/`.
8. Run `go test ./...` and resolve any compile issues.

## Notes
- Binary name stays `ftb-server-installer`.
- No public `pkg/` is needed because this remains CLI-only.

## Test Plan (Blackbox)

### Core Logic
- `computeUpdatedFiles` behavior across changed/removed/unchanged.
- `getLatestRelease` selection (release vs latest flag).
- `checkUpdate` downgrade/upgrade decision logic without prompts.

### Download Behavior
- Fake downloader to simulate mirror failures then success.
- Checksum verification branches (sha1/sha256/unsupported).

### Provider APIs
- Use `net/http/httptest` for FTB API responses.
- Verify parsing and error handling on non-200 responses.

### CLI Wiring
- Flags to options mapping; validate required values.

### Minimal DI Seams
- Introduce small interfaces only where necessary:
  - `Downloader` with `Do()` + checksum config.
  - `FS` wrapper for read/write/stat/remove.
- Keep production implementations in `internal/download` and `internal/support`.
