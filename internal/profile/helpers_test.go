package profile_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/davidsbond/autopgo/internal/profile"
)

var (
	//go:embed testdata/default.pgo
	validProfile []byte
)

func appKeyMatcher(app string) any {
	return mock.MatchedBy(func(s string) bool {
		return strings.HasPrefix(s, app)
	})
}

func uploadedEventMatcher(app string) any {
	return mock.MatchedBy(func(e profile.UploadedEvent) bool {
		return e.App == app && strings.HasPrefix(e.ProfileKey, app)
	})
}

type (
	WriteCloser struct {
		closeError error
		writeError error
	}

	ReadCloser struct {
		readError  error
		closeError error
		data       *bytes.Buffer
	}
)

func (n *WriteCloser) Write(b []byte) (int, error) {
	return len(b), n.writeError
}

func (n *WriteCloser) Close() error { return n.closeError }

func (n *ReadCloser) Read(b []byte) (int, error) {
	if n.readError != nil {
		return 0, n.readError
	}

	return n.data.Read(b)
}

func (n *ReadCloser) Close() error { return n.closeError }

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
