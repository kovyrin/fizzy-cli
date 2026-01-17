# TOON Output Support Plan for Fizzy CLI

## Summary

Add TOON (Token-Oriented Object Notation) as an alternative output format for all commands that emit structured data, selectable via config and a `--format` flag. Default remains JSON.

## Context / Current State

- Structured output is centralized in `internal/response/response.go`, which always emits JSON via `encoding/json`.
- Output formatting toggles only `--pretty` (JSON indentation) via `response.SetPrettyPrint` in `internal/commands/root.go`.
- Config supports `token`, `account`, `api_url`, `board` in `internal/config/config.go`.
- Commands that emit structured output go through `printSuccess*` / `exitWithError` in `internal/commands/root.go`.
- Interactive commands (`setup`, `skill`, etc.) print plain text directly and should remain unchanged.

## Requirements

- Add TOON output alongside JSON.
- Configurable via config file setting.
- `--format toon` should enable TOON for any command that emits structured output.

## Proposed Design

### 1) Config format

- Add a new field to `config.Config`, e.g.:

  - ```go
    OutputFormat string `yaml:"output"`
    ```

  - Accepted values: `"json"` (default), `"toon"`.
- Config precedence remains: flags > env > local config > global config > defaults.
- Add env override: `FIZZY_OUTPUT` accepts `json` or `toon`.

Example config:

```yaml
output: toon
```

### 2) CLI flag

- Add a persistent string flag on the root command:
  - `--format` (default empty, allowed: `json`, `toon`)
- In `PersistentPreRun`, if `--format` is set, override config output.
- This makes TOON available to all commands without per-command wiring.

### 3) Response formatting layer

- Extend `internal/response` to support multiple output formats:
  - Add `outputFormat` state + `SetOutputFormat(format string)`.
  - Default to JSON.
  - `Print()` switches on format:
    - JSON: current encoder path (with `SetIndent` when `--pretty`).
    - TOON: `toon.Marshal(...)` via `github.com/toon-format/toon-go` with options (see below).
- Add TOON struct tags to response types so field names remain `success`, `data`, `error`, etc.
  - `Response`, `ErrorDetail`, `Pagination` should all include `toon` tags mirroring JSON tags and `omitempty`.

TOON encoder options (initial suggestion):

- Use defaults (indent=2, comma delimiter).
- Do not enable length markers (keep output as compact as possible).
- `--pretty` will be ignored for TOON output (TOON already has indentation).

### 4) Documentation

- Update `README.md` to mention TOON output and the new config key / flag.
- Consider adding a short example TOON response in the Output Format section.

### 5) Tests

- Add tests in `internal/response/response_test.go` to validate TOON output:
  - Ensure format toggles to TOON and emits expected keys (lowercase, not struct field names).
  - Ensure HTML in data remains unescaped (if relevant in TOON).
- Add config tests to ensure `output` is loaded from global/local config.

## Implementation Steps

1. **Config layer**
   - Extend `config.Config` with `OutputFormat`.
   - Update `Load()` and `LoadGlobal()` to populate it.
   - Add env var override: `FIZZY_OUTPUT`.
2. **Root flags / wiring**
   - Add `cfgFormat` in `internal/commands/root.go`.
   - Add persistent `--format` flag.
   - In `PersistentPreRun`, compute effective format (config + flag) and call `response.SetOutputFormat`.
3. **Response package**
   - Add dependency on `github.com/toon-format/toon-go`.
   - Add output format handling + TOON marshal path.
   - Add `toon` tags to response structs.
   - Add tests for TOON output.
4. **Docs**
   - Update `README.md` with config + flag usage.
