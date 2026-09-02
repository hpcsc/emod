//go:build unit

package desktop_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// recordingMenu stands in for the shell's menu, which no test has: it only
// remembers each list it was shown, in the order it was shown them.
type recordingMenu struct {
	mu    sync.Mutex
	shown [][]string
}

func (m *recordingMenu) Show(paths []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shown = append(m.shown, append([]string{}, paths...))
}

func (m *recordingMenu) sequence() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([][]string(nil), m.shown...)
}

func (m *recordingMenu) latest() []string {
	shown := m.sequence()
	if len(shown) == 0 {
		return nil
	}

	return shown[len(shown)-1]
}

func listPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "recent-files.json")
}

// modelPaths answers n absolute paths, none of which need exist: recording is
// about where a model is, and only opening goes to disk.
func modelPaths(t *testing.T, n int) []string {
	t.Helper()

	dir := t.TempDir()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("model-%02d.emod", i))
	}

	return paths
}

func openRecent(t *testing.T, service *desktop.RecentFiles, path string) openedFile {
	t.Helper()

	var answer openedFile
	require.NoError(t, json.Unmarshal([]byte(service.Open(path)), &answer))

	return answer
}

func TestRecentFiles(t *testing.T) {
	t.Run("starting up", func(t *testing.T) {
		t.Run("holds nothing where no list is saved, and shows the menu that before anything else", func(t *testing.T) {
			menu := &recordingMenu{}

			desktop.NewRecentFiles(listPath(t), menu)

			require.Len(t, menu.sequence(), 1)
			require.Empty(t, menu.latest())
		})

		t.Run("works with no menu at all, so a shell that has none yet is a supported state", func(t *testing.T) {
			path := listPath(t)
			paths := modelPaths(t, 2)
			service := desktop.NewRecentFiles(path, nil)

			require.NoError(t, service.Record(paths[0]))
			require.NoError(t, service.Record(paths[1]))
			require.Contains(t, openRecent(t, service, paths[0]).Error, "no longer at")
			require.NoError(t, service.Clear())

			menu := &recordingMenu{}
			desktop.NewRecentFiles(path, menu)
			require.Empty(t, menu.latest())
		})

		t.Run("keeps the list for its own run only when given no path, and answers no failure", func(t *testing.T) {
			recorded := modelPaths(t, 1)[0]
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles("", menu)

			require.NoError(t, service.Record(recorded))
			require.Equal(t, []string{recorded}, menu.latest())
			require.NoError(t, service.Clear())

			later := &recordingMenu{}
			desktop.NewRecentFiles("", later)
			require.Empty(t, later.latest())
		})

		t.Run("reads back the list a previous run left, in the order it left it", func(t *testing.T) {
			path := listPath(t)
			paths := modelPaths(t, 3)
			earlier := desktop.NewRecentFiles(path, nil)
			for _, p := range paths {
				require.NoError(t, earlier.Record(p))
			}

			menu := &recordingMenu{}
			desktop.NewRecentFiles(path, menu)

			require.Equal(t, []string{paths[2], paths[1], paths[0]}, menu.latest())
		})

		t.Run("starts empty from a file that is not the list, and the next change replaces it", func(t *testing.T) {
			for _, tc := range []struct{ name, raw string }{
				{"an empty file", ""},
				{"a file that is not JSON", "not json at all"},
				{"a bare array where the list document should be", `["/models/bare.emod"]`},
				{"a document whose list is one string", `{"recent": "one path"}`},
				{"a document whose list holds numbers", `{"recent": [1, 2]}`},
				{"a document cut off part way through", `{"recent": ["/models/truncated.emod"`},
				{"a document of another shape", `{"other": []}`},
			} {
				t.Run(tc.name, func(t *testing.T) {
					path := listPath(t)
					require.NoError(t, os.WriteFile(path, []byte(tc.raw), 0o600))
					recorded := modelPaths(t, 1)[0]
					menu := &recordingMenu{}

					service := desktop.NewRecentFiles(path, menu)
					require.Empty(t, menu.latest())

					require.NoError(t, service.Record(recorded))
					later := &recordingMenu{}
					desktop.NewRecentFiles(path, later)
					require.Equal(t, []string{recorded}, later.latest())
				})
			}
		})

		t.Run("keeps from a hand-edited file only what the list could have written", func(t *testing.T) {
			path := listPath(t)
			paths := modelPaths(t, 12)
			edited := append([]string{"", "relative.emod", paths[0], paths[0]}, paths...)
			raw, err := json.Marshal(map[string][]string{"recent": edited})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, raw, 0o600))
			menu := &recordingMenu{}

			desktop.NewRecentFiles(path, menu)

			require.Equal(t, paths[:10], menu.latest())
		})
	})

	t.Run("recording", func(t *testing.T) {
		t.Run("puts the newest recording at the top", func(t *testing.T) {
			path := listPath(t)
			paths := modelPaths(t, 3)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)

			for _, p := range paths {
				require.NoError(t, service.Record(p))
			}

			require.Equal(t, []string{paths[2], paths[1], paths[0]}, menu.latest())
		})

		t.Run("moves a path already held to the top rather than listing it twice", func(t *testing.T) {
			paths := modelPaths(t, 3)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)
			for _, p := range paths {
				require.NoError(t, service.Record(p))
			}

			require.NoError(t, service.Record(paths[0]))

			require.Equal(t, []string{paths[0], paths[2], paths[1]}, menu.latest())
		})

		t.Run("holds ten, dropping the one recorded longest ago", func(t *testing.T) {
			path := listPath(t)
			paths := modelPaths(t, 11)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)

			for _, p := range paths {
				require.NoError(t, service.Record(p))
			}

			require.Len(t, menu.latest(), 10)
			require.Equal(t, paths[10], menu.latest()[0])
			require.Equal(t, paths[1], menu.latest()[9])
			require.NotContains(t, menu.latest(), paths[0])
			later := &recordingMenu{}
			desktop.NewRecentFiles(path, later)
			require.Equal(t, menu.latest(), later.latest())
		})

		t.Run("resolves a relative spelling to the same entry as the absolute one", func(t *testing.T) {
			dir := t.TempDir()
			absolute := writeFile(t, dir, "billing.emod", test.BillingPayments)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)
			t.Chdir(dir)

			require.NoError(t, service.Record(absolute))
			require.NoError(t, service.Record("billing.emod"))

			require.Equal(t, []string{absolute}, menu.latest())
		})

		t.Run("brings the list's directory into being rather than failing", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "emod", "state", "recent-files.json")
			recorded := modelPaths(t, 1)[0]
			service := desktop.NewRecentFiles(path, nil)

			require.NoError(t, service.Record(recorded))

			later := &recordingMenu{}
			desktop.NewRecentFiles(path, later)
			require.Equal(t, []string{recorded}, later.latest())
		})

		t.Run("answers the failure when the list cannot be written, while the list and the menu already hold the new order", func(t *testing.T) {
			path := t.TempDir()
			recorded := modelPaths(t, 1)[0]
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)

			err := service.Record(recorded)

			require.Error(t, err)
			require.Contains(t, err.Error(), "writing "+path)
			require.Contains(t, err.Error(), "is a directory")
			require.Equal(t, []string{recorded}, menu.latest())
		})
	})

	t.Run("opening", func(t *testing.T) {
		t.Run("answers the same document the file dialog's read answers for the same file", func(t *testing.T) {
			path := writeFile(t, t.TempDir(), "billing.emod", test.BillingPayments)
			service := desktop.NewRecentFiles(listPath(t), nil)

			answer := openRecent(t, service, path)

			require.Empty(t, answer.Error)
			require.Equal(t, "billing.emod", answer.Name)
			require.Equal(t, path, answer.Path)
			require.Equal(t, test.BillingPayments, answer.Content)
			require.Equal(t, (&desktop.FileService{}).Read(path), service.Open(path))
		})

		t.Run("refuses a file that is not valid UTF-8, exactly as the file dialog's read does", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "latin1.emod")
			require.NoError(t, os.WriteFile(path, []byte("emod 1\nmodel \"R\xe9servations\"\n"), 0o600))
			service := desktop.NewRecentFiles(listPath(t), nil)

			answer := openRecent(t, service, path)

			require.Contains(t, answer.Error, "not valid UTF-8")
			require.Contains(t, answer.Error, path)
			require.Empty(t, answer.Content)
			require.Equal(t, (&desktop.FileService{}).Read(path), service.Open(path))
		})

		t.Run("leaves the list's order alone, because opening is not recording", func(t *testing.T) {
			dir := t.TempDir()
			first := writeFile(t, dir, "first.emod", test.BillingPayments)
			second := writeFile(t, dir, "second.emod", test.BillingPayments)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)
			require.NoError(t, service.Record(first))
			require.NoError(t, service.Record(second))
			shown := len(menu.sequence())

			require.Empty(t, openRecent(t, service, first).Error)

			require.Equal(t, []string{second, first}, menu.latest())
			require.Len(t, menu.sequence(), shown)
		})

		t.Run("forgets an entry whose file is gone, says so, tells the menu and saves the shorter list", func(t *testing.T) {
			path := listPath(t)
			dir := t.TempDir()
			gone := writeFile(t, dir, "gone.emod", test.BillingPayments)
			kept := writeFile(t, dir, "kept.emod", test.BillingPayments)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)
			require.NoError(t, service.Record(kept))
			require.NoError(t, service.Record(gone))
			require.NoError(t, os.Remove(gone))

			answer := openRecent(t, service, gone)

			require.Contains(t, answer.Error, "gone.emod is no longer at "+gone)
			require.Contains(t, answer.Error, "removed from the recent files")
			require.Empty(t, answer.Content)
			require.Equal(t, []string{kept}, menu.latest())
			later := &recordingMenu{}
			desktop.NewRecentFiles(path, later)
			require.Equal(t, []string{kept}, later.latest())
		})

		t.Run("keeps an entry the filesystem refused for any other reason, and says why", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("root reads a mode 000 file whatever its permissions say")
			}
			secret := writeFile(t, t.TempDir(), "secret.emod", test.BillingPayments)
			require.NoError(t, os.Chmod(secret, 0o000))
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)
			require.NoError(t, service.Record(secret))
			shown := len(menu.sequence())

			answer := openRecent(t, service, secret)

			require.Contains(t, answer.Error, "permission denied")
			require.Contains(t, answer.Error, secret)
			require.NotContains(t, answer.Error, "removed")
			require.Equal(t, []string{secret}, menu.latest())
			require.Len(t, menu.sequence(), shown)
		})

		t.Run("reports a missing file that was never listed without touching the list", func(t *testing.T) {
			listed := modelPaths(t, 1)[0]
			unlisted := filepath.Join(t.TempDir(), "unlisted.emod")
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)
			require.NoError(t, service.Record(listed))
			shown := len(menu.sequence())

			answer := openRecent(t, service, unlisted)

			require.Contains(t, answer.Error, "unlisted.emod is no longer at "+unlisted)
			require.NotContains(t, answer.Error, "removed")
			require.Equal(t, []string{listed}, menu.latest())
			require.Len(t, menu.sequence(), shown)
		})

		t.Run("forgets a gone entry even when the shorter list cannot be saved, and says both", func(t *testing.T) {
			path := t.TempDir()
			gone := modelPaths(t, 1)[0]
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)
			require.Error(t, service.Record(gone))

			answer := openRecent(t, service, gone)

			require.Contains(t, answer.Error, "no longer at "+gone)
			require.Contains(t, answer.Error, "removed from the recent files")
			require.Contains(t, answer.Error, "could not be saved")
			require.Contains(t, answer.Error, "is a directory")
			require.Empty(t, menu.latest())
		})

		t.Run("states the path once in a reason, not twice the way the wrapped syscall error does", func(t *testing.T) {
			gone := filepath.Join(t.TempDir(), "absent.emod")
			service := desktop.NewRecentFiles(listPath(t), nil)

			answer := openRecent(t, service, gone)

			require.Equal(t, 1, strings.Count(answer.Error, gone), "the reason repeats the path: %s", answer.Error)
		})
	})

	t.Run("clearing", func(t *testing.T) {
		t.Run("empties the list, tells the menu, and a later run finds nothing", func(t *testing.T) {
			path := listPath(t)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)
			for _, p := range modelPaths(t, 3) {
				require.NoError(t, service.Record(p))
			}

			require.NoError(t, service.Clear())

			require.Empty(t, menu.latest())
			later := &recordingMenu{}
			desktop.NewRecentFiles(path, later)
			require.Empty(t, later.latest())
		})

		t.Run("answers the failure when the empty list cannot be written", func(t *testing.T) {
			path := t.TempDir()
			service := desktop.NewRecentFiles(path, nil)

			err := service.Clear()

			require.Error(t, err)
			require.Contains(t, err.Error(), "writing "+path)
		})
	})

	t.Run("showing the menu", func(t *testing.T) {
		t.Run("tells the menu every change, carrying the list as it now stands", func(t *testing.T) {
			paths := modelPaths(t, 2)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(listPath(t), menu)

			require.NoError(t, service.Record(paths[0]))
			require.NoError(t, service.Record(paths[1]))
			require.NoError(t, service.Clear())

			require.Equal(t, [][]string{
				{},
				{paths[0]},
				{paths[1], paths[0]},
				{},
			}, menu.sequence())
		})

		// Showing the menu outside the lock lets a second change write the list
		// while the first is still reaching the menu, so what the menu shows can
		// end up contradicting what the list holds. Ordering is what the lock
		// buys, and a count of shows cannot see it: this holds the menu open and
		// requires the next change to be unable to overtake it.
		t.Run("no change overtakes the one still reaching the menu", func(t *testing.T) {
			paths := modelPaths(t, 2)
			menu := newBlockingMenu()
			service := desktop.NewRecentFiles(listPath(t), menu)
			menu.arm()

			go service.Record(paths[0])
			select {
			case <-menu.entered:
			case <-time.After(time.Second):
				require.FailNow(t, "the menu was never reached")
			}

			overtaken := make(chan struct{})
			go func() {
				service.Record(paths[1])
				close(overtaken)
			}()

			select {
			case <-overtaken:
				require.FailNow(t, "a later change reached the list while the menu was still being shown")
			case <-time.After(100 * time.Millisecond):
			}

			close(menu.release)
			select {
			case <-overtaken:
			case <-time.After(time.Second):
				require.FailNow(t, "the change held behind the menu never landed")
			}
		})
	})

	// The frontend records over the bindings, which serve each call its own
	// goroutine, while the shell clears from a menu callback on another — so
	// the list is reached from several goroutines by construction.
	t.Run("concurrent use", func(t *testing.T) {
		t.Run("is safe to record, open and clear from several goroutines at once", func(t *testing.T) {
			path := listPath(t)
			present := writeFile(t, t.TempDir(), "present.emod", test.BillingPayments)
			paths := modelPaths(t, 5)
			menu := &recordingMenu{}
			service := desktop.NewRecentFiles(path, menu)

			var wg sync.WaitGroup
			for i := 0; i < 30; i++ {
				wg.Add(3)
				go func(p string) { defer wg.Done(); service.Record(p) }(paths[i%len(paths)])
				go func() { defer wg.Done(); service.Open(present) }()
				go func(i int) {
					defer wg.Done()
					if i%10 == 0 {
						service.Clear()
					} else {
						service.Open(paths[i%len(paths)])
					}
				}(i)
			}
			wg.Wait()

			later := &recordingMenu{}
			desktop.NewRecentFiles(path, later)
			require.Equal(t, menu.latest(), later.latest())
		})
	})
}

