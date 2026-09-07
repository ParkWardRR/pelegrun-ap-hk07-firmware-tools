package eyas

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner replies with canned output per exact command prefix match, in
// the order given — closest thing to the project's httptest-server pattern,
// but for a shell Runner instead of an HTTP client.
type fakeRunner struct {
	responses map[string]string // substring of cmd -> output
	err       error
}

func (f fakeRunner) run(_ context.Context, cmd string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	for substr, out := range f.responses {
		if strings.Contains(cmd, substr) {
			return out, nil
		}
	}
	return "", nil
}

func TestBoardIDFromUbus(t *testing.T) {
	// The Runner returns whatever the full piped command (ubus | sed) prints
	// on the device — already extracted, not raw JSON.
	fr := fakeRunner{responses: map[string]string{
		"ubus call": "engenius,ews377ap-v3\n",
	}}
	got, err := BoardID(context.Background(), fr.run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "engenius,ews377ap-v3" {
		t.Errorf("BoardID = %q, want engenius,ews377ap-v3", got)
	}
}

func TestBoardIDFallsBackToBoardJSON(t *testing.T) {
	// ubus produces nothing (e.g. minimal RAM-boot image); board.json fallback answers.
	fr := fakeRunner{responses: map[string]string{
		"board.json": "engenius,ews377ap-v3",
	}}
	got, err := BoardID(context.Background(), fr.run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "engenius,ews377ap-v3" {
		t.Errorf("BoardID = %q, want engenius,ews377ap-v3", got)
	}
}

func TestBoardIDReturnsErrWhenBothEmpty(t *testing.T) {
	fr := fakeRunner{responses: map[string]string{}}
	_, err := BoardID(context.Background(), fr.run)
	if !errors.Is(err, ErrNoBoardID) {
		t.Errorf("err = %v, want ErrNoBoardID", err)
	}
}

func TestBoardIDPropagatesRunnerError(t *testing.T) {
	wantErr := errors.New("ssh: connection refused")
	fr := fakeRunner{err: wantErr}
	_, err := BoardID(context.Background(), fr.run)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}
