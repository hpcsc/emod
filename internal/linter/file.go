package linter

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

// LintFile is Lint for a model read from the file at path, with the rules that
// look at the disk next to that file.
func LintFile(model *ast.Model, path string) []*diagnostic.Entry {
	diags := Lint(model)
	diags = append(diags, checkMissingImageAssets(model, filepath.Dir(path))...)
	slices.SortStableFunc(diags, byPosition)
	return diags
}

func checkMissingImageAssets(model *ast.Model, dir string) []*diagnostic.Entry {
	if model == nil {
		return nil
	}

	var diags []*diagnostic.Entry
	for _, screen := range model.Screens {
		if screen.Image == "" || leadsToFile(dir, screen.Image) {
			continue
		}
		diags = append(diags, warning(screen.ImagePos, "screen/missing-image-asset", fmt.Sprintf(
			"screen %q names image %q, which leads to no file", screen.Name, screen.Image)))
	}
	return diags
}

func leadsToFile(dir, image string) bool {
	path := image
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
