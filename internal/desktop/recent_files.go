package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/hpcsc/emod/internal/pipeline"
)

// RecentMenu shows the models opened most recently, newest first — File ▸ Open
// Recent on the desktop. The shell supplies one, because reaching a native menu
// means linking a GUI framework and this package deliberately links none.
//
// Show must not block: it is called while the write lock is held, so an
// implementation that waits on the UI thread — application.InvokeSync rather
// than InvokeAsync — deadlocks every later change behind it.
type RecentMenu interface {
	Show(paths []string)
}

// RecentFilesLimit is how many models the list holds; a shell that shows the
// list builds that many places for it.
const RecentFilesLimit = 10

// RecentFiles is the list of models opened most recently, newest first, kept on
// disk between runs. The frontend records what it has opened and reads entries
// back through the bindings; the shell clears the list from its menu and is
// shown every change. The list lives here rather than in the frontend because
// it outlives the page: a reload starts the page over, while the shell keeps
// what has been opened for the life of the process.
//
// Open answers the document envelope FileService.Read answers, so the frontend
// decodes both alike. Record and Clear have nothing to answer but whether the
// list was saved, so each returns that as an error, which the bindings hand the
// frontend as a rejection.
type RecentFiles struct {
	path string
	menu RecentMenu

	mu      sync.Mutex
	entries []string
}

// NewRecentFiles reads the list kept at path — nothing there, or something
// there that is not the list, starts it empty — and shows the menu what it found
// before anything can change it, so a menu built before the app runs is already
// populated. A nil menu is allowed and is shown nothing. An empty path keeps the
// list for this run only, for a shell that has nowhere to keep it.
func NewRecentFiles(path string, menu RecentMenu) *RecentFiles {
	s := &RecentFiles{path: path, menu: menu, entries: loadRecentFiles(path)}
	s.show()

	return s
}

// Record puts path at the top of the list, dropping the entry recorded longest
// ago once the list is full, and puts the list back on disk. The list and the
// menu already hold the new order when the write is refused; the error says
// only that the next run will not find it.
func (s *RecentFiles) Record(path string) error {
	absolute, err := resolveModelPath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append([]string{absolute}, without(s.entries, absolute)...)
	if len(s.entries) > RecentFilesLimit {
		s.entries = s.entries[:RecentFilesLimit]
	}

	return s.changed()
}

// Open answers the listed file exactly as FileService.Read answers a chosen
// one, and no more: where the file then sits in the list is for whoever adopts
// it to record. A file that is no longer there is taken off the list and the
// reason says so; a file the filesystem refuses for any other reason stays
// listed, because it may be back — a volume mounted again, a permission
// restored.
func (s *RecentFiles) Open(path string) string {
	absolute, err := resolveModelPath(path)
	if err != nil {
		return pipeline.ErrorJSON(err.Error())
	}

	opened, err := readModelFile(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return pipeline.ErrorJSON(s.forget(absolute))
	}
	if err != nil {
		return pipeline.ErrorJSON(err.Error())
	}

	return opened.document()
}

// Clear empties the list and puts the empty list back on disk.
func (s *RecentFiles) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = nil

	return s.changed()
}

// RecentLabels names each listed model for a menu: by its file name, and by its
// directory as well wherever two entries share a name — the one case a name
// alone cannot tell apart.
func RecentLabels(paths []string) []string {
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

// forget answers the reason a file that has gone could not be opened, having
// taken it off the list if it was there. A refused write is folded into the
// reason rather than lost: the answer has one channel back to the user, and a
// list that will come back holding the entry is worth a sentence.
func (s *RecentFiles) forget(absolute string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	reason := fmt.Sprintf("%s is no longer at %s", filepath.Base(absolute), absolute)
	remaining := without(s.entries, absolute)
	if len(remaining) == len(s.entries) {
		return reason
	}

	s.entries = remaining
	reason += "; it has been removed from the recent files"
	if err := s.changed(); err != nil {
		reason += ", but the list could not be saved: " + err.Error()
	}

	return reason
}

// changed shows the menu and then writes the list, so a write the filesystem
// refuses never leaves the menu behind what the list holds. Caller must hold mu.
func (s *RecentFiles) changed() error {
	s.show()

	return saveRecentFiles(s.path, s.entries)
}

func (s *RecentFiles) show() {
	if s.menu != nil {
		s.menu.Show(append([]string{}, s.entries...))
	}
}

func without(entries []string, path string) []string {
	kept := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry != path {
			kept = append(kept, entry)
		}
	}

	return kept
}

type recentFilesDocument struct {
	Recent []string `json:"recent"`
}

// loadRecentFiles answers the list a file holds, or nothing where there is no
// file or the file is not the list. What it does hold is filtered to what the
// list could have written — absolute, distinct, no more than the limit — so a
// hand-edited file cannot put the list in a state its own writes never reach.
func loadRecentFiles(path string) []string {
	if path == "" {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var document recentFilesDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil
	}

	var entries []string
	for _, entry := range document.Recent {
		if entry == "" || !filepath.IsAbs(entry) || slices.Contains(entries, entry) {
			continue
		}
		entries = append(entries, entry)
		if len(entries) == RecentFilesLimit {
			break
		}
	}

	return entries
}

// saveRecentFiles writes the list the way a model is written — complete, then
// renamed over the file — so a run interrupted mid-write finds the previous
// list rather than a truncated one.
func saveRecentFiles(path string, entries []string) error {
	if path == "" {
		return nil
	}

	raw, err := json.MarshalIndent(recentFilesDocument{Recent: append([]string{}, entries...)}, "", "  ")
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("writing %s: %w", path, failureReason(err))
	}
	if err := replaceFile(path, string(raw)+"\n"); err != nil {
		return fmt.Errorf("writing %s: %w", path, failureReason(err))
	}

	return nil
}
