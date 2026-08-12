package oaichat

import (
	"testing"

	"github.com/Lorry-San/nbapi/dto"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsResponseToResponsesPreservesTextToolCallsAndUsage(t *testing.T) {
	chat := &dto.OpenAITextResponse{
		Id:      "chatcmpl_1",
		Model:   "gpt-test",
		Created: 456,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Message:      assistantMessageWithTool("I will call.", "call_1", "lookup", `{"q":"x"}`),
				FinishReason: "tool_calls",
			},
		},
		Usage: dto.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
	}

	resp, usage, err := ChatCompletionsResponseToResponsesResponse(chat, "resp_1")
	require.NoError(t, err)
	require.NotNil(t, usage)

	assert.Equal(t, "resp_1", resp.ID)
	assert.Equal(t, "response", resp.Object)
	assert.Equal(t, `"completed"`, string(resp.Status))
	assert.Equal(t, 3, resp.Usage.InputTokens)
	assert.Equal(t, 5, resp.Usage.OutputTokens)
	require.Len(t, resp.Output, 2)
	assert.Equal(t, responsesOutputTypeMessage, resp.Output[0].Type)
	assert.Equal(t, "I will call.", resp.Output[0].Content[0].Text)
	assert.Equal(t, responsesOutputTypeFunctionCall, resp.Output[1].Type)
	assert.Equal(t, "call_1", resp.Output[1].CallId)
	assert.Equal(t, "lookup", resp.Output[1].Name)
	assert.Equal(t, `"{\"q\":\"x\"}"`, string(resp.Output[1].Arguments))
}

func TestChatCompletionsResponseToResponsesOptionsControlReasoningVisibility(t *testing.T) {
	reasoning := "private reasoning"
	chat := &dto.OpenAITextResponse{
		Id:    "chatcmpl_1",
		Model: "gpt-test",
		Choices: []dto.OpenAITextResponseChoice{
			{
				Message: dto.Message{
					Role:             "assistant",
					Content:          "final answer",
					ReasoningContent: &reasoning,
				},
			},
		},
	}

	hidden, _, err := ChatCompletionsResponseToResponsesResponseWithOptions(chat, "resp_hidden", ChatCompletionsToResponsesOptions{})
	require.NoError(t, err)
	require.Len(t, hidden.Output, 1)
	assert.Equal(t, responsesOutputTypeMessage, hidden.Output[0].Type)
	assert.Equal(t, "final answer", hidden.Output[0].Content[0].Text)

	visible, _, err := ChatCompletionsResponseToResponsesResponseWithOptions(chat, "resp_visible", ChatCompletionsToResponsesOptions{
		ThinkingToContent: true,
	})
	require.NoError(t, err)
	require.Len(t, visible.Output, 1)
	assert.Equal(t, "<think>\nprivate reasoning\n</think>\nfinal answer", visible.Output[0].Content[0].Text)
}

func TestChatCompletionsResponseToResponsesMapsIncompleteFinishReasons(t *testing.T) {
	tests := []struct {
		name         string
		finishReason string
		wantReason   string
	}{
		{name: "length", finishReason: "length", wantReason: responsesIncompleteReasonMaxTokens},
		{name: "content filter", finishReason: "content_filter", wantReason: responsesIncompleteReasonContentFilter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _, err := ChatCompletionsResponseToResponsesResponse(&dto.OpenAITextResponse{
				Id:    "chatcmpl_1",
				Model: "gpt-test",
				Choices: []dto.OpenAITextResponseChoice{
					{
						Message:      dto.Message{Role: "assistant", Content: "partial"},
						FinishReason: tt.finishReason,
					},
				},
			}, "resp_1")
			require.NoError(t, err)

			assert.Equal(t, `"incomplete"`, string(resp.Status))
			require.NotNil(t, resp.IncompleteDetails)
			assert.Equal(t, tt.wantReason, resp.IncompleteDetails.Reason)
			require.Len(t, resp.Output, 1)
			assert.Equal(t, "incomplete", resp.Output[0].Status)
		})
	}
}

