package relay

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	appconstant "github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/dto"
	"github.com/Lorry-San/nbapi/relay/channel"
	"github.com/Lorry-San/nbapi/relay/channel/claude"
	openaichannel "github.com/Lorry-San/nbapi/relay/channel/openai"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	relayconstant "github.com/Lorry-San/nbapi/relay/constant"
	"github.com/Lorry-San/nbapi/service"
	"github.com/Lorry-San/nbapi/types"

	"github.com/gin-gonic/gin"
)

func responsesViaChatCompletions(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.OpenAIResponsesRequest) (*dto.Usage, *types.NBAPIError) {
	chatReq, err := service.ResponsesRequestToChatCompletionsRequest(request)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	if chatReq.MaxCompletionTokens != nil && chatReq.MaxTokens == nil {
		chatReq.MaxTokens = chatReq.MaxCompletionTokens
		chatReq.MaxCompletionTokens = nil
	}
	if info.SupportStreamOptions && info.IsStream && appconstant.ForceStreamOption {
		if chatReq.StreamOptions == nil {
			chatReq.StreamOptions = &dto.StreamOptions{}
		}
		chatReq.StreamOptions.IncludeUsage = true
	}

	applySystemPromptIfNeeded(c, info, chatReq)

	savedRelayMode := info.RelayMode
	savedRequestURLPath := info.RequestURLPath
	defer func() {
		info.RelayMode = savedRelayMode
		info.RequestURLPath = savedRequestURLPath
	}()

	info.RelayMode = relayconstant.RelayModeChatCompletions
	info.RequestURLPath = "/v1/chat/completions"
	info.AppendRequestConversion(types.RelayFormatOpenAI)

	convertedRequest, err := adaptor.ConvertOpenAIRequest(c, info, chatReq)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	info.FinalRequestRelayFormat = types.RelayFormatOpenAI
	if format, ok := relaycommon.GuessRelayFormatFromRequest(convertedRequest); ok {
		info.FinalRequestRelayFormat = format
	}

	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, false)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return nil, nbapiErrorFromParamOverride(err)
		}
	}

	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer closer.Close()
	jsonData = nil
	info.UpstreamRequestBodySize = size
	var requestBody io.Reader = body

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	if resp == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	httpResp := resp.(*http.Response)
	info.IsStream = info.IsStream || strings.Contains(strings.ToLower(httpResp.Header.Get("Content-Type")), "text/event-stream")
	if httpResp.StatusCode != http.StatusOK {
		nbapiErr := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(nbapiErr, statusCodeMappingStr)
		return nil, nbapiErr
	}

	usage, nbapiErr := handleResponsesViaChatCompletionsResponse(c, info, httpResp)
	if nbapiErr != nil {
		service.ResetStatusCode(nbapiErr, statusCodeMappingStr)
		return nil, nbapiErr
	}
	return usage, nil
}

func handleResponsesViaChatCompletionsResponse(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NBAPIError) {
	switch info.GetFinalRequestRelayFormat() {
	case types.RelayFormatClaude:
		if info.IsStream {
			return claude.ChatCompletionsToResponsesStreamHandler(c, info, resp)
		}
		return claude.ChatCompletionsToResponsesHandler(c, info, resp)
	default:
		if info.IsStream {
			return openaichannel.OaiChatToResponsesStreamHandler(c, info, resp)
		}
		return openaichannel.OaiChatToResponsesHandler(c, info, resp)
	}
}
