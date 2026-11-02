package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/hpcsc/emod/internal/desktop"
)

// openFileEvents tells the page what the open file's watch found at its path.
// Each event carries the path, so a page that has opened another model since
// can tell the report is not about the file it has open.
type openFileEvents struct {
	window *application.WebviewWindow
}

var _ desktop.OpenFileListener = (*openFileEvents)(nil)

func (e *openFileEvents) ChangedOnDisk(path string) {
	e.window.EmitEvent("file:changed-on-disk", path)
}

func (e *openFileEvents) MissingFromDisk(path string) {
	e.window.EmitEvent("file:missing-from-disk", path)
}
