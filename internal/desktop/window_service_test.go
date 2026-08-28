//go:build unit

package desktop_test

import (
	"sync"
	"testing"

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
			service := &desktop.WindowService{}

			require.False(t, service.Modified())
		})

		t.Run("reports back the answer it was last given", func(t *testing.T) {
			service := &desktop.WindowService{}

			service.SetModified(true)
			require.True(t, service.Modified())

			service.SetModified(false)
			require.False(t, service.Modified())
		})
	})

	t.Run("marking the window", func(t *testing.T) {
		t.Run("hands the marker every answer it is given, in the order it was given them", func(t *testing.T) {
			marker := &recordingMarker{}
			service := &desktop.WindowService{Marker: marker}

			service.SetModified(true)
			service.SetModified(false)
			service.SetModified(true)

			require.Equal(t, []bool{true, false, true}, marker.sequence())
		})

		// The shell supplies the marker, and nothing but the shell does — so a
		// service standing on its own has to answer rather than panic.
		t.Run("still records the answer when no marker was supplied", func(t *testing.T) {
			service := &desktop.WindowService{}

			require.NotPanics(t, func() { service.SetModified(true) })
			require.True(t, service.Modified())
		})
	})

	// The frontend writes this over the bindings while the shell reads it from
	// the thread deciding whether a close would discard work, so the two reach
	// it from different goroutines by construction.
	t.Run("concurrent use", func(t *testing.T) {
		t.Run("is safe to write and read from several goroutines at once", func(t *testing.T) {
			marker := &recordingMarker{}
			service := &desktop.WindowService{Marker: marker}

			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(2)
				go func(edited bool) { defer wg.Done(); service.SetModified(edited) }(i%2 == 0)
				go func() { defer wg.Done(); service.Modified() }()
			}
			wg.Wait()

			require.Len(t, marker.sequence(), 50)
		})
	})
}