func TestChatCompletionsResponseToResponsesRejectsEmptyChoices(t *testing.T) {
	_, _, err := ChatCompletionsResponseToResponsesResponse(&dto.OpenAITextResponse{
		Model: "kimi-k2.7",
	}, "resp_empty")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestChatCompletionsResponseToResponsesRestoresCodexToolKinds(t *testing.T) {
	message := dto.Message{Role: "assistant"}
	message.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_shell",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "shell_command__run",
				Arguments: `{"cmd":"pwd"}`,
			},
		},
		{
			ID:   "call_patch",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "apply_patch",
				Arguments: `{"input":"*** Begin Patch\n*** End Patch"}`,
			},
		},
		{
			ID:   "call_search",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "tool_search",
				Arguments: `{"query":"shell","limit":5}`,
			},
		},
	})

	resp, _, err := ChatCompletionsResponseToResponsesResponseWithOptions(&dto.OpenAITextResponse{
		Model: "kimi-k2.7",
		Choices: []dto.OpenAITextResponseChoice{
			{Message: message, FinishReason: "tool_calls"},
		},
	}, "resp_1", ChatCompletionsToResponsesOptions{
		ToolMappings: map[string]dto.ResponsesToolMapping{
			"shell_command__run": {
				Kind:      dto.ResponsesToolKindNamespace,
				Name:      "run",
				Namespace: "shell_command",
			},
			"apply_patch": {
				Kind: dto.ResponsesToolKindCustom,
				Name: "apply_patch",
			},
			"tool_search": {
				Kind: dto.ResponsesToolKindToolSearch,
				Name: "tool_search",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, resp.Output, 3)
	assert.Equal(t, responsesOutputTypeFunctionCall, resp.Output[0].Type)
	assert.Equal(t, "fc_call_shell", resp.Output[0].ID)
	assert.Equal(t, "call_shell", resp.Output[0].CallId)
	assert.Equal(t, "shell_command", resp.Output[0].Namespace)
	assert.Equal(t, "run", resp.Output[0].Name)
	assert.Equal(t, `"{\"cmd\":\"pwd\"}"`, string(resp.Output[0].Arguments))
	assert.Equal(t, responsesOutputTypeCustomToolCall, resp.Output[1].Type)
	assert.Equal(t, "ctc_call_patch", resp.Output[1].ID)
	assert.Equal(t, "apply_patch", resp.Output[1].Name)
	assert.Equal(t, "*** Begin Patch\n*** End Patch", resp.Output[1].Input)
	assert.Equal(t, responsesOutputTypeToolSearchCall, resp.Output[2].Type)
	assert.Equal(t, "call_search", resp.Output[2].CallId)
	assert.Equal(t, "client", resp.Output[2].Execution)
	assert.JSONEq(t, `{"query":"shell","limit":5}`, string(resp.Output[2].Arguments))
}

func TestChatCompletionsResponseToResponsesRejectsOnlyUnnamedToolCalls(t *testing.T) {
	message := dto.Message{Role: "assistant", Content: "I will continue."}
	message.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_bad",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "   ",
				Arguments: `{}`,
			},
		},
	})

	_, _, err := ChatCompletionsResponseToResponsesResponse(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{
			{Message: message, FinishReason: "tool_calls"},
		},
	}, "resp_1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "without a function name")
}

func TestChatCompletionsResponseToResponsesKeepsValidToolBesideUnnamedTool(t *testing.T) {
	message := dto.Message{Role: "assistant"}
	message.SetToolCalls([]dto.ToolCallRequest{
		{ID: "call_bad", Type: "function", Function: dto.FunctionRequest{Arguments: `{}`}},
		{ID: "call_good", Type: "function", Function: dto.FunctionRequest{Name: "read_file", Arguments: `{"path":"README.md"}`}},
	})

	resp, _, err := ChatCompletionsResponseToResponsesResponse(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{
			{Message: message, FinishReason: "tool_calls"},
		},
	}, "resp_1")

	require.NoError(t, err)
	require.Len(t, resp.Output, 1)
	assert.Equal(t, "call_good", resp.Output[0].CallId)
	assert.Equal(t, "read_file", resp.Output[0].Name)
}

