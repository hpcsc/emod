// Command emod-desktop runs the emod viewer as a native desktop application.
// It is the only binary in the repository that links CGO, and it is quarantined
// in its own package and its own build task for that reason.
package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/hpcsc/emod/internal/desktop"
)

// frontend is written by `task build:desktop` and is not tracked, so this
// package does not compile until that task has assembled it — which is why it
// is excluded from the package sets `task test:unit` and `task test:integration`
// enumerate.
//
//go:embed all:frontend
var frontend embed.FS

// applicationMenu extends the framework's default menu rather than replacing it:
// a menu built from scratch has no Quit, no Copy and no Paste, which on macOS
// are the application's only bindings for them.
//
// Every item here reaches the frontend by event, because that is where the
// pickers and the model's own source live. Raising them here instead would be
// the shorter path, and is rejected: the reads and writes behind them belong to
// internal/desktop, which imports no GUI framework so that it stays testable,
// and this package is in no test target at all. internal/desktop's event-name
// guard pins each name against the frontend's subscription.
func applicationMenu(window *application.WebviewWindow, openRecent *application.MenuItem) *application.Menu {
	menu := application.DefaultApplicationMenu()

	open := application.NewMenuItem("Open…").
		SetAccelerator("CmdOrCtrl+O").
		OnClick(func(*application.Context) {
			window.EmitEvent("file:open-requested")
		})
	// The framework's own items, so Save and Save As carry whatever each
	// platform's standard accelerator for them is rather than this file's idea
	// of it. Open is built by hand because the framework's carries a role, and
	// a role is answered by the framework instead of reaching the frontend.
	save := application.NewSaveMenuItem().
		OnClick(func(*application.Context) {
			window.EmitEvent("file:save-requested")
		})
	saveAs := application.NewSaveAsMenuItem().
		SetLabel("Save As…").
		OnClick(func(*application.Context) {
			window.EmitEvent("file:save-as-requested")
		})

	fileItems := application.NewMenuFromItems(open, openRecent, save, saveAs)
	fileItems.AddSeparator()
	menu.FindByLabel("File").GetSubmenu().Prepend(fileItems)

	return menu
}

// recentFilesPath is where the models opened most recently are listed between
// runs: the platform's own per-user configuration directory, which is what the
// CLI's ~/.config/emod resolves to on Linux and is somewhere else on macOS and
// Windows. A machine that cannot name that directory keeps the list for this
// run only rather than somewhere shared: a fixed path under the temporary
// directory is one another user could plant a link at for the write to follow.
func recentFilesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("recent files: %s; keeping the list for this run only", err)

		return ""
	}

	return filepath.Join(dir, "emod", "recent-files.json")
}

func main() {
	// Both are assigned below and read only from callbacks the framework runs
	// after Run, which is what lets a quit veto built here consult a window and
	// a service that do not exist yet. Nothing enforces that ordering, so the
	// veto answers for itself rather than trusting it: a quit that somehow
	// arrived first would have no window to have unsaved work in.
	var window *application.WebviewWindow
	var windowState *desktop.WindowService

	app := application.New(application.Options{
		Name:        "emod",
		Description: "Event model viewer",
		Services: []application.Service{
			application.NewService(&desktop.ModelService{}),
			application.NewService(&desktop.FileService{}),
		},
		Assets: application.AssetOptions{
			// Bundled rather than plain: the generated bindings import
			// /wails/runtime.js, which only this server answers.
			Handler: application.BundledAssetFileServer(frontend),
		},
		// A quit carrying unsaved work is refused rather than confirmed: the
		// confirmation is a dialog the frontend raises, and this has to answer
		// before one could be shown, let alone answered. The frontend asks to
		// quit again once it has one, by which point this state has moved.
		ShouldQuit: func() bool {
			if windowState == nil || !windowState.Modified() {
				return true
			}
			window.EmitEvent("app:quit-requested")

			return false
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "emod",
		Width:  1400,
		Height: 900,
		// Windows adopts an application menu into a window only on this flag.
		// macOS and Linux find it without one — macOS because its menu is
		// always global, Linux because a window with no menu of its own falls
		// back to it — so leaving this off loses File ▸ Open on Windows alone.
		UseApplicationMenu: true,
		// Without this the webview handles a file drag itself and hands the page
		// contents with no location, which is what a browser can already do and
		// is not what a drop on this window is for. With it, the platform's own
		// drag destination takes the drag and answers the real paths below.
		EnableFileDrop: true,
	})

	// Registered after the window rather than in Options.Services, because the
	// marker needs the window and the window needs the app. Wails binds every
	// service registered before Run.
	windowState = desktop.NewWindowService(&windowMarker{window: window})
	app.RegisterService(application.NewService(windowState))

	// Built over the menu so the saved list is on it before the app runs, and
	// the menu clears through a service that does not exist until then, the way
	// the quit veto above reads state assigned after it. Wails binds every
	// service registered before Run.
	var recent *desktop.RecentFiles
	openRecent := newRecentMenu(window, func() error { return recent.Clear() })
	recent = desktop.NewRecentFiles(recentFilesPath(), openRecent)
	app.RegisterService(application.NewService(recent))
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		openRecent.started()
	})

	// A listener rather than a hook: the framework dispatches a resolved file
	// drop only to the listeners OnWindowEvent appends to, so a hook registered
	// for it — the shape the close veto below takes, for a different reason —
	// would never be called at all.
	//
	// The paths go straight on to the frontend because everything that decides
	// what to do with them is there: which of several files to open, what to say
	// about one it cannot, and whether opening it would discard unsaved work.
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		window.EmitEvent("file:dropped", event.Context().DroppedFiles())
	})

	// A hook runs ahead of the listener that destroys the window, and
	// cancelling stops it — so this is where a close carrying unsaved work is
	// refused. The frontend asks to close again once it has an answer, and by
	// then this hook has nothing to refuse over and lets it through.
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !windowState.Modified() {
			return
		}
		event.Cancel()
		window.EmitEvent("window:close-requested")
	})

	app.Menu.SetApplicationMenu(applicationMenu(window, application.NewSubmenu("Open Recent", openRecent.items)))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
