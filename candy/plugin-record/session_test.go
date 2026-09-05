package record

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/opencharly/plugin-record/candy/plugin-record/params"
	"github.com/opencharly/sdk"
	"github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
	"google.golang.org/grpc"
)

// session_test.go covers the record session method (Cutover A instrument model —
// venue-side session; the tmux transport IS the session, already invocation-surviving).
// Unlike the plain record methods (which need a live reverse-channel executor and are
// exercised by the R10 bed), sessionDispatch is unit-tested end to end THROUGH dispatch()
// against a FAKE ExecutorServiceClient standing in for the host's venue executor — start
// → status → stop round trip, the GetFile artifact pull, and the evidence row.json.

// tmuxSessionRe extracts the session name a tmux script targets (e.g. record-vm-a-live
// from a has-session/new-session/send-keys script). Session names are venue-scoped:
// recordSessionName(<venue>-<id>) → record-<venue>-<id>, so the fake keys its session
// set on the exact tmux names the dispatcher uses.
var tmuxSessionRe = regexp.MustCompile(`record-[A-Za-z0-9_.-]+`)

func tmuxSessionFromScript(script string) (string, bool) {
	m := tmuxSessionRe.FindString(script)
	return m, m != ""
}

// fakeExecutor is a script-responding fake of the host's reverse-channel
// ExecutorServiceClient (pb.ExecutorServiceClient): it simulates the venue's tmux
// server (session create/list/has/send-keys-exit/kill), tool probes, the recording
// dir, the .mode metadata, and the GetFile artifact pull.
type fakeExecutor struct {
	mu       sync.Mutex
	sessions map[string]string // tmux session name -> mode
	cast     []byte
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{
		sessions: map[string]string{},
		cast:     []byte("{\"version\": 2, \"events\": []}"),
	}
}

func (f *fakeExecutor) RunCapture(_ context.Context, req *proto.RunRequest, _ ...grpc.CallOption) (*proto.CaptureReply, error) {
	s := req.Script
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case strings.Contains(s, "command -v tmux"):
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "command -v pixelflux-record"), strings.Contains(s, "command -v wf-recorder"):
		return &proto.CaptureReply{ExitCode: 1}, nil
	case strings.Contains(s, "command -v asciinema"):
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "has-session"):
		name, _ := tmuxSessionFromScript(s)
		if _, ok := f.sessions[name]; ok {
			return &proto.CaptureReply{ExitCode: 0}, nil
		}
		return &proto.CaptureReply{ExitCode: 1}, nil
	case strings.Contains(s, "new-session"):
		name, _ := tmuxSessionFromScript(s)
		f.sessions[name] = "terminal"
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "list-sessions"):
		names := make([]string, 0, len(f.sessions))
		for n := range f.sessions {
			names = append(names, n)
		}
		sort.Strings(names)
		return &proto.CaptureReply{Stdout: strings.Join(names, "\n"), ExitCode: 0}, nil
	case strings.Contains(s, "send-keys"):
		name, _ := tmuxSessionFromScript(s)
		if strings.Contains(s, "exit") {
			delete(f.sessions, name) // asciinema exits its shell → session ends
		}
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "kill-session"):
		name, _ := tmuxSessionFromScript(s)
		delete(f.sessions, name)
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "pgrep -x"):
		if len(f.sessions) > 0 {
			return &proto.CaptureReply{ExitCode: 0}, nil
		}
		return &proto.CaptureReply{ExitCode: 1}, nil
	case strings.Contains(s, "mkdir -p"), strings.Contains(s, "printf"), strings.Contains(s, "rm -f"):
		return &proto.CaptureReply{ExitCode: 0}, nil
	case strings.Contains(s, "cat "):
		return &proto.CaptureReply{Stdout: "terminal", ExitCode: 0}, nil
	}
	return &proto.CaptureReply{ExitCode: 0}, nil
}

func (f *fakeExecutor) GetFile(_ context.Context, _ *proto.GetFileRequest, _ ...grpc.CallOption) (*proto.GetFileReply, error) {
	return &proto.GetFileReply{Content: f.cast}, nil
}

