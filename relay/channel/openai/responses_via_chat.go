package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func ChatCompletionsToResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var chatResp dto.OpenAITextResponse
	if err := common.Unmarshal(body, &chatResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiErr := chatResp.GetOpenAIError(); oaiErr != nil && oaiErr.Type != "" {
		return nil, types.WithOpenAIError(*oaiErr, resp.StatusCode)
	}

	responseID := responseIDForResponses(c)
	responsesResp, usage, err := service.ChatCompletionsResponseToResponsesResponseWithOptions(
		&chatResp,
		responseID,
		service.ChatCompletionsToResponsesOptions{
			ThinkingToContent: info != nil && info.ChannelSetting.ThinkingToContent,
		},
	)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if usage == nil || usage.TotalTokens == 0 {
		text := extractChatResponseText(&chatResp)
		usage = service.ResponseText2Usage(c, text, info.UpstreamModelName, info.GetEstimatePromptTokens())
		responsesResp.Usage = usage
	}

	responseBody, err := common.Marshal(responsesResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, responseBody)
	return usage, nil
}

func ChatCompletionsToResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	responseID := responseIDForResponses(c)
	createdAt := int(time.Now().Unix())
	model := info.UpstreamModelName
	messageID := "msg_" + responseID
	contentIndex := 0
	messageOutputIndex := -1
	nextOutputIndex := 0

	usage := &dto.Usage{}
	var outputText strings.Builder
	var usageText strings.Builder
	var streamErr *types.NewAPIError
	var sentCreated bool
	var sentMessageAdded bool
	var sentContentAdded bool
	var sentThinkingStart bool
	var sentThinkingEnd bool
	toolCallOutputIndexByID := make(map[string]int)
	toolCallIDByChatIndex := make(map[int]string)
	toolCallNameByID := make(map[string]string)
	toolCallArgsByID := make(map[string]string)
	toolCallStarted := make(map[string]bool)

	sendEvent := func(eventType string, payload dto.ResponsesStreamResponse) bool {
		payload.Type = eventType
		data, err := common.Marshal(payload)
		if err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
			return false
		}
		helper.ResponseChunkData(c, payload, string(data))
		return true
	}

	sendCreatedIfNeeded := func() bool {
		if sentCreated {
			return true
		}
		respObj := &dto.OpenAIResponsesResponse{
			ID:        responseID,
			Object:    "response",
			CreatedAt: createdAt,
			Status:    json.RawMessage(`"in_progress"`),
			Model:     model,
		}
		if !sendEvent("response.created", dto.ResponsesStreamResponse{Response: respObj}) {
			return false
		}
		sentCreated = true
		return true
	}

	sendMessageStartIfNeeded := func() bool {
		if !sendCreatedIfNeeded() {
			return false
		}
		if messageOutputIndex < 0 {
			messageOutputIndex = nextOutputIndex
			nextOutputIndex++
		}
		outputIndex := messageOutputIndex
		if !sentMessageAdded {
			item := &dto.ResponsesOutput{
				Type:   "message",
				ID:     messageID,
				Status: "in_progress",
				Role:   "assistant",
			}
			if !sendEvent("response.output_item.added", dto.ResponsesStreamResponse{OutputIndex: &outputIndex, Item: item}) {
				return false
			}
			sentMessageAdded = true
		}
		if !sentContentAdded {
			if !sendEvent("response.content_part.added", dto.ResponsesStreamResponse{OutputIndex: &outputIndex, ContentIndex: &contentIndex}) {
				return false
			}
			sentContentAdded = true
		}
		return true
	}

	sendTextDeltaWithUsage := func(delta string, includeUsage bool) bool {
		if delta == "" {
			return true
		}
		if !sendMessageStartIfNeeded() {
			return false
		}
		outputText.WriteString(delta)
		if includeUsage {
			usageText.WriteString(delta)
		}
		outputIndex := messageOutputIndex
		return sendEvent("response.output_text.delta", dto.ResponsesStreamResponse{
			OutputIndex:  &outputIndex,
			ContentIndex: &contentIndex,
			Delta:        delta,
		})
	}

	sendTextDelta := func(delta string) bool {
		return sendTextDeltaWithUsage(delta, true)
	}

	sendReasoningDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		usageText.WriteString(delta)
		if info == nil || !info.ChannelSetting.ThinkingToContent {
			return true
		}
		if !sentThinkingStart {
			if !sendTextDeltaWithUsage("<think>\n", false) {
				return false
			}
			sentThinkingStart = true
		}
		return sendTextDeltaWithUsage(delta, false)
	}

	closeThinkingIfNeeded := func() bool {
		if !sentThinkingStart || sentThinkingEnd {
			return true
		}
		if !sendTextDeltaWithUsage("\n</think>\n", false) {
			return false
		}
		sentThinkingEnd = true
		return true
	}

	sendToolCallDelta := func(call dto.ToolCallResponse) bool {
		if !sendCreatedIfNeeded() {
			return false
		}
		chatIdx := len(toolCallIDByChatIndex)
		if call.Index != nil {
			chatIdx = *call.Index
		}
		callID := strings.TrimSpace(call.ID)
		if callID == "" {
			callID = toolCallIDByChatIndex[chatIdx]
		}
		if callID == "" {
			callID = fmt.Sprintf("call_%d", chatIdx)
		}
		toolCallIDByChatIndex[chatIdx] = callID

		outputIdx, ok := toolCallOutputIndexByID[callID]
		if !ok {
			outputIdx = nextOutputIndex
			toolCallOutputIndexByID[callID] = outputIdx
			nextOutputIndex++
		}

		if name := strings.TrimSpace(call.Function.Name); name != "" {
			toolCallNameByID[callID] = name
		}
		name := toolCallNameByID[callID]
		itemID := "fc_" + callID
		if !toolCallStarted[callID] {
			item := &dto.ResponsesOutput{
				Type:   "function_call",
				ID:     itemID,
				Status: "in_progress",
				CallId: callID,
				Name:   name,
			}
			if !sendEvent("response.output_item.added", dto.ResponsesStreamResponse{OutputIndex: &outputIdx, Item: item}) {
				return false
			}
			toolCallStarted[callID] = true
		}

		if call.Function.Arguments != "" {
			toolCallArgsByID[callID] += call.Function.Arguments
			usageText.WriteString(call.Function.Arguments)
			return sendEvent("response.function_call_arguments.delta", dto.ResponsesStreamResponse{
				OutputIndex: &outputIdx,
				ItemID:      itemID,
				Delta:       call.Function.Arguments,
			})
		}
		return true
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		if streamErr != nil {
			sr.Stop(streamErr)
			return
		}

		var chunk dto.ChatCompletionsStreamResponse
		if err := common.UnmarshalJsonStr(data, &chunk); err != nil {
			logger.LogError(c, "failed to unmarshal chat completions stream event: "+err.Error())
			sr.Error(err)
			return
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		if chunk.Created != 0 {
			createdAt = int(chunk.Created)
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
			if usage.TotalTokens == 0 {
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			}
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.GetContentString() != "" {
				if !closeThinkingIfNeeded() {
					sr.Stop(streamErr)
					return
				}
			}
			if !sendTextDelta(choice.Delta.GetContentString()) {
				sr.Stop(streamErr)
				return
			}
			if reasoning := choice.Delta.GetReasoningContent(); reasoning != "" {
				if !sendReasoningDelta(reasoning) {
					sr.Stop(streamErr)
					return
				}
			}
			for _, call := range choice.Delta.ToolCalls {
				if !sendToolCallDelta(call) {
					sr.Stop(streamErr)
					return
				}
			}
		}
	})

	if streamErr != nil {
		return nil, streamErr
	}

	if !closeThinkingIfNeeded() {
		return nil, streamErr
	}

	if usage == nil || usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, usageText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}
	usage.InputTokens = usage.PromptTokens
	usage.OutputTokens = usage.CompletionTokens
	if usage.InputTokensDetails == nil {
		details := usage.PromptTokensDetails
		usage.InputTokensDetails = &details
	}

	if !sendCreatedIfNeeded() {
		return nil, streamErr
	}
	if outputText.Len() > 0 {
		if !sendMessageStartIfNeeded() {
			return nil, streamErr
		}
		text := outputText.String()
		itemDone := &dto.ResponsesOutput{
			Type:   "message",
			ID:     messageID,
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{
					Type: "output_text",
					Text: text,
				},
			},
		}
		outputIndex := messageOutputIndex
		if !sendEvent("response.content_part.done", dto.ResponsesStreamResponse{OutputIndex: &outputIndex, ContentIndex: &contentIndex}) {
			return nil, streamErr
		}
		if !sendEvent(dto.ResponsesOutputTypeItemDone, dto.ResponsesStreamResponse{OutputIndex: &outputIndex, Item: itemDone}) {
			return nil, streamErr
		}
	}

	completedOutputByIndex := make(map[int]dto.ResponsesOutput, 1+len(toolCallOutputIndexByID))
	if outputText.Len() > 0 {
		completedOutputByIndex[messageOutputIndex] = dto.ResponsesOutput{
			Type:   "message",
			ID:     messageID,
			Status: "completed",
			Role:   "assistant",
			Content: []dto.ResponsesOutputContent{
				{
					Type: "output_text",
					Text: outputText.String(),
				},
			},
		}
	}
	for _, callID := range sortedToolCallIDsByIndex(toolCallOutputIndexByID) {
		idx := toolCallOutputIndexByID[callID]
		args := jsonRawMessageOrEmpty(toolCallArgsByID[callID])
		item := &dto.ResponsesOutput{
			Type:      "function_call",
			ID:        "fc_" + callID,
			Status:    "completed",
			CallId:    callID,
			Name:      toolCallNameByID[callID],
			Arguments: args,
		}
		outputIdx := idx
		if !sendEvent("response.function_call_arguments.done", dto.ResponsesStreamResponse{
			OutputIndex: &outputIdx,
			ItemID:      item.ID,
			Delta:       toolCallArgsByID[callID],
		}) {
			return nil, streamErr
		}
		if !sendEvent(dto.ResponsesOutputTypeItemDone, dto.ResponsesStreamResponse{OutputIndex: &outputIdx, Item: item}) {
			return nil, streamErr
		}
		completedOutputByIndex[idx] = *item
	}
	completedOutput := responsesOutputInIndexOrder(completedOutputByIndex)

	completed := &dto.OpenAIResponsesResponse{
		ID:        responseID,
		Object:    "response",
		CreatedAt: createdAt,
		Status:    json.RawMessage(`"completed"`),
		Model:     model,
		Output:    completedOutput,
		Usage:     usage,
	}
	if !sendEvent("response.completed", dto.ResponsesStreamResponse{Response: completed}) {
		return nil, streamErr
	}
	helper.Done(c)
	return usage, nil
}

