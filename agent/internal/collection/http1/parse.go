package http1

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
)

const (
	DirRecv = 0
	DirSend = 1

	KindSocket  = 0
	KindOpenSSL = 1
	KindGoTLS   = 2

	incompleteCap = 8 << 10 // 8KiB
)

// Message is one HTTP/1 request-line or status-line extracted from a prefix.
type Message struct {
	Request bool
	Method  string
	Path    string
	Status  int
	Connect bool
	HTTP2   bool
	Opaque  bool
}

var emitMethods = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"PUT":     {},
	"HEAD":    {},
	"DELETE":  {},
	"PATCH":   {},
	"OPTIONS": {},
}

var http2Preface = []byte("PRI * HTTP/2.0")

// httpStartTokens are full first-line prefixes. Short buffers that are a
// prefix of one of these are also treated as HTTP so a split request-line
// is leftover-buffered instead of dropped.
var httpStartTokens = [][]byte{
	[]byte("GET "),
	[]byte("POST "),
	[]byte("PUT "),
	[]byte("HEAD "),
	[]byte("DELETE "),
	[]byte("PATCH "),
	[]byte("OPTIONS "),
	[]byte("CONNECT "),
	[]byte("PRI "),
	[]byte("HTTP/1"),
}

// ParsePrefix extracts complete first-lines from a syscall/uprobe prefix.
// leftover is an incomplete first line (no \n yet). opaque means stop HTTP/1.
func ParsePrefix(data []byte) (msgs []Message, leftover []byte, opaque bool) {
	if len(data) == 0 {
		return nil, nil, false
	}
	if bytes.HasPrefix(data, http2Preface) {
		return nil, nil, true
	}

	rest := data
	for len(rest) > 0 {
		line, after, ok := cutLine(rest)
		if !ok {
			if looksLikeHTTPStart(rest) {
				return msgs, rest, false
			}
			return msgs, nil, false
		}
		msg, skip := parseFirstLine(line)
		if skip {
			rest = after
			continue
		}
		if msg.HTTP2 || msg.Opaque {
			return msgs, nil, true
		}
		msgs = append(msgs, msg)
		if msg.Connect {
			return msgs, nil, true
		}

		// Pipeline next message only when headers ends in this prefix.
		_, after0, ok := bytes.Cut(after, []byte("\r\n\r\n"))
		if !ok {
			return msgs, nil, false
		}
		rest = after0
		if len(rest) == 0 {
			return msgs, nil, false
		}
		if !looksLikeHTTPStart(rest) {
			return msgs, nil, false
		}
	}
	return msgs, nil, false
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

func looksLikeHTTPStart(data []byte) bool {
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
	if bytes.HasPrefix(line, http2Preface) || bytes.Equal(line, []byte("PRI * HTTP/2.0")) {
		return Message{HTTP2: true, Opaque: true}, false
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
		return Message{Request: true, Method: methodStr, Path: targetStr, Connect: true, Opaque: true}, false
	}
	if _, ok := emitMethods[methodStr]; !ok {
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

func statusClass(status int) string {
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
