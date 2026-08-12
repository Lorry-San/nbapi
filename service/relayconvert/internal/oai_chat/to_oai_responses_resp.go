package oaichat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
)

const (
	chatFinishReasonLength        = "length"
	chatFinishReasonContentFilter = "content_filter"

	responsesEventCreated                  = "response.created"
	responsesEventCompleted                = "response.completed"
	responsesEventIncomplete               = "response.incomplete"
	responsesEventFailed                   = "response.failed"
	responsesEventOutputTextDelta          = "response.output_text.delta"
	responsesEventOutputItemAdded          = "response.output_item.added"
	responsesEventOutputItemDone           = "response.output_item.done"
	responsesEventFunctionArgsDelta        = "response.function_call_arguments.delta"
	responsesEventFunctionArgsDone         = "response.function_call_arguments.done"
	responsesEventCustomToolInputDelta     = "response.custom_tool_call_input.delta"
	responsesEventCustomToolInputDone      = "response.custom_tool_call_input.done"
	responsesEventReasoningSummaryDelta    = "response.reasoning_summary_text.delta"
	responsesEventReasoningSummaryDone     = "response.reasoning_summary_text.done"
	responsesOutputTypeFunctionCall        = "function_call"
	responsesOutputTypeCustomToolCall      = "custom_tool_call"
	responsesOutputTypeToolSearchCall      = "tool_search_call"
	responsesOutputTypeMessage             = "message"
	responsesOutputTypeReasoning           = "reasoning"
	responsesIncompleteReasonContentFilter = "content_filter"
	responsesIncompleteReasonMaxTokens     = "max_output_tokens"
	responsesIncompleteReasonStreamCut     = "stream_truncated"
)

type ChatCompletionsToResponsesOptions struct {
	ThinkingToContent bool
	PreserveReasoning bool
	ToolMappings      map[string]dto.ResponsesToolMapping
}

func ChatCompletionsResponseToResponsesResponse(resp *dto.OpenAITextResponse, id string) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	return ChatCompletionsResponseToResponsesResponseWithOptions(resp, id, ChatCompletionsToResponsesOptions{
		PreserveReasoning: true,
	})
}

func ChatCompletionsResponseToResponsesResponseWithOptions(resp *dto.OpenAITextResponse, id string, options ChatCompletionsToResponsesOptions) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	if resp == nil {
		return nil, nil, errors.New("response is nil")
	}

	usage := UsageFromChatUsage(&resp.Usage)
	out := &dto.OpenAIResponsesResponse{
		ID:        id,
		Object:    "response",
		CreatedAt: chatCreatedAt(resp.Created),
		Status:    []byte(`"completed"`),
		Model:     resp.Model,
		Output:    make([]dto.ResponsesOutput, 0),
		Usage:     usage,
	}

	if len(resp.Choices) == 0 {
		return nil, usage, errors.New("upstream Chat Completions response has no choices")
	}

	choice := resp.Choices[0]
	if status, details := ResponsesStatusFromChatFinishReason(choice.FinishReason); status != "" {
		out.Status = []byte(fmt.Sprintf("%q", status))
		out.IncompleteDetails = details
	}

	text := choice.Message.StringContent()
	reasoning := choice.Message.GetReasoningContent()
	if options.ThinkingToContent {
		text = joinVisibleReasoningAndContent(reasoning, text)
	}
	if text != "" {
		out.Output = append(out.Output, dto.ResponsesOutput{
			Type:   responsesOutputTypeMessage,
			ID:     fmt.Sprintf("%s_msg_0", id),
			Status: responseOutputStatus(out),
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{
					Type:        "output_text",
					Text:        text,
					Annotations: []interface{}{},
				},
			},
		})
	}
	if reasoning != "" && options.PreserveReasoning && !options.ThinkingToContent {
		out.Output = append(out.Output, dto.ResponsesOutput{
			Type:   responsesOutputTypeReasoning,
			ID:     fmt.Sprintf("%s_reasoning_0", id),
			Status: responseOutputStatus(out),
			Content: []dto.ResponsesOutputContent{
				{
					Type: "summary_text",
					Text: reasoning,
				},
			},
		})
	}

	toolCalls := choice.Message.ParseToolCalls()
	validToolCalls := 0
	droppedToolCalls := 0
	for i, toolCall := range toolCalls {
		toolOutput, ok, err := chatToolCallToResponsesOutput(toolCall, id, i, responseOutputStatus(out), options.ToolMappings)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			droppedToolCalls++
			continue
		}
		validToolCalls++
		out.Output = append(out.Output, toolOutput)
	}
	if responseStatusString(out) == "completed" && droppedToolCalls > 0 && validToolCalls == 0 {
		return nil, nil, fmt.Errorf("upstream returned %d tool call(s) without a function name, leaving no usable tool call in this turn", droppedToolCalls)
	}

	return out, usage, nil
}

