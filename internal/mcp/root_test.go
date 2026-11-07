//go:build unit

package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/mcp"
)

func writeFile(t *testing.T, path, text string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}

func TestRoot(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		t.Run("a root made from a symbolic link reads the files of the directory that it links to", func(t *testing.T) {
			base := t.TempDir()
			writeFile(t, filepath.Join(base, "models", "ordering.emod"), "emod = 1\n")
			require.NoError(t, os.Symlink(filepath.Join(base, "models"), filepath.Join(base, "link")))

			root, err := mcp.NewRoot(filepath.Join(base, "link"))
			require.NoError(t, err)
			text, err := root.Read("ordering.emod")

			require.NoError(t, err)
			require.Equal(t, "emod = 1\n", text)
		})

		t.Run("refuses a directory that does not exist, and names it", func(t *testing.T) {
			missing := filepath.Join(t.TempDir(), "missing")

			_, err := mcp.NewRoot(missing)

			require.Error(t, err)
			require.Contains(t, err.Error(), missing)
			require.Contains(t, err.Error(), "no such file or directory")
		})

		t.Run("refuses a loop of symbolic links, and names it", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Symlink("b", filepath.Join(base, "a")))
			require.NoError(t, os.Symlink("a", filepath.Join(base, "b")))

			_, err := mcp.NewRoot(filepath.Join(base, "a"))

			require.Error(t, err)
			require.Equal(t, "root "+filepath.Join(base, "a")+": too many levels of symbolic links", err.Error())
		})

		t.Run("refuses a file, and names it", func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "ordering.emod")
			writeFile(t, file, "emod = 1\n")

			_, err := mcp.NewRoot(file)

			require.Error(t, err)
			require.Contains(t, err.Error(), file)
			require.Contains(t, err.Error(), "is not a directory")
		})
	})

	t.Run("read", func(t *testing.T) {
		t.Run("reads a file under the root by a path relative to the root", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "ordering.emod"), "model ordering\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read("models/ordering.emod")

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("reads a file under the root by an absolute path", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "billing.emod"), "model billing\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(dir, "billing.emod"))

			require.NoError(t, err)
			require.Equal(t, "model billing\n", text)
		})

		t.Run("refuses a path that climbs out of the root, and names the root but not the text of the file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "secret.emod"), "SECRET-TEXT")
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read("../secret.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
			require.NotContains(t, err.Error(), "SECRET-TEXT")
		})

		t.Run("refuses an absolute path in another directory, and names the root but not the text of the file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "other", "secret.emod"), "SECRET-TEXT")
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read(filepath.Join(base, "other", "secret.emod"))

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
			require.NotContains(t, err.Error(), "SECRET-TEXT")
		})

		t.Run("refuses a symbolic link under the root to a file outside it, and names the root but not the text of the file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "secret.emod"), "SECRET-TEXT")
			require.NoError(t, os.Symlink("../secret.emod", filepath.Join(project, "link.emod")))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read("link.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
			require.NotContains(t, err.Error(), "SECRET-TEXT")
		})

		t.Run("refuses the parent of the root, and names the root", func(t *testing.T) {
			project := filepath.Join(t.TempDir(), "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read("..")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
		})

		t.Run("refuses a path outside the root that does not exist as outside, so a host cannot learn which files exist there", func(t *testing.T) {
			project := filepath.Join(t.TempDir(), "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read("../missing.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
		})

		t.Run("a missing file named through the link that made the root is missing, not outside", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "models"), 0o755))
			link := filepath.Join(base, "link")
			require.NoError(t, os.Symlink(filepath.Join(base, "models"), link))
			root, err := mcp.NewRoot(link)
			require.NoError(t, err)

			_, err = root.Read(filepath.Join(link, "missing.emod"))

			require.Error(t, err)
			require.Contains(t, err.Error(), "reading "+filepath.Join(link, "missing.emod")+": no such file or directory")
		})

		t.Run("a loop of symbolic links gives an error that names the path", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Symlink("b.emod", filepath.Join(dir, "a.emod")))
			require.NoError(t, os.Symlink("a.emod", filepath.Join(dir, "b.emod")))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read("a.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "reading a.emod: too many levels of symbolic links")
		})

		t.Run("refuses a missing file behind a link to a directory outside the root as outside, so a host cannot learn which files exist there", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "outside", "exists.emod"), "SECRET-TEXT")
			require.NoError(t, os.Symlink("../outside", filepath.Join(project, "outlink")))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, errExisting := root.Read("outlink/exists.emod")
			_, errMissing := root.Read("outlink/missing.emod")

			require.Equal(t, `path "outlink/exists.emod" is outside the root `+project, errExisting.Error())
			require.Equal(t, `path "outlink/missing.emod" is outside the root `+project, errMissing.Error())
		})

		t.Run("reads a relative symbolic link to a file under the root", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "ordering.emod"), "model ordering\n")
			require.NoError(t, os.Symlink("models/ordering.emod", filepath.Join(dir, "link.emod")))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read("link.emod")

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("refuses an absolute symbolic link, also to a file under the root, because the root follows only relative links", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), "model ordering\n")
			require.NoError(t, os.Symlink(filepath.Join(dir, "ordering.emod"), filepath.Join(dir, "link.emod")))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read("link.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+dir)
		})

		t.Run("reads the file that the system reads for .. after a link, by a relative and by an absolute path", func(t *testing.T) {
			project := t.TempDir()
			writeFile(t, filepath.Join(project, "a.emod"), "TOP")
			writeFile(t, filepath.Join(project, "real", "a.emod"), "REAL")
			require.NoError(t, os.MkdirAll(filepath.Join(project, "real", "models"), 0o755))
			require.NoError(t, os.Symlink("real/models", filepath.Join(project, "models")))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			relative, errRelative := root.Read("models/../a.emod")
			absolute, errAbsolute := root.Read(project + "/models/../a.emod")

			require.NoError(t, errRelative)
			require.NoError(t, errAbsolute)
			require.Equal(t, []string{"REAL", "REAL"}, []string{relative, absolute})
		})

		t.Run("reads an absolute path through another name of the directory of the root", func(t *testing.T) {
			base := t.TempDir()
			writeFile(t, filepath.Join(base, "models", "ordering.emod"), "model ordering\n")
			require.NoError(t, os.Symlink("models", filepath.Join(base, "alias")))
			root, err := mcp.NewRoot(filepath.Join(base, "models"))
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(base, "alias", "ordering.emod"))

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("through another name of the directory of the root, a missing file is missing, not outside", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "models"), 0o755))
			require.NoError(t, os.Symlink("models", filepath.Join(base, "alias")))
			root, err := mcp.NewRoot(filepath.Join(base, "models"))
			require.NoError(t, err)
			missing := filepath.Join(base, "alias", "missing.emod")

			_, err = root.Read(missing)

			require.Error(t, err)
			require.Equal(t, "reading "+missing+": no such file or directory", err.Error())
		})

		t.Run("a root made from a link reads an absolute path through the directory that the link points to", func(t *testing.T) {
			base := t.TempDir()
			writeFile(t, filepath.Join(base, "models", "ordering.emod"), "model ordering\n")
			link := filepath.Join(base, "link")
			require.NoError(t, os.Symlink("models", link))
			root, err := mcp.NewRoot(link)
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(base, "models", "ordering.emod"))

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("a root made from a link names the link in the error for a path outside it", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "models"), 0o755))
			link := filepath.Join(base, "link")
			require.NoError(t, os.Symlink("models", link))
			root, err := mcp.NewRoot(link)
			require.NoError(t, err)

			_, err = root.Read("../outside.emod")

			require.Error(t, err)
			require.Equal(t, `path "../outside.emod" is outside the root `+link, err.Error())
		})

		t.Run("reads an absolute path that spells the directory of the root in another letter case, where the file system ignores case", func(t *testing.T) {
			base := t.TempDir()
			writeFile(t, filepath.Join(base, "project", "ordering.emod"), "model ordering\n")
			if _, err := os.Stat(filepath.Join(base, "PROJECT")); err != nil {
				t.Skip("the file system tells letter case apart")
			}
			root, err := mcp.NewRoot(filepath.Join(base, "project"))
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(base, "PROJECT", "ordering.emod"))

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("a root made from . reads an absolute path under the working directory", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), "model ordering\n")
			t.Chdir(dir)
			root, err := mcp.NewRoot(".")
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(dir, "ordering.emod"))

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("a missing directory under the root, named by an absolute path, is missing, not outside", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "models"), 0o755))
			link := filepath.Join(base, "link")
			require.NoError(t, os.Symlink("models", link))
			root, err := mcp.NewRoot(link)
			require.NoError(t, err)
			path := filepath.Join(link, "drafts", "ordering.emod")

			_, err = root.Read(path)

			require.Error(t, err)
			require.Equal(t, "reading "+path+": no such file or directory", err.Error())
		})

		t.Run("refuses an absolute path in a directory that does not exist outside the root", func(t *testing.T) {
			project := filepath.Join(t.TempDir(), "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)
			path := filepath.Join(filepath.Dir(project), "nowhere", "ordering.emod")

			_, err = root.Read(path)

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
		})

		t.Run("refuses the directory of the root itself, named by an absolute path, as not a regular file", func(t *testing.T) {
			dir := t.TempDir()
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read(dir)

			require.Error(t, err)
			require.Equal(t, "reading "+dir+": not a regular file", err.Error())
		})

		t.Run("a root whose directory is gone gives an error that names the path", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "project")
			require.NoError(t, os.Mkdir(dir, 0o755))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)
			require.NoError(t, os.Remove(dir))

			_, err = root.Read("ordering.emod")

			require.Error(t, err)
			require.Equal(t, "reading ordering.emod: no such file or directory", err.Error())
		})

		t.Run("a root at the top of the file system reads an absolute path", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), "model ordering\n")
			root, err := mcp.NewRoot(string(filepath.Separator))
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(dir, "ordering.emod"))

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("reads an absolute path with a repeated separator after the root", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), "model ordering\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read(dir + "//ordering.emod")

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
		})

		t.Run("through another name of the directory of the root, reads the file that the system reads for .. after a link", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			writeFile(t, filepath.Join(project, "a.emod"), "TOP")
			writeFile(t, filepath.Join(project, "real", "a.emod"), "REAL")
			require.NoError(t, os.MkdirAll(filepath.Join(project, "real", "models"), 0o755))
			require.NoError(t, os.Symlink("real/models", filepath.Join(project, "models")))
			require.NoError(t, os.Symlink("project", filepath.Join(base, "alias")))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			text, err := root.Read(filepath.Join(base, "alias") + "/models/../a.emod")

			require.NoError(t, err)
			require.Equal(t, "REAL", text)
		})

		t.Run("through another name of the directory of the root, a missing directory is missing, not outside", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "project"), 0o755))
			alias := filepath.Join(base, "alias")
			require.NoError(t, os.Symlink("project", alias))
			root, err := mcp.NewRoot(filepath.Join(base, "project"))
			require.NoError(t, err)
			missing := filepath.Join(alias, "drafts", "ordering.emod")

			_, err = root.Read(missing)

			require.Error(t, err)
			require.Equal(t, "reading "+missing+": no such file or directory", err.Error())
		})

		t.Run("through another name of the directory of the root, the directory itself is not a regular file", func(t *testing.T) {
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "project"), 0o755))
			alias := filepath.Join(base, "alias")
			require.NoError(t, os.Symlink("project", alias))
			root, err := mcp.NewRoot(filepath.Join(base, "project"))
			require.NoError(t, err)

			_, err = root.Read(alias)

			require.Error(t, err)
			require.Equal(t, "reading "+alias+": not a regular file", err.Error())
		})

		t.Run("refuses a sibling directory whose name starts with the name of the root", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "projectile", "secret.emod"), "SECRET-TEXT")
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			_, err = root.Read(project + "ile/secret.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "is outside the root "+project)
		})

		t.Run("refuses a directory as not a regular file", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "models"), 0o755))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read("models")

			require.Error(t, err)
			require.Equal(t, "reading models: not a regular file", err.Error())
		})

		t.Run("refuses a named pipe as not a regular file, and does not wait for a writer", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe.emod"), 0o644))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			read := make(chan error, 1)
			go func() {
				_, err := root.Read("pipe.emod")
				read <- err
			}()

			select {
			case err := <-read:
				require.Error(t, err)
				require.Equal(t, "reading pipe.emod: not a regular file", err.Error())
			case <-time.After(5 * time.Second):
				require.Fail(t, "the read of a named pipe still waits for a writer after 5 seconds")
			}
		})

		t.Run("reads a file of exactly 1 MiB", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "large.emod"), strings.Repeat("#", 1<<20))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read("large.emod")

			require.NoError(t, err)
			require.Len(t, text, 1<<20)
		})

		t.Run("refuses a file larger than 1 MiB", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "large.emod"), strings.Repeat("#", 1<<20+1))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read("large.emod")

			require.Error(t, err)
			require.Equal(t, "reading large.emod: the file is larger than 1 MiB", err.Error())
		})

		t.Run("a path that does not exist gives an error that names the path and says that it does not exist", func(t *testing.T) {
			root, err := mcp.NewRoot(t.TempDir())
			require.NoError(t, err)

			_, err = root.Read("models/missing.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "reading models/missing.emod: no such file or directory")
		})

		t.Run("a file with no read permission gives an error that names the path and the cause", func(t *testing.T) {
			if os.Geteuid() == 0 {
				t.Skip("root reads a file with no read permission")
			}
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "locked.emod"), "model locked\n")
			require.NoError(t, os.Chmod(filepath.Join(dir, "locked.emod"), 0o000))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			_, err = root.Read("locked.emod")

			require.Error(t, err)
			require.Contains(t, err.Error(), "reading locked.emod: permission denied")
		})

		t.Run("a read changes neither the bytes nor the modification time of the file", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model ordering\n")
			written := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			require.NoError(t, os.Chtimes(path, written, written))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			text, err := root.Read("ordering.emod")

			require.NoError(t, err)
			require.Equal(t, "model ordering\n", text)
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.True(t, info.ModTime().Equal(written))
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "model ordering\n", string(content))
		})
	})

	t.Run("replace", func(t *testing.T) {
		t.Run("replaces the text of a file under the root by a relative path, and keeps the mode of the file", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "models", "ordering.emod")
			writeFile(t, path, "model   ordering\n")
			require.NoError(t, os.Chmod(path, 0o640))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("models/ordering.emod", "model   ordering\n", "model ordering\n")

			require.NoError(t, err)
			requireText(t, path, "model ordering\n")
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
		})

		t.Run("replaces the text of a file under the root by an absolute path", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   ordering\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace(path, "model   ordering\n", "model ordering\n")

			require.NoError(t, err)
			requireText(t, path, "model ordering\n")
		})

		t.Run("replaces a longer text with a shorter one, and leaves no rest of the longer text", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model ordering\n\n\n\n# the end\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("ordering.emod", "model ordering\n\n\n\n# the end\n", "model ordering\n")

			require.NoError(t, err)
			requireText(t, path, "model ordering\n")
		})

		t.Run("refuses a path that climbs out of the root as outside the root, and changes no file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "outside.emod"), "model   outside\n")
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			err = root.Replace("../outside.emod", "model   outside\n", "model outside\n")

			require.EqualError(t, err, `path "../outside.emod" is outside the root `+project)
			requireText(t, filepath.Join(base, "outside.emod"), "model   outside\n")
		})

		t.Run("refuses an absolute path in another directory as outside the root, and changes no file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			outside := filepath.Join(base, "other", "outside.emod")
			writeFile(t, outside, "model   outside\n")
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			err = root.Replace(outside, "model   outside\n", "model outside\n")

			require.EqualError(t, err, `path "`+outside+`" is outside the root `+project)
			requireText(t, outside, "model   outside\n")
		})

		t.Run("refuses a symbolic link under the root to a file outside it as outside the root, and changes no file", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "outside.emod"), "model   outside\n")
			require.NoError(t, os.Symlink("../outside.emod", filepath.Join(project, "link.emod")))
			root, err := mcp.NewRoot(project)
			require.NoError(t, err)

			err = root.Replace("link.emod", "model   outside\n", "model outside\n")

			require.EqualError(t, err, `path "link.emod" is outside the root `+project)
			requireText(t, filepath.Join(base, "outside.emod"), "model   outside\n")
		})

		t.Run("refuses an absolute symbolic link, also to a file under the root, because the root follows only relative links", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   ordering\n")
			require.NoError(t, os.Symlink(path, filepath.Join(dir, "link.emod")))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("link.emod", "model   ordering\n", "model ordering\n")

			require.EqualError(t, err, `path "link.emod" is outside the root `+dir)
			requireText(t, path, "model   ordering\n")
		})

		t.Run("writes the file that a relative symbolic link under the root points to, and keeps the link", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   ordering\n")
			require.NoError(t, os.Symlink("ordering.emod", filepath.Join(dir, "link.emod")))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("link.emod", "model   ordering\n", "model ordering\n")

			require.NoError(t, err)
			requireText(t, path, "model ordering\n")
			target, err := os.Readlink(filepath.Join(dir, "link.emod"))
			require.NoError(t, err)
			require.Equal(t, "ordering.emod", target)
		})

		t.Run("refuses a missing file and does not create it", func(t *testing.T) {
			dir := t.TempDir()
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("missing.emod", "", "model missing\n")

			require.EqualError(t, err, "writing missing.emod: no such file or directory")
			_, err = os.Stat(filepath.Join(dir, "missing.emod"))
			require.ErrorIs(t, err, os.ErrNotExist)
		})

		t.Run("refuses a directory as not a regular file", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "models"), 0o755))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("models", "", "model ordering\n")

			require.EqualError(t, err, "writing models: not a regular file")
		})

		t.Run("refuses a named pipe as not a regular file, and does not wait for a reader", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe.emod"), 0o644))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			replaced := make(chan error, 1)
			go func() {
				replaced <- root.Replace("pipe.emod", "", "model ordering\n")
			}()

			select {
			case err := <-replaced:
				require.EqualError(t, err, "writing pipe.emod: not a regular file")
			case <-time.After(5 * time.Second):
				require.Fail(t, "the replace of a named pipe still waits for a reader after 5 seconds")
			}
		})

		t.Run("refuses a file that no longer holds the text that the caller read, and the file keeps its text", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   ordering\n# saved by a person after the read\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("ordering.emod", "model   ordering\n", "model ordering\n")

			require.EqualError(t, err, "writing ordering.emod: the file changed after the tool read it")
			requireText(t, path, "model   ordering\n# saved by a person after the read\n")
		})

		t.Run("refuses a file that a person changed to a text of the same length, and the file keeps its text", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   orderinG\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("ordering.emod", "model   ordering\n", "model ordering\n")

			require.EqualError(t, err, "writing ordering.emod: the file changed after the tool read it")
			requireText(t, path, "model   orderinG\n")
		})

		t.Run("two replaces of one file at the same time never both write: one writes its text, and the other finds that the file changed", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			read := strings.Repeat("r", 400)
			first, second := strings.Repeat("f", 100), strings.Repeat("s", 200)
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			for range 200 {
				writeFile(t, path, read)
				start := make(chan struct{})
				replaced := make(chan error, 2)
				for _, text := range []string{first, second} {
					go func() {
						<-start
						replaced <- root.Replace("ordering.emod", read, text)
					}()
				}
				close(start)
				errs := []error{<-replaced, <-replaced}

				content, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Contains(t, []string{first, second}, string(content), "the file must hold the text of one replace, whole")
				var refused []string
				for _, err := range errs {
					if err != nil {
						refused = append(refused, err.Error())
					}
				}
				require.Equal(t, []string{"writing ordering.emod: the file changed after the tool read it"}, refused)
			}
		})

		t.Run("a read in parallel with a replace gives the old text or the new text, whole", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			old, new := strings.Repeat("o", 1<<20), strings.Repeat("n", 1<<20)
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			for range 20 {
				writeFile(t, path, old)
				replaced := make(chan struct{})
				torn := make(chan int, 4)
				for range 4 {
					go func() {
						count := 0
						for {
							select {
							case <-replaced:
								torn <- count
								return
							default:
							}
							text, err := root.Read("ordering.emod")
							if err == nil && text != old && text != new {
								count++
							}
						}
					}()
				}
				require.NoError(t, root.Replace("ordering.emod", old, new))
				close(replaced)
				for range 4 {
					require.Zero(t, <-torn, "a read must not give the start of one text and the end of the other")
				}
			}
		})

		t.Run("refuses a file that holds only the start of the text that the caller read, and the file keeps its text", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ordering.emod")
			writeFile(t, path, "model   ordering\n")
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("ordering.emod", "model   ordering\n# a line that a person removed\n", "model ordering\n")

			require.EqualError(t, err, "writing ordering.emod: the file changed after the tool read it")
			requireText(t, path, "model   ordering\n")
		})

		t.Run("a root whose directory is gone gives an error that names the path", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "project")
			require.NoError(t, os.Mkdir(dir, 0o755))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)
			require.NoError(t, os.Remove(dir))

			err = root.Replace("ordering.emod", "model   ordering\n", "model ordering\n")

			require.EqualError(t, err, "writing ordering.emod: no such file or directory")
		})

		t.Run("a file with no write permission gives an error that names the path and the cause, and keeps its text", func(t *testing.T) {
			if os.Geteuid() == 0 {
				t.Skip("root writes a file with no write permission")
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "locked.emod")
			writeFile(t, path, "model   locked\n")
			require.NoError(t, os.Chmod(path, 0o444))
			root, err := mcp.NewRoot(dir)
			require.NoError(t, err)

			err = root.Replace("locked.emod", "model   locked\n", "model locked\n")

			require.EqualError(t, err, "writing locked.emod: permission denied")
			requireText(t, path, "model   locked\n")
		})
	})
}

func requireText(t *testing.T, path, text string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, text, string(content))
}
