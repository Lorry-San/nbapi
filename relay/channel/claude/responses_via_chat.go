package claude

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/openai"
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

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	claudeInfo := &ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}

	var claudeResp dto.ClaudeResponse
	if err := common.Unmarshal(responseBody, &claudeResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if claudeError := claudeResp.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
		return nil, types.WithClaudeError(*claudeError, http.StatusInternalServerError)
	}
	maybeMarkClaudeRefusal(c, claudeResp.StopReason)

	if claudeResp.Usage != nil {
		claudeInfo.Usage.PromptTokens = claudeResp.Usage.InputTokens
		claudeInfo.Usage.CompletionTokens = claudeResp.Usage.OutputTokens
		claudeInfo.Usage.TotalTokens = claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens
		claudeInfo.Usage.UsageSemantic = "anthropic"
		claudeInfo.Usage.PromptTokensDetails.CachedTokens = claudeResp.Usage.CacheReadInputTokens
		claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens = claudeResp.Usage.CacheCreationInputTokens
		claudeInfo.Usage.ClaudeCacheCreation5mTokens = claudeResp.Usage.GetCacheCreation5mTokens()
		claudeInfo.Usage.ClaudeCacheCreation1hTokens = claudeResp.Usage.GetCacheCreation1hTokens()
	}
	if claudeResp.Usage != nil && claudeResp.Usage.ServerToolUse != nil && claudeResp.Usage.ServerToolUse.WebSearchRequests > 0 {
		c.Set("claude_web_search_requests", claudeResp.Usage.ServerToolUse.WebSearchRequests)
	}

	chatResp := ResponseClaude2OpenAI(&claudeResp)
	chatResp.Usage = buildOpenAIStyleUsageFromClaudeUsage(claudeInfo.Usage)

	responseID := "resp_" + strings.TrimPrefix(helper.GetResponseID(c), "chatcmpl-")
	responsesResp, _, err := service.ChatCompletionsResponseToResponsesResponseWithOptions(
		chatResp,
		responseID,
		service.ChatCompletionsToResponsesOptions{
			ThinkingToContent: info != nil && info.ChannelSetting.ThinkingToContent,
		},
	)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if responsesResp.Usage == nil || responsesResp.Usage.TotalTokens == 0 {
		usage := service.ResponseText2Usage(c, extractClaudeChatResponseText(chatResp), info.UpstreamModelName, info.GetEstimatePromptTokens())
		responsesResp.Usage = usage
		claudeInfo.Usage = usage
	}

	responseData, err := common.Marshal(responsesResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, responseData)
	return claudeInfo.Usage, nil
}

func extractClaudeChatResponseText(resp *dto.OpenAITextResponse) string {
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

func ChatCompletionsToResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	reader, writer := io.Pipe()
	chatResp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       reader,
	}
	chatResp.Header.Set("Content-Type", "text/event-stream")

	type streamResult struct {
		usage *dto.Usage
		err   *types.NewAPIError
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		usage, newAPIError := openai.OaiChatToResponsesStreamHandler(c, info, chatResp)
		resultCh <- streamResult{usage: usage, err: newAPIError}
	}()

	claudeInfo := &ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      common.GetTimestamp(),
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}

	writeChatChunk := func(chunk *dto.ChatCompletionsStreamResponse) error {
		if chunk == nil {
			return nil
		}
		raw, err := common.Marshal(chunk)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(writer, "data: %s\n\n", raw)
		return err
	}

	streamErr := scanClaudeStreamAsChatCompletions(c, info, resp.Body, claudeInfo, writeChatChunk)
	if streamErr == nil {
		finalizeClaudeResponsesFallbackUsage(c, info, claudeInfo)
		openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(claudeInfo.Usage)
		finalUsage := helper.GenerateFinalUsageResponse(claudeInfo.ResponseId, claudeInfo.Created, info.UpstreamModelName, openAIUsage)
		if err := writeChatChunk(finalUsage); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		} else if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	}

	if streamErr != nil {
		_ = writer.CloseWithError(streamErr)
	} else {
		_ = writer.Close()
	}

	result := <-resultCh
	if result.err != nil {
		return nil, result.err
	}
	if streamErr != nil {
		return nil, streamErr
	}
	return claudeInfo.Usage, nil
}

func scanClaudeStreamAsChatCompletions(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	body io.Reader,
	claudeInfo *ClaudeResponseInfo,
	writeChatChunk func(*dto.ChatCompletionsStreamResponse) error,
) *types.NewAPIError {
	scanner := bufio.NewScanner(body)
	maxBufferSize := helper.DefaultMaxScannerBufferSize
	if constant.StreamScannerMaxBufferMB > 0 {
		maxBufferSize = constant.StreamScannerMaxBufferMB << 20
	}
	scanner.Buffer(make([]byte, helper.InitialScannerBufferSize), maxBufferSize)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 6 || (!strings.HasPrefix(line, "data:") && !strings.HasPrefix(line, "[DONE]")) {
			continue
		}
		if strings.HasPrefix(line, "[DONE]") {
			break
		}
		data := strings.TrimSpace(line[5:])
		if data == "" {
			continue
		}
		if strings.HasPrefix(data, "[DONE]") {
			break
		}
		info.SetFirstResponseTime()
		info.ReceivedResponseCount++

		var claudeResp dto.ClaudeResponse
		if err := common.UnmarshalJsonStr(data, &claudeResp); err != nil {
			return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		if claudeError := claudeResp.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
			return types.WithClaudeError(*claudeError, http.StatusInternalServerError)
		}
		if claudeResp.StopReason != "" {
			maybeMarkClaudeRefusal(c, claudeResp.StopReason)
		}
		if claudeResp.Delta != nil && claudeResp.Delta.StopReason != nil {
			maybeMarkClaudeRefusal(c, *claudeResp.Delta.StopReason)
		}
		if claudeResp.Type == "message_start" && claudeResp.Message != nil {
			info.UpstreamModelName = claudeResp.Message.Model
		}

		chatChunk := StreamResponseClaude2OpenAI(&claudeResp)
		if !FormatClaudeResponseInfo(&claudeResp, chatChunk, claudeInfo) {
			continue
		}
		if err := writeChatChunk(chatChunk); err != nil {
			return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	return nil
}

func finalizeClaudeResponsesFallbackUsage(c *gin.Context, info *relaycommon.RelayInfo, claudeInfo *ClaudeResponseInfo) {
	if claudeInfo == nil {
		return
	}
	if claudeInfo.Usage == nil {
		claudeInfo.Usage = &dto.Usage{}
	}
	if claudeInfo.Usage.CompletionTokens == 0 || !claudeInfo.Done {
		fallback := service.ResponseText2Usage(c, claudeInfo.ResponseText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
		if claudeInfo.Usage.CompletionTokens == 0 ||
			(!claudeInfo.Done && fallback.CompletionTokens > claudeInfo.Usage.CompletionTokens) {
			claudeInfo.Usage.CompletionTokens = fallback.CompletionTokens
		}
		if claudeInfo.Usage.PromptTokens == 0 {
			claudeInfo.Usage.PromptTokens = fallback.PromptTokens
		}
		claudeInfo.Usage.TotalTokens = claudeInfo.Usage.PromptTokens + claudeInfo.Usage.CompletionTokens
	}
	claudeInfo.Usage.UsageSemantic = "anthropic"
}
