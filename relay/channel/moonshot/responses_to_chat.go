package moonshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
)

const moonshotChatToolTypeFunction = "function"

func convertMoonshotResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}
	if err := validateMoonshotResponsesChatUnsupportedFields(req); err != nil {
		return nil, err
	}

	messages, err := moonshotResponsesInputToChatMessages(req.Input)
	if err != nil {
		return nil, err
	}
	if moonshotRawJSONPresent(req.Instructions) {
		instructions := moonshotRawJSONToString(req.Instructions)
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

	tools := moonshotResponsesToolsToChatTools(req.Tools)
	out := &dto.GeneralOpenAIRequest{
		Model:               req.Model,
		Messages:            messages,
		Stream:              req.Stream,
		StreamOptions:       req.StreamOptions,
		MaxCompletionTokens: req.MaxOutputTokens,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		TopLogProbs:         req.TopLogProbs,
		Tools:               tools,
		ToolChoice:          moonshotResponsesToolChoiceToChatToolChoice(req.ToolChoice, tools),
		User:                req.User,
	}

	if len(tools) > 0 && moonshotRawJSONPresent(req.ParallelToolCalls) {
		var parallel bool
		if err := common.Unmarshal(req.ParallelToolCalls, &parallel); err == nil {
			out.ParallelTooCalls = &parallel
		}
	}
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		out.ReasoningEffort = req.Reasoning.Effort
	}
	if responseFormat := moonshotResponsesTextToChatResponseFormat(req.Text); responseFormat != nil {
		out.ResponseFormat = responseFormat
	}

	return out, nil
}

func validateMoonshotResponsesChatUnsupportedFields(req *dto.OpenAIResponsesRequest) error {
	unsupported := make([]string, 0, 4)
	if moonshotRawJSONPresent(req.Conversation) {
		unsupported = append(unsupported, "conversation")
	}
	if strings.TrimSpace(req.PreviousResponseID) != "" {
		unsupported = append(unsupported, "previous_response_id")
	}
	if moonshotRawJSONPresent(req.Prompt) {
		unsupported = append(unsupported, "prompt")
	}
	if moonshotRawJSONPresent(req.ContextManagement) {
		unsupported = append(unsupported, "context_management")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("moonshot responses to chat conversion does not support stateful fields: %s", strings.Join(unsupported, ", "))
	}
	return nil
}

func moonshotResponsesInputToChatMessages(raw json.RawMessage) ([]dto.Message, error) {
	if !moonshotRawJSONPresent(raw) {
		return nil, nil
	}

	switch common.GetJsonType(raw) {
	case "string":
		return []dto.Message{{
			Role:    "user",
			Content: moonshotRawJSONToString(raw),
		}}, nil
	case "array":
		var items []map[string]any
		if err := common.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("invalid responses input: %w", err)
		}
		return moonshotResponsesInputItemsToChatMessages(items), nil
	case "object":
		var item map[string]any
		if err := common.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("invalid responses input: %w", err)
		}
		return moonshotResponsesInputItemsToChatMessages([]map[string]any{item}), nil
	default:
		return []dto.Message{{
			Role:    "user",
			Content: string(raw),
		}}, nil
	}
}

func moonshotResponsesInputItemsToChatMessages(items []map[string]any) []dto.Message {
	messages := make([]dto.Message, 0, len(items))
	knownFunctionCallIDs := make(map[string]struct{})

	for i := 0; i < len(items); i++ {
		item := items[i]
		itemType := strings.TrimSpace(common.Interface2String(item["type"]))
		switch itemType {
		case "reasoning":
			// reasoning 默认隐藏：历史推理内容不应转换为聊天消息发给上游。
			continue
		case "function_call":
			toolCalls := make([]dto.ToolCallRequest, 0, 1)
			for ; i < len(items); i++ {
				nextItem := items[i]
				if strings.TrimSpace(common.Interface2String(nextItem["type"])) != "function_call" {
					break
				}
				toolCall, ok := moonshotResponsesFunctionCallToChatToolCall(nextItem)
				if ok {
					toolCalls = append(toolCalls, toolCall)
					knownFunctionCallIDs[toolCall.ID] = struct{}{}
				}
			}
			i--
			if len(toolCalls) == 0 {
				continue
			}
			msg := dto.Message{
				Role:    "assistant",
				Content: nil,
			}
			msg.SetToolCalls(toolCalls)
			messages = append(messages, msg)
		case "function_call_output":
			callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
			if _, ok := knownFunctionCallIDs[callID]; callID != "" && ok {
				messages = append(messages, dto.Message{
					Role:       "tool",
					ToolCallId: callID,
					Content:    moonshotRawAnyToString(item["output"]),
				})
				continue
			}
			messages = append(messages, dto.Message{
				Role:    "user",
				Content: moonshotRawAnyToString(item),
			})
		case "message", "":
			role := moonshotResponsesRoleToChatRole(common.Interface2String(item["role"]))
			content := moonshotResponsesContentToChatContent(item["content"], role)
			if role == "system" {
				if text, ok := content.(string); !ok || strings.TrimSpace(text) == "" {
					// 上游拒绝空 system 消息（400: system must not be empty），跳过而不是原样发送。
					continue
				}
			}
			messages = append(messages, dto.Message{
				Role:    role,
				Content: content,
			})
		default:
			messages = append(messages, dto.Message{
				Role:    "user",
				Content: moonshotRawAnyToString(item),
			})
		}
	}

	return messages
}

