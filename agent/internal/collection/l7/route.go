package l7

import (
	"strconv"
	"strings"
)

const (
	maxRouteDepth    = 12
	maxRouteLen      = 128
	maxRoutesPerDest = 64
)

// Router templates REST paths per destination and caps cardinality with /{other}.
// Callers must serialize access (Tracker holds t.mu around Template).
type Router struct {
	byDest map[string]map[string]struct{}
}

func NewRouter() *Router {
	return &Router{byDest: make(map[string]map[string]struct{})}
}

func (r *Router) Template(dest, path string) string {
	templated := templatePath(path)
	if templated == "" {
		templated = "/"
	}

	routes := r.byDest[dest]
	if routes == nil {
		routes = make(map[string]struct{})
		r.byDest[dest] = routes
	}
	if _, ok := routes[templated]; ok {
		return templated
	}
	if len(routes) >= maxRoutesPerDest {
		return "/{other}"
	}
	routes[templated] = struct{}{}
	return templated
}

func DestKey(dstPodUID, actualIP string, actualPort int) string {
	if dstPodUID != "" {
		return "uid:" + dstPodUID
	}
	return "ip:" + actualIP + ":" + strconv.Itoa(actualPort)
}

func CapRoute(path string) string {
	if path == "" {
		return "/"
	}
	if len(path) > maxRouteLen {
		return path[:maxRouteLen]
	}
	return path
}

func templatePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	segments := strings.Split(path, "/")
	out := make([]string, 0, len(segments))
	depth := 0
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		depth++
		if depth > maxRouteDepth {
			out = append(out, "{other}")
			break
		}
		if isIDSegment(seg) {
			out = append(out, "{id}")
			continue
		}
		out = append(out, seg)
	}
	result := "/" + strings.Join(out, "/")
	if len(result) > maxRouteLen {
		result = result[:maxRouteLen]
	}
	return result
}

func isIDSegment(seg string) bool {
	if isUUID(seg) || isULID(seg) || isObjectID(seg) {
		return true
	}
	if isAllDigits(seg) {
		return true
	}
	if len(seg) >= 16 && isAllHex(seg) {
		return true
	}
	return false
}

func isUUID(seg string) bool {
	if len(seg) != 36 {
		return false
	}
	for i, c := range seg {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !isHexRune(c) {
				return false
			}
		}
	}
	return true
}

func isULID(seg string) bool {
	if len(seg) != 26 {
		return false
	}
	for _, c := range seg {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		default:
			return false
		}
	}
	return true
}

func isObjectID(seg string) bool {
	return len(seg) == 24 && isAllHex(seg)
}

func isAllDigits(seg string) bool {
	if seg == "" {
		return false
	}
	for _, c := range seg {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isAllHex(seg string) bool {
	if seg == "" {
		return false
	}
	for _, c := range seg {
		if !isHexRune(c) {
			return false
		}
	}
	return true
}

func isHexRune(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
