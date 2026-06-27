package openaicompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}
	if strings.TrimSpace(req.PreviousResponseID) != "" {
		return nil, errors.New("previous_response_id is not supported when responses is proxied through chat completions")
	}

	messages, err := convertResponsesInputToChatMessages(req.Input)
	if err != nil {
		return nil, err
	}
	if len(req.Instructions) > 0 {
		instructions := rawJSONToString(req.Instructions)
		if strings.TrimSpace(instructions) != "" {
			messages = append([]dto.Message{{
				Role:    "system",
				Content: instructions,
			}}, messages...)
		}
	}
	if len(messages) == 0 {
		messages = []dto.Message{{
			Role:    "user",
			Content: "",
		}}
	}

	out := &dto.GeneralOpenAIRequest{
		Model:               req.Model,
		Messages:            messages,
		Stream:              req.Stream,
		StreamOptions:       req.StreamOptions,
		MaxCompletionTokens: req.MaxOutputTokens,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		TopLogProbs:         req.TopLogProbs,
		Tools:               convertResponsesToolsToChatTools(req.Tools),
		ToolChoice:          convertResponsesToolChoiceToChatToolChoice(req.ToolChoice),
		User:                req.User,
		ServiceTier:         stringToRawMessage(req.ServiceTier),
		Store:               req.Store,
		PromptCacheKey:       rawMessageToPlainString(req.PromptCacheKey),
		PromptCacheRetention: req.PromptCacheRetention,
		SafetyIdentifier:     req.SafetyIdentifier,
		Metadata:             req.Metadata,
	}

	if len(req.ParallelToolCalls) > 0 {
		var parallel bool
		if err := common.Unmarshal(req.ParallelToolCalls, &parallel); err == nil {
			out.ParallelTooCalls = &parallel
		}
	}

	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		out.ReasoningEffort = req.Reasoning.Effort
	}

	if responseFormat := convertResponsesTextToChatResponseFormat(req.Text); responseFormat != nil {
		out.ResponseFormat = responseFormat
	}

	return out, nil
}

func convertResponsesInputToChatMessages(raw json.RawMessage) ([]dto.Message, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	switch common.GetJsonType(raw) {
	case "string":
		return []dto.Message{{
			Role:    "user",
			Content: rawJSONToString(raw),
		}}, nil
	case "array":
		var items []map[string]any
		if err := common.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("invalid responses input: %w", err)
		}
		return convertResponsesInputItemsToChatMessages(items), nil
	case "object":
		var item map[string]any
		if err := common.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("invalid responses input: %w", err)
		}
		return convertResponsesInputItemsToChatMessages([]map[string]any{item}), nil
	default:
		return []dto.Message{{
			Role:    "user",
			Content: string(raw),
		}}, nil
	}
}

func convertResponsesInputItemsToChatMessages(items []map[string]any) []dto.Message {
	messages := make([]dto.Message, 0, len(items))
	for _, item := range items {
		itemType := strings.TrimSpace(common.Interface2String(item["type"]))
		switch itemType {
		case "function_call_output":
			callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
			output := rawAnyToString(item["output"])
			if callID == "" {
				messages = append(messages, dto.Message{
					Role:    "user",
					Content: fmt.Sprintf("[tool_output_missing_call_id] %s", output),
				})
				continue
			}
			messages = append(messages, dto.Message{
				Role:       "tool",
				ToolCallId: callID,
				Content:    output,
			})
		case "function_call":
			name := strings.TrimSpace(common.Interface2String(item["name"]))
			callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
			if callID == "" {
				callID = strings.TrimSpace(common.Interface2String(item["id"]))
			}
			if name == "" || callID == "" {
				continue
			}
			msg := dto.Message{
				Role:    "assistant",
				Content: nil,
			}
			msg.SetToolCalls([]dto.ToolCallRequest{{
				ID:   callID,
				Type: "function",
				Function: dto.FunctionRequest{
					Name:      name,
					Arguments: rawAnyToString(item["arguments"]),
				},
			}})
			messages = append(messages, msg)
		case "message", "":
			role := strings.TrimSpace(common.Interface2String(item["role"]))
			if role == "" {
				role = "user"
			}
			messages = append(messages, dto.Message{
				Role:    responsesRoleToChatRole(role),
				Content: convertResponsesContentToChatContent(item["content"], role),
			})
		default:
			messages = append(messages, dto.Message{
				Role:    "user",
				Content: rawAnyToString(item),
			})
		}
	}
	return messages
}

