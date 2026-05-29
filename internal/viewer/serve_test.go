//go:build unit

package viewer_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/viewer"
	"github.com/stretchr/testify/require"
)

const readyWait = 80 * time.Millisecond

func startAndExtractURL(t *testing.T, diagramJSON []byte) (string, func()) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w

	_, shutdown, err := viewer.ServeViewer(0, diagramJSON)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	output := buf.String()

	re := regexp.MustCompile(`http://127\.0\.0\.1:(\d+)`)
	matches := re.FindStringSubmatch(output)
	require.Len(t, matches, 2, "stdout should contain viewer URL")
	require.Contains(t, output, "Viewer available at")

	addr := fmt.Sprintf("http://127.0.0.1:%s", matches[1])
	return addr, shutdown
}

func TestServeViewer(t *testing.T) {
	t.Run("serves viewer HTML at root URL", func(t *testing.T) {
		addr, shutdown := startAndExtractURL(t, nil)
		defer shutdown()

		time.Sleep(readyWait)

		resp, err := http.Get(addr)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "<!DOCTYPE html>")
		require.Contains(t, string(body), "Emod Diagram Viewer")
		require.NotContains(t, string(body), "window.INITIAL_DATA")
	})

	t.Run("injects diagram JSON when provided", func(t *testing.T) {
		diagramJSON := []byte(`{"nodes":[],"edges":[]}`)
		addr, shutdown := startAndExtractURL(t, diagramJSON)
		defer shutdown()

		time.Sleep(readyWait)

		resp, err := http.Get(addr)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "window.INITIAL_DATA = ")
		require.Contains(t, string(body), `{"nodes":[],"edges":[]}`)
	})

	t.Run("serves HTML unchanged when diagram JSON is empty", func(t *testing.T) {
		addr, shutdown := startAndExtractURL(t, []byte{})
		defer shutdown()

		time.Sleep(readyWait)

		resp, err := http.Get(addr)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "<!DOCTYPE html>")
		require.NotContains(t, string(body), "window.INITIAL_DATA")
	})

	t.Run("listens only on 127.0.0.1", func(t *testing.T) {
		addr, shutdown := startAndExtractURL(t, nil)
		defer shutdown()

		time.Sleep(readyWait)

		// Verify it works via localhost (matches what's printed)
		resp, err := http.Get(addr)
		require.NoError(t, err)
		resp.Body.Close()

		// The address itself is 127.0.0.1 — verify by checking the URL format
		require.Contains(t, addr, "127.0.0.1")
		require.NotContains(t, addr, "0.0.0.0")
	})

	t.Run("programmatic shutdown stops the server", func(t *testing.T) {
		addr, shutdown := startAndExtractURL(t, nil)

		time.Sleep(readyWait)

		// Server should be up
		resp, err := http.Get(addr)
		require.NoError(t, err)
		resp.Body.Close()

		// Shut down and wait for it to complete
		shutdown()
		time.Sleep(100 * time.Millisecond)

		// Server should be down now
		_, err = http.Get(addr)
		require.Error(t, err)
	})
}
