package record

import (
	"strings"
	"testing"
	"time"

	"github.com/opencharly/plugin-record/candy/plugin-record/params"
	"github.com/opencharly/sdk"
	"github.com/opencharly/spec/spec"
)

// methods_test.go covers the PLUGIN-side helpers ported out-of-process from
// charly/record.go (the deleted host-side RecordCmd): the pure path/name builders and the
// required-modifier check that moved here from the host's former in-proc live-verb contract.
// The venue-driving methods (start/stop/list/cmd) need a live executor reverse channel and
// are exercised by the R10 bed (the sway-browser-vnc `record: start`), not these unit tests.

func TestRecordSessionName(t *testing.T) {
	cases := []struct{ name, want string }{
		{"default", "record-default"},
		{"demo", "record-demo"},
		{"my-recording", "record-my-recording"},
		{"test_123", "record-test_123"},
	}
	for _, tc := range cases {
		if got := recordSessionName(tc.name); got != tc.want {
			t.Errorf("recordSessionName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRecordingFilePath(t *testing.T) {
	cases := []struct{ name, mode, want string }{
		{"demo", "terminal", "/tmp/charly-recordings/demo.cast"},
		{"demo", "desktop", "/tmp/charly-recordings/demo.mp4"},
		{"walkthrough", "desktop", "/tmp/charly-recordings/walkthrough.mp4"},
		{"test-1", "terminal", "/tmp/charly-recordings/test-1.cast"},
	}
	for _, tc := range cases {
		if got := recordingFilePath(tc.name, tc.mode); got != tc.want {
			t.Errorf("recordingFilePath(%q, %q) = %q, want %q", tc.name, tc.mode, got, tc.want)
		}
	}
}

// TestRecordName covers the CLI `-n` default (empty record_name → "default").
func TestRecordName(t *testing.T) {
	if got := recordName(&params.RecordInput{}); got != "default" {
		t.Errorf("recordName(empty) = %q, want default", got)
	}
	if got := recordName(&params.RecordInput{RecordName: "demo"}); got != "demo" {
		t.Errorf("recordName(demo) = %q, want demo", got)
	}
}

// TestRecordFps covers the CLI Fps default (0/unset → 30).
func TestRecordFps(t *testing.T) {
	if got := recordFps(&params.RecordInput{}); got != 30 {
		t.Errorf("recordFps(unset) = %d, want 30", got)
	}
	if got := recordFps(&params.RecordInput{RecordFps: 60}); got != 60 {
		t.Errorf("recordFps(60) = %d, want 60", got)
	}
}

// TestRecorderEnv covers the record_env merge contract for desktop recorders: the
// container-shaped defaults (XDG_RUNTIME_DIR=/tmp, WAYLAND_DISPLAY=wayland-0) apply
// when unset, authored values override them, extra keys pass through, and the env
// string is deterministic (sorted keys, shellquoted values).
func TestRecorderEnv(t *testing.T) {
	cases := []struct {
		name string
		in   *params.RecordInput
		want []string // substrings, in any order, of the built env prefix
		not  []string // substrings that must NOT appear
	}{
		{"empty defaults", &params.RecordInput{},
			[]string{"env ", "XDG_RUNTIME_DIR='/tmp'", "WAYLAND_DISPLAY='wayland-0'"},
			nil},
		{"authored overrides", &params.RecordInput{RecordEnv: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000", "WAYLAND_DISPLAY": "wayland-1"}},
			[]string{"XDG_RUNTIME_DIR='/run/user/1000'", "WAYLAND_DISPLAY='wayland-1'"},
			[]string{"'/tmp'", "wayland-0"}},
		{"extra keys pass through", &params.RecordInput{RecordEnv: map[string]string{"MY_REC_EXTRA": "x"}},
			[]string{"MY_REC_EXTRA='x'", "XDG_RUNTIME_DIR='/tmp'"},
			nil},
	}
	for _, tc := range cases {
		got := recorderEnv(tc.in)
		for _, want := range tc.want {
			if !strings.Contains(got, want) {
				t.Errorf("%s: recorderEnv missing %q in %q", tc.name, want, got)
			}
		}
		for _, n := range tc.not {
			if strings.Contains(got, n) {
				t.Errorf("%s: recorderEnv unexpectedly contains %q in %q", tc.name, n, got)
			}
		}
		if !strings.HasPrefix(got, "env ") {
			t.Errorf("%s: recorderEnv must start with the env prefix, got %q", tc.name, got)
		}
	}
	// determinism: same input, same output
	in := &params.RecordInput{RecordEnv: map[string]string{"B": "2", "A": "1"}}
	if recorderEnv(in) != recorderEnv(in) {
		t.Errorf("recorderEnv is not deterministic for map input")
	}
}

// TestRunSettle covers the record: run settle-duration contract: the named default
// (recordRunDefaultSettle) when settle_ms is unset, and the authored override when
// set. This is the deterministic core of recordRun (the venue-driving send/sleep
// path is exercised by the R10 bed) — it FAILS without the settle_ms behavior.
func TestRunSettle(t *testing.T) {
	defaultWant := recordRunDefaultSettle
	if got := runSettle(&params.RecordInput{}); got != defaultWant {
		t.Errorf("runSettle(unset) = %v, want default %v", got, defaultWant)
	}
	if got := runSettle(&params.RecordInput{SettleMs: 2000}); got != 2000*time.Millisecond {
		t.Errorf("runSettle(2000) = %v, want 2000ms", got)
	}
	if got := runSettle(&params.RecordInput{SettleMs: 0}); got != defaultWant {
		t.Errorf("runSettle(0) = %v, want default %v", got, defaultWant)
	}
}

// TestAggArgs covers the agg CLI option builder for record: gif: unset options are
// omitted (agg's own defaults + the recording's embedded theme apply), every set
// option maps to its agg flag with a shellquoted value, and the output is
// deterministic (fixed option order). This is the deterministic core of recordGif
// (the venue-driving agg run + GetFile pull is exercised by the R10 bed).
func TestAggArgs(t *testing.T) {
	cases := []struct {
		name string
		in   *params.RecordInput
		want string // exact expected arg prefix
	}{
		{"empty", &params.RecordInput{}, ""},
		{"theme only", &params.RecordInput{Theme: "monokai"}, "--theme 'monokai'"},
		{"font size", &params.RecordInput{FontSize: 20}, "--font-size 20"},
		{"speed int", &params.RecordInput{Speed: 2}, "--speed 2"},
		{"speed float", &params.RecordInput{Speed: 0.5}, "--speed 0.5"},
		{"idle limit", &params.RecordInput{IdleTimeLimit: 1}, "--idle-time-limit 1"},
		{"fps cap", &params.RecordInput{FpsCap: 15}, "--fps-cap 15"},
		{"select range", &params.RecordInput{Select: "5..30"}, "--select '5..30'"},
		{"select marker", &params.RecordInput{Select: "marker:build..marker:test"}, "--select 'marker:build..marker:test'"},
		{"geometry", &params.RecordInput{Cols: 100, Rows: 30}, "--cols 100 --rows 30"},
		{"no loop", &params.RecordInput{NoLoop: true}, "--no-loop"},
		{"last frame", &params.RecordInput{LastFrameDuration: 2}, "--last-frame-duration 2"},
		{"renderer", &params.RecordInput{Renderer: "resvg"}, "--renderer 'resvg'"},
		{"full", &params.RecordInput{Theme: "dracula", FontSize: 20, Speed: 2, IdleTimeLimit: 1, FpsCap: 15, Select: "5..30", Cols: 100, Rows: 30, NoLoop: true, LastFrameDuration: 2, Renderer: "resvg"},
			"--theme 'dracula' --font-size 20 --speed 2 --idle-time-limit 1 --fps-cap 15 --select '5..30' --cols 100 --rows 30 --no-loop --last-frame-duration 2 --renderer 'resvg'"},
	}
	for _, tc := range cases {
		if got := aggArgs(tc.in); got != tc.want {
			t.Errorf("%s: aggArgs = %q, want %q", tc.name, got, tc.want)
		}
	}
	// determinism: same input, same output
	in := &params.RecordInput{Theme: "monokai", Speed: 2, NoLoop: true}
	if aggArgs(in) != aggArgs(in) {
		t.Errorf("aggArgs is not deterministic")
	}
}

// TestRequireModifiers mirrors the in-tree recordMethods Required specs that moved
// here: `stop` needs an artifact, `cmd` needs the text line; list/start need nothing.
// The modifiers ride the desugared plugin_input map (op.PluginInput) since the
// schema-compaction cutover, so the fixtures set input maps, not Op fields.
func TestRequireModifiers(t *testing.T) {
	cases := []struct {
		method  string
		op      spec.Op
		wantErr string // substring; "" means no error
	}{
		{"list", spec.Op{PluginInput: map[string]any{"method": "list"}}, ""},
		{"start", spec.Op{PluginInput: map[string]any{"method": "start"}}, ""},
		{"stop", spec.Op{PluginInput: map[string]any{"method": "stop"}}, "artifact"},
		{"stop", spec.Op{PluginInput: map[string]any{"method": "stop", "artifact": "/tmp/x.cast"}}, ""},
		{"cmd", spec.Op{PluginInput: map[string]any{"method": "cmd"}}, "text"},
		{"cmd", spec.Op{PluginInput: map[string]any{"method": "cmd", "text": "echo hi"}}, ""},
		{"run", spec.Op{PluginInput: map[string]any{"method": "run"}}, "text"},
		{"run", spec.Op{PluginInput: map[string]any{"method": "run", "text": "echo hi"}}, ""},
		{"gif", spec.Op{PluginInput: map[string]any{"method": "gif"}}, "artifact"},
		{"gif", spec.Op{PluginInput: map[string]any{"method": "gif", "artifact": "/tmp/x.gif"}}, ""},
	}
	for _, tc := range cases {
		err := sdk.RequireModifiers(tc.method, &tc.op, requiredModifiers)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("%s: unexpected error: %v", tc.method, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: expected error containing %q, got %v", tc.method, tc.wantErr, err)
		}
	}
}