func TestChatCompletionsStreamToResponsesEventsAggregatesUsageAndToolArgs(t *testing.T) {
	state := NewChatToResponsesStreamState("resp_1", "gpt-test")
	state.Created = 123
	toolIndex := 0

	var events []ChatToResponsesStreamEvent
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Id:      "chatcmpl_1",
		Model:   "gpt-test",
		Created: 123,
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Role: "assistant"}},
		},
	})...)
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: lo.ToPtr("hello")}},
		},
	})...)
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{Index: &toolIndex, ID: "call_1", Type: "function", Function: dto.FunctionResponse{Name: "lookup"}},
			}}},
		},
	})...)
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{Index: &toolIndex, Function: dto.FunctionResponse{Arguments: `{"q":"x"}`}},
			}}},
		},
	})...)
	finishReason := "tool_calls"
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, FinishReason: &finishReason},
		},
	})...)
	events = append(events, mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Usage: &dto.Usage{PromptTokens: 2, CompletionTokens: 4, TotalTokens: 6},
	})...)
	events = append(events, FinalizeChatCompletionsStreamToResponses(state)...)

	require.Len(t, events, 10)
	assert.Equal(t, responsesEventCreated, events[0].Type)
	assert.Equal(t, responsesEventOutputTextDelta, events[2].Type)
	assert.Equal(t, "hello", events[2].Payload.Delta)
	assert.Equal(t, responsesEventFunctionArgsDelta, events[4].Type)
	assert.Equal(t, `{"q":"x"}`, events[4].Payload.Delta)
	assert.Equal(t, responsesEventCompleted, events[9].Type)
	require.NotNil(t, events[9].Payload.Response)
	assert.Equal(t, 6, events[9].Payload.Response.Usage.TotalTokens)
	require.Len(t, events[9].Payload.Response.Output, 2)
	assert.Equal(t, "hello", events[9].Payload.Response.Output[0].Content[0].Text)
	assert.Equal(t, `"{\"q\":\"x\"}"`, string(events[9].Payload.Response.Output[1].Arguments))
}

func TestChatCompletionsStreamToResponsesOptionsControlReasoningVisibility(t *testing.T) {
	hidden := NewChatToResponsesStreamStateWithOptions("resp_hidden", "gpt-test", ChatCompletionsToResponsesOptions{})
	hiddenEvents := mustResponsesEventsFromChatChunk(t, hidden, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: lo.ToPtr("private")}},
		},
	})
	hiddenEvents = append(hiddenEvents, FinalizeChatCompletionsStreamToResponses(hidden)...)
	for _, event := range hiddenEvents {
		assert.NotEqual(t, responsesEventReasoningSummaryDelta, event.Type)
		if event.Payload.Item != nil {
			assert.NotEqual(t, responsesOutputTypeReasoning, event.Payload.Item.Type)
		}
	}
	require.NotEmpty(t, hidden.UsageText())

	visible := NewChatToResponsesStreamStateWithOptions("resp_visible", "gpt-test", ChatCompletionsToResponsesOptions{
		ThinkingToContent: true,
	})
	visibleEvents := mustResponsesEventsFromChatChunk(t, visible, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ReasoningContent: lo.ToPtr("private")}},
		},
	})
	visibleEvents = append(visibleEvents, mustResponsesEventsFromChatChunk(t, visible, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{Index: 0, Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: lo.ToPtr("answer")}},
		},
	})...)
	visibleEvents = append(visibleEvents, FinalizeChatCompletionsStreamToResponses(visible)...)

	var text string
	for _, event := range visibleEvents {
		if event.Type == responsesEventOutputTextDelta {
			text += event.Payload.Delta
		}
	}
	assert.Equal(t, "<think>\nprivate\n</think>\nanswer", text)
}

