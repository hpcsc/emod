//go:build unit

package desktop_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/stretchr/testify/require"
)

type diskReport struct {
	event string
	path  string
}

type recordingListener struct {
	mu    sync.Mutex
	heard []diskReport
}

func (l *recordingListener) ChangedOnDisk(path string) {
	l.record(diskReport{event: "changed", path: path})
}

func (l *recordingListener) MissingFromDisk(path string) {
	l.record(diskReport{event: "missing", path: path})
}

func (l *recordingListener) record(report diskReport) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.heard = append(l.heard, report)
}

func (l *recordingListener) forget() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.heard = nil
}

func (l *recordingListener) reports() []diskReport {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]diskReport(nil), l.heard...)
}

type stoppingListener struct {
	recordingListener
	watch *desktop.OpenFileWatch
}

func (l *stoppingListener) ChangedOnDisk(path string) {
	l.recordingListener.ChangedOnDisk(path)
	l.watch.Stop()
}

func changed(path string) diskReport { return diskReport{event: "changed", path: path} }

func missing(path string) diskReport { return diskReport{event: "missing", path: path} }

func watching(t *testing.T, path string) (*desktop.OpenFileWatch, *recordingListener) {
	t.Helper()

	listener := &recordingListener{}
	watch := desktop.NewOpenFileWatch(listener)
	watch.Watch(path)
	watch.Check()
	listener.forget()

	return watch, listener
}

// writeOldFile writes a file whose modification time is long past, so a
// check finds nothing new in it until it is written again.
func writeOldFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := writeFile(t, dir, name, content)
	setModTime(t, path, time.Now().Add(-time.Hour))

	return path
}

func setModTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()

	require.NoError(t, os.Chtimes(path, modTime, modTime))
}

func modTimeOf(t *testing.T, path string) time.Time {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)

	return info.ModTime()
}

