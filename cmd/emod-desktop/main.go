// Command emod-desktop runs the emod viewer as a native desktop application.
// It is the only binary in the repository that links CGO, and it is quarantined
// in its own package and its own build task for that reason.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

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
func applicationMenu(window *application.WebviewWindow) *application.Menu {
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

	fileItems := application.NewMenuFromItems(open, save, saveAs)
	fileItems.AddSeparator()
	menu.FindByLabel("File").GetSubmenu().Prepend(fileItems)

	return menu
}

func main() {
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
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "emod",
		Width:  1400,
		Height: 900,
		// Windows adopts an application menu into a window only on this flag.
		// macOS and Linux find it without one — macOS because its menu is
		// always global, Linux because a window with no menu of its own falls
		// back to it — so leaving this off loses File ▸ Open on Windows alone.
		UseApplicationMenu: true,
	})

	// Registered after the window rather than in Options.Services, because the
	// marker needs the window and the window needs the app. Wails binds every
	// service registered before Run.
	app.RegisterService(application.NewService(&desktop.WindowService{
		Marker: &windowMarker{window: window},
	}))

	app.Menu.SetApplicationMenu(applicationMenu(window))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
