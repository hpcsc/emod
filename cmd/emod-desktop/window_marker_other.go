//go:build !darwin

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/hpcsc/emod/internal/desktop"
)

// windowMarker does nothing outside macOS: no other platform this app targets
// has a window-level convention for unsaved work, so the frontend puts a `*`
// ahead of the window's name and that is the whole marker.
type windowMarker struct {
	window *application.WebviewWindow
}

var _ desktop.WindowMarker = (*windowMarker)(nil)

func (m *windowMarker) MarkEdited(bool) {}
