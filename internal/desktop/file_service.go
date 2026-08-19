package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/pipeline"
)

// FileService is how a native shell reads a model off disk and puts one back.
// It answers the same {"error": "..."} envelope as ModelService, so the frontend
// reports a file it could not read, or could not write, the way it reports
// source it could not parse.
type FileService struct{}

type openedFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Read answers the file at path with its base name, its absolute path and its
// contents byte for byte — including contents no emod stage would accept,
// because what is wrong with a model is the pipeline's answer to give, not this
// one's. A file that is not valid UTF-8 is refused instead, because it is the
// one thing this cannot carry unaltered. The path is resolved before it is read
// so a caller holding the answer can act on it without knowing this process's
// working directory.
func (s *FileService) Read(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return pipeline.ErrorJSON(fmt.Sprintf("resolving %s: %s", path, err))
	}

	content, err := os.ReadFile(absolute)
	if err != nil {
		return pipeline.ErrorJSON(fmt.Sprintf("reading %s: %s", absolute, failureReason(err)))
	}

	// json.Marshal rewrites every byte it cannot decode as U+FFFD, so carrying
	// such a file would hand the viewer altered text and, once Save writes that
	// text back, silently destroy what it could not read.
	if !utf8.Valid(content) {
		return pipeline.ErrorJSON(fmt.Sprintf("reading %s: not valid UTF-8", absolute))
	}

	b, _ := json.Marshal(openedFile{
		Name:    filepath.Base(absolute),
		Path:    absolute,
		Content: string(content),
	})

	return string(b)
}

type savedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Write puts content at path and answers that file's base name and absolute
// path. The bytes reach the target only once all of them are on disk: they go to
// a working file beside it which is then renamed over it, so a write that fails
// part way through leaves the model already there intact rather than truncated.
// A target that exists is opened for writing first, because that rename is
// permitted by the directory rather than by the file and would otherwise replace
// a model the filesystem is refusing to let anyone write.
func (s *FileService) Write(path string, content string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return pipeline.ErrorJSON(fmt.Sprintf("resolving %s: %s", path, err))
	}

	if err := replaceFile(absolute, content); err != nil {
		return pipeline.ErrorJSON(fmt.Sprintf("writing %s: %s", absolute, failureReason(err)))
	}

	b, _ := json.Marshal(savedFile{
		Name: filepath.Base(absolute),
		Path: absolute,
	})

	return string(b)
}

func replaceFile(path string, content string) (err error) {
	target, mode, created, err := prepareTarget(path)
	if err != nil {
		return err
	}
	// A target this call brought into being has to go again if the write does
	// not land, or a refused save leaves an empty model where there was none.
	defer func() {
		if err != nil && created {
			os.Remove(target)
		}
	}()

	working, err := os.CreateTemp(filepath.Dir(target), workingName(target))
	if err != nil {
		return err
	}
	// Removed by name rather than by handle: the rename below leaves nothing at
	// this name, and every way out before it has to take the working file along.
	defer os.Remove(working.Name())

	if _, err := working.WriteString(content); err != nil {
		working.Close()
		return err
	}
	if err := working.Close(); err != nil {
		return err
	}

	// A working file is created owner-only, so a model anyone could read would
	// come back readable by nobody else without this.
	if err := os.Chmod(working.Name(), mode); err != nil {
		return err
	}

	return os.Rename(working.Name(), target)
}

// prepareTarget answers the path the replacement should land on, the mode it
// should carry, and whether it had to be brought into being to find out.
//
// Symlinks are resolved first because the replacement renames over its target:
// left unresolved, saving a linked model would put a regular file where the link
// was and leave the file it named untouched, which is the "my edits did not land
// in my working copy" failure saving in place exists to prevent.
//
// Opening the target is what asks the filesystem whether this process may write
// it, and nothing else in the replacement asks: a rename is permitted by the
// directory rather than by the file it replaces, so a read-only model would
// otherwise be overwritten without complaint.
func prepareTarget(path string) (string, os.FileMode, bool, error) {
	path, err := resolveLinks(path)
	if err != nil {
		return "", 0, false, err
	}

	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		return createTarget(path)
	}
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}

	return path, info.Mode().Perm(), false, nil
}

// filepath.EvalSymlinks refuses a link whose target is not there, so a save
// through one would otherwise be reported as "file exists" — what O_EXCL says
// about the link itself — rather than creating the model the link names. Each
// hop is followed by hand in that case, bounded because a cycle has no end.
func resolveLinks(path string) (string, error) {
	for hop := 0; hop < 16; hop++ {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		link, err := os.Readlink(path)
		if err != nil {
			return path, nil
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(path), link)
		}
		path = link
	}

	return "", errors.New("too many levels of symbolic links")
}

// The mode is asked for rather than stated because the umask filters what open
// creates and does not filter the chmod that gives the replacement its mode: a
// literal here would hand every new model the same permissions whatever the user
// asked their shell for.
func createTarget(path string) (string, os.FileMode, bool, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, true, err
	}

	return path, info.Mode().Perm(), true, nil
}

// os.CreateTemp puts a dot in front of this and ten digits behind it, and a
// filesystem refuses a name longer than its limit — so without a bound a model
// whose own name is near that limit could be read and never saved back.
func workingName(path string) string {
	const longestBase = 243

	base := filepath.Base(path)
	// Cut on a rune boundary: os.CreateTemp refuses a pattern that is not valid
	// UTF-8, so halving a multi-byte character here would fail every save of a
	// long name that happens to hold one.
	for len(base) > longestBase {
		_, size := utf8.DecodeLastRuneInString(base)
		base = base[:len(base)-size]
	}

	return "." + base + ".*"
}

// A *os.PathError prints as "open <path>: <reason>" and a *os.LinkError names
// two paths, so reporting either beside the path the caller already knows states
// that path more than once in one sentence.
func failureReason(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}

	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return linkErr.Err
	}

	return err
}
