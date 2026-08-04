package oaichat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
	"github.com/samber/lo"
)

func normalizeChatImageURLToString(v any) any {
	switch vv := v.(type) {
	case string:
		return vv
	case map[string]any:
		if url := common.Interface2String(vv["url"]); url != "" {
			return url
		}
		return v
	case dto.MessageImageUrl:
		if vv.Url != "" {
			return vv.Url
		}
		return v
	case *dto.MessageImageUrl:
		if vv != nil && vv.Url != "" {
			return vv.Url
		}
		return v
	default:
		return v
	}
}

func convertChatResponseFormatToResponsesText(reqFormat *dto.ResponseFormat) json.RawMessage {
	if reqFormat == nil || strings.TrimSpace(reqFormat.Type) == "" {
		return nil
	}

	format := map[string]any{
		"type": reqFormat.Type,
	}

	if reqFormat.Type == "json_schema" && len(reqFormat.JsonSchema) > 0 {
		var chatSchema map[string]any
		if err := common.Unmarshal(reqFormat.JsonSchema, &chatSchema); err == nil {
			for key, value := range chatSchema {
				if key == "type" {
					continue
				}
				format[key] = value
			}

			if nested, ok := format["json_schema"].(map[string]any); ok {
				for key, value := range nested {
					if _, exists := format[key]; !exists {
						format[key] = value
					}
				}
				delete(format, "json_schema")
			}
		} else {
			format["json_schema"] = reqFormat.JsonSchema
		}
	}

	textRaw, _ := common.Marshal(map[string]any{
		"format": format,
	})
	return textRaw
}

func ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if lo.FromPtrOr(req.N, 1) > 1 {
		return nil, fmt.Errorf("n>1 is not supported in responses compatibility mode")
	}

	var instructionsParts []string
	inputItems := make([]map[string]any, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}

		if role == "tool" || role == "function" {
			callID := strings.TrimSpace(msg.ToolCallId)

			var output any
			if msg.Content == nil {
				output = ""
			} else if msg.IsStringContent() {
				output = msg.StringContent()
			} else {
				if b, err := common.Marshal(msg.Content); err == nil {
					output = string(b)
				} else {
					output = fmt.Sprintf("%v", msg.Content)
				}
			}

			if callID == "" {
				inputItems = append(inputItems, map[string]any{
					"role":    "user",
					"content": fmt.Sprintf("[tool_output_missing_call_id] %v", output),
				})
				continue
			}

			inputItems = append(inputItems, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  output,
			})
			continue
		}

		// Prefer mapping system/developer messages into `instructions`.
		if role == "system" || role == "developer" {
			if msg.Content == nil {
				continue
			}
			if msg.IsStringContent() {
				if s := strings.TrimSpace(msg.StringContent()); s != "" {
					instructionsParts = append(instructionsParts, s)
				}
				continue
			}
			parts := msg.ParseContent()
			var sb strings.Builder
			for _, part := range parts {
				if part.Type == dto.ContentTypeText && strings.TrimSpace(part.Text) != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(part.Text)
				}
			}
			if s := strings.TrimSpace(sb.String()); s != "" {
				instructionsParts = append(instructionsParts, s)
			}
			continue
		}

		item := map[string]any{
			"role": role,
		}

		if msg.Content == nil {
			item["content"] = ""
			inputItems = append(inputItems, item)

			if role == "assistant" {
				for _, tc := range msg.ParseToolCalls() {
					if strings.TrimSpace(tc.ID) == "" {
						continue
					}
					if tc.Type != "" && tc.Type != "function" {
						continue
					}
					name := strings.TrimSpace(tc.Function.Name)
					if name == "" {
						continue
					}
					inputItems = append(inputItems, map[string]any{
						"type":      "function_call",
						"call_id":   tc.ID,
						"name":      name,
						"arguments": tc.Function.Arguments,
					})
				}
			}
			continue
		}

		if msg.IsStringContent() {
			item["content"] = msg.StringContent()
			inputItems = append(inputItems, item)

			if role == "assistant" {
				for _, tc := range msg.ParseToolCalls() {
					if strings.TrimSpace(tc.ID) == "" {
						continue
					}
					if tc.Type != "" && tc.Type != "function" {
						continue
					}
					name := strings.TrimSpace(tc.Function.Name)
					if name == "" {
						continue
					}
					inputItems = append(inputItems, map[string]any{
						"type":      "function_call",
						"call_id":   tc.ID,
						"name":      name,
						"arguments": tc.Function.Arguments,
					})
				}
			}
			continue
		}

		parts := msg.ParseContent()
		contentParts := make([]map[string]any, 0, len(parts))
		for _, part := range parts {
			switch part.Type {
			case dto.ContentTypeText:
				textType := "input_text"
				if role == "assistant" {
					textType = "output_text"
				}
				contentParts = append(contentParts, map[string]any{
					"type": textType,
					"text": part.Text,
				})
			case dto.ContentTypeImageURL:
				contentParts = append(contentParts, map[string]any{
					"type":      "input_image",
					"image_url": normalizeChatImageURLToString(part.ImageUrl),
				})
			case dto.ContentTypeInputAudio:
				contentParts = append(contentParts, map[string]any{
					"type":        "input_audio",
					"input_audio": part.InputAudio,
				})
			case dto.ContentTypeFile:
				contentParts = append(contentParts, map[string]any{
					"type": "input_file",
					"file": part.File,
				})
			case dto.ContentTypeVideoUrl:
				contentParts = append(contentParts, map[string]any{
					"type":      "input_video",
					"video_url": part.VideoUrl,
				})
			default:
				contentParts = append(contentParts, map[string]any{
					"type": part.Type,
				})
			}
		}
		item["content"] = contentParts
		inputItems = append(inputItems, item)

		if role == "assistant" {
			for _, tc := range msg.ParseToolCalls() {
				if strings.TrimSpace(tc.ID) == "" {
					continue
				}
				if tc.Type != "" && tc.Type != "function" {
					continue
				}
				name := strings.TrimSpace(tc.Function.Name)
				if name == "" {
					continue
				}
				inputItems = append(inputItems, map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      name,
					"arguments": tc.Function.Arguments,
				})
			}
		}
	}

	inputRaw, err := common.Marshal(inputItems)
	if err != nil {
		return nil, err
	}

	var instructionsRaw json.RawMessage
	if len(instructionsParts) > 0 {
		instructions := strings.Join(instructionsParts, "\n\n")
		instructionsRaw, _ = common.Marshal(instructions)
	}

	toolsRaw := convertChatToolsToResponsesTools(req)

	toolChoiceRaw := convertChatToolChoiceToResponsesToolChoice(req)

	var parallelToolCallsRaw json.RawMessage
	if req.ParallelTooCalls != nil {
		parallelToolCallsRaw, _ = common.Marshal(*req.ParallelTooCalls)
	}

	textRaw := convertChatResponseFormatToResponsesText(req.ResponseFormat)
	serviceTier := rawMessageToString(req.ServiceTier)

	maxOutputTokens := lo.FromPtrOr(req.MaxTokens, uint(0))
	maxCompletionTokens := lo.FromPtrOr(req.MaxCompletionTokens, uint(0))
	if maxCompletionTokens > maxOutputTokens {
		maxOutputTokens = maxCompletionTokens
	}
	// OpenAI Responses API rejects max_output_tokens < 16 when explicitly provided.
	//if maxOutputTokens > 0 && maxOutputTokens < 16 {
	//	maxOutputTokens = 16
	//}

	var topP *float64
	if req.TopP != nil {
		topP = common.GetPointer(lo.FromPtr(req.TopP))
	}

	out := &dto.OpenAIResponsesRequest{
		Model:             req.Model,
		Input:             inputRaw,
		Instructions:      instructionsRaw,
		Stream:            req.Stream,
		Temperature:       req.Temperature,
		Text:              textRaw,
		ToolChoice:        toolChoiceRaw,
		Tools:             toolsRaw,
		TopLogProbs:       req.TopLogProbs,
		TopP:              topP,
		User:              req.User,
		ParallelToolCalls: parallelToolCallsRaw,
		Store:             req.Store,
		Metadata:          req.Metadata,
		ServiceTier:       serviceTier,
		StreamOptions:     req.StreamOptions,
		PromptCacheKey:       stringToRawMessage(req.PromptCacheKey),
		PromptCacheRetention: req.PromptCacheRetention,
		SafetyIdentifier:     req.SafetyIdentifier,
	}
	if req.MaxTokens != nil || req.MaxCompletionTokens != nil {
		out.MaxOutputTokens = lo.ToPtr(maxOutputTokens)
	}

	if req.ReasoningEffort != "" {
		out.Reasoning = &dto.Reasoning{
			Effort:  req.ReasoningEffort,
			Summary: "detailed",
		}
	}

	return out, nil
}

