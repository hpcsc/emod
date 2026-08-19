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

// FileService is how a native shell reads a model off disk. It answers the same
// {"error": "..."} envelope as ModelService, so the frontend reports a file it
// could not read the way it reports source it could not parse.
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
		return pipeline.ErrorJSON(fmt.Sprintf("reading %s: %s", absolute, readFailureReason(err)))
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

// A *os.PathError prints as "open <path>: <reason>", so reporting it beside the
// path the caller already knows names that path twice in one sentence.
func readFailureReason(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}

	return err
}
