package viewer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/validator"
)

const maxShutdownTime = 5 * time.Second

const maxBodyBytes = 1 << 20 // 1 MB

// ServeViewer starts an HTTP server on 127.0.0.1:<port> serving the embedded
// viewer HTML. If diagramJSON is non-empty, it is injected into the HTML as
// window.INITIAL_DATA. On SIGINT/SIGTERM the server shuts down gracefully.
// Returns the server URL, a shutdown function for programmatic use, and any error.
func ServeViewer(port int, diagramJSON []byte) (string, func(), error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", nil, fmt.Errorf("viewer: %w", err)
	}

	html := buildHTML(diagramJSON)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})
	mux.HandleFunc("/parse", handleParse)

	srv := &http.Server{Handler: mux}

	shutdownCh := make(chan struct{})
	sigCtx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCtx.Done():
		case <-shutdownCh:
		}
		sigStop()

		ctx, cancel := context.WithTimeout(context.Background(), maxShutdownTime)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	go func() {
		srv.Serve(listener)
	}()

	addr := fmt.Sprintf("http://%s", listener.Addr().String())
	fmt.Printf("Viewer available at %s\n", addr)

	return addr, func() {
		close(shutdownCh)
	}, nil
}

func buildHTML(diagramJSON []byte) string {
	if len(diagramJSON) == 0 {
		return ViewerHTML
	}

	html := strings.TrimSuffix(ViewerHTML, "\n")
	injection := fmt.Sprintf("\nwindow.INITIAL_DATA = %s;", string(diagramJSON))
	return strings.Replace(html, "</script>", injection+"</script>", 1)
}

type parseRequest struct {
	Source string `json:"source"`
}

type parseResponse struct {
	Diagnostics []*jsonDiagnostic `json:"diagnostics"`
	Diagram     json.RawMessage   `json:"diagram"`
}

type jsonDiagnostic struct {
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

func handleParse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body := io.LimitReader(r.Body, maxBodyBytes)
	raw, err := io.ReadAll(body)
	if err != nil {
		jsonError(w, fmt.Sprintf("reading body: %s", err), http.StatusInternalServerError)
		return
	}

	var req parseRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		jsonError(w, fmt.Sprintf("invalid JSON: %s", err), http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		jsonError(w, "missing source field", http.StatusBadRequest)
		return
	}

	tokens, diags := lexer.Scan(req.Source, "input.emod")

	p := parser.New(tokens, "input.emod")
	model, parserDiags := p.Parse()
	diags = append(diags, parserDiags...)

	validatorDiags := validator.Validate(model)
	diags = append(diags, validatorDiags...)

	lintDiags := linter.Lint(model)
	diags = append(diags, lintDiags...)

	diagramJSON, exportErr := export.ExportDiagramJSONDiagnostics(model, diags)
	if exportErr != nil {
		jsonError(w, fmt.Sprintf("export: %s", exportErr), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(diagramJSON)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
