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
// Open reaches the frontend by event because that is where the picker lives —
// internal/desktop imports no GUI framework, so the dialog cannot be raised
// from Go. internal/desktop's event-name guard pins the name against the
// frontend's subscription.
func applicationMenu(window *application.WebviewWindow) *application.Menu {
	menu := application.DefaultApplicationMenu()

	openItems := application.NewMenu()
	openItems.Add("Open…").
		SetAccelerator("CmdOrCtrl+O").
		OnClick(func(*application.Context) {
			window.EmitEvent("file:open-requested")
		})
	openItems.AddSeparator()
	menu.FindByLabel("File").GetSubmenu().Prepend(openItems)

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
	})

	app.Menu.SetApplicationMenu(applicationMenu(window))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
