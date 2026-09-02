//go:build unit

package desktop_test

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/stretchr/testify/require"
)

// recordingSlot stands in for a native menu item, which no test has: it only
// remembers what it was last told.
type recordingSlot struct {
	mu     sync.Mutex
	label  string
	hidden bool
}

func (s *recordingSlot) SetLabel(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = label
}

func (s *recordingSlot) SetHidden(hidden bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hidden = hidden
}

type shownSlot struct {
	Label  string
	Hidden bool
}

func slotsOf(n int) ([]desktop.MenuSlot, func() []shownSlot) {
	recorders := make([]*recordingSlot, n)
	slots := make([]desktop.MenuSlot, n)
	for i := range recorders {
		recorders[i] = &recordingSlot{}
		slots[i] = recorders[i]
	}

	return slots, func() []shownSlot {
		shown := make([]shownSlot, n)
		for i, r := range recorders {
			r.mu.Lock()
			shown[i] = shownSlot{Label: r.label, Hidden: r.hidden}
			r.mu.Unlock()
		}

		return shown
	}
}

func TestRecentSlots(t *testing.T) {
	t.Run("showing", func(t *testing.T) {
		t.Run("labels one slot per path, in order, and hides the slots left over", func(t *testing.T) {
			slots, shown := slotsOf(5)
			recent := desktop.NewRecentSlots(slots)

			recent.Show([]string{"/models/billing.emod", "/archive/hotel.emod", "/work/library.emod"})

			require.Equal(t, []shownSlot{
				{Label: "billing.emod"},
				{Label: "hotel.emod"},
				{Label: "library.emod"},
				{Hidden: true},
				{Hidden: true},
			}, shown())
		})

		t.Run("tells two entries sharing a file name apart by their directories, and those alone", func(t *testing.T) {
			slots, shown := slotsOf(3)
			recent := desktop.NewRecentSlots(slots)

			recent.Show([]string{"/work/model.emod", "/models/hotel.emod", "/archive/model.emod"})

			require.Equal(t, []shownSlot{
				{Label: "model.emod — /work"},
				{Label: "hotel.emod"},
				{Label: "model.emod — /archive"},
			}, shown())
		})

		t.Run("hides every slot again once the list is cleared", func(t *testing.T) {
			slots, shown := slotsOf(3)
			recent := desktop.NewRecentSlots(slots)
			recent.Show([]string{"/models/billing.emod", "/archive/hotel.emod"})

			recent.Show(nil)

			for _, slot := range shown() {
				require.True(t, slot.Hidden)
			}
		})

		t.Run("shows no more paths than it has slots", func(t *testing.T) {
			slots, shown := slotsOf(2)
			recent := desktop.NewRecentSlots(slots)

			recent.Show([]string{"/models/billing.emod", "/archive/hotel.emod", "/work/library.emod"})

			require.Equal(t, []shownSlot{{Label: "billing.emod"}, {Label: "hotel.emod"}}, shown())
			require.Equal(t, "", recent.PathAt(2))
		})
	})

	t.Run("answering a slot's path", func(t *testing.T) {
		t.Run("answers the path shown in the slot, and nothing for a slot nothing is shown in", func(t *testing.T) {
			slots, _ := slotsOf(4)
			recent := desktop.NewRecentSlots(slots)
			recent.Show([]string{"/models/billing.emod", "/archive/hotel.emod"})

			require.Equal(t, "/archive/hotel.emod", recent.PathAt(1))
			require.Equal(t, "", recent.PathAt(2))
			require.Equal(t, "", recent.PathAt(-1))
		})

		t.Run("answers the path of the order shown last", func(t *testing.T) {
			slots, _ := slotsOf(4)
			recent := desktop.NewRecentSlots(slots)
			recent.Show([]string{"/models/billing.emod", "/archive/hotel.emod"})

			recent.Show([]string{"/archive/hotel.emod", "/models/billing.emod"})

			require.Equal(t, "/archive/hotel.emod", recent.PathAt(0))
		})
	})

	// The shell shows changes from the main thread and answers clicks from the
	// goroutine the framework runs each callback on.
	t.Run("concurrent use", func(t *testing.T) {
		t.Run("leaves every slot's label naming the path it answers, however the calls interleave", func(t *testing.T) {
			slots, shown := slotsOf(3)
			recent := desktop.NewRecentSlots(slots)
			orders := [][]string{{"/a.emod"}, {"/b.emod", "/a.emod"}, nil}

			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(2)
				go func(order []string) { defer wg.Done(); recent.Show(order) }(orders[i%len(orders)])
				go func() { defer wg.Done(); recent.PathAt(1) }()
			}
			wg.Wait()

			for i, slot := range shown() {
				path := recent.PathAt(i)
				if slot.Hidden {
					require.Equal(t, "", path, "slot %d is hidden yet answers a path", i)
				} else {
					require.Equal(t, filepath.Base(path), slot.Label, "slot %d names one file and answers another", i)
				}
			}
		})

		// Labelling the slots outside the lock lets a click read the new order
		// while the menu still shows the old one, so the file opened is not the
		// one the label named. Ordering is what the lock buys, and the invariant
		// above can hold by luck: this holds a label open and requires the path
		// behind it to be unanswerable until it is shown.
		t.Run("answers no path from an order still being shown", func(t *testing.T) {
			blocking := newBlockingSlot()
			recent := desktop.NewRecentSlots([]desktop.MenuSlot{blocking})

			go recent.Show([]string{"/new.emod"})
			select {
			case <-blocking.entered:
			case <-time.After(time.Second):
				require.FailNow(t, "the slot was never labelled")
			}

			answered := make(chan string, 1)
			go func() { answered <- recent.PathAt(0) }()

			select {
			case <-answered:
				require.FailNow(t, "a path was answered while its order was still being shown")
			case <-time.After(100 * time.Millisecond):
			}

			close(blocking.release)
			select {
			case path := <-answered:
				require.Equal(t, "/new.emod", path)
			case <-time.After(time.Second):
				require.FailNow(t, "the path held behind the label never came")
			}
		})
	})
}

// blockingSlot holds the first label it is given until the test lets go, so
// what another goroutine can do meanwhile is observable. Only the first call
// blocks, and deliberately not through sync.Once: Do makes every caller wait
// for the first to finish, which would serialise the callers and hide the very
// overtaking this exists to catch.
type blockingSlot struct {
	entered chan struct{}
	release chan struct{}

	mu      sync.Mutex
	blocked bool
}

func newBlockingSlot() *blockingSlot {
	return &blockingSlot{entered: make(chan struct{}), release: make(chan struct{})}
}

func (s *blockingSlot) SetLabel(string) {
	s.mu.Lock()
	first := !s.blocked
	s.blocked = true
	s.mu.Unlock()

	if !first {
		return
	}
	close(s.entered)
	<-s.release
}

func (s *blockingSlot) SetHidden(bool) {}
