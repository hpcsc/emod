//go:build unit && !windows

package desktop_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// How open a new file is belongs to the umask, and os.Chmod is not filtered by
// it — so a literal mode in the write path is invisible wherever the umask is
// 022, which is this machine's default and CI's. Setting one is what makes the
// question decidable; it lives apart from the rest of the umbrella only because
// syscall.Umask does not exist on Windows.
func TestFileServiceUnderAUmask(t *testing.T) {
	t.Run("write", func(t *testing.T) {
		t.Run("gives a file it creates only the permissions a restrictive umask allows", func(t *testing.T) {
			previous := syscall.Umask(0o077)
			t.Cleanup(func() { syscall.Umask(previous) })

			path := filepath.Join(t.TempDir(), "billing.emod")

			service := &desktop.FileService{}
			require.NotContains(t, service.Write(path, test.BillingPayments), "error")

			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		})

		t.Run("still leaves an existing file the permissions it already had, whatever the umask says", func(t *testing.T) {
			previous := syscall.Umask(0o077)
			t.Cleanup(func() { syscall.Umask(previous) })

			dir := t.TempDir()
			path := filepath.Join(dir, "billing.emod")
			require.NoError(t, os.WriteFile(path, []byte("emod 1\n"), 0o600))
			require.NoError(t, os.Chmod(path, 0o664))

			service := &desktop.FileService{}
			require.NotContains(t, service.Write(path, test.BillingPayments), "error")

			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o664), info.Mode().Perm())
		})
	})
}
