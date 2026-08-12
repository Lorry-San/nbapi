package oaichat

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/dto"
)

type ChatToResponsesStreamEvent struct {
	Type    string
	Payload dto.ResponsesStreamResponse
}

type ChatToResponsesStreamState struct {
	ID      string
	Model   string
	Created int64
	Usage   *dto.Usage

	status            string
	incompleteDetails *dto.IncompleteDetails
	errorDetails      map[string]any
	sawFinishReason   bool
	finishReason      string
	sawToolDelta      bool
	droppedToolCalls  int
	sentCreated       bool
	textOutputIndex   int
	textStarted       bool
	textDone          bool
	reasoningIndex    int
	reasoningStarted  bool
	reasoningDone     bool
	finalized         bool
	nextOutputIndex   int
	toolsByIndex      map[int]*chatToResponsesStreamTool
	outputOrder       []chatToResponsesOutputRef
	text              strings.Builder
	reasoning         strings.Builder
	usageText         strings.Builder
	thinkingToContent bool
	preserveReasoning bool
	visibleReasoning  bool
	toolMappings      map[string]dto.ResponsesToolMapping
}

type chatToResponsesStreamTool struct {
	ChatIndex   int
	OutputIndex int
	ID          string
	Name        string
	Arguments   strings.Builder
	Added       bool
	Done        bool
	Dropped     bool
}

type chatToResponsesOutputRef struct {
	Kind      string
	ToolIndex int
}

func NewChatToResponsesStreamState(id string, model string) *ChatToResponsesStreamState {
	return NewChatToResponsesStreamStateWithOptions(id, model, ChatCompletionsToResponsesOptions{
		PreserveReasoning: true,
	})
}

func NewChatToResponsesStreamStateWithOptions(id string, model string, options ChatCompletionsToResponsesOptions) *ChatToResponsesStreamState {
	return &ChatToResponsesStreamState{
		ID:                id,
		Model:             model,
		Created:           time.Now().Unix(),
		Usage:             &dto.Usage{},
		status:            "in_progress",
		textOutputIndex:   -1,
		reasoningIndex:    -1,
		toolsByIndex:      make(map[int]*chatToResponsesStreamTool),
		thinkingToContent: options.ThinkingToContent,
		preserveReasoning: options.PreserveReasoning,
		toolMappings:      cloneResponsesToolMappings(options.ToolMappings),
	}
}

func ChatCompletionsStreamChunkToResponsesEvents(chunk *dto.ChatCompletionsStreamResponse, state *ChatToResponsesStreamState) ([]ChatToResponsesStreamEvent, error) {
	if chunk == nil || state == nil {
		return nil, nil
	}
	if state.ID == "" {
		state.ID = chunk.Id
	}
	if state.Model == "" {
		state.Model = chunk.Model
	}
	if state.Created == 0 {
		state.Created = chunk.Created
	}
	if chunk.Usage != nil {
		state.Usage = UsageFromChatUsage(chunk.Usage)
	}

	events := make([]ChatToResponsesStreamEvent, 0)
	if !state.sentCreated {
		state.sentCreated = true
		events = append(events, responsesStreamEvent(responsesEventCreated, dto.ResponsesStreamResponse{
			Type:     responsesEventCreated,
			Response: state.createdResponse(),
		}))
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.GetReasoningContent() != "" {
			events = append(events, state.appendReasoningDelta(choice.Delta.GetReasoningContent())...)
		}
		if choice.Delta.GetContentString() != "" {
			events = append(events, state.closeVisibleReasoningIfNeeded()...)
			events = append(events, state.appendTextDelta(choice.Delta.GetContentString())...)
		}
		for _, toolCall := range choice.Delta.ToolCalls {
			toolEvents, err := state.appendToolCallDelta(toolCall)
			if err != nil {
				return nil, err
			}
			events = append(events, toolEvents...)
		}
		if choice.FinishReason != nil && strings.TrimSpace(*choice.FinishReason) != "" {
			state.applyFinishReason(*choice.FinishReason)
			events = append(events, state.doneDeltaEvents()...)
		}
	}
	return events, nil
}

