package test

import (
	"errors"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const module = "github.com/hpcsc/emod"

func ImportsReachedFrom(t *testing.T, goos, goarch, start string) map[string]bool {
	t.Helper()

	root := moduleRoot(t)
	buildContext := build.Default
	buildContext.GOOS = goos
	buildContext.GOARCH = goarch
	buildContext.CgoEnabled = false

	reached := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		if reached[path] {
			continue
		}
		reached[path] = true

		relative, inModule := strings.CutPrefix(path, module+"/")
		if !inModule {
			continue
		}
		pkg, err := buildContext.ImportDir(filepath.Join(root, relative), 0)
		require.NoError(t, err, path)
		queue = append(queue, pkg.Imports...)
	}

	return reached
}

func RequireNoneWithPrefix(t *testing.T, reached map[string]bool, prefixes ...string) {
	t.Helper()

	for path := range reached {
		for _, prefix := range prefixes {
			require.False(t, path == prefix || strings.HasPrefix(path, prefix+"/"), "%s must not be reached, but %s is", prefix, path)
		}
	}
}

func PackagesImporting(t *testing.T, prefix string) []string {
	t.Helper()

	root := moduleRoot(t)
	var importers []string
	err := filepath.WalkDir(root, func(dir string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if dir != root {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}

		pkg, err := build.Default.ImportDir(dir, 0)
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil
		}
		if err != nil {
			return err
		}

		for _, path := range pkg.Imports {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				relative, err := filepath.Rel(root, dir)
				if err != nil {
					return err
				}
				importers = append(importers, module+"/"+filepath.ToSlash(relative))
				break
			}
		}
		return nil
	})
	require.NoError(t, err)

	return importers
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "no go.mod above the working directory of the test")
		dir = parent
	}
}