func convertResponsesContentToChatContent(content any, role string) any {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		parts := make([]any, 0, len(v))
		var text strings.Builder
		allText := true
		for _, partAny := range v {
			part, ok := partAny.(map[string]any)
			if !ok {
				allText = false
				continue
			}
			contentPart := responsesContentPartToChat(part, role)
			if contentPart.Type == dto.ContentTypeText {
				text.WriteString(contentPart.Text)
			} else {
				allText = false
			}
			parts = append(parts, contentPart)
		}
		if allText {
			return text.String()
		}
		return parts
	default:
		return rawAnyToString(v)
	}
}

func responsesContentPartToChat(part map[string]any, role string) dto.MediaContent {
	partType := strings.TrimSpace(common.Interface2String(part["type"]))
	switch partType {
	case "input_text", "output_text", "text":
		return dto.MediaContent{
			Type: dto.ContentTypeText,
			Text: common.Interface2String(part["text"]),
		}
	case "input_image":
		return dto.MediaContent{
			Type:     dto.ContentTypeImageURL,
			ImageUrl: normalizeResponsesImageURL(part["image_url"]),
		}
	case "input_file":
		return dto.MediaContent{
			Type: dto.ContentTypeFile,
			File: normalizeResponsesFile(part),
		}
	default:
		if text := common.Interface2String(part["text"]); text != "" {
			return dto.MediaContent{Type: dto.ContentTypeText, Text: text}
		}
		return dto.MediaContent{Type: dto.ContentTypeText, Text: rawAnyToString(part)}
	}
}

func normalizeResponsesImageURL(v any) any {
	switch vv := v.(type) {
	case string:
		return &dto.MessageImageUrl{Url: vv}
	case map[string]any:
		return &dto.MessageImageUrl{
			Url:    common.Interface2String(vv["url"]),
			Detail: common.Interface2String(vv["detail"]),
		}
	default:
		return v
	}
}

func normalizeResponsesFile(part map[string]any) any {
	file := map[string]any{}
	for _, key := range []string{"file_id", "filename", "file_name", "file_data"} {
		if value, ok := part[key]; ok {
			file[key] = value
		}
	}
	if len(file) > 0 {
		return file
	}
	if value, ok := part["file"]; ok {
		return value
	}
	return part
}

func convertResponsesToolsToChatTools(raw json.RawMessage) []dto.ToolCallRequest {
	if len(raw) == 0 {
		return nil
	}
	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil
	}
	out := make([]dto.ToolCallRequest, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(common.Interface2String(tool["type"])) != "function" {
			continue
		}
		name := strings.TrimSpace(common.Interface2String(tool["name"]))
		if name == "" {
			continue
		}
		out = append(out, dto.ToolCallRequest{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:        name,
				Description: common.Interface2String(tool["description"]),
				Parameters:  tool["parameters"],
			},
		})
	}
	return out
}

func convertResponsesToolChoiceToChatToolChoice(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	if common.GetJsonType(raw) == "string" {
		return rawJSONToString(raw)
	}
	var m map[string]any
	if err := common.Unmarshal(raw, &m); err != nil {
		return raw
	}
	if common.Interface2String(m["type"]) == "function" {
		name := strings.TrimSpace(common.Interface2String(m["name"]))
		if name == "" {
			return raw
		}
		return map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": name,
			},
		}
	}
	return raw
}

func convertResponsesTextToChatResponseFormat(raw json.RawMessage) *dto.ResponseFormat {
	if len(raw) == 0 {
		return nil
	}
	var text map[string]any
	if err := common.Unmarshal(raw, &text); err != nil {
		return nil
	}
	formatAny, ok := text["format"]
	if !ok {
		return nil
	}
	format, ok := formatAny.(map[string]any)
	if !ok {
		return nil
	}
	formatType := strings.TrimSpace(common.Interface2String(format["type"]))
	if formatType == "" {
		return nil
	}
	out := &dto.ResponseFormat{Type: formatType}
	if formatType == "json_schema" {
		rawSchema, _ := common.Marshal(format)
		out.JsonSchema = rawSchema
	}
	return out
}

func responsesRoleToChatRole(role string) string {
	switch role {
	case "developer":
		return "system"
	default:
		return role
	}
}

func rawJSONToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := common.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func rawMessageToPlainString(raw json.RawMessage) string {
	return strings.TrimSpace(rawJSONToString(raw))
}

func rawAnyToString(v any) string {
	switch vv := v.(type) {
	case nil:
		return ""
	case string:
		return vv
	case json.RawMessage:
		return rawJSONToString(vv)
	default:
		b, err := common.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