func FinalizeChatCompletionsStreamToResponses(state *ChatToResponsesStreamState) []ChatToResponsesStreamEvent {
	if state == nil || state.finalized {
		return nil
	}
	if !state.sawFinishReason {
		if state.hasSubstantiveOutput() {
			state.status = "incomplete"
			state.incompleteDetails = &dto.IncompleteDetails{Reason: responsesIncompleteReasonStreamCut}
		} else {
			state.setFailed("Upstream Chat Completions stream ended before sending finish_reason", "stream_truncated")
		}
	}
	events := state.doneDeltaEvents()
	if state.status == "completed" && state.droppedToolCalls > 0 && !state.hasEmittedToolCall() {
		state.setFailed(
			fmt.Sprintf("Upstream returned %d tool call(s) without a function name, leaving no usable tool call in this turn", state.droppedToolCalls),
			"upstream_tool_call_dropped",
		)
	}
	state.finalized = true
	resp := state.finalResponse()
	eventType := responsesEventCompleted
	switch state.status {
	case "incomplete":
		eventType = responsesEventIncomplete
	case "failed":
		eventType = responsesEventFailed
	}
	events = append(events, responsesStreamEvent(eventType, dto.ResponsesStreamResponse{
		Type:     eventType,
		Response: resp,
	}))
	return events
}

func (s *ChatToResponsesStreamState) UsageText() string {
	if s == nil {
		return ""
	}
	if s.usageText.Len() > 0 {
		return s.usageText.String()
	}
	return s.text.String()
}

func (s *ChatToResponsesStreamState) appendTextDelta(delta string) []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0, 2)
	if !s.textStarted {
		s.textStarted = true
		s.textOutputIndex = s.nextIndex("message", -1)
		events = append(events, responsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemAdded,
			OutputIndex: intPtr(s.textOutputIndex),
			Item: &dto.ResponsesOutput{
				Type:    responsesOutputTypeMessage,
				ID:      s.messageID(),
				Status:  "in_progress",
				Role:    "assistant",
				Content: []dto.ResponsesOutputContent{},
			},
		}))
	}
	s.text.WriteString(delta)
	s.usageText.WriteString(delta)
	events = append(events, responsesStreamEvent(responsesEventOutputTextDelta, dto.ResponsesStreamResponse{
		Type:         responsesEventOutputTextDelta,
		OutputIndex:  intPtr(s.textOutputIndex),
		ContentIndex: intPtr(0),
		Delta:        delta,
		ItemID:       s.messageID(),
	}))
	return events
}

func (s *ChatToResponsesStreamState) appendReasoningDelta(delta string) []ChatToResponsesStreamEvent {
	if delta == "" {
		return nil
	}
	if s.thinkingToContent {
		events := make([]ChatToResponsesStreamEvent, 0, 2)
		if !s.visibleReasoning {
			s.visibleReasoning = true
			events = append(events, s.appendTextDelta("<think>\n")...)
		}
		events = append(events, s.appendTextDelta(delta)...)
		return events
	}
	s.usageText.WriteString(delta)
	if !s.preserveReasoning {
		return nil
	}
	events := make([]ChatToResponsesStreamEvent, 0, 2)
	if !s.reasoningStarted {
		s.reasoningStarted = true
		s.reasoningIndex = s.nextIndex("reasoning", -1)
		events = append(events, responsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemAdded,
			OutputIndex: intPtr(s.reasoningIndex),
			Item: &dto.ResponsesOutput{
				Type:    responsesOutputTypeReasoning,
				ID:      s.reasoningID(),
				Status:  "in_progress",
				Content: []dto.ResponsesOutputContent{},
			},
		}))
	}
	s.reasoning.WriteString(delta)
	events = append(events, responsesStreamEvent(responsesEventReasoningSummaryDelta, dto.ResponsesStreamResponse{
		Type:         responsesEventReasoningSummaryDelta,
		OutputIndex:  intPtr(s.reasoningIndex),
		SummaryIndex: intPtr(0),
		Delta:        delta,
		ItemID:       s.reasoningID(),
	}))
	return events
}

func (s *ChatToResponsesStreamState) closeVisibleReasoningIfNeeded() []ChatToResponsesStreamEvent {
	if !s.visibleReasoning {
		return nil
	}
	s.visibleReasoning = false
	return s.appendTextDelta("\n</think>\n")
}