func TestChatCompletionsStreamToResponsesRestoresCodexToolKinds(t *testing.T) {
	state := NewChatToResponsesStreamStateWithOptions("resp_tools", "kimi-k2.7", ChatCompletionsToResponsesOptions{
		ToolMappings: map[string]dto.ResponsesToolMapping{
			"shell_command__run": {Kind: dto.ResponsesToolKindNamespace, Name: "run", Namespace: "shell_command"},
			"apply_patch":        {Kind: dto.ResponsesToolKindCustom, Name: "apply_patch"},
			"tool_search":        {Kind: dto.ResponsesToolKindToolSearch, Name: "tool_search"},
		},
	})
	indexes := []int{0, 1, 2}
	finishReason := "tool_calls"

	var events []ChatToResponsesStreamEvent
	for _, chunk := range []*dto.ChatCompletionsStreamResponse{
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &indexes[0], ID: "call_shell", Type: "function", Function: dto.FunctionResponse{Name: "shell_command__run"}},
				}},
			}},
		},
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &indexes[0], Function: dto.FunctionResponse{Arguments: `{"cmd":"pwd"}`}},
				}},
			}},
		},
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &indexes[1], ID: "call_patch", Type: "function", Function: dto.FunctionResponse{Name: "apply_patch"}},
				}},
			}},
		},
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &indexes[1], Function: dto.FunctionResponse{Arguments: `{"input":"*** Begin Patch\n*** End Patch"}`}},
				}},
			}},
		},
		{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &indexes[2], ID: "call_search", Type: "function", Function: dto.FunctionResponse{Name: "tool_search", Arguments: `{"query":"shell"}`}},
				}},
			}},
		},
		{Choices: []dto.ChatCompletionsStreamResponseChoice{{FinishReason: &finishReason}}},
	} {
		events = append(events, mustResponsesEventsFromChatChunk(t, state, chunk)...)
	}
	events = append(events, FinalizeChatCompletionsStreamToResponses(state)...)

	var final *dto.OpenAIResponsesResponse
	customInputEvents := 0
	customFunctionArgumentEvents := 0
	for _, event := range events {
		if event.Type == responsesEventCustomToolInputDelta || event.Type == responsesEventCustomToolInputDone {
			customInputEvents++
			assert.Equal(t, "ctc_call_patch", event.Payload.ItemID)
		}
		if event.Type == responsesEventFunctionArgsDelta && event.Payload.ItemID == "ctc_call_patch" {
			customFunctionArgumentEvents++
		}
		if event.Type == responsesEventCompleted {
			final = event.Payload.Response
		}
	}
	require.NotNil(t, final)
	require.Len(t, final.Output, 3)
	assert.Equal(t, "fc_call_shell", final.Output[0].ID)
	assert.Equal(t, "shell_command", final.Output[0].Namespace)
	assert.Equal(t, "run", final.Output[0].Name)
	assert.Equal(t, "ctc_call_patch", final.Output[1].ID)
	assert.Equal(t, responsesOutputTypeCustomToolCall, final.Output[1].Type)
	assert.Equal(t, "*** Begin Patch\n*** End Patch", final.Output[1].Input)
	assert.Equal(t, responsesOutputTypeToolSearchCall, final.Output[2].Type)
	assert.Equal(t, "client", final.Output[2].Execution)
	assert.Equal(t, 2, customInputEvents)
	assert.Zero(t, customFunctionArgumentEvents)
}

func TestChatCompletionsStreamToResponsesHandlesLateNameAndMissingIndexes(t *testing.T) {
	state := NewChatToResponsesStreamState("resp_late", "kimi-k2.7")

	first := mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{ID: "call_a", Type: "function", Function: dto.FunctionResponse{Arguments: `{"path":"`}},
			}},
		}},
	})
	for _, event := range first {
		assert.NotEqual(t, responsesEventOutputItemAdded, event.Type)
	}
	second := mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{Function: dto.FunctionResponse{Name: "read_file", Arguments: `README.md"}`}},
			}},
		}},
	})
	require.Len(t, second, 2)
	assert.Equal(t, responsesEventOutputItemAdded, second[0].Type)
	assert.Equal(t, "call_a", second[0].Payload.ItemID)
	assert.Equal(t, responsesEventFunctionArgsDelta, second[1].Type)
	assert.Equal(t, `{"path":"README.md"}`, second[1].Payload.Delta)

	mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
				{ID: "call_b", Type: "function", Function: dto.FunctionResponse{Name: "shell_command", Arguments: `{"cmd":"pwd"}`}},
			}},
		}},
	})
	finishReason := "tool_calls"
	mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{FinishReason: &finishReason}},
	})
	finalEvents := FinalizeChatCompletionsStreamToResponses(state)
	require.Len(t, finalEvents, 1)
	require.NotNil(t, finalEvents[0].Payload.Response)
	require.Len(t, finalEvents[0].Payload.Response.Output, 2)
	assert.Equal(t, "call_a", finalEvents[0].Payload.Response.Output[0].CallId)
	assert.Equal(t, "call_b", finalEvents[0].Payload.Response.Output[1].CallId)
}

