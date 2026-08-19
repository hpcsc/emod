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

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "emod",
		Width:  1400,
		Height: 900,
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
