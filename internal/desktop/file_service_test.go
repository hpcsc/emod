//go:build unit

package desktop_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

type savedFile struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

func saveFile(t *testing.T, path, content string) savedFile {
	t.Helper()

	var answer savedFile
	service := &desktop.FileService{}
	require.NoError(t, json.Unmarshal([]byte(service.Write(path, content)), &answer))

	return answer
}

func entriesIn(t *testing.T, dir string) []string {
	t.Helper()

	found, err := os.ReadDir(dir)
	require.NoError(t, err)

	var names []string
	for _, entry := range found {
		names = append(names, entry.Name())
	}

	return names
}

func readBack(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(raw)
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// A rename over a target this process may open for writing still needs
// permission to delete that target, and only an ACL separates the two. This is
// the one failure reachable after the working file exists, which is what makes
// the replacement's cleanup testable at all. /bin/chmod by absolute path: a GNU
// chmod earlier on PATH does not understand +a.
func denyDelete(t *testing.T, path string, inherit bool) {
	t.Helper()

	if runtime.GOOS != "darwin" {
		t.Skip("denying delete on its own needs a darwin ACL")
	}
	rule := os.Getenv("USER") + " deny delete"
	if inherit {
		rule += ",file_inherit"
	}
	require.NoError(t, exec.Command("/bin/chmod", "+a", rule, path).Run())
	t.Cleanup(func() { exec.Command("/bin/chmod", "-R", "-N", path).Run() })
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

	t.Run("write", func(t *testing.T) {
		t.Run("creates a file that was not there, holding exactly the bytes it was given", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "billing.emod")

			answer := saveFile(t, path, test.BillingPayments)

			require.Equal(t, savedFile{Name: "billing.emod", Path: path}, answer)
			require.Equal(t, test.BillingPayments, readBack(t, path))
			require.Equal(t, []string{"billing.emod"}, entriesIn(t, filepath.Dir(path)),
				"the write left something beside the file it was asked for")
		})

		t.Run("replaces what a file held, and leaves the permission bits it held alone", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "billing.emod")
			require.NoError(t, os.WriteFile(path, []byte("emod 1\nmodel \"Stale\"\n"), 0o640))
			require.NoError(t, os.Chmod(path, 0o640))

			require.Equal(t, savedFile{Name: "billing.emod", Path: path}, saveFile(t, path, test.BillingPayments))

			require.Equal(t, test.BillingPayments, readBack(t, path))
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
		})

		t.Run("puts back what the read answered byte for byte, line endings, trailing blank lines and non-ASCII prose included", func(t *testing.T) {
			content := "emod 1\r\n\r\nmodel \"Réservations — Hôtel\"\n\n\n"
			dir := t.TempDir()
			source := writeFile(t, dir, "crlf.emod", content)
			target := filepath.Join(dir, "copy.emod")

			require.Empty(t, saveFile(t, target, readFile(t, source).Content).Error)

			require.Equal(t, content, readBack(t, target))
		})

		t.Run("gives a file it creates the permissions the umask allows, not permissions of its own", func(t *testing.T) {
			dir := t.TempDir()
			reference, err := os.OpenFile(filepath.Join(dir, "reference"), os.O_WRONLY|os.O_CREATE, 0o666)
			require.NoError(t, err)
			require.NoError(t, reference.Close())
			expected, err := os.Stat(filepath.Join(dir, "reference"))
			require.NoError(t, err)

			path := filepath.Join(dir, "billing.emod")
			require.Empty(t, saveFile(t, path, test.BillingPayments).Error)

			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, expected.Mode().Perm(), info.Mode().Perm())
		})

		t.Run("writes through a symlink to the model it points at, leaving the link a link", func(t *testing.T) {
			dir := t.TempDir()
			model := writeFile(t, dir, "billing.emod", "emod 1\nmodel \"Stale\"\n")
			link := filepath.Join(dir, "current.emod")
			require.NoError(t, os.Symlink(model, link))

			require.Empty(t, saveFile(t, link, test.BillingPayments).Error)

			require.Equal(t, test.BillingPayments, readBack(t, model))
			info, err := os.Lstat(link)
			require.NoError(t, err)
			require.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink,
				"the save replaced the link instead of writing through it")
		})

		t.Run("saves a model whose long name holds non-ASCII prose, cutting its working name on a rune", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, strings.Repeat("é", 130)+".emod")

			answer := saveFile(t, path, test.BillingPayments)

			require.Empty(t, answer.Error)
			require.Equal(t, test.BillingPayments, readBack(t, path))
		})

		t.Run("creates the model a dangling symlink names rather than reporting the link exists", func(t *testing.T) {
			dir := t.TempDir()
			link := filepath.Join(dir, "current.emod")
			require.NoError(t, os.Symlink(filepath.Join(dir, "absent.emod"), link))

			answer := saveFile(t, link, test.BillingPayments)

			require.Empty(t, answer.Error)
			require.Equal(t, test.BillingPayments, readBack(t, filepath.Join(dir, "absent.emod")))
		})

		// A rename over a target this process may open for writing still needs
		// permission to delete it, which an ACL can withhold on its own. It is
		// the one failure reachable after the working file exists, so it is what
		// proves the working file is taken away again.
		t.Run("takes its working file away again when the replacement cannot land", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\nmodel \"Stale\"\n")
			denyDelete(t, path, false)

			answer := saveFile(t, path, test.BillingPayments)

			require.Contains(t, answer.Error, "permission denied")
			require.Equal(t, "emod 1\nmodel \"Stale\"\n", readBack(t, path))
			require.Equal(t, []string{"billing.emod"}, entriesIn(t, dir),
				"the refused write left its working file behind")
		})

		// The same refusal states the path once, which the rename's own error
		// does not: it names both the working file and the target.
		t.Run("states the path once when a rename is what was refused", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", test.BillingPayments)
			denyDelete(t, path, false)

			answer := saveFile(t, path, "emod 1\nmodel \"Replacement\"\n")

			require.Equal(t, 1, strings.Count(answer.Error, path),
				"the reason names the path more than once: %s", answer.Error)
			require.NotContains(t, answer.Error, "rename ")
		})

		t.Run("saves a model whose name is as long as the filesystem allows", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, strings.Repeat("a", 250)+".emod")

			answer := saveFile(t, path, test.BillingPayments)

			require.Empty(t, answer.Error)
			require.Equal(t, test.BillingPayments, readBack(t, path))
		})

		t.Run("answers an absolute path for a relative one, so a caller need not know the working directory", func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			answer := saveFile(t, "billing.emod", test.BillingPayments)

			require.Empty(t, answer.Error)
			require.True(t, filepath.IsAbs(answer.Path), "path %q is not absolute", answer.Path)
			require.Equal(t, "billing.emod", answer.Name)
			require.Equal(t, test.BillingPayments, readBack(t, answer.Path))
		})

		t.Run("refuses a file it may not write, and leaves what that file held alone", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("root writes a mode 444 file whatever its permissions say")
			}
			path := writeFile(t, t.TempDir(), "readonly.emod", test.BillingPayments)
			require.NoError(t, os.Chmod(path, 0o444))

			answer := saveFile(t, path, "emod 1\nmodel \"Replacement\"\n")

			require.Contains(t, answer.Error, "permission denied")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
			require.Equal(t, test.BillingPayments, readBack(t, path))
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o444), info.Mode().Perm(),
				"the refusal left the file writable")
		})

		// The target is writable and the directory holding it is not, which is the
		// failure a write that truncated its target before the new contents existed
		// would report only after destroying them. It stands in for running out of
		// space, which a test cannot produce.
		t.Run("refuses a target whose directory it may not write into, and the writable target still holds what it held", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("root writes into a mode 555 directory whatever its permissions say")
			}
			dir := filepath.Join(t.TempDir(), "models")
			require.NoError(t, os.Mkdir(dir, 0o755))
			path := writeFile(t, dir, "billing.emod", test.BillingPayments)
			require.NoError(t, os.Chmod(dir, 0o555))
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

			answer := saveFile(t, path, "emod 1\nmodel \"Replacement\"\n")

			require.Contains(t, answer.Error, "permission denied")
			require.Empty(t, answer.Path)
			require.Equal(t, test.BillingPayments, readBack(t, path))
		})

		t.Run("names a directory as a directory rather than writing a file into it", func(t *testing.T) {
			dir := t.TempDir()

			answer := saveFile(t, dir, test.BillingPayments)

			require.Contains(t, answer.Error, "is a directory")
			require.Contains(t, answer.Error, dir)
			require.Empty(t, answer.Name)
			require.Empty(t, answer.Path)
		})

		t.Run("names a parent directory that is not there rather than creating it", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "absent", "billing.emod")

			answer := saveFile(t, path, test.BillingPayments)

			require.Contains(t, answer.Error, "no such file or directory")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Path)
			require.Empty(t, entriesIn(t, dir))
		})

		t.Run("states the path once in the reason, not twice the way the wrapped syscall error does", func(t *testing.T) {
			dir := t.TempDir()

			answer := saveFile(t, dir, test.BillingPayments)

			require.Equal(t, 1, strings.Count(answer.Error, dir),
				"the reason repeats the path: %s", answer.Error)
		})
	})
}