func (s *ChatToResponsesStreamState) appendToolCallDelta(toolCall dto.ToolCallResponse) ([]ChatToResponsesStreamEvent, error) {
	s.sawToolDelta = true
	chatIndex := s.resolveToolIndex(toolCall)
	tool := s.toolsByIndex[chatIndex]
	if tool == nil {
		tool = &chatToResponsesStreamTool{
			ChatIndex:   chatIndex,
			OutputIndex: -1,
			ID:          strings.TrimSpace(toolCall.ID),
			Name:        strings.TrimSpace(toolCall.Function.Name),
		}
		s.toolsByIndex[chatIndex] = tool
	}
	wasAdded := tool.Added
	if id := strings.TrimSpace(toolCall.ID); id != "" && (!tool.Added || tool.ID == "") {
		tool.ID = id
	}
	if name := strings.TrimSpace(toolCall.Function.Name); name != "" && (!tool.Added || tool.Name == "") {
		tool.Name = name
	}
	events := make([]ChatToResponsesStreamEvent, 0, 3)
	if toolCall.Function.Arguments != "" {
		tool.Arguments.WriteString(toolCall.Function.Arguments)
		s.usageText.WriteString(toolCall.Function.Arguments)
		if wasAdded && s.toolMapping(tool).Kind != dto.ResponsesToolKindCustom {
			itemID := chatToolCallItemID(tool.ID, s.toolMapping(tool))
			events = append(events, responsesStreamEvent(responsesEventFunctionArgsDelta, dto.ResponsesStreamResponse{
				Type:        responsesEventFunctionArgsDelta,
				OutputIndex: intPtr(tool.OutputIndex),
				ItemID:      itemID,
				Delta:       toolCall.Function.Arguments,
			}))
		}
	}
	events = append(events, s.flushReadyTools()...)
	return events, nil
}

func (s *ChatToResponsesStreamState) doneDeltaEvents() []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0)
	events = append(events, s.closeVisibleReasoningIfNeeded()...)
	status := s.outputStatus()
	if s.textStarted && !s.textDone {
		s.textDone = true
		events = append(events, responsesStreamEvent("response.output_text.done", dto.ResponsesStreamResponse{
			Type:         "response.output_text.done",
			OutputIndex:  intPtr(s.textOutputIndex),
			ContentIndex: intPtr(0),
			ItemID:       s.messageID(),
		}))
		events = append(events, responsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemDone,
			OutputIndex: intPtr(s.textOutputIndex),
			Item:        s.messageOutput(status),
		}))
	}
	if s.reasoningStarted && !s.reasoningDone {
		s.reasoningDone = true
		events = append(events, responsesStreamEvent(responsesEventReasoningSummaryDone, dto.ResponsesStreamResponse{
			Type:         responsesEventReasoningSummaryDone,
			OutputIndex:  intPtr(s.reasoningIndex),
			SummaryIndex: intPtr(0),
			ItemID:       s.reasoningID(),
			Part: &dto.ResponsesReasoningSummaryPart{
				Type: "summary_text",
				Text: s.reasoning.String(),
			},
		}))
		events = append(events, responsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemDone,
			OutputIndex: intPtr(s.reasoningIndex),
			Item:        s.reasoningOutput(status),
		}))
	}
	events = append(events, s.finalizeTools(status)...)
	return events
}

func (s *ChatToResponsesStreamState) applyFinishReason(finishReason string) {
	s.sawFinishReason = true
	s.finishReason = strings.TrimSpace(finishReason)
	if status, details := ResponsesStatusFromChatFinishReason(finishReason); status != "" {
		s.status = status
		s.incompleteDetails = details
	}
}

func (s *ChatToResponsesStreamState) finalResponse() *dto.OpenAIResponsesResponse {
	output := make([]dto.ResponsesOutput, 0, len(s.outputOrder))
	status := s.outputStatus()
	for _, ref := range s.outputOrder {
		switch ref.Kind {
		case "message":
			output = append(output, *s.messageOutput(status))
		case "reasoning":
			output = append(output, *s.reasoningOutput(status))
		case "tool":
			if tool := s.toolsByIndex[ref.ToolIndex]; tool != nil && tool.Added && !tool.Dropped {
				output = append(output, *s.toolOutput(tool, status))
			}
		}
	}
	return &dto.OpenAIResponsesResponse{
		ID:                s.ID,
		Object:            "response",
		CreatedAt:         int(s.Created),
		Status:            []byte(fmt.Sprintf("%q", s.status)),
		Error:             s.errorDetails,
		IncompleteDetails: s.incompleteDetails,
		Model:             s.Model,
		Output:            output,
		Usage:             s.Usage,
	}
}

func (s *ChatToResponsesStreamState) createdResponse() *dto.OpenAIResponsesResponse {
	return &dto.OpenAIResponsesResponse{
		ID:        s.ID,
		Object:    "response",
		CreatedAt: int(s.Created),
		Status:    []byte(`"in_progress"`),
		Model:     s.Model,
		Output:    []dto.ResponsesOutput{},
	}
}