func TestChatCompletionsStreamToResponsesReportsMalformedAndTruncatedTurns(t *testing.T) {
	t.Run("unnamed tool fails", func(t *testing.T) {
		state := NewChatToResponsesStreamState("resp_bad", "kimi-k2.7")
		index := 0
		finishReason := "tool_calls"
		mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &index, ID: "call_bad", Type: "function", Function: dto.FunctionResponse{Name: "   ", Arguments: `{}`}},
				}},
				FinishReason: &finishReason,
			}},
		})
		final := FinalizeChatCompletionsStreamToResponses(state)
		require.Len(t, final, 1)
		assert.Equal(t, responsesEventFailed, final[0].Type)
		require.NotNil(t, final[0].Payload.Response)
		assert.Equal(t, `"failed"`, string(final[0].Payload.Response.Status))
		assert.Equal(t, "upstream_tool_call_dropped", final[0].Payload.Response.Error.(map[string]any)["code"])
	})

	t.Run("length remains incomplete", func(t *testing.T) {
		state := NewChatToResponsesStreamState("resp_length", "kimi-k2.7")
		index := 0
		finishReason := "length"
		mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{ToolCalls: []dto.ToolCallResponse{
					{Index: &index, ID: "call_cut", Type: "function", Function: dto.FunctionResponse{Arguments: `{"pa`}},
				}},
				FinishReason: &finishReason,
			}},
		})
		final := FinalizeChatCompletionsStreamToResponses(state)
		require.Len(t, final, 1)
		assert.Equal(t, responsesEventIncomplete, final[0].Type)
		require.NotNil(t, final[0].Payload.Response.IncompleteDetails)
		assert.Equal(t, responsesIncompleteReasonMaxTokens, final[0].Payload.Response.IncompleteDetails.Reason)
	})

	t.Run("stream cut with output is incomplete", func(t *testing.T) {
		state := NewChatToResponsesStreamState("resp_cut", "kimi-k2.7")
		mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{
			Choices: []dto.ChatCompletionsStreamResponseChoice{{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: lo.ToPtr("partial")},
			}},
		})
		final := FinalizeChatCompletionsStreamToResponses(state)
		require.NotEmpty(t, final)
		last := final[len(final)-1]
		assert.Equal(t, responsesEventIncomplete, last.Type)
		require.NotNil(t, last.Payload.Response.IncompleteDetails)
		assert.Equal(t, responsesIncompleteReasonStreamCut, last.Payload.Response.IncompleteDetails.Reason)
	})

	t.Run("stream cut without output fails", func(t *testing.T) {
		state := NewChatToResponsesStreamState("resp_empty", "kimi-k2.7")
		mustResponsesEventsFromChatChunk(t, state, &dto.ChatCompletionsStreamResponse{})
		final := FinalizeChatCompletionsStreamToResponses(state)
		require.Len(t, final, 1)
		assert.Equal(t, responsesEventFailed, final[0].Type)
		require.NotNil(t, final[0].Payload.Response.Error)
	})
}

func mustResponsesEventsFromChatChunk(t *testing.T, state *ChatToResponsesStreamState, chunk *dto.ChatCompletionsStreamResponse) []ChatToResponsesStreamEvent {
	t.Helper()
	events, err := ChatCompletionsStreamChunkToResponsesEvents(chunk, state)
	require.NoError(t, err)
	return events
}
