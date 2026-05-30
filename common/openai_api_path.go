package common

import "strings"

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
