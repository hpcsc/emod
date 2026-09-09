//go:build unit

package desktop_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hpcsc/emod/internal/desktop"
	"github.com/stretchr/testify/require"
)

func TestOpenRequests(t *testing.T) {
	t.Run("taking a request", func(t *testing.T) {
		t.Run("answers nothing when the app was asked to open no file", func(t *testing.T) {
			requests := &desktop.OpenRequests{}

			require.Empty(t, requests.Take())
		})

		t.Run("answers the path once, and nothing after that", func(t *testing.T) {
			requests := &desktop.OpenRequests{}
			requests.Hold("/models/billing.emod")

			require.Equal(t, "/models/billing.emod", requests.Take())
			require.Empty(t, requests.Take(),
				"the page takes again on every reload, and a model it has already opened "+
					"must not open a second time over whatever the reader has typed since")
		})
	})

	t.Run("holding a request", func(t *testing.T) {
		t.Run("replaces one nobody has taken yet", func(t *testing.T) {
			requests := &desktop.OpenRequests{}

			requests.Hold("/models/first.emod")
			requests.Hold("/models/second.emod")

			require.Equal(t, "/models/second.emod", requests.Take(),
				"the newest request is the one the user made")
			require.Empty(t, requests.Take())
		})

		t.Run("waits again after the page has taken the one before it", func(t *testing.T) {
			requests := &desktop.OpenRequests{}
			requests.Hold("/models/first.emod")
			requests.Take()

			requests.Hold("/models/second.emod")

			require.Equal(t, "/models/second.emod", requests.Take())
		})
	})

	// The operating system hands a path over on the application's event goroutine
	// while the page takes over a binding the shell serves its own goroutine, so
	// the two reach this from different goroutines by construction.
	t.Run("concurrent use", func(t *testing.T) {
		t.Run("hands one held path to exactly one of many takers at once", func(t *testing.T) {
			requests := &desktop.OpenRequests{}
			requests.Hold("/models/billing.emod")

			var mu sync.Mutex
			var taken []string
			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if path := requests.Take(); path != "" {
						mu.Lock()
						defer mu.Unlock()
						taken = append(taken, path)
					}
				}()
			}
			wg.Wait()

			require.Equal(t, []string{"/models/billing.emod"}, taken)
		})

		t.Run("never answers a path nothing held, whatever arrives while the page takes", func(t *testing.T) {
			requests := &desktop.OpenRequests{}
			var held []string
			for i := 0; i < 50; i++ {
				held = append(held, fmt.Sprintf("/models/model-%d.emod", i))
			}

			var mu sync.Mutex
			var taken []string
			var wg sync.WaitGroup
			for _, path := range held {
				wg.Add(2)
				go func() { defer wg.Done(); requests.Hold(path) }()
				go func() {
					defer wg.Done()
					if answered := requests.Take(); answered != "" {
						mu.Lock()
						defer mu.Unlock()
						taken = append(taken, answered)
					}
				}()
			}
			wg.Wait()

			// Subset alone holds when nothing was taken at all, which is what a
			// Take that answers nothing would do.
			require.NotEmpty(t, taken)
			require.Subset(t, held, taken)
		})
	})
}
