# plugin-record

Recording for OpenCharly — the `record:` check verb.

`record:` manages recording sessions inside a running deployment: `list`, `start`,
`stop`, `cmd`, and `gif`. It is served **out-of-process** — charly's loader
fetches this repo, host-builds the provider binary, and serves it over go-plugin
gRPC, so the asciinema/wf-recorder/pixelflux recording driver lives here, out of
charly's core check surface.

It is EXEC-based: the host attaches its live `DeployExecutor` over the reverse
channel and the plugin dials back through the SDK to run the recording commands
in-container via tmux and pull the produced `.cast`/`.mp4` artifact back to the
host. It owns no podman/SSH machinery.

- `record: start` verifies the recorder process is alive after spawn (no
  false-positive starts); `record: stop` pulls the artifact host-side before the
  artifact validators run.
- `record: gif` renders a **stopped** terminal recording (`.cast`) to an animated
  GIF with `agg` (installed by the `asciinema` candy) and pulls the `.gif` to the
  artifact path.
- Desktop (wf-recorder) recording needs the compositor session env — author
  `record_env:` (e.g. `{XDG_RUNTIME_DIR: /run/user/1000, WAYLAND_DISPLAY:
  wayland-1}`) to override the container defaults (`/tmp` + `wayland-0`).

## What it provides

| Capability | Surface |
|---|---|
| `verb:record` | the `record:` check verb — `list`, `start`, `stop`, `cmd`, `gif` |

## How to use it

Compose the plugin candy in a box or check bed's `candy:` list:

```yaml
- '@github.com/opencharly/plugin-record/candy/plugin-record:<tag>'
```

Then author the verb in a plan:

```yaml
- check: no recording sessions are active
  record: list
  context: [runtime]
```

The R10 consumer is a desktop pod bed whose check composes this plugin (the
`sway-browser-vnc` bed's `record: start`).

## Layout

- `candy/plugin-record/` — the plugin module: `plugin.go` / `provider.go` (the
  verb + provider), `methods.go` (the session methods), `schema/record.cue`
  (the self-contained `#RecordInput`), `params/cue_types_gen.go`,
  `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-check:record` — the `record:` verb (terminal/desktop
  recording), served out-of-process by this candy.
- `/charly-internals:plugin` — the out-of-process plugin model.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