func moonshotResponsesFunctionCallToChatToolCall(item map[string]any) (dto.ToolCallRequest, bool) {
	name := strings.TrimSpace(common.Interface2String(item["name"]))
	callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
	if callID == "" {
		callID = strings.TrimSpace(common.Interface2String(item["id"]))
	}
	if !moonshotValidChatFunctionName(name) || callID == "" {
		return dto.ToolCallRequest{}, false
	}
	return dto.ToolCallRequest{
		ID:   callID,
		Type: moonshotChatToolTypeFunction,
		Function: dto.FunctionRequest{
			Name:      name,
			Arguments: moonshotRawAnyToString(item["arguments"]),
		},
	}, true
}

func moonshotResponsesContentToChatContent(content any, role string) any {
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
			contentPart := moonshotResponsesContentPartToChat(part, role)
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
		return moonshotRawAnyToString(v)
	}
}

func moonshotResponsesContentPartToChat(part map[string]any, role string) dto.MediaContent {
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
			ImageUrl: moonshotNormalizeResponsesImageURL(part),
		}
	case "input_file":
		return dto.MediaContent{
			Type: dto.ContentTypeFile,
			File: moonshotNormalizeResponsesFile(part),
		}
	default:
		if text := common.Interface2String(part["text"]); text != "" {
			return dto.MediaContent{Type: dto.ContentTypeText, Text: text}
		}
		return dto.MediaContent{Type: dto.ContentTypeText, Text: moonshotRawAnyToString(part)}
	}
}

func moonshotNormalizeResponsesImageURL(part map[string]any) any {
	if imageURL, ok := part["image_url"]; ok {
		switch value := imageURL.(type) {
		case string:
			return &dto.MessageImageUrl{Url: value}
		case map[string]any:
			return &dto.MessageImageUrl{
				Url:    common.Interface2String(value["url"]),
				Detail: common.Interface2String(value["detail"]),
			}
		default:
			return imageURL
		}
	}
	return &dto.MessageImageUrl{
		Url:    common.Interface2String(part["url"]),
		Detail: common.Interface2String(part["detail"]),
	}
}

func moonshotNormalizeResponsesFile(part map[string]any) any {
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

func moonshotResponsesToolsToChatTools(raw json.RawMessage) []dto.ToolCallRequest {
	if !moonshotRawJSONPresent(raw) {
		return nil
	}
	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil
	}

	out := make([]dto.ToolCallRequest, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(common.Interface2String(tool["type"])) != moonshotChatToolTypeFunction {
			continue
		}
		name := moonshotResponsesToolName(tool)
		if !moonshotValidChatFunctionName(name) {
			continue
		}
		out = append(out, dto.ToolCallRequest{
			Type: moonshotChatToolTypeFunction,
			Function: dto.FunctionRequest{
				Name:        name,
				Description: moonshotResponsesToolDescription(tool),
				Parameters:  moonshotResponsesToolParameters(tool),
			},
		})
	}
	return out
}

func moonshotResponsesToolChoiceToChatToolChoice(raw json.RawMessage, tools []dto.ToolCallRequest) any {
	if !moonshotRawJSONPresent(raw) || len(tools) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		allowed[tool.Function.Name] = struct{}{}
	}

	if common.GetJsonType(raw) == "string" {
		choice := moonshotRawJSONToString(raw)
		switch choice {
		case "auto", "none", "required":
			return choice
		default:
			return nil
		}
	}

	var choice map[string]any
	if err := common.Unmarshal(raw, &choice); err != nil {
		return nil
	}
	if strings.TrimSpace(common.Interface2String(choice["type"])) != moonshotChatToolTypeFunction {
		return nil
	}
	name := moonshotResponsesToolName(choice)
	if _, ok := allowed[name]; !ok {
		return nil
	}
	return map[string]any{
		"type": moonshotChatToolTypeFunction,
		"function": map[string]any{
			"name": name,
		},
	}
}

func moonshotResponsesTextToChatResponseFormat(raw json.RawMessage) *dto.ResponseFormat {
	if !moonshotRawJSONPresent(raw) {
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

func moonshotResponsesRoleToChatRole(role string) string {
	switch strings.TrimSpace(role) {
	case "developer":
		return "system"
	case "system", "user", "assistant", "tool":
		return strings.TrimSpace(role)
	default:
		return "user"
	}
}

func moonshotResponsesToolName(tool map[string]any) string {
	for _, key := range []string{"name", "function_name"} {
		if name := strings.TrimSpace(common.Interface2String(tool[key])); name != "" {
			return name
		}
	}
	if function, ok := tool["function"].(map[string]any); ok {
		return strings.TrimSpace(common.Interface2String(function["name"]))
	}
	return ""
}

func moonshotResponsesToolDescription(tool map[string]any) string {
	if description := common.Interface2String(tool["description"]); description != "" {
		return description
	}
	if function, ok := tool["function"].(map[string]any); ok {
		return common.Interface2String(function["description"])
	}
	return ""
}

func moonshotResponsesToolParameters(tool map[string]any) any {
	for _, key := range []string{"parameters", "input_schema", "schema"} {
		if value, ok := tool[key]; ok {
			return value
		}
	}
	if function, ok := tool["function"].(map[string]any); ok {
		if value, ok := function["parameters"]; ok {
			return value
		}
	}
	return nil
}

func moonshotValidChatFunctionName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func moonshotRawJSONToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := common.Unmarshal(raw, &value); err == nil {
		return value
	}
	return string(raw)
}

func moonshotRawAnyToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.RawMessage:
		return moonshotRawJSONToString(typed)
	default:
		raw, err := common.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(raw)
	}
}

func moonshotRawJSONPresent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	return common.GetJsonType(raw) != "null"
}
