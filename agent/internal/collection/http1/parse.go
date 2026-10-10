package http1

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
)

const incompleteCap = 8 << 10

type Message struct {
	Request     bool
	Method      string
	Path        string
	Status      int
	Opaque      bool
	Traceparent string
}

// httpStartTokens are full first line prefixes. Short buffers that are a
// prefix of one of these are also treated as HTTP so a split request line
// is leftover buffered instead of dropped.
var httpStartTokens = [][]byte{
	[]byte("GET "),
	[]byte("POST "),
	[]byte("PUT "),
	[]byte("HEAD "),
	[]byte("DELETE "),
	[]byte("PATCH "),
	[]byte("OPTIONS "),
	[]byte("CONNECT "),
	[]byte("HTTP/1"),
}

var traceparentName = []byte("traceparent")

// ParsePrefix extracts complete messages from a syscall/uprobe prefix.
// Requests wait for the header terminator (\r\n\r\n) so traceparent can be
// scanned. Responses stay first-line only for pairing. leftover is an
// incomplete request line or incomplete request header block. opaque means
// stop HTTP/1 (CONNECT or 101).
func ParsePrefix(data []byte) (msgs []Message, leftover []byte, opaque bool) {
	if len(data) == 0 {
		return nil, nil, false
	}

	rest := data
	for len(rest) > 0 {
		messageStart := rest
		line, after, ok := cutLine(rest)
		if !ok {
			if LooksLikeStart(rest) {
				return msgs, rest, false
			}
			return msgs, nil, false
		}
		msg, skip := parseFirstLine(line)
		if skip {
			rest = after
			continue
		}
		if msg.Opaque {
			msgs = append(msgs, msg)
			return msgs, nil, true
		}

		if !msg.Request {
			// Responses: first-line only. Skip a complete header block when
			// present so pipelined status lines can continue. Otherwise drop
			// trailing header bytes rather than stall pairing.
			msgs = append(msgs, msg)
			_, after0, ok := bytes.Cut(after, []byte("\r\n\r\n"))
			if !ok {
				return msgs, nil, false
			}
			rest = after0
			if len(rest) == 0 {
				return msgs, nil, false
			}
			if !LooksLikeStart(rest) {
				return msgs, nil, false
			}
			continue
		}

		headers, after0, ok := bytes.Cut(after, []byte("\r\n\r\n"))
		if !ok {
			leftover = messageStart
			if len(leftover) > incompleteCap {
				// Give-up: emit without context so hop RED still exists.
				msgs = append(msgs, msg)
				return msgs, nil, false
			}
			return msgs, leftover, false
		}
		msg.Traceparent = scanTraceparent(headers)
		msgs = append(msgs, msg)
		rest = after0
		if len(rest) == 0 {
			return msgs, nil, false
		}
		if !LooksLikeStart(rest) {
			return msgs, nil, false
		}
	}
	return msgs, nil, false
}

func scanTraceparent(headers []byte) string {
	for line := range bytes.SplitSeq(headers, []byte("\r\n")) {
		name, value, ok := bytes.Cut(line, []byte{':'})
		if !ok {
			continue
		}
		if !bytes.EqualFold(bytes.TrimSpace(name), traceparentName) {
			continue
		}
		return string(bytes.TrimSpace(value))
	}
	return ""
}

func cutLine(data []byte) (line, after []byte, ok bool) {
	before, after, ok := bytes.Cut(data, []byte{'\n'})
	if !ok {
		return nil, nil, false
	}
	line = before
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, after, true
}

func LooksLikeStart(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	switch data[0] {
	case 'G', 'P', 'H', 'D', 'O', 'C':
	default:
		return false
	}
	for _, token := range httpStartTokens {
		if bytes.HasPrefix(data, token) {
			return true
		}
		if len(data) < len(token) && bytes.Equal(data, token[:len(data)]) {
			return true
		}
	}
	return false
}

func parseFirstLine(line []byte) (Message, bool) {
	if len(line) == 0 {
		return Message{}, true
	}
	if bytes.HasPrefix(line, []byte("HTTP/1.")) {
		return parseStatusLine(line)
	}
	return parseRequestLine(line)
}

func parseRequestLine(line []byte) (Message, bool) {
	method, rest, ok := bytes.Cut(line, []byte{' '})
	if !ok {
		return Message{}, true
	}
	target, _, _ := bytes.Cut(rest, []byte{' '})
	methodStr := string(method)
	targetStr := string(target)
	if methodStr == "CONNECT" {
		return Message{Request: true, Method: methodStr, Path: targetStr, Opaque: true}, false
	}
	if !EmitMethod(methodStr) {
		return Message{}, true
	}
	path := requestPath(targetStr)
	if path == "" {
		return Message{}, true
	}
	return Message{Request: true, Method: methodStr, Path: path}, false
}

func parseStatusLine(line []byte) (Message, bool) {
	version, rest, ok := bytes.Cut(line, []byte{' '})
	if !ok {
		return Message{}, true
	}
	if !bytes.HasPrefix(version, []byte("HTTP/1.")) {
		return Message{}, true
	}
	code, _, _ := bytes.Cut(rest, []byte{' '})
	status, err := strconv.Atoi(string(code))
	if err != nil || status < 100 || status > 599 {
		return Message{}, true
	}
	msg := Message{Request: false, Status: status}
	if status == 101 {
		msg.Opaque = true
	}
	return msg, false
}

func requestPath(target string) string {
	if target == "" || target == "*" {
		return ""
	}
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		u, err := url.Parse(target)
		if err != nil || u.Path == "" {
			return "/"
		}
		return stripQuery(u.Path)
	}
	return stripQuery(target)
}

func stripQuery(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if path == "" {
		return "/"
	}
	return path
}

func EmitMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "HEAD", "DELETE", "PATCH", "OPTIONS":
		return true
	default:
		return false
	}
}

func StatusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500 && status < 600:
		return "5xx"
	default:
		return ""
	}
}