// --- the rest of the ExecutorServiceClient interface (unused by record) ---

func (f *fakeExecutor) Venue(_ context.Context, _ *proto.Empty, _ ...grpc.CallOption) (*proto.VenueReply, error) {
	return &proto.VenueReply{}, nil
}
func (f *fakeExecutor) RunSystem(_ context.Context, _ *proto.RunRequest, _ ...grpc.CallOption) (*proto.RunReply, error) {
	return &proto.RunReply{}, nil
}
func (f *fakeExecutor) RunUser(_ context.Context, _ *proto.RunRequest, _ ...grpc.CallOption) (*proto.RunReply, error) {
	return &proto.RunReply{}, nil
}
func (f *fakeExecutor) PutFile(_ context.Context, _ *proto.PutFileRequest, _ ...grpc.CallOption) (*proto.PutFileReply, error) {
	return &proto.PutFileReply{}, nil
}
func (f *fakeExecutor) RunInteractive(_ context.Context, _ *proto.RunRequest, _ ...grpc.CallOption) (*proto.LiveReply, error) {
	return &proto.LiveReply{}, nil
}
func (f *fakeExecutor) RunStream(_ context.Context, _ *proto.RunRequest, _ ...grpc.CallOption) (*proto.LiveReply, error) {
	return &proto.LiveReply{}, nil
}
func (f *fakeExecutor) RunHostStep(_ context.Context, _ *proto.HostStepRequest, _ ...grpc.CallOption) (*proto.HostStepReply, error) {
	return &proto.HostStepReply{}, nil
}
func (f *fakeExecutor) InvokeProvider(_ context.Context, _ *proto.InvokeProviderRequest, _ ...grpc.CallOption) (*proto.InvokeReply, error) {
	return &proto.InvokeReply{}, nil
}
func (f *fakeExecutor) HostBuild(_ context.Context, _ *proto.HostBuildRequest, _ ...grpc.CallOption) (*proto.HostBuildReply, error) {
	return &proto.HostBuildReply{}, nil
}
func (f *fakeExecutor) DescribeProvider(_ context.Context, _ *proto.DescribeProviderRequest, _ ...grpc.CallOption) (*proto.DescribeProviderReply, error) {
	return &proto.DescribeProviderReply{}, nil
}

