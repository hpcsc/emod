package desktop

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"time"
)

// checksBeforeMissing is how many checks in a row must find no file before the
// file is reported missing. An editor that keeps a backup renames the original
// away and only then writes its replacement, and one check can come between the
// two.
const checksBeforeMissing = 2

// coarsestTimeStep is the step of the coarsest modification time a supported
// filesystem keeps: FAT keeps times in two-second steps.
const coarsestTimeStep = 2 * time.Second

// OpenFileListener hears what a check finds at the path of the file the window
// has open. The shell supplies one, because reaching the page means linking a
// GUI framework and this package deliberately links none.
type OpenFileListener interface {
	ChangedOnDisk(path string)
	MissingFromDisk(path string)
}

// Each check stats the path, because an editor that renames a replacement over
// the file leaves the old file unwritten. A reported change only starts the
// page's comparison of the bytes, which is what finds the app's own saves.
type OpenFileWatch struct {
	listener OpenFileListener

	mu            sync.Mutex
	path          string
	last          os.FileInfo
	lastSeenAt    time.Time
	missingChecks int
}

func NewOpenFileWatch(listener OpenFileListener) *OpenFileWatch {
	return &OpenFileWatch{listener: listener}
}

// The first check after Watch reports the file as a change, so the page also
// sees a write made between its read of the file and this call.
func (w *OpenFileWatch) Watch(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.path = path
	w.last = nil
	w.missingChecks = 0
}

// Stop ends the watch, so a check that starts after it reports nothing.
func (w *OpenFileWatch) Stop() {
	w.Watch("")
}

// Check looks at the watched path once and tells the listener what is new
// there. Neither the stat nor the listener runs under the lock that Watch and
// Stop take: a stat on a volume that has stopped answering can take minutes, and
// the shell's listener waits on the UI thread.
//
//wails:ignore
func (w *OpenFileWatch) Check() {
	path := w.watched()
	if path == "" {
		return
	}

	checkedAt := time.Now()
	info, err := os.Stat(path)
	if tell := w.update(path, info, err, checkedAt); tell != nil {
		tell(w.listener, path)
	}
}

func (w *OpenFileWatch) watched() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.path
}

// update keeps what a stat of path found and answers the listener method that
// reports what is new there, or nil.
func (w *OpenFileWatch) update(path string, info os.FileInfo, err error, checkedAt time.Time) func(OpenFileListener, string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if path != w.path {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		w.missingChecks++
		if w.missingChecks != checksBeforeMissing {
			return nil
		}
		w.last = nil

		return OpenFileListener.MissingFromDisk
	}
	if err != nil {
		return nil
	}

	w.missingChecks = 0
	if w.provesUnwritten(info) {
		return nil
	}
	w.last = info
	w.lastSeenAt = checkedAt

	return OpenFileListener.ChangedOnDisk
}

// provesUnwritten answers whether info is the file the last report saw, with
// nothing to show it was written since. A match proves nothing while that report
// came less than one time step from the file's modification time, because a
// write in that same step can keep both the size and the time. Caller must
// hold mu.
func (w *OpenFileWatch) provesUnwritten(info os.FileInfo) bool {
	if w.last == nil || !os.SameFile(w.last, info) || w.last.Size() != info.Size() || !w.last.ModTime().Equal(info.ModTime()) {
		return false
	}
	sinceWritten := w.lastSeenAt.Sub(w.last.ModTime())

	return sinceWritten <= -coarsestTimeStep || sinceWritten >= coarsestTimeStep
}