// The one hard requirement RecentMenu states is not something the compiler can
// hold anyone to, and its only implementation lives in cmd/emod-desktop — a
// package no test target builds and no other guard reads. Waiting on the main
// thread there deadlocks every later change behind the service's lock, which is
// a hang rather than a failure and so would reach a user before it reached a
// test.
func TestRecentMenuImplementation(t *testing.T) {
	t.Run("the shell's menu changes its items on the main thread rather than waiting on it", func(t *testing.T) {
		// The whole file rather than Show's body: a wait one call deeper, in a
		// helper Show reaches, hangs the service exactly as one in Show would.
		source, err := os.ReadFile("../../cmd/emod-desktop/recent_menu.go")
		require.NoError(t, err)
		require.NotContains(t, string(source), "InvokeSync",
			"InvokeSync waits for the main thread while the service holds its lock")

		body := methodBody(t, "../../cmd/emod-desktop/recent_menu.go", "recentMenu", "Show")
		// Presence of the dispatch is not enough: a Show that changed the items on
		// the caller's goroutine and dispatched something else would pass that
		// and still touch the native menu off the main thread.
		handedOff := regexp.MustCompile(`(?s)application\.InvokeAsync\(func\(\) \{(.*?)\}\)`).FindStringSubmatch(body)
		require.Len(t, handedOff, 2, "Show must hand a closure to application.InvokeAsync")
		require.Contains(t, handedOff[1], "apply(",
			"the change must be applied inside the closure handed to the main thread")
		// The one place the change may be applied where it is made is before the
		// app runs, and only there: the branch has to be the not-running one, or
		// an inverted test would apply every change off the main thread and hand
		// off none.
		beforeRun := regexp.MustCompile(`(?s)if !m\.running\.Load\(\) \{(.*?)\n\t\}`).FindStringSubmatch(body)
		require.Len(t, beforeRun, 2, "Show must apply the change in place only while the app is not yet running")
		require.Contains(t, beforeRun[1], "apply(")
		require.NotContains(t, beforeRun[1], "InvokeAsync",
			"before the app runs there is no main thread to hand to")
	})

	t.Run("the change handed to the main thread reaches the slots", func(t *testing.T) {
		apply := methodBody(t, "../../cmd/emod-desktop/recent_menu.go", "recentMenu", "apply")

		require.Contains(t, apply, "shown.Show(",
			"a change that never reaches RecentSlots leaves the menu showing the old order")
	})
}

// blockingMenu holds the first change shown to it after arming inside Show until
// the test lets go, so what another goroutine can do meanwhile is observable.
// Arming is what lets the constructor's own show through. Only one call blocks,
// and deliberately not through sync.Once: Do makes every caller wait for the
// first to finish, which would serialise the two callers here and hide the very
// overtaking this exists to catch.
type blockingMenu struct {
	entered chan struct{}
	release chan struct{}

	mu      sync.Mutex
	armed   bool
	blocked bool
}

func newBlockingMenu() *blockingMenu {
	return &blockingMenu{entered: make(chan struct{}), release: make(chan struct{})}
}

func (m *blockingMenu) arm() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.armed = true
}

func (m *blockingMenu) Show([]string) {
	m.mu.Lock()
	first := m.armed && !m.blocked
	if first {
		m.blocked = true
	}
	m.mu.Unlock()

	if !first {
		return
	}
	close(m.entered)
	<-m.release
}