// TestSessionDispatchRoundTrip drives the session lifecycle (start → status → stop)
// THROUGH dispatch() with Method="session" against the fake executor and asserts the
// artifact pull (the .cast written to in.Artifact by recordStop's GetFile) and the
// evidence row.json (the Cutover A wire shape) written to state_dir on stop.
func TestSessionDispatchRoundTrip(t *testing.T) {
	f := newFakeExecutor()
	ex := sdk.NewInProcExecutor(f)
	ctx := context.Background()
	stateDir := t.TempDir()
	artifact := filepath.Join(t.TempDir(), "vm-a-live.cast")

	base := params.RecordInput{
		Method:    "session",
		SessionId: "vm-a-live",
		Venue:     "vm-a",
		Phase:     "live",
	}

	// start
	start := base
	start.Action = "start"
	out, err := dispatch(ctx, ex, &spec.Op{}, &start)
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	if !strings.Contains(out, "session vm-a-live started") {
		t.Errorf("start output = %q, want it to name the venue-scoped session", out)
	}
	// the tmux session must be the venue-scoped name, not the input id alone
	if _, ok := f.sessions["record-vm-a-live"]; !ok {
		t.Errorf("fake venue has no tmux session record-vm-a-live (got %v)", f.sessions)
	}

	// status
	status := base
	status.Action = "status"
	out, err = dispatch(ctx, ex, &spec.Op{}, &status)
	if err != nil {
		t.Fatalf("session status: %v", err)
	}
	if !strings.Contains(out, "vm-a-live") {
		t.Errorf("status output = %q, want the active session listed", out)
	}

	// stop
	stop := base
	stop.Action = "stop"
	stop.Artifact = artifact
	stop.StateDir = stateDir
	out, err = dispatch(ctx, ex, &spec.Op{}, &stop)
	if err != nil {
		t.Fatalf("session stop: %v", err)
	}
	if !strings.Contains(out, "session vm-a-live stopped") {
		t.Errorf("stop output = %q, want the session named", out)
	}
	if _, ok := f.sessions["record-vm-a-live"]; ok {
		t.Errorf("tmux session record-vm-a-live still active after stop")
	}

	// artifact pulled (recordStop GetFile → in.Artifact)
	data, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("artifact %s not written: %v", artifact, err)
	}
	if len(data) == 0 {
		t.Errorf("artifact %s is empty", artifact)
	}

	// evidence row
	rowBytes, err := os.ReadFile(filepath.Join(stateDir, "row.json"))
	if err != nil {
		t.Fatalf("evidence row.json not written in %s: %v", stateDir, err)
	}
	var row sessionEvidenceRow
	if err := json.Unmarshal(rowBytes, &row); err != nil {
		t.Fatalf("evidence row is not JSON: %v\n%s", err, rowBytes)
	}
	if row.Instrument != "vm-a-live" || row.Origin != "session" || row.Verb != "record" {
		t.Errorf("evidence row provenance = %+v, want instrument=vm-a-live origin=session verb=record", row)
	}
	if row.Venue != "vm-a" || row.Phase != "live" {
		t.Errorf("evidence row venue/phase = %+v, want vm-a/live", row)
	}
	if len(row.Artifact) != 1 || row.Artifact[0].Path != artifact || row.Artifact[0].Kind != "cast" {
		t.Errorf("evidence row artifact = %+v, want [{path: %s kind: cast}]", row.Artifact, artifact)
	}
}

// TestSessionStatusNotActive asserts status fails honestly when the venue-scoped session
// is absent (the venue-scoped name must not leak across venues).
func TestSessionStatusNotActive(t *testing.T) {
	f := newFakeExecutor()
	ex := sdk.NewInProcExecutor(f)
	in := params.RecordInput{Method: "session", SessionId: "vm-b-absent", Action: "status"}
	_, err := dispatch(context.Background(), ex, &spec.Op{}, &in)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("status on absent session: want error containing 'not active', got %v", err)
	}
}

// TestSessionStopRequiresArtifactAndStateDir asserts the stop guardrails (the .cast
// needs an artifact path to be pulled to; the evidence row needs a state dir).
func TestSessionStopRequiresArtifactAndStateDir(t *testing.T) {
	f := newFakeExecutor()
	ex := sdk.NewInProcExecutor(f)
	ctx := context.Background()

	in := params.RecordInput{Method: "session", SessionId: "vm-a-live", Action: "stop", StateDir: t.TempDir()}
	if _, err := dispatch(ctx, ex, &spec.Op{}, &in); err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("stop without artifact: want error mentioning artifact, got %v", err)
	}

	in = params.RecordInput{Method: "session", SessionId: "vm-a-live", Action: "stop", Artifact: filepath.Join(t.TempDir(), "x.cast")}
	if _, err := dispatch(ctx, ex, &spec.Op{}, &in); err == nil || !strings.Contains(err.Error(), "state_dir") {
		t.Fatalf("stop without state_dir: want error mentioning state_dir, got %v", err)
	}
}

// TestSessionDoubleStartRejected asserts the venue-scoped tmux name collision guard: a
// second start for the same venue+session must fail honestly instead of clobbering.
func TestSessionDoubleStartRejected(t *testing.T) {
	f := newFakeExecutor()
	ex := sdk.NewInProcExecutor(f)
	ctx := context.Background()
	in := params.RecordInput{Method: "session", SessionId: "vm-a-live", Action: "start"}
	if _, err := dispatch(ctx, ex, &spec.Op{}, &in); err != nil {
		t.Fatalf("first start: %v", err)
	}
	_, err := dispatch(ctx, ex, &spec.Op{}, &in)
	if err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("second start: want error containing 'already active', got %v", err)
	}
}
