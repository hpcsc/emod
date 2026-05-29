package viewer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
	if len(diagramJSON) == 0 {
		return ViewerHTML
	}

	html := strings.TrimSuffix(ViewerHTML, "\n")
	injection := fmt.Sprintf("\nwindow.INITIAL_DATA = %s;", string(diagramJSON))
	return strings.Replace(html, "</script>", injection+"</script>", 1)
}
