package common

import (
	"net/url"
	"strings"
)

func NormalizeOpenAIAPIPath(apiPath string) string {
	path := strings.TrimSpace(apiPath)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		return ""
	}
	return path
}

func OpenAIAPIPathOrDefault(apiPath string) string {
	path := NormalizeOpenAIAPIPath(apiPath)
	if path == "" {
		return "/v1"
	}
	return path
}

func ApplyOpenAIAPIPath(requestPath string, apiPath string) string {
	path := NormalizeOpenAIAPIPath(apiPath)
	if path == "" || path == "/v1" {
		return requestPath
	}
	if requestPath == "/v1" {
		return path
	}
	if strings.HasPrefix(requestPath, "/v1/") ||
		strings.HasPrefix(requestPath, "/v1?") {
		return path + strings.TrimPrefix(requestPath, "/v1")
	}
	return requestPath
}

func IsOpenAIAPIVersionSegment(segment string) bool {
	segment = strings.ToLower(strings.TrimSpace(segment))
	if len(segment) < 2 || segment[0] != 'v' || segment[1] < '0' || segment[1] > '9' {
		return false
	}
	for _, r := range segment[2:] {
		if (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func normalizedURLPath(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err == nil {
		return "/" + strings.Trim(strings.TrimRight(parsed.Path, "/"), "/")
	}
	path := strings.Split(rawURL, "?")[0]
	return "/" + strings.Trim(strings.TrimRight(path, "/"), "/")
}

func OpenAIBaseURLHasVersionPath(baseURL string) bool {
	path := strings.Trim(normalizedURLPath(baseURL), "/")
	if path == "" {
		return false
	}
	segments := strings.Split(path, "/")
	return IsOpenAIAPIVersionSegment(segments[len(segments)-1])
}

func openAIBaseURLPathHasSuffix(baseURL string, apiPath string) bool {
	basePath := normalizedURLPath(baseURL)
	suffix := NormalizeOpenAIAPIPath(apiPath)
	return suffix != "" && (basePath == suffix || strings.HasSuffix(basePath, suffix))
}

func StripOpenAIRequestVersionPrefix(requestPath string) string {
	if !strings.HasPrefix(requestPath, "/") {
		return requestPath
	}
	rest := strings.TrimPrefix(requestPath, "/")
	if rest == "" {
		return requestPath
	}

	end := len(rest)
	if slash := strings.Index(rest, "/"); slash >= 0 && slash < end {
		end = slash
	}
	if query := strings.Index(rest, "?"); query >= 0 && query < end {
		end = query
	}
	segment := rest[:end]
	if !IsOpenAIAPIVersionSegment(segment) {
		return requestPath
	}
	suffix := rest[end:]
	if suffix == "" {
		return ""
	}
	if strings.HasPrefix(suffix, "/") || strings.HasPrefix(suffix, "?") {
		return suffix
	}
	return requestPath
}

func BuildOpenAIRequestURL(baseURL string, requestPath string, apiPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ApplyOpenAIAPIPath(requestPath, apiPath)
	}

	normalizedAPIPath := NormalizeOpenAIAPIPath(apiPath)
	var path string
	switch {
	case normalizedAPIPath != "" && openAIBaseURLPathHasSuffix(baseURL, normalizedAPIPath):
		path = StripOpenAIRequestVersionPrefix(requestPath)
	case normalizedAPIPath != "":
		path = ApplyOpenAIAPIPath(requestPath, normalizedAPIPath)
	case strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com"):
		path = StripOpenAIRequestVersionPrefix(requestPath)
	case OpenAIBaseURLHasVersionPath(baseURL):
		path = StripOpenAIRequestVersionPrefix(requestPath)
	default:
		path = requestPath
	}
	return baseURL + path
}
