//go:build unit

package lsp_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestTransport(t *testing.T) {
	t.Run("read message with Content-Length header", func(t *testing.T) {
		input := "Content-Length: 33\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"ping\"}"
		r := strings.NewReader(input)

		msg, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.Equal(t, "ping", msg.Method)
	})

	t.Run("read message with Content-Length and ID", func(t *testing.T) {
		input := "Content-Length: 58\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}"
		r := strings.NewReader(input)

		msg, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.NotNil(t, msg.ID)
		require.Equal(t, 1, *msg.ID)
		require.Equal(t, "initialize", msg.Method)
		require.NotNil(t, msg.Params)
	})

	t.Run("read message with result", func(t *testing.T) {
		input := "Content-Length: 53\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"capabilities\":{}}}"
		r := strings.NewReader(input)

		msg, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.NotNil(t, msg.ID)
		require.Equal(t, 1, *msg.ID)
		require.NotNil(t, msg.Result)
	})

	t.Run("read message with error object", func(t *testing.T) {
		input := "Content-Length: 77\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32601,\"message\":\"method not found\"}}"
		r := strings.NewReader(input)

		msg, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.NotNil(t, msg.Error)
		require.Equal(t, -32601, msg.Error.Code)
		require.Equal(t, "method not found", msg.Error.Message)
	})

	t.Run("read handles multiple messages sequentially", func(t *testing.T) {
		input := "Content-Length: 14\r\n\r\n{\"method\":\"a\"}Content-Length: 14\r\n\r\n{\"method\":\"b\"}"
		r := strings.NewReader(input)

		msg1, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.Equal(t, "a", msg1.Method)

		msg2, err := lsp.ReadMessage(r)
		require.NoError(t, err)
		require.Equal(t, "b", msg2.Method)
	})

	t.Run("read returns error with missing Content-Length header", func(t *testing.T) {
		input := "Content-Type: application/json\r\n\r\n{}"
		r := strings.NewReader(input)

		_, err := lsp.ReadMessage(r)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing Content-Length")
	})

	t.Run("read returns error with invalid Content-Length value", func(t *testing.T) {
		input := "Content-Length: abc\r\n\r\n{}"
		r := strings.NewReader(input)

		_, err := lsp.ReadMessage(r)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse content-length")
	})

	t.Run("read returns error with truncated body", func(t *testing.T) {
		input := "Content-Length: 100\r\n\r\n{\"jsonrpc\":\"2.0\"}"
		r := strings.NewReader(input)

		_, err := lsp.ReadMessage(r)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("write message with Content-Length header", func(t *testing.T) {
		var buf bytes.Buffer
		msg := &lsp.Message{
			JSONRPC: "2.0",
			Method:  "ping",
		}

		err := lsp.WriteMessage(&buf, msg)
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "Content-Length: ")
		require.Contains(t, output, "\"jsonrpc\":\"2.0\"")
		require.Contains(t, output, "\"method\":\"ping\"")
	})

	t.Run("round-trip request message", func(t *testing.T) {
		id := 1
		original := &lsp.Message{
			JSONRPC: "2.0",
			ID:      &id,
			Method:  "initialize",
			Params:  []byte("{}"),
		}

		var buf bytes.Buffer
		err := lsp.WriteMessage(&buf, original)
		require.NoError(t, err)

		msg, err := lsp.ReadMessage(&buf)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.NotNil(t, msg.ID)
		require.Equal(t, 1, *msg.ID)
		require.Equal(t, "initialize", msg.Method)
	})

	t.Run("round-trip notification message has no ID", func(t *testing.T) {
		original := &lsp.Message{
			JSONRPC: "2.0",
			Method:  "textDocument/didOpen",
			Params:  []byte("{}"),
		}

		var buf bytes.Buffer
		err := lsp.WriteMessage(&buf, original)
		require.NoError(t, err)

		msg, err := lsp.ReadMessage(&buf)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.Equal(t, "textDocument/didOpen", msg.Method)
		require.Nil(t, msg.ID)
	})

	t.Run("round-trip response message with result", func(t *testing.T) {
		id := 42
		original := &lsp.Message{
			JSONRPC: "2.0",
			ID:      &id,
			Result:  []byte(`{"capabilities":{"textDocumentSync":1}}`),
		}

		var buf bytes.Buffer
		err := lsp.WriteMessage(&buf, original)
		require.NoError(t, err)

		msg, err := lsp.ReadMessage(&buf)
		require.NoError(t, err)
		require.Equal(t, "2.0", msg.JSONRPC)
		require.NotNil(t, msg.ID)
		require.Equal(t, 42, *msg.ID)
		require.NotNil(t, msg.Result)
	})

	t.Run("round-trip response message with error", func(t *testing.T) {
		id := 1
		original := &lsp.Message{
			JSONRPC: "2.0",
			ID:      &id,
			Error: &lsp.ErrorObject{
				Code:    -32601,
				Message: "method not found",
			},
		}

		var buf bytes.Buffer
		err := lsp.WriteMessage(&buf, original)
		require.NoError(t, err)

		msg, err := lsp.ReadMessage(&buf)
		require.NoError(t, err)
		require.NotNil(t, msg.Error)
		require.Equal(t, -32601, msg.Error.Code)
		require.Equal(t, "method not found", msg.Error.Message)
	})
}