func convertChatToolsToResponsesTools(req *dto.GeneralOpenAIRequest) json.RawMessage {
	if req == nil {
		return nil
	}

	tools := make([]map[string]any, 0, len(req.Tools))
	for _, tool := range req.Tools {
		switch tool.Type {
		case "function":
			tools = append(tools, map[string]any{
				"type":        "function",
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			})
		default:
			// Best-effort: keep original tool shape for unknown types.
			var m map[string]any
			if b, err := common.Marshal(tool); err == nil {
				_ = common.Unmarshal(b, &m)
			}
			if len(m) == 0 {
				m = map[string]any{"type": tool.Type}
			}
			tools = append(tools, m)
		}
	}

	if len(req.Functions) > 0 {
		var functions []dto.FunctionRequest
		if err := common.Unmarshal(req.Functions, &functions); err == nil {
			for _, fn := range functions {
				if strings.TrimSpace(fn.Name) == "" {
					continue
				}
				tools = append(tools, map[string]any{
					"type":        "function",
					"name":        fn.Name,
					"description": fn.Description,
					"parameters":  fn.Parameters,
				})
			}
		} else {
			var rawFunctions []map[string]any
			if err := common.Unmarshal(req.Functions, &rawFunctions); err == nil {
				for _, fn := range rawFunctions {
					if strings.TrimSpace(common.Interface2String(fn["name"])) == "" {
						continue
					}
					fn["type"] = "function"
					tools = append(tools, fn)
				}
			}
		}
	}

	if len(tools) == 0 {
		return nil
	}
	toolsRaw, _ := common.Marshal(tools)
	return toolsRaw
}

func convertChatToolChoiceToResponsesToolChoice(req *dto.GeneralOpenAIRequest) json.RawMessage {
	if req == nil {
		return nil
	}
	if req.ToolChoice != nil {
		return convertToolChoiceValueToResponses(req.ToolChoice)
	}
	if len(req.FunctionCall) == 0 {
		return nil
	}

	var functionCall any
	if err := common.Unmarshal(req.FunctionCall, &functionCall); err != nil {
		return req.FunctionCall
	}

	switch v := functionCall.(type) {
	case string:
		if v == "none" || v == "auto" || v == "required" {
			raw, _ := common.Marshal(v)
			return raw
		}
		raw, _ := common.Marshal(map[string]any{
			"type": "function",
			"name": v,
		})
		return raw
	case map[string]any:
		if name := strings.TrimSpace(common.Interface2String(v["name"])); name != "" {
			raw, _ := common.Marshal(map[string]any{
				"type": "function",
				"name": name,
			})
			return raw
		}
	}
	return req.FunctionCall
}

func convertToolChoiceValueToResponses(toolChoice any) json.RawMessage {
	switch v := toolChoice.(type) {
	case string:
		raw, _ := common.Marshal(v)
		return raw
	default:
		var m map[string]any
		if b, err := common.Marshal(v); err == nil {
			_ = common.Unmarshal(b, &m)
		}
		if m == nil {
			raw, _ := common.Marshal(v)
			return raw
		}
		if t, _ := m["type"].(string); t == "function" {
			// Chat: {"type":"function","function":{"name":"..."}}
			// Responses: {"type":"function","name":"..."}
			if name := strings.TrimSpace(common.Interface2String(m["name"])); name != "" {
				raw, _ := common.Marshal(map[string]any{
					"type": "function",
					"name": name,
				})
				return raw
			}
			if fn, ok := m["function"].(map[string]any); ok {
				if name := strings.TrimSpace(common.Interface2String(fn["name"])); name != "" {
					raw, _ := common.Marshal(map[string]any{
						"type": "function",
						"name": name,
					})
					return raw
				}
			}
		}
		raw, _ := common.Marshal(v)
		return raw
	}
}

func rawMessageToString(raw json.RawMessage) string {
	return common.JsonRawMessageToString(raw)
}

func stringToRawMessage(s string) json.RawMessage {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	raw, _ := common.Marshal(s)
	return raw
}
