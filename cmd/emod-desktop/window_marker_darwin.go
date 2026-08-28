package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

static void setWindowDocumentEdited(void *window, bool edited) {
	NSWindow *nsWindow = (__bridge NSWindow *)window;
	[nsWindow setDocumentEdited:edited];
}
*/
import "C"

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/hpcsc/emod/internal/desktop"
)

// windowMarker shows unsaved work the way macOS does, as a dot in the close
// button. Wails exposes no document-edited state of its own — the framework's
// answer to an edited window is to rename it — so the marker goes through the
// NSWindow behind the webview.
type windowMarker struct {
	window *application.WebviewWindow
}

var _ desktop.WindowMarker = (*windowMarker)(nil)

// MarkEdited must run on the main thread, because it touches AppKit, and it is
// called from whichever goroutine served the frontend's binding call.
func (m *windowMarker) MarkEdited(edited bool) {
	application.InvokeAsync(func() {
		native := m.window.NativeWindow()
		if native == nil {
			return
		}
		C.setWindowDocumentEdited(unsafe.Pointer(native), C.bool(edited))
	})
}
