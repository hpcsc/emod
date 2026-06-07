//go:build unit

package lsp_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestDocumentManager(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Run("stores content for a URI", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "model User {}")

			content, ok := dm.GetContent("file:///test.emod")
			require.True(t, ok)
			require.Equal(t, "model User {}", content)
		})

		t.Run("replaces existing content for the same URI", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "original content")
			dm.Open("file:///test.emod", "replaced content")

			content, ok := dm.GetContent("file:///test.emod")
			require.True(t, ok)
			require.Equal(t, "replaced content", content)
		})
	})

	t.Run("update", func(t *testing.T) {
		t.Run("modifies content for an existing document", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "original")
			dm.Update("file:///test.emod", "updated")

			content, ok := dm.GetContent("file:///test.emod")
			require.True(t, ok)
			require.Equal(t, "updated", content)
		})

		t.Run("stores content for a URI that was not previously opened", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Update("file:///new.emod", "new content")

			content, ok := dm.GetContent("file:///new.emod")
			require.True(t, ok)
			require.Equal(t, "new content", content)
		})
	})

	t.Run("close", func(t *testing.T) {
		t.Run("removes a document from the store", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "content")
			dm.Close("file:///test.emod")

			_, ok := dm.GetContent("file:///test.emod")
			require.False(t, ok)
		})

		t.Run("is a no-op when the URI was never opened", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			require.NotPanics(t, func() {
				dm.Close("file:///nonexistent.emod")
			})
		})
	})

	t.Run("get content", func(t *testing.T) {
		t.Run("returns content for an open document", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "model User {}")

			content, ok := dm.GetContent("file:///test.emod")
			require.True(t, ok)
			require.Equal(t, "model User {}", content)
		})

		t.Run("returns empty string and false for a URI that was never opened", func(t *testing.T) {
			dm := lsp.NewDocumentManager()

			content, ok := dm.GetContent("file:///unknown.emod")
			require.False(t, ok)
			require.Empty(t, content)
		})

		t.Run("returns empty string and false after close", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///test.emod", "content")
			dm.Close("file:///test.emod")

			content, ok := dm.GetContent("file:///test.emod")
			require.False(t, ok)
			require.Empty(t, content)
		})

		t.Run("manages multiple documents independently", func(t *testing.T) {
			dm := lsp.NewDocumentManager()
			dm.Open("file:///a.emod", "content a")
			dm.Open("file:///b.emod", "content b")

			contentA, okA := dm.GetContent("file:///a.emod")
			require.True(t, okA)
			require.Equal(t, "content a", contentA)

			contentB, okB := dm.GetContent("file:///b.emod")
			require.True(t, okB)
			require.Equal(t, "content b", contentB)
		})
	})
}
