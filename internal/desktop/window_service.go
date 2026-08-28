package desktop

import "sync"

// WindowMarker shows a window as holding work that is not on disk — a dot in
// the close button on macOS, and nothing at all where the platform has no such
// convention and the title carries the answer instead. The shell supplies one,
// because reaching a native window means linking a GUI framework and this
// package deliberately links none.
//
// MarkEdited must not block: it is called while the write lock is held, so an
// implementation that waits on the UI thread — application.InvokeSync rather
// than InvokeAsync — deadlocks every later answer behind it.
type WindowMarker interface {
	MarkEdited(edited bool)
}

// WindowService is what the frontend tells that the model on screen no longer
// matches the file behind it. The shell reads that answer back when it has to
// decide whether closing the window would discard work, which is why the state
// lives here rather than in the frontend that derives it: a window is closed
// from outside the page, and the decision cannot wait on a round trip into it.
type WindowService struct {
	marker WindowMarker

	mu       sync.RWMutex
	modified bool
}

// NewWindowService takes the marker rather than exposing it, so it is settled
// once at construction and nothing can swap it afterwards. A nil marker is
// allowed and does nothing, which is what a service built before there is a
// window to mark needs.
func NewWindowService(marker WindowMarker) *WindowService {
	return &WindowService{marker: marker}
}

// SetModified records the frontend's answer and shows it on the window. The
// frontend states the answer whenever it changes, so this is called once per
// change rather than once per edit.
//
// The marker is told while the lock is held, so two answers cannot reach the
// window in the opposite order to the state they wrote — which would leave the
// dot on the close button contradicting what Modified reports. That is why
// MarkEdited must not block; see WindowMarker above.
func (s *WindowService) SetModified(modified bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.modified = modified
	if s.marker != nil {
		s.marker.MarkEdited(modified)
	}
}

// Modified answers whether closing the window now would discard work.
func (s *WindowService) Modified() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.modified
}
