package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// JSON-RPC protocol version.
const Version = "2.0"

// Message represents a generic JSON-RPC 2.0 message envelope.
// It can carry a request, response, or notification depending on which fields are set.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error object.
type ErrorObject struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    json.RawMessage  `json:"data,omitempty"`
}

// ReadMessage reads a Content-Length framed JSON-RPC message from r.
// The format is: "Content-Length: N\r\n\r\n" followed by exactly N bytes of JSON body.
func ReadMessage(r io.Reader) (*Message, error) {
	contentLength, err := readHeaders(r)
	if err != nil {
		return nil, err
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	msg := &Message{}
	if err := json.Unmarshal(body, msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return msg, nil
}

// readHeaders reads the Content-Length header from r by scanning for "\r\n\r\n".
// It reads byte-by-byte to avoid over-reading into a buffer, which would
// interfere with subsequent reads from the same io.Reader.
func readHeaders(r io.Reader) (int, error) {
	var buf bytes.Buffer
	one := make([]byte, 1)

	for {
		if _, err := r.Read(one); err != nil {
			return 0, fmt.Errorf("read header: %w", err)
		}
		buf.WriteByte(one[0])
		if buf.Len() >= 4 {
			b := buf.Bytes()
			if b[buf.Len()-4] == '\r' && b[buf.Len()-3] == '\n' &&
				b[buf.Len()-2] == '\r' && b[buf.Len()-1] == '\n' {
				break
			}
		}
	}

	headerBlock := buf.String()
	headerBlock = headerBlock[:len(headerBlock)-4] // strip trailing \r\n\r\n

	var contentLength int
	for _, line := range strings.Split(headerBlock, "\r\n") {
		if strings.HasPrefix(line, "Content-Length: ") {
			n, err := strconv.Atoi(line[len("Content-Length: "):])
			if err != nil {
				return 0, fmt.Errorf("parse content-length: %w", err)
			}
			contentLength = n
		}
	}

	if contentLength == 0 {
		return 0, fmt.Errorf("missing Content-Length header")
	}

	return contentLength, nil
}

// WriteMessage writes a Content-Length framed JSON-RPC message to w.
// The format is: "Content-Length: N\r\n\r\n" followed by the JSON body.
func WriteMessage(w io.Writer, msg *Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(w, header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}

	return nil
}