func (s *ChatToResponsesStreamState) nextIndex(kind string, toolIndex int) int {
	index := s.nextOutputIndex
	s.nextOutputIndex++
	s.outputOrder = append(s.outputOrder, chatToResponsesOutputRef{Kind: kind, ToolIndex: toolIndex})
	return index
}

func (s *ChatToResponsesStreamState) sortedTools() []*chatToResponsesStreamTool {
	indexes := make([]int, 0, len(s.toolsByIndex))
	for index := range s.toolsByIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	tools := make([]*chatToResponsesStreamTool, 0, len(indexes))
	for _, index := range indexes {
		tools = append(tools, s.toolsByIndex[index])
	}
	return tools
}

func (s *ChatToResponsesStreamState) outputStatus() string {
	if s.status == "incomplete" {
		return "incomplete"
	}
	return "completed"
}

func (s *ChatToResponsesStreamState) messageID() string {
	return fmt.Sprintf("%s_msg_0", s.ID)
}

func (s *ChatToResponsesStreamState) reasoningID() string {
	return fmt.Sprintf("%s_reasoning_0", s.ID)
}

func (s *ChatToResponsesStreamState) messageOutput(status string) *dto.ResponsesOutput {
	return &dto.ResponsesOutput{
		Type:   responsesOutputTypeMessage,
		ID:     s.messageID(),
		Status: status,
		Role:   "assistant",
		Content: []dto.ResponsesOutputContent{
			{
				Type:        "output_text",
				Text:        s.text.String(),
				Annotations: []interface{}{},
			},
		},
	}
}

func (s *ChatToResponsesStreamState) reasoningOutput(status string) *dto.ResponsesOutput {
	return &dto.ResponsesOutput{
		Type:   responsesOutputTypeReasoning,
		ID:     s.reasoningID(),
		Status: status,
		Content: []dto.ResponsesOutputContent{
			{
				Type: "summary_text",
				Text: s.reasoning.String(),
			},
		},
	}
}

func (s *ChatToResponsesStreamState) toolOutput(tool *chatToResponsesStreamTool, status string) *dto.ResponsesOutput {
	output := chatToolCallOutputFromMapping(tool.ID, tool.Name, tool.Arguments.String(), status, s.toolMapping(tool))
	return &output
}

func (s *ChatToResponsesStreamState) resolveToolIndex(toolCall dto.ToolCallResponse) int {
	if toolCall.Index != nil {
		return *toolCall.Index
	}
	id := strings.TrimSpace(toolCall.ID)
	if id != "" {
		for index, tool := range s.toolsByIndex {
			if tool != nil && tool.ID == id {
				return index
			}
		}
		if lastIndex, tool, ok := s.lastTool(); ok && tool != nil && tool.ID == "" && !tool.Added && !tool.Done {
			return lastIndex
		}
		return s.nextFreeToolIndex()
	}
	if lastIndex, _, ok := s.lastTool(); ok {
		return lastIndex
	}
	return 0
}

func (s *ChatToResponsesStreamState) nextFreeToolIndex() int {
	if len(s.toolsByIndex) == 0 {
		return 0
	}
	maxIndex := 0
	for index := range s.toolsByIndex {
		if index > maxIndex {
			maxIndex = index
		}
	}
	return maxIndex + 1
}

func (s *ChatToResponsesStreamState) lastTool() (int, *chatToResponsesStreamTool, bool) {
	if len(s.toolsByIndex) == 0 {
		return 0, nil, false
	}
	indexes := make([]int, 0, len(s.toolsByIndex))
	for index := range s.toolsByIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	index := indexes[len(indexes)-1]
	return index, s.toolsByIndex[index], true
}

