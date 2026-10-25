package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The direct /v1/messages path forwards the client's own thinking object, so the
// channel display setting must apply to whatever thinking the client sent.
func TestApplyClaudeThinkingDisplay(t *testing.T) {
	tests := []struct {
		name        string
		thinking    *Thinking
		display     string
		wantDisplay string
	}{
		{name: "adaptive gets summarized", thinking: &Thinking{Type: "adaptive"}, display: ClaudeThinkingDisplaySummarized, wantDisplay: "summarized"},
		{name: "enabled gets omitted", thinking: &Thinking{Type: "enabled"}, display: ClaudeThinkingDisplayOmitted, wantDisplay: "omitted"},
		{name: "channel setting overrides client display", thinking: &Thinking{Type: "adaptive", Display: "omitted"}, display: ClaudeThinkingDisplaySummarized, wantDisplay: "summarized"},
		{name: "unset keeps client display", thinking: &Thinking{Type: "adaptive", Display: "omitted"}, display: "", wantDisplay: "omitted"},
		{name: "disabled thinking is left alone", thinking: &Thinking{Type: "disabled"}, display: ClaudeThinkingDisplaySummarized, wantDisplay: ""},
		{name: "between_tools accepts no other field", thinking: &Thinking{Type: "between_tools"}, display: ClaudeThinkingDisplaySummarized, wantDisplay: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &ClaudeRequest{Thinking: tt.thinking}
			ApplyClaudeThinkingDisplay(request, tt.display)
			assert.Equal(t, tt.wantDisplay, request.Thinking.Display)
		})
	}

	request := &ClaudeRequest{}
	ApplyClaudeThinkingDisplay(request, ClaudeThinkingDisplaySummarized)
	assert.Nil(t, request.Thinking, "a request without thinking gains none")
	ApplyClaudeThinkingDisplay(nil, ClaudeThinkingDisplaySummarized)
}