func joinVisibleReasoningAndContent(reasoning string, content string) string {
	if reasoning == "" {
		return content
	}
	visibleReasoning := "<think>\n" + reasoning + "\n</think>"
	if content == "" {
		return visibleReasoning
	}
	return visibleReasoning + "\n" + content
}

func ResponsesStatusFromChatFinishReason(finishReason string) (string, *dto.IncompleteDetails) {
	switch strings.TrimSpace(finishReason) {
	case chatFinishReasonLength:
		return "incomplete", &dto.IncompleteDetails{Reason: responsesIncompleteReasonMaxTokens}
	case chatFinishReasonContentFilter:
		return "incomplete", &dto.IncompleteDetails{Reason: responsesIncompleteReasonContentFilter}
	default:
		return "completed", nil
	}
}

func UsageFromChatUsage(src *dto.Usage) *dto.Usage {
	usage := &dto.Usage{}
	if src == nil {
		return usage
	}
	usage.UsageSemantic = src.UsageSemantic
	usage.UsageSource = src.UsageSource
	usage.BillingUsage = dto.CloneBillingUsage(src.BillingUsage)
	if usage.BillingUsage == nil {
		usage.BillingUsage = dto.NewOpenAIChatBillingUsage(src)
	}
	usage.Cost = src.Cost
	if src.PromptTokens != 0 {
		usage.PromptTokens = src.PromptTokens
		usage.InputTokens = src.PromptTokens
	}
	if src.CompletionTokens != 0 {
		usage.CompletionTokens = src.CompletionTokens
		usage.OutputTokens = src.CompletionTokens
	}
	if src.TotalTokens != 0 {
		usage.TotalTokens = src.TotalTokens
	} else {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	if src.PromptTokensDetails.CachedTokens != 0 ||
		src.PromptTokensDetails.ImageTokens != 0 ||
		src.PromptTokensDetails.AudioTokens != 0 ||
		src.PromptTokensDetails.CachedCreationTokens != 0 ||
		src.PromptTokensDetails.CacheWriteTokens != 0 ||
		src.PromptTokensDetails.TextTokens != 0 {
		details := src.PromptTokensDetails
		usage.InputTokensDetails = &details
	}
	if src.CompletionTokenDetails.ReasoningTokens != 0 ||
		src.CompletionTokenDetails.TextTokens != 0 ||
		src.CompletionTokenDetails.AudioTokens != 0 ||
		src.CompletionTokenDetails.ImageTokens != 0 {
		usage.CompletionTokenDetails = src.CompletionTokenDetails
	}
	usage.ClaudeCacheCreation5mTokens = src.ClaudeCacheCreation5mTokens
	usage.ClaudeCacheCreation1hTokens = src.ClaudeCacheCreation1hTokens
	return usage
}

func responseOutputStatus(resp *dto.OpenAIResponsesResponse) string {
	if resp == nil || responseStatusString(resp) != "incomplete" {
		return "completed"
	}
	return "incomplete"
}

func responseStatusString(resp *dto.OpenAIResponsesResponse) string {
	if resp == nil || len(resp.Status) == 0 {
		return ""
	}
	var status string
	_ = common.Unmarshal(resp.Status, &status)
	return strings.TrimSpace(status)
}

func chatToolCallToResponsesOutput(toolCall dto.ToolCallRequest, responseID string, index int, status string, mappings map[string]dto.ResponsesToolMapping) (dto.ResponsesOutput, bool, error) {
	callID := strings.TrimSpace(toolCall.ID)
	if callID == "" {
		callID = fmt.Sprintf("%s_call_%d", responseID, index)
	}
	if toolCall.Type == "" || toolCall.Type == "function" {
		chatName := strings.TrimSpace(toolCall.Function.Name)
		if chatName == "" {
			return dto.ResponsesOutput{}, false, nil
		}
		return chatToolCallOutputFromMapping(callID, chatName, toolCall.Function.Arguments, status, mappings[chatName]), true, nil
	}
	return dto.ResponsesOutput{
		Type:      toolCall.Type,
		ID:        callID,
		Status:    status,
		CallId:    callID,
		Arguments: toolCall.Custom,
	}, true, nil
}

func chatToolCallOutputFromMapping(callID string, chatName string, arguments string, status string, mapping dto.ResponsesToolMapping) dto.ResponsesOutput {
	itemID := chatToolCallItemID(callID, mapping)
	switch mapping.Kind {
	case dto.ResponsesToolKindCustom:
		return dto.ResponsesOutput{
			Type:   responsesOutputTypeCustomToolCall,
			ID:     itemID,
			Status: status,
			CallId: callID,
			Name:   mapping.Name,
			Input:  customToolInputFromChatArguments(arguments),
		}
	case dto.ResponsesToolKindToolSearch:
		return dto.ResponsesOutput{
			Type:      responsesOutputTypeToolSearchCall,
			Status:    status,
			CallId:    callID,
			Execution: "client",
			Arguments: chatToolArgumentsObjectRaw(arguments),
		}
	default:
		name := strings.TrimSpace(mapping.Name)
		if name == "" {
			name = chatName
		}
		return dto.ResponsesOutput{
			Type:      responsesOutputTypeFunctionCall,
			ID:        itemID,
			Status:    status,
			CallId:    callID,
			Name:      name,
			Namespace: mapping.Namespace,
			Arguments: chatArgumentsRawMessage(arguments),
		}
	}
}

func chatToolCallItemID(callID string, mapping dto.ResponsesToolMapping) string {
	if mapping.Kind == "" {
		return callID
	}
	if mapping.Kind == dto.ResponsesToolKindCustom {
		return "ctc_" + callID
	}
	return "fc_" + callID
}

func customToolInputFromChatArguments(arguments string) string {
	if strings.TrimSpace(arguments) == "" {
		return ""
	}
	var value map[string]any
	if err := common.Unmarshal([]byte(arguments), &value); err == nil {
		if input, ok := value["input"].(string); ok {
			return input
		}
	}
	return arguments
}

func chatToolArgumentsObjectRaw(arguments string) json.RawMessage {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	var value map[string]any
	if err := common.Unmarshal([]byte(trimmed), &value); err == nil {
		if raw, err := common.Marshal(value); err == nil {
			return raw
		}
	}
	raw, _ := common.Marshal(map[string]any{"query": arguments})
	return raw
}

func chatArgumentsRawMessage(arguments string) []byte {
	raw, err := common.Marshal(arguments)
	if err != nil {
		return []byte(`""`)
	}
	return raw
}

func chatCreatedAt(created any) int {
	switch v := created.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if parsed := common.String2Int(v); parsed != 0 {
			return parsed
		}
	}
	return int(time.Now().Unix())
}

func responsesStreamEvent(eventType string, payload dto.ResponsesStreamResponse) ChatToResponsesStreamEvent {
	payload.Type = eventType
	return ChatToResponsesStreamEvent{
		Type:    eventType,
		Payload: payload,
	}
}

func intPtr(v int) *int {
	return &v
}