func (s *ChatToResponsesStreamState) flushReadyTools() []ChatToResponsesStreamEvent {
	events := make([]ChatToResponsesStreamEvent, 0)
	for _, tool := range s.sortedTools() {
		if tool == nil || tool.Done || tool.Dropped || tool.Added {
			continue
		}
		if strings.TrimSpace(tool.ID) == "" || strings.TrimSpace(tool.Name) == "" {
			break
		}
		tool.OutputIndex = s.nextIndex("tool", tool.ChatIndex)
		tool.Added = true
		itemValue := chatToolCallOutputFromMapping(tool.ID, tool.Name, "", "in_progress", s.toolMapping(tool))
		item := &itemValue
		itemID := item.ID
		if itemID == "" {
			itemID = chatToolCallItemID(tool.ID, s.toolMapping(tool))
		}
		events = append(events, responsesStreamEvent(responsesEventOutputItemAdded, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemAdded,
			OutputIndex: intPtr(tool.OutputIndex),
			ItemID:      itemID,
			Item:        item,
		}))
		if tool.Arguments.Len() > 0 && s.toolMapping(tool).Kind != dto.ResponsesToolKindCustom {
			events = append(events, responsesStreamEvent(responsesEventFunctionArgsDelta, dto.ResponsesStreamResponse{
				Type:        responsesEventFunctionArgsDelta,
				OutputIndex: intPtr(tool.OutputIndex),
				ItemID:      itemID,
				Delta:       tool.Arguments.String(),
			}))
		}
	}
	return events
}

func (s *ChatToResponsesStreamState) finalizeTools(status string) []ChatToResponsesStreamEvent {
	for _, tool := range s.sortedTools() {
		if tool == nil || tool.Done || tool.Dropped {
			continue
		}
		if strings.TrimSpace(tool.Name) == "" {
			tool.Dropped = true
			tool.Done = true
			s.droppedToolCalls++
			continue
		}
		if strings.TrimSpace(tool.ID) == "" {
			tool.ID = fmt.Sprintf("%s_call_%d", s.ID, tool.ChatIndex)
		}
	}

	events := s.flushReadyTools()
	for _, tool := range s.sortedTools() {
		if tool == nil || tool.Done || tool.Dropped || !tool.Added {
			continue
		}
		output := s.toolOutput(tool, status)
		itemID := output.ID
		if itemID == "" {
			itemID = chatToolCallItemID(tool.ID, s.toolMapping(tool))
		}
		if s.toolMapping(tool).Kind == dto.ResponsesToolKindCustom {
			if output.Input != "" {
				events = append(events, responsesStreamEvent(responsesEventCustomToolInputDelta, dto.ResponsesStreamResponse{
					Type:        responsesEventCustomToolInputDelta,
					OutputIndex: intPtr(tool.OutputIndex),
					ItemID:      itemID,
					Delta:       output.Input,
				}))
			}
			events = append(events, responsesStreamEvent(responsesEventCustomToolInputDone, dto.ResponsesStreamResponse{
				Type:        responsesEventCustomToolInputDone,
				OutputIndex: intPtr(tool.OutputIndex),
				ItemID:      itemID,
				Input:       output.Input,
			}))
		} else {
			events = append(events, responsesStreamEvent(responsesEventFunctionArgsDone, dto.ResponsesStreamResponse{
				Type:        responsesEventFunctionArgsDone,
				OutputIndex: intPtr(tool.OutputIndex),
				ItemID:      itemID,
				Arguments:   tool.Arguments.String(),
			}))
		}
		events = append(events, responsesStreamEvent(responsesEventOutputItemDone, dto.ResponsesStreamResponse{
			Type:        responsesEventOutputItemDone,
			OutputIndex: intPtr(tool.OutputIndex),
			Item:        output,
		}))
		tool.Done = true
	}
	return events
}

func (s *ChatToResponsesStreamState) toolMapping(tool *chatToResponsesStreamTool) dto.ResponsesToolMapping {
	if s == nil || tool == nil || s.toolMappings == nil {
		return dto.ResponsesToolMapping{}
	}
	return s.toolMappings[tool.Name]
}

func (s *ChatToResponsesStreamState) hasEmittedToolCall() bool {
	for _, tool := range s.toolsByIndex {
		if tool != nil && tool.Added && !tool.Dropped {
			return true
		}
	}
	return false
}

func (s *ChatToResponsesStreamState) hasSubstantiveOutput() bool {
	return strings.TrimSpace(s.text.String()) != "" ||
		strings.TrimSpace(s.reasoning.String()) != "" ||
		s.sawToolDelta
}

func (s *ChatToResponsesStreamState) setFailed(message string, code string) {
	s.status = "failed"
	s.incompleteDetails = nil
	s.errorDetails = map[string]any{
		"message": message,
		"type":    code,
		"code":    code,
	}
}

func cloneResponsesToolMappings(mappings map[string]dto.ResponsesToolMapping) map[string]dto.ResponsesToolMapping {
	if len(mappings) == 0 {
		return nil
	}
	out := make(map[string]dto.ResponsesToolMapping, len(mappings))
	for name, mapping := range mappings {
		out[name] = mapping
	}
	return out
}
