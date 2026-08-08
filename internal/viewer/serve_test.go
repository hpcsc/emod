//go:build unit

package viewer_test

import (
	"io"
	"net/http"
	"strings"
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
		// The placeholder it replaces sits between two <script> tags rather than inside
		// one, so an assignment injected bare is document text: the browser lays the
		// JSON out down the page and the viewer loads with no model. Asserting the
		// substring alone cannot tell those two apart, which is how that shipped.
		t.Run("injects the diagram JSON as script the browser will run", func(t *testing.T) {
			diagramJSON := []byte(`{"nodes":[],"edges":[]}`)

			addr := startViewer(t, diagramJSON)

			require.Regexp(t,
				`(?s)<script>\s*window\.INITIAL_DATA = \{"nodes":\[\],"edges":\[\]\};\s*</script>`,
				body(t, get(t, addr)))
		})

		t.Run("a model that spells the closing script tag cannot end the block early", func(t *testing.T) {
			diagramJSON := []byte(`{"model_name":"</script><img src=x>","nodes":[]}`)

			addr := startViewer(t, diagramJSON)

			pageBody := body(t, get(t, addr))
			injected := pageBody[strings.Index(pageBody, "window.INITIAL_DATA"):]
			// Everything up to the block's own terminator is one script: the model's
			// text must not have contributed a `</script>` of its own along the way.
			// Its `<img>` still appears in the page — as characters inside a JS string,
			// which is exactly the point, so asserting its absence would be wrong.
			require.NotContains(t, injected[:strings.Index(injected, "\n</script>")], "</script>")
			require.Contains(t, pageBody, `"model_name":"<\/script><img src=x>"`)
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
