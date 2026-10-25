package dto

import (
	"fmt"
	"strings"

	"github.com/Lorry-San/nbapi/common"
)

const (
	// ClaudeThinkingModeInherit keeps the historical suffix and budget behavior.
	ClaudeThinkingModeInherit = "inherit"
	// ClaudeThinkingModeAdaptive maps a normal reasoning effort to adaptive thinking.
	ClaudeThinkingModeAdaptive = "adaptive"
	// ClaudeThinkingModeEnabled maps a normal reasoning effort to enabled thinking budgets.
	ClaudeThinkingModeEnabled = "enabled"
	// ClaudeThinkingModeDisabled removes thinking from the request.
	ClaudeThinkingModeDisabled = "disabled"
)

const (
	// ClaudeThinkingDisplaySummarized asks Anthropic for readable summarized thinking.
	ClaudeThinkingDisplaySummarized = "summarized"
	// ClaudeThinkingDisplayOmitted asks Anthropic to return empty thinking blocks with signatures only.
	ClaudeThinkingDisplayOmitted = "omitted"
)

// ValidClaudeThinkingDisplay reports whether display is a value the channel
// setting accepts. Empty keeps the upstream model default.
func ValidClaudeThinkingDisplay(display string) bool {
	switch strings.TrimSpace(display) {
	case "", ClaudeThinkingDisplaySummarized, ClaudeThinkingDisplayOmitted:
		return true
	default:
		return false
	}
}

// ApplyClaudeThinkingDisplay sets thinking.display on a request that thinks.
// An empty display leaves the request unchanged so the model default applies;
// a request without thinking, or with disabled thinking, has nothing to display.
func ApplyClaudeThinkingDisplay(request *ClaudeRequest, display string) {
	display = strings.TrimSpace(display)
	if request == nil || request.Thinking == nil || display == "" {
		return
	}
	if request.Thinking.Type != "adaptive" && request.Thinking.Type != "enabled" {
		return
	}
	request.Thinking.Display = display
}

// claudeEnabledThinkingMinimumBudget is the smallest enabled-thinking budget
// Anthropic accepts.
const claudeEnabledThinkingMinimumBudget = 1280

// claudeEnabledThinkingBudgets maps a normal reasoning effort to an enabled
// thinking budget, so a client effort stays observable upstream.
var claudeEnabledThinkingBudgets = map[string]int{
	"minimal": 1024,
	"low":     1280,
	"medium":  2048,
	"high":    4096,
	"xhigh":   8192,
	"max":     16384,
}

// ValidClaudeThinkingMode reports whether mode is one this gateway understands.
func ValidClaudeThinkingMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "", ClaudeThinkingModeInherit, ClaudeThinkingModeAdaptive, ClaudeThinkingModeEnabled, ClaudeThinkingModeDisabled:
		return true
	default:
		return false
	}
}

// ApplyClaudeThinking maps one channel thinking mode plus the client's normal
// reasoning effort onto a Claude request.
//
// An empty or inherit mode keeps the historical behavior: an effort produces
// enabled thinking with a fixed budget. Explicit modes let a Claude channel
// choose adaptive thinking (the effort becomes output_config.effort) or enabled
// thinking with a budget that still reflects the requested effort.
func ApplyClaudeThinking(request *ClaudeRequest, mode string, effort string, budgetRatio float64) error {
	if request == nil {
		return nil
	}
	mode = strings.TrimSpace(mode)
	effort = strings.TrimSpace(effort)

	switch mode {
	case ClaudeThinkingModeDisabled:
		request.Thinking = nil
		request.OutputConfig = nil
		return nil
	case ClaudeThinkingModeAdaptive:
		request.Thinking = &Thinking{Type: "adaptive"}
		request.OutputConfig = nil
		if effort != "" {
			outputConfig, err := common.Marshal(map[string]string{"effort": effort})
			if err != nil {
				return fmt.Errorf("marshal claude output_config: %w", err)
			}
			request.OutputConfig = outputConfig
		}
		return nil
	case "", ClaudeThinkingModeInherit, ClaudeThinkingModeEnabled:
		// Fall through to the enabled-thinking budget paths below.
	default:
		return fmt.Errorf("unknown claude thinking mode %q", mode)
	}

	budget, ok := claudeEnabledThinkingBudgets[effort]
	if !ok {
		// No recognized effort: an explicitly enabled channel still turns
		// thinking on at the configured share of the output cap.
		if effort != "" || mode != ClaudeThinkingModeEnabled {
			return nil
		}
		if request.MaxTokens == nil || *request.MaxTokens == 0 || budgetRatio <= 0 {
			return nil
		}
		budget = int(float64(common.BoundedRequestTokens(*request.MaxTokens)) * budgetRatio)
		if budget < claudeEnabledThinkingMinimumBudget {
			budget = claudeEnabledThinkingMinimumBudget
		}
	}

	maxTokens := 0
	if request.MaxTokens != nil {
		maxTokens = common.BoundedRequestTokens(*request.MaxTokens)
	}
	if maxTokens == 0 {
		maxTokens = budget + 1
	}
	// Anthropic requires budget_tokens < max_tokens. Keep a caller-provided
	// output cap when it can hold a legal thinking budget; only raise very small
	// caps to the minimum valid envelope.
	if maxTokens <= budget {
		budget = maxTokens - 1
	}
	if budget < claudeEnabledThinkingMinimumBudget {
		maxTokens = claudeEnabledThinkingMinimumBudget + 1
		budget = claudeEnabledThinkingMinimumBudget
	}
	request.MaxTokens = pointerTo(uint(maxTokens))

	request.Thinking = &Thinking{
		Type:         "enabled",
		BudgetTokens: pointerTo(budget),
	}
	request.OutputConfig = nil
	return nil
}

// NormalizeClaudeAdaptiveSampling aligns sampling fields with the thinking mode
// Anthropic accepts for the request's model family.
func NormalizeClaudeAdaptiveSampling(request *ClaudeRequest) {
	if request == nil || request.Thinking == nil || request.Thinking.Type != "adaptive" {
		return
	}
	switch {
	case strings.HasPrefix(request.Model, "claude-opus-4-7"),
		strings.HasPrefix(request.Model, "claude-opus-4-8"):
		// Opus 4.7/4.8 reject non-default temperature/top_p/top_k and omit the
		// visible summary by default.
		request.Thinking.Display = "summarized"
		request.Temperature = nil
		request.TopP = nil
		request.TopK = nil
	case strings.HasPrefix(request.Model, "claude-opus-4-6"):
		request.Temperature = pointerTo[float64](1.0)
	}
}

func pointerTo[T any](value T) *T {
	return &value
}
