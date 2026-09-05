// The `record` plugin's OWN CUE schema — the typed plugin_input for the
// `record` session-recording check verb. It is the SINGLE SOURCE for this
// plugin's params, used two ways (the same contract core `spec` and the http
// plugin use):
//
//  1. GENERATE the Go param struct — `cue exp gengotypes` (driven by the cue:gen
//     pipeline, which wraps this with `package params` + `@go(params)`) emits
//     ../params/cue_types_gen.go, so the provider decodes plugin_input into a
//     TYPED struct, never a hand-parsed map.
//  2. VALIDATE authored input AT RUNTIME — the plugin serves this source over the
//     Describe channel; the host splices it onto the base (base ++ plugin) and
//     validates every authored `record:` step's plugin_input against
//     #RecordInput.
//
// Since the schema-compaction cutover the per-verb fields LEFT core #Op: an
// authored `record: <method>` step (scalar sugar) or `record: {method: …,
// record_name: …}` (map form) desugars to the INTERNAL plugin/plugin_input
// envelope, and every record-exclusive modifier lives HERE — the former core
// #RecordMethod enum is this def's `method` field. The shared assertion
// matchers (exit_status/stdout/stderr) and the general `timeout` stay on core
// #Op, read off the step Op by the provider.
//
// SELF-CONTAINED: it references NO base def, so it compiles standalone
// (gengotypes + the load-gate compile) AND splices onto the base (base ++ plugin
// is a def-name collision check, not a base-reference resolver).
#RecordInput: {
	// method — the record method to dispatch (the former core #RecordMethod
	// enum; also the scalar-sugar primary: `record: <method>`).
	method: "list" | "start" | "stop" | "cmd" | "run" | "gif" | "session"
	// record_name — the recording session name (default "default").
	record_name?: string @go(RecordName)
	// record_mode — terminal (asciinema) / desktop (pixelflux-record or
	// wf-recorder); empty means auto-detect from the venue's tools.
	record_mode?: string @go(RecordMode)
	// record_fps — the desktop-recorder frame rate (default 30).
	record_fps?: int & >=0 @go(RecordFps,type=int)
	// record_audio — capture audio with the desktop recording.
	record_audio?: bool @go(RecordAudio)
	// record_env — extra environment for the recorder process (desktop mode needs the
	// compositor session on VM/desktop venues: record_env: {XDG_RUNTIME_DIR: /run/user/1000,
	// WAYLAND_DISPLAY: wayland-1}; container defaults are /tmp + wayland-0). Every stated
	// key overrides the default; extra keys pass through. Values are static strings.
	record_env?: { [string]: string } @go(RecordEnv,type=map[string]string)
	// text — the command line `cmd`/`run` sends into the recording's tmux session.
	text?: string
	// settle_ms — how long `run` waits after sending the text before returning
	// (default 1500). The command's output becomes part of the recording.
	settle_ms?: int & >=0 @go(SettleMs,type=int)
	// artifact — the host path `stop` copies the recording to, and `gif`
	// copies the rendered .gif to.
	artifact?: string
	// artifact_min_bytes / artifact_min_cast_events — the post-run
	// artifact-reality assertions (sdk.RunArtifactValidators).
	artifact_min_bytes?:       int & >=0 @go(ArtifactMinBytes,type=int)
	artifact_min_cast_events?: int & >=0 @go(ArtifactMinCastEvents,type=int)

	// --- gif method (agg) — render a stopped terminal recording to an animated GIF ---
	// theme — agg color theme (asciinema, dracula, monokai, github-dark,
	// solarized-dark, ...); empty uses the recording's embedded theme when present.
	theme?: string @go(Theme)
	// font_size — agg font size in px (default 16).
	font_size?: int & >=0 @go(FontSize,type=int)
	// speed — agg playback speed multiplier (default 1; >1 speeds up, <1 slows down).
	speed?: number & >0 @go(Speed,type=float64)
	// idle_time_limit — agg cap on any single inactive period, in seconds
	// (default 5), so long pauses don't bloat the GIF.
	idle_time_limit?: int & >=0 @go(IdleTimeLimit,type=int)
	// fps_cap — agg maximum GIF frame rate (default 30); lower values produce
	// smaller files at the cost of motion smoothness.
	fps_cap?: int & >=0 @go(FpsCap,type=int)
	// select — agg frame selection (e.g. "5..30", "50%", "marker:build..marker:test",
	// "12.5"); renders only part of the recording or discrete terminal states.
	select?: string @go(Select)
	// cols / rows — agg terminal size override (re-render at a different geometry).
	cols?: int & >0 @go(Cols,type=int)
	rows?: int & >0 @go(Rows,type=int)
	// no_loop — agg plays the GIF once instead of looping forever.
	no_loop?: bool @go(NoLoop)
	// last_frame_duration — agg holds the final frame for this many seconds
	// (default 3) before the GIF loops or ends.
	last_frame_duration?: int & >=0 @go(LastFrameDuration,type=int)
	// renderer — agg rendering backend: "swash" (default) or "resvg".
	renderer?: string @go(Renderer)

	// --- session method (Cutover A instrument model — venue-side session; the
	// tmux transport IS the session, already invocation-surviving) ---
	// action — the session lifecycle action (start/stop/status); empty means start
	// (the default lifecycle action, mirroring the instrument phase bracket).
	action?: "start" | "stop" | "status" @go(Action,type=string)
	// session_id — the venue-scoped session identity. When set it BECOMES the
	// record_name (tmux session names must not collide across venues); falls back
	// to record_name when empty.
	session_id?: string @go(SessionId)
	// state_dir — the run's state directory; stop writes the instrument evidence
	// row (<state_dir>/row.json) here.
	state_dir?: string @go(StateDir)
	// venue — the venue the session runs on (evidence-row provenance).
	venue?: string @go(Venue)
	// phase — the instrument phase bracket (evidence-row provenance).
	phase?: string @go(Phase)
}
