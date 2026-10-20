package oaichat

import (
	"encoding/json"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
)

// claudeCacheControl is the Anthropic prompt-cache marker. It carries no request data, so
// identical prefixes stay byte-identical across turns and the upstream can reuse them.
var claudeCacheControl = json.RawMessage(`{"type":"ephemeral"}`)

// Claude models cache at most four breakpoints per request; leave the first three to the
// caller and use the last for the moving conversation tail.
const claudeMaxCacheBreakpoints = 2

// isClaudeCacheModel reports whether the routed model charges for prompt caching, which is
// every Claude model. Other families keep their own cache conventions untouched.
func isClaudeCacheModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "claude")
}

// hasClaudeCacheControl reports whether the caller already marked any breakpoint, in which
// case its own layout is authoritative and the gateway adds nothing.
func hasClaudeCacheControl(request *dto.ClaudeRequest) bool {
	if len(request.CacheControl) > 0 {
		return true
	}
	for _, message := range request.Messages {
		if claudeMessageHasCacheControl(message) {
			return true
		}
	}
	switch system := request.System.(type) {
	case []dto.ClaudeMediaMessage:
		for i := range system {
			if len(system[i].CacheControl) > 0 {
				return true
			}
		}
	case []any:
		for _, item := range system {
			if media, ok := item.(dto.ClaudeMediaMessage); ok && len(media.CacheControl) > 0 {
				return true
			}
		}
	}
	return false
}

func claudeMessageHasCacheControl(message dto.ClaudeMessage) bool {
	switch content := message.Content.(type) {
	case []dto.ClaudeMediaMessage:
		for i := range content {
			if len(content[i].CacheControl) > 0 {
				return true
			}
		}
	case []any:
		for _, item := range content {
			if media, ok := item.(dto.ClaudeMediaMessage); ok && len(media.CacheControl) > 0 {
				return true
			}
		}
	}
	return false
}

// ApplyClaudeCacheControl marks the stable prefix (system prompt, last tool) and the moving
// conversation tail so a multi-turn request can be served from the upstream cache. A request
// that already carries markers keeps them. Requests without a Claude model are untouched.
func ApplyClaudeCacheControl(request *dto.ClaudeRequest) {
	if request == nil || !isClaudeCacheModel(request.Model) || hasClaudeCacheControl(request) {
		return
	}
	markedPrefix := markClaudeSystemTail(request)
	// Tools are a separate prefix segment: mark them independently of the system prompt.
	markedPrefix = markClaudeToolTail(request) || markedPrefix
	if !markedPrefix {
		// Without either prefix anchor the request has no reusable segment worth marking.
		return
	}
	markClaudeMessageTail(request)
}

// markClaudeSystemTail marks the last system block, which is where Anthropic expects the
// prefix breakpoint. The system prompt is identical across turns of one conversation.
func markClaudeSystemTail(request *dto.ClaudeRequest) bool {
	switch system := request.System.(type) {
	case []dto.ClaudeMediaMessage:
		for i := len(system) - 1; i >= 0; i-- {
			if system[i].Type == "text" {
				system[i].CacheControl = claudeCacheControl
				request.System = system
				return true
			}
		}
	case []any:
		for i := len(system) - 1; i >= 0; i-- {
			media, ok := system[i].(dto.ClaudeMediaMessage)
			if !ok || media.Type != "text" {
				continue
			}
			media.CacheControl = claudeCacheControl
			system[i] = media
			request.System = system
			return true
		}
	}
	return false
}

// markClaudeToolTail marks the final tool definition, caching the whole tool block with it.
// The converter builds `[]any` of `*dto.Tool`, with web search appended as its own type.
func markClaudeToolTail(request *dto.ClaudeRequest) bool {
	tools, ok := request.Tools.([]any)
	if !ok || len(tools) == 0 {
		return false
	}
	for i := len(tools) - 1; i >= 0; i-- {
		if tool, ok := tools[i].(*dto.Tool); ok && tool != nil {
			tool.CacheControl = claudeCacheControl
			return true
		}
	}
	return false
}

// markClaudeMessageTail marks the last message block so each new turn extends the cache
// instead of repaying the whole transcript. A plain string tail becomes a text block first:
// Anthropic only accepts cache_control on content blocks.
func markClaudeMessageTail(request *dto.ClaudeRequest) {
	for i := len(request.Messages) - 1; i >= 0; i-- {
		message := &request.Messages[i]
		switch content := message.Content.(type) {
		case string:
			if content == "" {
				continue
			}
			message.Content = []dto.ClaudeMediaMessage{{
				Type:         "text",
				Text:         common.GetPointer(content),
				CacheControl: claudeCacheControl,
			}}
			return
		case []dto.ClaudeMediaMessage:
			if len(content) == 0 {
				continue
			}
			content[len(content)-1].CacheControl = claudeCacheControl
			message.Content = content
			return
		case []any:
			if len(content) == 0 {
				continue
			}
			media, ok := content[len(content)-1].(dto.ClaudeMediaMessage)
			if !ok {
				continue
			}
			media.CacheControl = claudeCacheControl
			content[len(content)-1] = media
			message.Content = content
			return
		}
	}
}
