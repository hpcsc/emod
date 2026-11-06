package mcp

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const maxModelSize = 1 << 20

// os keeps the error of Root unexported, so its text is the only way to tell an escape apart.
const escapeMessage = "path escapes from parent"

type Root struct {
	absolute string
}

func NewRoot(dir string) (Root, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return Root{}, fmt.Errorf("root %s: %w", dir, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return Root{}, fmt.Errorf("root %s: %w", dir, causeOf(err))
	}
	if !info.IsDir() {
		return Root{}, fmt.Errorf("root %s is not a directory", dir)
	}
	return Root{absolute: absolute}, nil
}

func (r Root) Read(path string) (string, error) {
	relative, inside := r.relativeOf(path)
	if !inside {
		return "", r.outside(path)
	}

	root, err := os.OpenRoot(r.absolute)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, causeOf(err))
	}
	defer root.Close()

	file, err := root.OpenFile(relative, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) && pathErr.Err.Error() == escapeMessage {
			return "", r.outside(path)
		}
		return "", fmt.Errorf("reading %s: %w", path, causeOf(err))
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, causeOf(err))
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("reading %s: not a regular file", path)
	}

	var text strings.Builder
	if _, err := io.Copy(&text, io.LimitReader(file, maxModelSize+1)); err != nil {
		return "", fmt.Errorf("reading %s: %w", path, causeOf(err))
	}
	if text.Len() > maxModelSize {
		return "", fmt.Errorf("reading %s: the file is larger than 1 MiB", path)
	}
	return text.String(), nil
}

func (r Root) relativeOf(path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return path, true
	}
	if relative, found := r.relativeTo(r.absolute, path); found {
		return relative, true
	}
	rootInfo, err := os.Stat(r.absolute)
	if err != nil {
		return "", false
	}
	parts := strings.Split(path, string(filepath.Separator))
	for end := len(parts); end > 0; end-- {
		prefix := strings.Join(parts[:end], string(filepath.Separator))
		if info, err := os.Stat(prefix); err == nil && os.SameFile(info, rootInfo) {
			return r.relativeTo(prefix, path)
		}
	}
	return "", false
}

func (r Root) relativeTo(dir, path string) (string, bool) {
	rest, found := strings.CutPrefix(path, strings.TrimSuffix(dir, string(filepath.Separator)))
	if !found || (rest != "" && rest[0] != filepath.Separator) {
		return "", false
	}
	if relative := strings.TrimLeft(rest, string(filepath.Separator)); relative != "" {
		return relative, true
	}
	return ".", true
}

func (r Root) outside(path string) error {
	return fmt.Errorf("path %q is outside the root %s", path, r.absolute)
}

func causeOf(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}