func responseIDForResponses(c *gin.Context) string {
	id := helper.GetResponseID(c)
	return "resp_" + strings.TrimPrefix(id, "chatcmpl-")
}

func extractChatResponseText(resp *dto.OpenAITextResponse) string {
	if resp == nil {
		return ""
	}
	var sb strings.Builder
	for _, choice := range resp.Choices {
		sb.WriteString(choice.Message.StringContent())
		sb.WriteString(choice.Message.GetReasoningContent())
	}
	return sb.String()
}

func jsonRawMessageOrEmpty(s string) json.RawMessage {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	raw := json.RawMessage(trimmed)
	if json.Valid(raw) {
		return raw
	}
	quoted, err := common.Marshal(s)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return quoted
}

func sortedToolCallIDsByIndex(indexByID map[string]int) []string {
	type indexedCall struct {
		id    string
		index int
	}
	calls := make([]indexedCall, 0, len(indexByID))
	for id, idx := range indexByID {
		calls = append(calls, indexedCall{id: id, index: idx})
	}
	sort.SliceStable(calls, func(i, j int) bool {
		if calls[i].index == calls[j].index {
			return calls[i].id < calls[j].id
		}
		return calls[i].index < calls[j].index
	})
	ids := make([]string, 0, len(calls))
	for _, call := range calls {
		ids = append(ids, call.id)
	}
	return ids
}

func responsesOutputInIndexOrder(outputByIndex map[int]dto.ResponsesOutput) []dto.ResponsesOutput {
	indexes := make([]int, 0, len(outputByIndex))
	for idx := range outputByIndex {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	out := make([]dto.ResponsesOutput, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, outputByIndex[idx])
	}
	return out
}
