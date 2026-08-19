//go:build unit

package desktop_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

type openedFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Error   string `json:"error"`
}

func readFile(t *testing.T, path string) openedFile {
	t.Helper()

	var answer openedFile
	service := &desktop.FileService{}
	require.NoError(t, json.Unmarshal([]byte(service.Read(path)), &answer))

	return answer
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestFileService(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		t.Run("answers the file's base name, absolute path and contents", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", test.BillingPayments)

			answer := readFile(t, path)

			require.Empty(t, answer.Error)
			require.Equal(t, "billing.emod", answer.Name)
			require.Equal(t, path, answer.Path)
			require.Equal(t, test.BillingPayments, answer.Content)
		})

		t.Run("hands back source the pipeline would reject, leaving what is wrong with it to the reader", func(t *testing.T) {
			source := "emod 1\nmodel \"Billing\"\ncontext {\n"
			path := writeFile(t, t.TempDir(), "broken.emod", source)

			answer := readFile(t, path)

			require.Empty(t, answer.Error)
			require.Equal(t, source, answer.Content)
		})

		t.Run("preserves the bytes exactly, line endings and trailing blank lines included", func(t *testing.T) {
			content := "emod 1\r\n\r\nmodel \"Billing\"\n\n\n"
			path := writeFile(t, t.TempDir(), "crlf.emod", content)

			require.Equal(t, content, readFile(t, path).Content)
		})

		t.Run("answers an empty file as empty content rather than as a failure", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "empty.emod", "")

			answer := readFile(t, path)

			require.Empty(t, answer.Error)
			require.Equal(t, "empty.emod", answer.Name)
			require.Equal(t, "", answer.Content)
		})

		t.Run("answers an absolute path for a relative one, so a caller need not know the working directory", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "billing.emod", test.BillingPayments)
			t.Chdir(dir)

			answer := readFile(t, "billing.emod")

			require.Empty(t, answer.Error)
			require.True(t, filepath.IsAbs(answer.Path), "path %q is not absolute", answer.Path)
			require.Equal(t, "billing.emod", filepath.Base(answer.Path))
			require.Equal(t, test.BillingPayments, answer.Content)
		})

		t.Run("names a missing file as missing rather than failing generically", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "absent.emod")

			answer := readFile(t, path)

			require.Contains(t, answer.Error, "no such file or directory")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Content)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
		})

		t.Run("names a file it may not read as a permission failure", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("root reads a mode 000 file whatever its permissions say")
			}
			path := writeFile(t, t.TempDir(), "secret.emod", test.BillingPayments)
			require.NoError(t, os.Chmod(path, 0o000))

			answer := readFile(t, path)

			require.Contains(t, answer.Error, "permission denied")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Content)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
		})

		t.Run("names a directory as a directory rather than reporting it missing", func(t *testing.T) {
			dir := t.TempDir()

			answer := readFile(t, dir)

			require.Contains(t, answer.Error, "is a directory")
			require.Contains(t, answer.Error, dir)
			require.Empty(t, answer.Content)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
		})

		t.Run("refuses a file it cannot carry unaltered rather than substituting replacement characters", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "latin1.emod")
			require.NoError(t, os.WriteFile(path, []byte("emod 1\nmodel \"R\xe9servations\"\n"), 0o600))

			answer := readFile(t, path)

			require.Contains(t, answer.Error, "not valid UTF-8")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Content)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
		})

		t.Run("carries non-ASCII UTF-8 through byte for byte, because a quoted name may hold any prose", func(t *testing.T) {
			content := "emod 1\nmodel \"Réservations — Hôtel\"\n"
			path := writeFile(t, t.TempDir(), "utf8.emod", content)

			answer := readFile(t, path)

			require.Empty(t, answer.Error)
			require.Equal(t, content, answer.Content)
		})

		t.Run("states the path once in the reason, not twice the way the wrapped syscall error does", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "absent.emod")

			answer := readFile(t, path)

			require.Equal(t, 1, strings.Count(answer.Error, path),
				"the reason repeats the path: %s", answer.Error)
		})
	})
}
