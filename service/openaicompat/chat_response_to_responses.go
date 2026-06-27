package openaicompat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/dto"
)

type ChatCompletionsToResponsesOptions struct {
	ThinkingToContent bool
}

func ChatCompletionsResponseToResponsesResponse(resp *dto.OpenAITextResponse, id string) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	return ChatCompletionsResponseToResponsesResponseWithOptions(resp, id, ChatCompletionsToResponsesOptions{})
}

func ChatCompletionsResponseToResponsesResponseWithOptions(resp *dto.OpenAITextResponse, id string, options ChatCompletionsToResponsesOptions) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	if resp == nil {
		return nil, nil, fmt.Errorf("chat completions response is nil")
	}
	if strings.TrimSpace(id) == "" {
		id = strings.TrimSpace(resp.Id)
	}
	if id == "" {
		id = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}

	createdAt := int(time.Now().Unix())
	switch v := resp.Created.(type) {
	case int:
		if v != 0 {
			createdAt = v
		}
	case int64:
		if v != 0 {
			createdAt = int(v)
		}
	case float64:
		if v != 0 {
			createdAt = int(v)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil && n != 0 {
			createdAt = int(n)
		}
	}

	output := make([]dto.ResponsesOutput, 0, 2)
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		text := choice.Message.StringContent()
		if options.ThinkingToContent {
			text = JoinVisibleReasoningAndContent(choice.Message.GetReasoningContent(), text)
		}
		if text != "" {
			output = append(output, dto.ResponsesOutput{
				Type:   "message",
				ID:     "msg_" + id,
				Status: "completed",
				Role:   "assistant",
				Content: []dto.ResponsesOutputContent{
					{
						Type: "output_text",
						Text: text,
					},
				},
			})
		}
		for _, tc := range choice.Message.ParseToolCalls() {
			if strings.TrimSpace(tc.Function.Name) == "" {
				continue
			}
			callID := strings.TrimSpace(tc.ID)
			if callID == "" {
				callID = fmt.Sprintf("call_%d", len(output)+1)
			}
			output = append(output, dto.ResponsesOutput{
				Type:      "function_call",
				ID:        "fc_" + callID,
				Status:    "completed",
				CallId:    callID,
				Name:      tc.Function.Name,
				Arguments: chatToolArgumentsToResponsesRaw(tc.Function.Arguments),
			})
		}
	}

	usage := resp.Usage
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	usage.InputTokens = usage.PromptTokens
	usage.OutputTokens = usage.CompletionTokens
	if usage.InputTokensDetails == nil {
		details := usage.PromptTokensDetails
		usage.InputTokensDetails = &details
	}

	out := &dto.OpenAIResponsesResponse{
		ID:        id,
		Object:    "response",
		CreatedAt: createdAt,
		Status:    json.RawMessage(`"completed"`),
		Model:     resp.Model,
		Output:    output,
		Usage:     &usage,
	}
	return out, &usage, nil
}

func JoinVisibleReasoningAndContent(reasoning string, content string) string {
	if reasoning == "" {
		return content
	}
	visibleReasoning := "<think>\n" + reasoning + "\n</think>"
	if content == "" {
		return visibleReasoning
	}
	return visibleReasoning + "\n" + content
}

func chatToolArgumentsToResponsesRaw(arguments string) json.RawMessage {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	raw := json.RawMessage(trimmed)
	if json.Valid(raw) {
		return raw
	}
	quoted, err := json.Marshal(arguments)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return quoted
}
