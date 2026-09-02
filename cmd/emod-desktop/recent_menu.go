package main

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/hpcsc/emod/internal/desktop"
)

// recentMenu is File ▸ Open Recent: what the shell shows of desktop.RecentFiles.
// It is built once, with an item for every place the list can fill, and shows
// each change by relabelling and hiding those items where they are. Rebuilding
// the application menu is the only way the framework can add or remove an item
// once the app runs, and on macOS a rebuild discards the items AppKit added to
// the menu at launch — Start Dictation, Emoji & Symbols, AutoFill, Close All
// and the fullscreen toggle — so nothing here rebuilds.
//
// Choosing an entry reaches the frontend by event, like Open does, because the
// read and the delivery live there; Clear Menu reaches the service directly,
// because nothing about clearing needs the page.
type recentMenu struct {
	window    *application.WebviewWindow
	items     *application.Menu
	slots     []*application.MenuItem
	clear     *application.MenuItem
	clearList func() error
	running   atomic.Bool

	mu    sync.Mutex
	paths []string
}

var _ desktop.RecentMenu = (*recentMenu)(nil)

// clearList is called when Clear Menu is chosen, which can only be once the app
// runs — so it may reach a service built after this menu, which it has to: the
// service is constructed over this menu and shows it the saved list before
// returning.
func newRecentMenu(window *application.WebviewWindow, clearList func() error) *recentMenu {
	m := &recentMenu{window: window, items: application.NewMenu(), clearList: clearList}
	for slot := 0; slot < desktop.RecentFilesLimit; slot++ {
		m.slots = append(m.slots, m.items.Add("").SetHidden(true).OnClick(func(*application.Context) {
			m.open(slot)
		}))
	}
	m.items.AddSeparator()
	// A refused write leaves the list cleared and the menu showing it, and the
	// next launch brings the entries back. Only the log can carry that from
	// here — nothing in the shell reaches the bar a refused recording goes to —
	// and a directory that refuses this write refuses every recording, which the
	// bar does report.
	m.clear = m.items.Add("Clear Menu").SetEnabled(false).OnClick(func(*application.Context) {
		if err := m.clearList(); err != nil {
			log.Printf("recent files: %s", err)
		}
	})

	return m
}

// started records that the app is running, which is when the framework's main
// thread dispatch exists. Before that the saved list is shown while the menu is
// still being built, ahead of Run, and a change is applied where it is made:
// nothing else can reach the items until the app runs.
func (m *recentMenu) started() {
	m.running.Store(true)
}

// Show must not wait on the main thread — it is called while the service holds
// its lock — so the change is handed off to it, where the framework's own item
// calls belong.
func (m *recentMenu) Show(paths []string) {
	if !m.running.Load() {
		m.apply(paths)
		return
	}
	application.InvokeAsync(func() { m.apply(paths) })
}

// The paths and the labels move together, in one turn of the main thread, so
// the menu never shows one order over another. A click the main thread has
// already dispatched when the list moves opens the entry now in that place;
// the window's name says which.
func (m *recentMenu) apply(paths []string) {
	m.mu.Lock()
	m.paths = append([]string(nil), paths...)
	m.mu.Unlock()

	labels := desktop.RecentLabels(paths)
	for i, slot := range m.slots {
		if i < len(labels) {
			slot.SetLabel(labels[i])
			slot.SetHidden(false)
		} else {
			slot.SetHidden(true)
		}
	}
	m.clear.SetEnabled(len(paths) > 0)
}

func (m *recentMenu) open(slot int) {
	m.mu.Lock()
	path := ""
	if slot < len(m.paths) {
		path = m.paths[slot]
	}
	m.mu.Unlock()

	if path == "" {
		return
	}
	m.window.EmitEvent("file:open-recent-requested", path)
}