func TestOpenFileWatch(t *testing.T) {
	t.Run("check", func(t *testing.T) {
		t.Run("reports a file that is there at the first check as changed", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			listener := &recordingListener{}
			watch := desktop.NewOpenFileWatch(listener)
			watch.Watch(path)

			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports nothing when the file is as the last check found it", func(t *testing.T) {
			path := writeOldFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			watch.Check()
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports a write in place as a change", func(t *testing.T) {
			path := writeOldFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 1\nmodel \"Billing\"\n")
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports a write that keeps the size as a change", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			setModTime(t, path, time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
			watch, listener := watching(t, path)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 2\n")
			setModTime(t, path, time.Date(2026, 1, 1, 9, 0, 1, 0, time.UTC))
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports a file renamed over the path with the size and the time of the one it replaces as a change", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			working := writeFile(t, dir, ".billing.emod.1234", "emod 2\n")
			setModTime(t, working, modTimeOf(t, path))
			require.NoError(t, os.Rename(working, path))
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports nothing more after an atomic save until the file is written again", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)
			working := writeOldFile(t, dir, ".billing.emod.1234", "emod 2\n")
			require.NoError(t, os.Rename(working, path))
			watch.Check()
			watch.Check()
			require.Equal(t, []diskReport{changed(path)}, listener.reports())

			writeFile(t, dir, "billing.emod", "emod 2\nmodel \"Billing\"\n")
			watch.Check()

			require.Equal(t, []diskReport{changed(path), changed(path)}, listener.reports())
		})

		t.Run("reports a path missing at one check and back at the next as a change, never as missing", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.Rename(path, filepath.Join(dir, "billing.emod~")))
			watch.Check()
			writeFile(t, dir, "billing.emod", "emod 2\n")
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports nothing when a path missing at one check is back as it was at the next", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)
			aside := filepath.Join(dir, "billing.emod~")

			require.NoError(t, os.Rename(path, aside))
			watch.Check()
			require.NoError(t, os.Rename(aside, path))
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports a write that keeps the time as a change, as on a filesystem with coarse times", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			kept := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
			setModTime(t, path, kept)
			watch, listener := watching(t, path)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 1\nmodel \"Billing\"\n")
			setModTime(t, path, kept)
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports a write that keeps the size in the time step the last check saw", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			written := time.Now().Add(-1500 * time.Millisecond)
			setModTime(t, path, written)
			watch, listener := watching(t, path)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 2\n")
			setModTime(t, path, written)
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("measures the time step from the check that saw the file, not from the check now", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			written := time.Now().Add(-1200 * time.Millisecond)
			setModTime(t, path, written)
			watch, listener := watching(t, path)
			time.Sleep(time.Second)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 2\n")
			setModTime(t, path, written)
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports a write that keeps the size in the time step of a modification time just after the check", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			written := time.Now().Add(500 * time.Millisecond)
			setModTime(t, path, written)
			watch, listener := watching(t, path)

			writeFile(t, filepath.Dir(path), "billing.emod", "emod 2\n")
			setModTime(t, path, written)
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})

		t.Run("reports nothing for an unwritten file whose modification time is later than the check", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			setModTime(t, path, time.Now().Add(time.Hour))
			watch, listener := watching(t, path)

			watch.Check()
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports nothing for a file unwritten since a check more than one time step after its last write", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			setModTime(t, path, time.Now().Add(-2500*time.Millisecond))
			watch, listener := watching(t, path)

			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports the same file moved back after it was reported missing as a change", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)
			aside := filepath.Join(dir, "billing.emod~")
			require.NoError(t, os.Rename(path, aside))
			watch.Check()
			watch.Check()

			require.NoError(t, os.Rename(aside, path))
			watch.Check()

			require.Equal(t, []diskReport{missing(path), changed(path)}, listener.reports())
		})

		t.Run("counts only checks in a row that find no file", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)
			aside := filepath.Join(dir, "billing.emod~")

			require.NoError(t, os.Rename(path, aside))
			watch.Check()
			require.NoError(t, os.Rename(aside, path))
			watch.Check()
			require.NoError(t, os.Rename(path, aside))
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports a file whose directory was replaced by a file as missing", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "models")
			require.NoError(t, os.Mkdir(dir, 0o700))
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.RemoveAll(dir))
			require.NoError(t, os.WriteFile(dir, []byte("not a directory\n"), 0o600))
			watch.Check()
			watch.Check()

			require.Equal(t, []diskReport{missing(path)}, listener.reports())
		})

		t.Run("reports nothing while the directory cannot be searched", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "models")
			require.NoError(t, os.Mkdir(dir, 0o700))
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.Chmod(dir, 0o000))
			t.Cleanup(func() { os.Chmod(dir, 0o700) })
			_, err := os.Stat(path)
			require.ErrorIs(t, err, os.ErrPermission, "the fixture must refuse the stat, not report the file missing")
			watch.Check()
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("reports a deleted file as missing once a second check still finds nothing there", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.Remove(path))
			watch.Check()
			require.Empty(t, listener.reports(), "one check that finds nothing can be an editor between a rename and a write")

			watch.Check()
			require.Equal(t, []diskReport{missing(path)}, listener.reports())
		})

		t.Run("reports a file renamed away as missing", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.Rename(path, filepath.Join(dir, "invoicing.emod")))
			watch.Check()
			watch.Check()

			require.Equal(t, []diskReport{missing(path)}, listener.reports())
		})

		t.Run("reports a missing file once, however many checks find it missing", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			require.NoError(t, os.Remove(path))
			for range 5 {
				watch.Check()
			}

			require.Equal(t, []diskReport{missing(path)}, listener.reports())
		})

		t.Run("reports a file that appears after it was reported missing as a change", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)
			require.NoError(t, os.Remove(path))
			watch.Check()
			watch.Check()

			writeFile(t, dir, "billing.emod", "emod 1\n")
			watch.Check()

			require.Equal(t, []diskReport{missing(path), changed(path)}, listener.reports())
		})

		t.Run("reports a path with no file there from the start as missing", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "billing.emod")
			listener := &recordingListener{}
			watch := desktop.NewOpenFileWatch(listener)
			watch.Watch(path)

			watch.Check()
			watch.Check()

			require.Equal(t, []diskReport{missing(path)}, listener.reports())
		})

		t.Run("reports nothing that happens to other files in the directory", func(t *testing.T) {
			dir := t.TempDir()
			path := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			other := writeFile(t, dir, "invoicing.emod", "emod 1\n")
			watch.Check()
			writeFile(t, dir, "invoicing.emod", "emod 2\nmodel \"Invoicing\"\n")
			watch.Check()
			require.NoError(t, os.Rename(other, filepath.Join(dir, "shipping.emod")))
			watch.Check()
			require.NoError(t, os.Remove(filepath.Join(dir, "shipping.emod")))
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("follows a symbolic link to the file it names, and reports the path it was asked to watch", func(t *testing.T) {
			dir := t.TempDir()
			model := writeOldFile(t, dir, "billing.emod", "emod 1\n")
			link := filepath.Join(dir, "current.emod")
			require.NoError(t, os.Symlink(model, link))
			watch, listener := watching(t, link)
			watch.Check()
			require.Empty(t, listener.reports(), "the link itself is new, and only the file it names is unwritten")

			writeFile(t, dir, "billing.emod", "emod 1\nmodel \"Billing\"\n")
			watch.Check()

			require.Equal(t, []diskReport{changed(link)}, listener.reports())
		})

		t.Run("reports nothing before any path is watched", func(t *testing.T) {
			listener := &recordingListener{}
			watch := desktop.NewOpenFileWatch(listener)

			watch.Check()
			watch.Check()

			require.Empty(t, listener.reports())
		})
		t.Run("tells a listener that stops the watch, and reports nothing after it", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			listener := &stoppingListener{}
			watch := desktop.NewOpenFileWatch(listener)
			listener.watch = watch
			watch.Watch(path)

			checked := make(chan struct{})
			go func() {
				watch.Check()
				close(checked)
			}()
			select {
			case <-checked:
			case <-time.After(time.Second):
				require.FailNow(t, "the check never returned to a listener that stopped the watch")
			}
			writeFile(t, dir, "billing.emod", "emod 2\nmodel \"Billing\"\n")
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})
	})

	t.Run("watch", func(t *testing.T) {
		t.Run("watching a second path ends the watch on the first", func(t *testing.T) {
			dir := t.TempDir()
			first := writeFile(t, dir, "billing.emod", "emod 1\n")
			second := writeOldFile(t, dir, "invoicing.emod", "emod 1\n")
			watch, listener := watching(t, first)

			watch.Watch(second)
			writeFile(t, dir, "billing.emod", "emod 2\nmodel \"Billing\"\n")
			require.NoError(t, os.Remove(first))
			watch.Check()
			watch.Check()

			require.Equal(t, []diskReport{changed(second)}, listener.reports())
		})

		t.Run("a new watch needs checks of its own in a row to report its path missing", func(t *testing.T) {
			dir := t.TempDir()
			first := filepath.Join(dir, "billing.emod")
			second := filepath.Join(dir, "invoicing.emod")
			listener := &recordingListener{}
			watch := desktop.NewOpenFileWatch(listener)
			watch.Watch(first)
			watch.Check()

			watch.Watch(second)
			watch.Check()

			require.Empty(t, listener.reports())
		})

		t.Run("watching a path again reports the file as it is", func(t *testing.T) {
			path := writeOldFile(t, t.TempDir(), "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			watch.Watch(path)
			watch.Check()

			require.Equal(t, []diskReport{changed(path)}, listener.reports())
		})
	})

	t.Run("stop", func(t *testing.T) {
		t.Run("reports nothing once stopped, whatever happens to the file", func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "billing.emod", "emod 1\n")
			watch, listener := watching(t, path)

			watch.Stop()
			writeFile(t, dir, "billing.emod", "emod 2\nmodel \"Billing\"\n")
			watch.Check()
			require.NoError(t, os.Remove(path))
			watch.Check()
			watch.Check()

			require.Empty(t, listener.reports())
		})
	})

	t.Run("concurrent use", func(t *testing.T) {
		t.Run("is safe to watch and stop from several goroutines while one checks", func(t *testing.T) {
			dir := t.TempDir()
			first := writeFile(t, dir, "billing.emod", "emod 1\n")
			second := writeFile(t, dir, "invoicing.emod", "emod 1\n")
			listener := &recordingListener{}
			watch := desktop.NewOpenFileWatch(listener)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 50 {
					watch.Check()
				}
			}()
			for i := range 50 {
				wg.Add(2)
				go func() {
					defer wg.Done()
					if i%2 == 0 {
						watch.Watch(first)
					} else {
						watch.Watch(second)
					}
				}()
				go func() { defer wg.Done(); watch.Stop() }()
			}
			wg.Wait()

			for _, report := range listener.reports() {
				require.Contains(t, []diskReport{changed(first), changed(second)}, report)
			}
		})
	})
}
