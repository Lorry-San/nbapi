package moonshot

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/dto"
)

const (
	moonshotChatToolTypeFunction  = "function"
	moonshotToolSearchName        = "tool_search"
	moonshotCustomInputField      = "input"
	moonshotChatToolNameMaxLength = 64
)

type moonshotToolContext struct {
	chatTools          []dto.ToolCallRequest
	mappings           map[string]dto.ResponsesToolMapping
	responseNameToChat map[string]string
}

func convertMoonshotResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, map[string]dto.ResponsesToolMapping, error) {
	if req == nil {
		return nil, nil, errors.New("request is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, nil, errors.New("model is required")
	}
	if err := validateMoonshotResponsesChatUnsupportedFields(req); err != nil {
		return nil, nil, err
	}

	toolContext := buildMoonshotToolContext(req.Tools, req.Input)
	messages, err := moonshotResponsesInputToChatMessages(req.Input, toolContext)
	if err != nil {
		return nil, nil, err
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

	tools := toolContext.chatTools
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
		ToolChoice:          toolContext.toolChoiceToChat(req.ToolChoice),
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

	return out, toolContext.cloneMappings(), nil
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

func moonshotResponsesInputToChatMessages(raw json.RawMessage, toolContext *moonshotToolContext) ([]dto.Message, error) {
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
		return moonshotResponsesInputItemsToChatMessages(items, toolContext), nil
	case "object":
		var item map[string]any
		if err := common.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("invalid responses input: %w", err)
		}
		return moonshotResponsesInputItemsToChatMessages([]map[string]any{item}, toolContext), nil
	default:
		return []dto.Message{{
			Role:    "user",
			Content: string(raw),
		}}, nil
	}
}

func moonshotResponsesInputItemsToChatMessages(items []map[string]any, toolContext *moonshotToolContext) []dto.Message {
	messages := make([]dto.Message, 0, len(items))
	knownFunctionCallIDs := make(map[string]struct{})

	for i := 0; i < len(items); i++ {
		item := items[i]
		itemType := strings.TrimSpace(common.Interface2String(item["type"]))
		switch itemType {
		case "reasoning":
			// reasoning 默认隐藏：历史推理内容不应转换为聊天消息发给上游。
			continue
		case "function_call", "custom_tool_call", "tool_search_call":
			toolCalls := make([]dto.ToolCallRequest, 0, 1)
			for ; i < len(items); i++ {
				nextItem := items[i]
				nextType := strings.TrimSpace(common.Interface2String(nextItem["type"]))
				if nextType != "function_call" && nextType != "custom_tool_call" && nextType != "tool_search_call" {
					break
				}
				toolCall, ok := moonshotResponsesCallToChatToolCall(nextItem, toolContext)
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
		case "custom_tool_call_output", "tool_search_output":
			callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
			if _, ok := knownFunctionCallIDs[callID]; callID != "" && ok {
				// These Responses items carry useful data outside `output` (notably
				// tool_search_output.tools), so preserve the complete item in history.
				messages = append(messages, dto.Message{
					Role:       "tool",
					ToolCallId: callID,
					Content:    moonshotRawAnyToString(item),
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

func moonshotResponsesCallToChatToolCall(item map[string]any, toolContext *moonshotToolContext) (dto.ToolCallRequest, bool) {
	itemType := strings.TrimSpace(common.Interface2String(item["type"]))
	callID := strings.TrimSpace(common.Interface2String(item["call_id"]))
	if callID == "" {
		callID = strings.TrimSpace(common.Interface2String(item["id"]))
	}
	if callID == "" {
		return dto.ToolCallRequest{}, false
	}

	name := strings.TrimSpace(common.Interface2String(item["name"]))
	namespace := strings.TrimSpace(common.Interface2String(item["namespace"]))
	arguments := ""
	mapping := dto.ResponsesToolMapping{Kind: dto.ResponsesToolKindFunction, Name: name, Namespace: namespace}
	if namespace != "" {
		mapping.Kind = dto.ResponsesToolKindNamespace
	}

	switch itemType {
	case "custom_tool_call":
		mapping.Kind = dto.ResponsesToolKindCustom
		input := moonshotRawAnyToString(item["input"])
		rawArguments, _ := common.Marshal(map[string]any{moonshotCustomInputField: input})
		arguments = string(rawArguments)
	case "tool_search_call":
		mapping.Kind = dto.ResponsesToolKindToolSearch
		mapping.Name = moonshotToolSearchName
		arguments = moonshotRawAnyToString(item["arguments"])
		if strings.TrimSpace(arguments) == "" {
			arguments = "{}"
		}
	default:
		arguments = moonshotRawAnyToString(item["arguments"])
	}

	if mapping.Name == "" {
		return dto.ToolCallRequest{}, false
	}
	chatName := moonshotMappedChatToolName(mapping.Namespace, mapping.Name)
	if toolContext != nil {
		chatName = toolContext.ensureMapping(mapping)
	}
	return dto.ToolCallRequest{
		ID:   callID,
		Type: moonshotChatToolTypeFunction,
		Function: dto.FunctionRequest{
			Name:      chatName,
			Arguments: arguments,
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

func buildMoonshotToolContext(toolsRaw json.RawMessage, inputRaw json.RawMessage) *moonshotToolContext {
	ctx := &moonshotToolContext{
		mappings:           make(map[string]dto.ResponsesToolMapping),
		responseNameToChat: make(map[string]string),
	}
	if moonshotRawJSONPresent(toolsRaw) {
		var tools []any
		if err := common.Unmarshal(toolsRaw, &tools); err == nil {
			for _, tool := range tools {
				ctx.addResponseTool(tool)
			}
		}
	}
	if moonshotRawJSONPresent(inputRaw) {
		var input any
		if err := common.Unmarshal(inputRaw, &input); err == nil {
			ctx.collectToolSearchOutputTools(input)
		}
	}
	return ctx
}

func (c *moonshotToolContext) addResponseTool(raw any) {
	if name, ok := raw.(string); ok {
		c.addCustomTool(map[string]any{"type": dto.ResponsesToolKindCustom, "name": name})
		return
	}
	tool, ok := raw.(map[string]any)
	if !ok {
		return
	}
	switch strings.TrimSpace(common.Interface2String(tool["type"])) {
	case moonshotChatToolTypeFunction:
		c.addFunctionTool(tool, "")
	case dto.ResponsesToolKindNamespace:
		c.addNamespaceTool(tool)
	case dto.ResponsesToolKindCustom:
		c.addCustomTool(tool)
	case dto.ResponsesToolKindToolSearch:
		c.addToolSearchTool()
	}
}

func (c *moonshotToolContext) addFunctionTool(tool map[string]any, namespace string) {
	name := moonshotResponsesToolName(tool)
	if name == "" {
		return
	}
	kind := dto.ResponsesToolKindFunction
	if namespace != "" {
		kind = dto.ResponsesToolKindNamespace
	}
	mapping := dto.ResponsesToolMapping{Kind: kind, Name: name, Namespace: namespace}
	chatName := moonshotMappedChatToolName(namespace, name)
	parameters := moonshotNormalizeFunctionParameters(moonshotResponsesToolParameters(tool))
	c.addChatTool(chatName, mapping, dto.ToolCallRequest{
		Type: moonshotChatToolTypeFunction,
		Function: dto.FunctionRequest{
			Name:        chatName,
			Description: moonshotResponsesToolDescription(tool),
			Parameters:  parameters,
			Strict:      moonshotResponsesToolStrict(tool),
		},
	})
}

func (c *moonshotToolContext) addNamespaceTool(tool map[string]any) {
	namespace := moonshotResponsesToolName(tool)
	if namespace == "" {
		return
	}
	nested := moonshotResponsesNestedTools(tool)
	if len(nested) == 0 {
		return
	}

	before := len(c.chatTools)
	for _, child := range nested {
		childType := strings.TrimSpace(common.Interface2String(child["type"]))
		if childType == "" || childType == moonshotChatToolTypeFunction {
			c.addFunctionTool(child, namespace)
		}
	}
	if len(c.chatTools) == before+1 {
		c.responseNameToChat[moonshotResponseToolKey(dto.ResponsesToolKindNamespace, "", namespace)] = c.chatTools[before].Function.Name
	}
}

func (c *moonshotToolContext) addCustomTool(tool map[string]any) {
	name := moonshotResponsesToolName(tool)
	if name == "" {
		return
	}
	mapping := dto.ResponsesToolMapping{Kind: dto.ResponsesToolKindCustom, Name: name}
	chatName := moonshotMappedChatToolName("", name)
	description := "Original tool definition:\n"
	if raw, err := common.Marshal(tool); err == nil {
		description += string(raw)
	}
	c.addChatTool(chatName, mapping, dto.ToolCallRequest{
		Type: moonshotChatToolTypeFunction,
		Function: dto.FunctionRequest{
			Name:        chatName,
			Description: description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					moonshotCustomInputField: map[string]any{"type": "string"},
				},
				"required": []string{moonshotCustomInputField},
			},
		},
	})
}

func (c *moonshotToolContext) addToolSearchTool() {
	mapping := dto.ResponsesToolMapping{Kind: dto.ResponsesToolKindToolSearch, Name: moonshotToolSearchName}
	c.addChatTool(moonshotToolSearchName, mapping, dto.ToolCallRequest{
		Type: moonshotChatToolTypeFunction,
		Function: dto.FunctionRequest{
			Name:        moonshotToolSearchName,
			Description: "Search and load tools, plugins, connectors, and namespaces for the current task.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "integer"},
				},
				"required": []string{"query"},
			},
		},
	})
}

func (c *moonshotToolContext) addChatTool(chatName string, mapping dto.ResponsesToolMapping, tool dto.ToolCallRequest) string {
	if chatName == "" {
		return ""
	}
	key := moonshotResponseToolKey(mapping.Kind, mapping.Namespace, mapping.Name)
	if existing := c.responseNameToChat[key]; existing != "" {
		return existing
	}
	chatName = c.uniqueChatToolName(chatName, mapping)
	tool.Function.Name = chatName
	c.chatTools = append(c.chatTools, tool)
	c.mappings[chatName] = mapping
	c.responseNameToChat[key] = chatName
	return chatName
}

func (c *moonshotToolContext) ensureMapping(mapping dto.ResponsesToolMapping) string {
	key := moonshotResponseToolKey(mapping.Kind, mapping.Namespace, mapping.Name)
	if chatName := c.responseNameToChat[key]; chatName != "" {
		return chatName
	}
	chatName := c.uniqueChatToolName(moonshotMappedChatToolName(mapping.Namespace, mapping.Name), mapping)
	c.mappings[chatName] = mapping
	c.responseNameToChat[key] = chatName
	return chatName
}

func (c *moonshotToolContext) uniqueChatToolName(base string, mapping dto.ResponsesToolMapping) string {
	for attempt := 0; ; attempt++ {
		candidate := base
		if attempt > 0 {
			candidate = moonshotCollisionChatToolName(base, mapping, attempt)
		}
		existing, ok := c.mappings[candidate]
		if !ok || existing == mapping {
			return candidate
		}
	}
}

func (c *moonshotToolContext) cloneMappings() map[string]dto.ResponsesToolMapping {
	if c == nil || len(c.mappings) == 0 {
		return nil
	}
	out := make(map[string]dto.ResponsesToolMapping, len(c.mappings))
	for name, mapping := range c.mappings {
		out[name] = mapping
	}
	return out
}

func (c *moonshotToolContext) toolChoiceToChat(raw json.RawMessage) any {
	if c == nil || !moonshotRawJSONPresent(raw) || len(c.chatTools) == 0 {
		return nil
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
	choiceType := strings.TrimSpace(common.Interface2String(choice["type"]))
	name := moonshotResponsesToolName(choice)
	namespace := strings.TrimSpace(common.Interface2String(choice["namespace"]))
	kind := dto.ResponsesToolKindFunction
	if namespace != "" || choiceType == dto.ResponsesToolKindNamespace {
		kind = dto.ResponsesToolKindNamespace
	}
	if choiceType == dto.ResponsesToolKindCustom {
		kind = dto.ResponsesToolKindCustom
	}
	if choiceType == dto.ResponsesToolKindToolSearch {
		kind = dto.ResponsesToolKindToolSearch
		name = moonshotToolSearchName
	}
	chatName := c.responseNameToChat[moonshotResponseToolKey(kind, namespace, name)]
	if chatName == "" && choiceType == dto.ResponsesToolKindNamespace {
		chatName = c.responseNameToChat[moonshotResponseToolKey(dto.ResponsesToolKindNamespace, "", name)]
	}
	if chatName == "" {
		return nil
	}
	return map[string]any{
		"type":     moonshotChatToolTypeFunction,
		"function": map[string]any{"name": chatName},
	}
}

func (c *moonshotToolContext) collectToolSearchOutputTools(value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			c.collectToolSearchOutputTools(item)
		}
	case map[string]any:
		if strings.TrimSpace(common.Interface2String(typed["type"])) == "tool_search_output" {
			if tools, ok := typed["tools"].([]any); ok {
				for _, tool := range tools {
					c.addResponseTool(tool)
				}
			}
		}
		for _, child := range typed {
			c.collectToolSearchOutputTools(child)
		}
	}
}

func moonshotResponsesNestedTools(tool map[string]any) []map[string]any {
	for _, key := range []string{"tools", "children", "functions"} {
		if raw, ok := tool[key].([]any); ok {
			out := make([]map[string]any, 0, len(raw))
			for _, item := range raw {
				if child, ok := item.(map[string]any); ok {
					out = append(out, child)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
		if raw, ok := tool[key].([]map[string]any); ok && len(raw) > 0 {
			return raw
		}
	}
	return nil
}

func moonshotNormalizeFunctionParameters(value any) any {
	parameters, ok := value.(map[string]any)
	if !ok {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	copy := make(map[string]any, len(parameters)+1)
	for key, item := range parameters {
		copy[key] = item
	}
	if strings.TrimSpace(common.Interface2String(copy["type"])) != "object" {
		copy["type"] = "object"
	}
	return copy
}

func moonshotResponsesToolStrict(tool map[string]any) any {
	if strict, ok := tool["strict"]; ok {
		return strict
	}
	if function, ok := tool["function"].(map[string]any); ok {
		return function["strict"]
	}
	return nil
}

func moonshotMappedChatToolName(namespace string, name string) string {
	fullName := strings.TrimSpace(name)
	if namespace != "" {
		fullName = strings.TrimSpace(namespace) + "__" + fullName
	}
	if moonshotValidChatFunctionName(fullName) {
		return fullName
	}

	var safe strings.Builder
	for _, r := range fullName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			safe.WriteRune(r)
		} else {
			safe.WriteByte('_')
		}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(fullName)))[:16]
	suffix := "__" + digest
	prefix := safe.String()
	maxPrefix := moonshotChatToolNameMaxLength - len(suffix)
	if len(prefix) > maxPrefix {
		prefix = prefix[:maxPrefix]
	}
	prefix = strings.Trim(prefix, "_-")
	if prefix == "" {
		prefix = "tool"
	}
	return prefix + suffix
}

func moonshotCollisionChatToolName(base string, mapping dto.ResponsesToolMapping, attempt int) string {
	key := fmt.Sprintf("%s\x00%d", moonshotResponseToolKey(mapping.Kind, mapping.Namespace, mapping.Name), attempt)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))[:16]
	suffix := "__" + digest
	prefix := strings.TrimRight(base, "_-")
	maxPrefix := moonshotChatToolNameMaxLength - len(suffix)
	if len(prefix) > maxPrefix {
		prefix = prefix[:maxPrefix]
	}
	if prefix == "" {
		prefix = "tool"
	}
	return prefix + suffix
}

func moonshotResponseToolKey(kind string, namespace string, name string) string {
	return strings.TrimSpace(kind) + "\x00" + strings.TrimSpace(namespace) + "\x00" + strings.TrimSpace(name)
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
