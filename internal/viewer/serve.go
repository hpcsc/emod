package viewer

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hpcsc/emod/internal/frontend"
)

const maxShutdownTime = 5 * time.Second

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

	staticFS, err := fs.Sub(frontend.FS, "static")
	if err != nil {
		return "", nil, fmt.Errorf("viewer: fs sub: %w", err)
	}
	generatedFS, err := fs.Sub(frontend.FS, "generated")
	if err != nil {
		return "", nil, fmt.Errorf("viewer: fs sub: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.Handle("/generated/", http.StripPrefix("/generated/", http.FileServer(http.FS(generatedFS))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

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
	data, err := frontend.FS.ReadFile("static/viewer.html")
	if err != nil {
		return ""
	}
	html := string(data)
	if len(diagramJSON) > 0 {
		// The placeholder sits between two <script> tags, not inside one, so the
		// assignment has to bring its own or the browser lays the whole document
		// out as text and the viewer loads with no model.
		// `</` is escaped so a model that spells `</script>` in a name or description
		// cannot end the block early and have the rest of itself parsed as markup.
		// `\/` is a valid JSON string escape, and `/` occurs nowhere else in JSON.
		safe := strings.ReplaceAll(string(diagramJSON), "</", `<\/`)
		injection := fmt.Sprintf("<script>\nwindow.INITIAL_DATA = %s;\n</script>", safe)
		html = strings.Replace(html, "<!--INITIAL_DATA-->", injection, 1)
	} else {
		html = strings.Replace(html, "<!--INITIAL_DATA-->", "", 1)
	}
	return html
}
