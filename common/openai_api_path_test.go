package common

import "testing"

func TestNormalizeOpenAIAPIPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "spaces", in: "   ", want: ""},
		{name: "version without slash", in: "v3", want: "/v3"},
		{name: "version with slash", in: "/v3", want: "/v3"},
		{name: "nested path", in: " /api/v3/ ", want: "/api/v3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeOpenAIAPIPath(tt.in); got != tt.want {
				t.Fatalf("NormalizeOpenAIAPIPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestApplyOpenAIAPIPath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		apiPath     string
		want        string
	}{
		{
			name:        "empty keeps v1",
			requestPath: "/v1/chat/completions",
			apiPath:     "",
			want:        "/v1/chat/completions",
		},
		{
			name:        "explicit v1 keeps v1",
			requestPath: "/v1/models",
			apiPath:     "/v1",
			want:        "/v1/models",
		},
		{
			name:        "replace version prefix",
			requestPath: "/v1/chat/completions",
			apiPath:     "v3",
			want:        "/v3/chat/completions",
		},
		{
			name:        "replace nested path prefix",
			requestPath: "/v1/models",
			apiPath:     "/api/v3/",
			want:        "/api/v3/models",
		},
		{
			name:        "preserve query",
			requestPath: "/v1/responses?beta=true",
			apiPath:     "/v3",
			want:        "/v3/responses?beta=true",
		},
		{
			name:        "non v1 path unchanged",
			requestPath: "/openai/deployments/model/chat/completions",
			apiPath:     "/v3",
			want:        "/openai/deployments/model/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyOpenAIAPIPath(tt.requestPath, tt.apiPath); got != tt.want {
				t.Fatalf("ApplyOpenAIAPIPath(%q, %q) = %q, want %q", tt.requestPath, tt.apiPath, got, tt.want)
			}
		})
	}
}

func TestBuildOpenAIRequestURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestPath string
		apiPath     string
		want        string
	}{
		{
			name:        "origin defaults to v1",
			baseURL:     "https://api.example.com",
			requestPath: "/v1/models",
			apiPath:     "",
			want:        "https://api.example.com/v1/models",
		},
		{
			name:        "origin uses custom nested path",
			baseURL:     "https://ark.cn-beijing.volces.com",
			requestPath: "/v1/models",
			apiPath:     "/api/coding/v3",
			want:        "https://ark.cn-beijing.volces.com/api/coding/v3/models",
		},
		{
			name:        "versioned base url does not append v1",
			baseURL:     "https://ark.cn-beijing.volces.com/api/coding/v3",
			requestPath: "/v1/models",
			apiPath:     "",
			want:        "https://ark.cn-beijing.volces.com/api/coding/v3/models",
		},
		{
			name:        "versioned base url with matching api path does not duplicate path",
			baseURL:     "https://ark.cn-beijing.volces.com/api/coding/v3",
			requestPath: "/v1/chat/completions",
			apiPath:     "/api/coding/v3",
			want:        "https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions",
		},
		{
			name:        "base url ending in v1 accepts normal openai requests",
			baseURL:     "https://api.example.com/v1",
			requestPath: "/v1/responses?beta=true",
			apiPath:     "",
			want:        "https://api.example.com/v1/responses?beta=true",
		},
		{
			name:        "non versioned nested base keeps default v1 suffix",
			baseURL:     "https://proxy.example.com/openai",
			requestPath: "/v1/models",
			apiPath:     "",
			want:        "https://proxy.example.com/openai/v1/models",
		},
		{
			name:        "cloudflare gateway strips default v1 prefix",
			baseURL:     "https://gateway.ai.cloudflare.com/v1/account/gateway/openai",
			requestPath: "/v1/chat/completions",
			apiPath:     "",
			want:        "https://gateway.ai.cloudflare.com/v1/account/gateway/openai/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildOpenAIRequestURL(tt.baseURL, tt.requestPath, tt.apiPath); got != tt.want {
				t.Fatalf("BuildOpenAIRequestURL(%q, %q, %q) = %q, want %q", tt.baseURL, tt.requestPath, tt.apiPath, got, tt.want)
			}
		})
	}
}
