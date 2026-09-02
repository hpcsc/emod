package desktop

import (
	"path/filepath"
	"sync"
)

// MenuSlot is one place on a native menu that the recent list can fill. The
// shell supplies them, because reaching a native menu item means linking a GUI
// framework and this package deliberately links none.
type MenuSlot interface {
	SetLabel(label string)
	SetHidden(hidden bool)
}

// RecentSlots shows the recent list on a fixed set of menu slots, one for each
// entry the list can hold, by labelling the ones in use and hiding the rest —
// so a change never adds an item to the menu or removes one, which on macOS
// would discard the items AppKit adds to the application menu at launch.
//
// The paths shown move with the labels, under one lock, so a slot never
// answers a path from one order under a label from another. A click the menu
// has already dispatched when the list moves opens the entry now in that
// slot; the window's name says which.
type RecentSlots struct {
	slots []MenuSlot

	mu    sync.Mutex
	paths []string
}

func NewRecentSlots(slots []MenuSlot) *RecentSlots {
	return &RecentSlots{slots: slots}
}

// Show labels one slot per path, in order, telling two entries that share a
// file name apart by their directories, and hides the slots left over. Paths
// beyond the slots are not shown: the list is capped at RecentFilesLimit, which
// is how many slots a shell builds.
func (s *RecentSlots) Show(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(paths) > len(s.slots) {
		paths = paths[:len(s.slots)]
	}
	s.paths = append([]string(nil), paths...)

	labels := recentLabels(paths)
	for i, slot := range s.slots {
		if i < len(labels) {
			slot.SetLabel(labels[i])
			slot.SetHidden(false)
		} else {
			slot.SetHidden(true)
		}
	}
}

// PathAt answers the path shown in slot, or nothing for a slot nothing is shown
// in.
func (s *RecentSlots) PathAt(slot int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slot < 0 || slot >= len(s.paths) {
		return ""
	}

	return s.paths[slot]
}

// recentLabels names each entry by its file name, and by its directory as well
// wherever two entries share a name — the one case a name alone cannot tell
// apart.
func recentLabels(paths []string) []string {
	sharing := map[string]int{}
	for _, path := range paths {
		sharing[filepath.Base(path)]++
	}

	labels := make([]string, len(paths))
	for i, path := range paths {
		name := filepath.Base(path)
		if sharing[name] > 1 {
			name += " — " + filepath.Dir(path)
		}
		labels[i] = name
	}

	return labels
}
