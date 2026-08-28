package desktop

import "sync"

// WindowMarker shows a window as holding work that is not on disk — a dot in
// the close button on macOS, and nothing at all where the platform has no such
// convention and the title carries the answer instead. The shell supplies one,
// because reaching a native window means linking a GUI framework and this
// package deliberately links none.
type WindowMarker interface {
	MarkEdited(edited bool)
}

// WindowService is what the frontend tells that the model on screen no longer
// matches the file behind it. The shell reads that answer back when it has to
// decide whether closing the window would discard work, which is why the state
// lives here rather than in the frontend that derives it: a window is closed
// from outside the page, and the decision cannot wait on a round trip into it.
type WindowService struct {
	Marker WindowMarker

	mu       sync.RWMutex
	modified bool
}

// SetModified records the frontend's answer and shows it on the window. The
// frontend states the answer whenever it changes, so this is called once per
// change rather than once per edit.
func (s *WindowService) SetModified(modified bool) {
	s.mu.Lock()
	s.modified = modified
	marker := s.Marker
	s.mu.Unlock()

	if marker != nil {
		marker.MarkEdited(modified)
	}
}

// Modified answers whether closing the window now would discard work.
func (s *WindowService) Modified() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.modified
}
