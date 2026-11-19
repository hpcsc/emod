//go:build unit

package linter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

func modelFileIn(t *testing.T, dir, source string) (*ast.Model, string) {
	t.Helper()
	path := filepath.Join(dir, "shop.emod")
	model, diags := parser.Parse(source, path)
	require.Empty(t, diags)
	return model, path
}

func shownScreenWithImage(image string) string {
	return "screen \"CheckoutPage\" {\n  image = \"" + image + "\"\n}\n\n" +
		"context \"Shop\" {\n  aggregate \"Cart\" {\n    slice \"Check Out\" {\n      shows {\n        screen = CheckoutPage\n      }\n    }\n  }\n}\n"
}

func writeFileIn(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("png"), 0o644))
	return path
}

func missingImageMessages(diags []*diagnostic.Entry) []string {
	var messages []string
	for _, d := range diags {
		if d.RuleName == "screen/missing-image-asset" {
			messages = append(messages, d.Message)
		}
	}
	return messages
}

func TestLintFile(t *testing.T) {
	t.Run("a nil model gives nothing", func(t *testing.T) {
		require.Empty(t, linter.LintFile(nil, filepath.Join(t.TempDir(), "shop.emod")))
	})

	t.Run("screen/missing-image-asset", func(t *testing.T) {
		t.Run("warns at the image of a screen whose path leads to no file, naming the screen and the path as written", func(t *testing.T) {
			model, path := modelFileIn(t, t.TempDir(), shownScreenWithImage("./wireframes/checkout.png"))

			diags := linter.LintFile(model, path)

			require.Equal(t, []*diagnostic.Entry{{
				Filename: path,
				Line:     2,
				Column:   11,
				Severity: diagnostic.Warning,
				RuleName: "screen/missing-image-asset",
				Message:  `screen "CheckoutPage" names image "./wireframes/checkout.png", which leads to no file`,
			}}, diags)
		})

		t.Run("reports nothing once a file is at the path", func(t *testing.T) {
			dir := t.TempDir()
			writeFileIn(t, dir, "wireframes/checkout.png")
			model, path := modelFileIn(t, dir, shownScreenWithImage("./wireframes/checkout.png"))

			require.Empty(t, linter.LintFile(model, path))
		})

		t.Run("resolves a relative path against the directory of the model, not the working directory", func(t *testing.T) {
			t.Run("a file only beside the working directory is reported", func(t *testing.T) {
				modelDir, workDir := t.TempDir(), t.TempDir()
				writeFileIn(t, workDir, "checkout.png")
				t.Chdir(workDir)
				model, path := modelFileIn(t, modelDir, shownScreenWithImage("checkout.png"))

				require.Len(t, missingImageMessages(linter.LintFile(model, path)), 1)
			})

			t.Run("a file beside the model reports nothing while the working directory is elsewhere", func(t *testing.T) {
				modelDir := t.TempDir()
				writeFileIn(t, modelDir, "checkout.png")
				t.Chdir(t.TempDir())
				model, path := modelFileIn(t, modelDir, shownScreenWithImage("checkout.png"))

				require.Empty(t, linter.LintFile(model, path))
			})
		})

		t.Run("reports nothing for a path that goes up with .. to a file", func(t *testing.T) {
			root := t.TempDir()
			writeFileIn(t, root, "mockups/checkout.png")
			modelDir := filepath.Join(root, "models")
			require.NoError(t, os.Mkdir(modelDir, 0o755))
			model, path := modelFileIn(t, modelDir, shownScreenWithImage("../mockups/checkout.png"))

			require.Empty(t, linter.LintFile(model, path))
		})

		t.Run("checks an absolute path as it stands", func(t *testing.T) {
			elsewhere := t.TempDir()
			existing := writeFileIn(t, elsewhere, "checkout.png")
			missing := filepath.Join(elsewhere, "history.png")
			model, path := modelFileIn(t, t.TempDir(),
				shownScreenWithImage(existing)+"\nscreen \"OrderHistory\" {\n  image = \""+missing+"\"\n}\n")

			require.Equal(t,
				[]string{`screen "OrderHistory" names image "` + missing + `", which leads to no file`},
				missingImageMessages(linter.LintFile(model, path)))
		})

		t.Run("reports a directory and a link to nothing, and nothing for a link to a file", func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "folder.png"), 0o755))
			target := writeFileIn(t, dir, "real.png")
			require.NoError(t, os.Symlink(target, filepath.Join(dir, "link.png")))
			require.NoError(t, os.Symlink(filepath.Join(dir, "gone.png"), filepath.Join(dir, "broken.png")))
			model, path := modelFileIn(t, dir,
				"screen \"Folder\" {\n  image = \"folder.png\"\n}\n\n"+
					"screen \"Linked\" {\n  image = \"link.png\"\n}\n\n"+
					"screen \"Broken\" {\n  image = \"broken.png\"\n}\n")

			require.Equal(t, []string{
				`screen "Folder" names image "folder.png", which leads to no file`,
				`screen "Broken" names image "broken.png", which leads to no file`,
			}, missingImageMessages(linter.LintFile(model, path)))
		})

		t.Run("checks each screen on its own, and an empty image is no image", func(t *testing.T) {
			dir := t.TempDir()
			writeFileIn(t, dir, "drawer.png")
			model, path := modelFileIn(t, dir,
				"screen \"CheckoutPage\" {\n  image = \"checkout.png\"\n}\n\n"+
					"screen \"CartDrawer\" {\n  image = \"drawer.png\"\n}\n\n"+
					"screen \"OrderHistory\" {\n}\n\n"+
					"screen \"Receipt\" {\n  image = \"\"\n}\n")

			require.Equal(t,
				[]string{`screen "CheckoutPage" names image "checkout.png", which leads to no file`},
				missingImageMessages(linter.LintFile(model, path)))
		})
	})

	t.Run("the rules of Lint", func(t *testing.T) {
		const source = `screen "CheckoutPage" {
  image = "./checkout.png"
}

context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      shows {
        screen = CheckoutPage
      }

      event "OrderUpdated" {
      }
    }
  }
}
`

		t.Run("are given with the diagnostics of this rule, in position order", func(t *testing.T) {
			model, path := modelFileIn(t, t.TempDir(), source)
			fromLint := linter.Lint(model)
			require.NotEmpty(t, fromLint)
			require.Empty(t, missingImageMessages(fromLint), "Lint reads no disk, so it runs no rule that needs the file")

			diags := linter.LintFile(model, path)

			require.Equal(t, "screen/missing-image-asset", diags[0].RuleName)
			require.Equal(t, fromLint, diags[1:])
		})

		t.Run("are all that is given when every image exists", func(t *testing.T) {
			dir := t.TempDir()
			writeFileIn(t, dir, "checkout.png")
			model, path := modelFileIn(t, dir, source)

			require.Equal(t, linter.Lint(model), linter.LintFile(model, path))
		})
	})
}
