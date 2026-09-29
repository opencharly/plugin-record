# AGENTS.md — plugin-record

Standalone plugin repo for the `record` live-container check verb
(`verb:record`). The plugin is a Go module at `candy/plugin-record/` (module path
`github.com/opencharly/plugin-record/candy/plugin-record`); the root `charly.yml`
only declares `discover: candy` so the repo is a project and its candy is
scanned.

Canonical files:

- `candy/plugin-record/charly.yml` — the `plugin-record:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-record/plugin.go` / `provider.go` — the verb + provider.
- `candy/plugin-record/methods.go` — the session methods (`list`/`start`/`stop`/
  `cmd`/`gif`).
- `candy/plugin-record/schema/record.cue` — the self-contained `#RecordInput`.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, the out-of-process/EXEC shape, the
  per-plugin CUE-schema contract, placement. Load before touching the provider or
  schema.
- `/charly-check:record` — the `record:` verb this candy serves.
- `/charly-check:check` — the check orchestrator, beds and the R10 sequence.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-record/` — compile the plugin module.
- `go test ./...` in `candy/plugin-record/` — the plugin's Go tests
  (`methods_test.go`, `session_test.go`).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- R10 witness: a desktop pod bed whose check composes this plugin (the
  `sway-browser-vnc` bed's `record: start`).

## Modify this repo

- Edit the `plugin-record:` candy entity, the Go source, and `schema/record.cue`
  **together** — the schema is the single source for the `params/` struct, so a
  field change not mirrored in the schema desyncs the generated types.
- The plugin is **out-of-process** and EXEC-based; it dials the host executor
  back over the reverse channel rather than owning podman/SSH.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
