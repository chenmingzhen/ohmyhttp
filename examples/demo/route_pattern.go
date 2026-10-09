package demo

import "strings"

func matchesAnyRoute(patterns []string, requestPath string) bool {
	for _, pattern := range patterns {
		if matchRoutePattern(pattern, requestPath) {
			return true
		}
	}
	return false
}

func matchRoutePattern(pattern string, requestPath string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == requestPath || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
	}
	if !strings.Contains(pattern, "*") {
		return false
	}

	parts := strings.Split(pattern, "*")
	if first := parts[0]; first != "" {
		if !strings.HasPrefix(requestPath, first) {
			return false
		}
		requestPath = requestPath[len(first):]
	}
	for _, part := range parts[1 : len(parts)-1] {
		if part == "" {
			continue
		}
		index := strings.Index(requestPath, part)
		if index < 0 {
			return false
		}
		requestPath = requestPath[index+len(part):]
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(requestPath, last)
}
