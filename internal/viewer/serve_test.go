//go:build unit

package viewer_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/viewer"
	"github.com/stretchr/testify/require"
)

const (
	readyTimeout = 5 * time.Second
	readyPoll    = 5 * time.Millisecond
)

// startViewer serves diagramJSON and returns the URL once the server answers,
// shutting it down during cleanup.
func startViewer(t *testing.T, diagramJSON []byte) string {
	t.Helper()

	addr, shutdown, err := viewer.ServeViewer(0, diagramJSON)
	require.NoError(t, err)
	t.Cleanup(shutdown)

	require.Eventually(t, func() bool {
		resp, err := http.Get(addr)
		if err != nil {
			return false
		}
		resp.Body.Close()
		return true
	}, readyTimeout, readyPoll, "viewer did not start answering at %s", addr)

	return addr
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := http.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(raw)
}

func TestServeViewer(t *testing.T) {
	t.Run("serving", func(t *testing.T) {
		t.Run("returns the viewer page as HTML at the root URL", func(t *testing.T) {
			addr := startViewer(t, nil)

			resp := get(t, addr)

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
			require.Contains(t, body(t, resp), "Emod Diagram Viewer")
		})

		t.Run("listens on loopback only and reports that address", func(t *testing.T) {
			addr, shutdown, err := viewer.ServeViewer(0, nil)
			require.NoError(t, err)
			t.Cleanup(shutdown)

			// addr comes from the listener, so a wildcard bind (0.0.0.0) shows up here.
			require.Regexp(t, `^http://127\.0\.0\.1:\d+$`, addr)
		})
	})

	t.Run("initial data", func(t *testing.T) {
		t.Run("injects the diagram JSON for the viewer to load", func(t *testing.T) {
			diagramJSON := []byte(`{"nodes":[],"edges":[]}`)

			addr := startViewer(t, diagramJSON)

			require.Contains(t, body(t, get(t, addr)),
				`window.INITIAL_DATA = {"nodes":[],"edges":[]};`)
		})

		t.Run("omits the injection when no diagram JSON is given", func(t *testing.T) {
			addr := startViewer(t, nil)

			require.NotContains(t, body(t, get(t, addr)), "window.INITIAL_DATA")
		})

		t.Run("omits the injection for empty diagram JSON", func(t *testing.T) {
			addr := startViewer(t, []byte{})

			pageBody := body(t, get(t, addr))
			require.Contains(t, pageBody, "<!DOCTYPE html>")
			require.NotContains(t, pageBody, "window.INITIAL_DATA")
		})
	})

	t.Run("shutdown", func(t *testing.T) {
		t.Run("stops answering requests once shut down", func(t *testing.T) {
			addr, shutdown, err := viewer.ServeViewer(0, nil)
			require.NoError(t, err)

			require.Eventually(t, func() bool {
				resp, err := http.Get(addr)
				if err != nil {
					return false
				}
				resp.Body.Close()
				return true
			}, readyTimeout, readyPoll, "viewer never started")

			shutdown()

			require.Eventually(t, func() bool {
				_, err := http.Get(addr)
				return err != nil
			}, readyTimeout, readyPoll, "viewer kept answering after shutdown")
		})
	})
}
