package desktop

import "sync"

// OpenRequests holds the model the operating system last asked the app to open,
// until the page takes it. A launch the system started in order to open a file
// hands the path over before the page exists, so the request waits here rather
// than being sent at a window nothing is listening in yet.
//
// Nothing here records whether a page is listening. The page is reloaded while
// this lives for the whole process, so such a record would be wrong in exactly
// the window where the file is lost — between one page going and the next
// subscribing. A page that can receive asks instead, and asks again after every
// reload.
type OpenRequests struct {
	mu      sync.Mutex
	waiting string
}

// Hold records path as the model to open next. A request nobody has taken is
// replaced rather than queued: the newest is the one the user asked for, and
// this window holds one model at a time.
func (s *OpenRequests) Hold(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.waiting = path
}

// Take answers the path waiting for the page and forgets it, or answers nothing
// when no request is waiting. Two takers at once is the ordinary case — the page
// asks as it starts, and the shell tells it to ask again when a request arrives
// — and only one of them may be given the file.
func (s *OpenRequests) Take() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	waiting := s.waiting
	s.waiting = ""

	return waiting
}
