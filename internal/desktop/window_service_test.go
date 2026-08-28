//go:build unit

package desktop_test

import (
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/stretchr/testify/require"
)

// recordingMarker stands in for the window, which no test has: the shell's real
// marker reaches an NSWindow, and this one only remembers what it was told.
type recordingMarker struct {
	mu   sync.Mutex
	told []bool
}

func (m *recordingMarker) MarkEdited(edited bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.told = append(m.told, edited)
}

func (m *recordingMarker) sequence() []bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]bool(nil), m.told...)
}

func TestWindowService(t *testing.T) {
	t.Run("modified", func(t *testing.T) {
		t.Run("a window nobody has told anything holds no unsaved work", func(t *testing.T) {
			service := desktop.NewWindowService(nil)

			require.False(t, service.Modified())
		})

		t.Run("reports back the answer it was last given", func(t *testing.T) {
			service := desktop.NewWindowService(nil)

			service.SetModified(true)
			require.True(t, service.Modified())

			service.SetModified(false)
			require.False(t, service.Modified())
		})
	})

	t.Run("marking the window", func(t *testing.T) {
		t.Run("hands the marker every answer it is given, in the order it was given them", func(t *testing.T) {
			marker := &recordingMarker{}
			service := desktop.NewWindowService(marker)

			service.SetModified(true)
			service.SetModified(false)
			service.SetModified(true)

			require.Equal(t, []bool{true, false, true}, marker.sequence())
		})
	})

	// The frontend writes this over the bindings while the shell reads it from
	// the thread deciding whether a close would discard work, so the two reach
	// it from different goroutines by construction.
	t.Run("concurrent use", func(t *testing.T) {
		t.Run("is safe to write and read from several goroutines at once", func(t *testing.T) {
			marker := &recordingMarker{}
			service := desktop.NewWindowService(marker)

			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(2)
				go func(edited bool) { defer wg.Done(); service.SetModified(edited) }(i%2 == 0)
				go func() { defer wg.Done(); service.Modified() }()
			}
			wg.Wait()

			require.Len(t, marker.sequence(), 50)
		})

		// Marking the window outside the lock lets a second answer write the
		// state while the first is still reaching the window, so the dot on the
		// close button ends up contradicting what Modified reports. Ordering is
		// what the lock buys, and a count of marks cannot see it: this holds the
		// marker open and requires the next answer to be unable to overtake it.
		t.Run("no answer overtakes the one still reaching the window", func(t *testing.T) {
			marker := newBlockingMarker()
			service := desktop.NewWindowService(marker)

			go service.SetModified(true)
			select {
			case <-marker.entered:
			case <-time.After(time.Second):
				require.FailNow(t, "the marker was never reached")
			}

			overtaken := make(chan struct{})
			go func() {
				service.SetModified(false)
				close(overtaken)
			}()

			select {
			case <-overtaken:
				require.FailNow(t, "a later answer reached the state while the window was still being marked")
			case <-time.After(100 * time.Millisecond):
			}

			close(marker.release)
			select {
			case <-overtaken:
			case <-time.After(time.Second):
				require.FailNow(t, "the answer held behind the marker never landed")
			}
		})
	})
}

// blockingMarker holds the first SetModified inside MarkEdited until the test
// lets go, so what another goroutine can do meanwhile is observable. Only the
// first call blocks, and deliberately not through sync.Once: Do makes every
// caller wait for the first to finish, which would serialise the two callers
// here and hide the very overtaking this exists to catch.
type blockingMarker struct {
	entered chan struct{}
	release chan struct{}

	mu      sync.Mutex
	blocked bool
}

func newBlockingMarker() *blockingMarker {
	return &blockingMarker{entered: make(chan struct{}), release: make(chan struct{})}
}

func (m *blockingMarker) MarkEdited(bool) {
	m.mu.Lock()
	first := !m.blocked
	m.blocked = true
	m.mu.Unlock()

	if !first {
		return
	}
	close(m.entered)
	<-m.release
}

// The one hard requirement WindowMarker states is not something the compiler
// can hold anyone to, and its only implementation lives in cmd/emod-desktop —
// a package no test target builds and no other guard reads. Blocking there
// deadlocks every later answer behind the write lock, which is a hang rather
// than a failure and so would reach a user before it reached a test.
func TestWindowMarkerImplementation(t *testing.T) {
	t.Run("the shell's marker hands the window off rather than waiting on it", func(t *testing.T) {
		body := markEditedBody(t, "../../cmd/emod-desktop/window_marker_darwin.go")

		require.Contains(t, body, "application.InvokeAsync",
			"MarkEdited must dispatch onto the main thread, not run on it")
		require.NotContains(t, body, "InvokeSync",
			"InvokeSync waits for the main thread while SetModified holds the write lock")
	})
}

func markEditedBody(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	body := regexp.MustCompile(`(?s)func \(\w+ \*windowMarker\) MarkEdited\([^)]*\) \{(.*?)\n\}`).
		FindStringSubmatch(string(raw))
	require.Len(t, body, 2, path+" must declare a windowMarker.MarkEdited method")

	return body[1]
}
