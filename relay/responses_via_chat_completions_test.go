package relay

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lorry-San/nbapi/common"
	appconstant "github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/dto"
	openaichannel "github.com/Lorry-San/nbapi/relay/channel/openai"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	relayconstant "github.com/Lorry-San/nbapi/relay/constant"
	"github.com/Lorry-San/nbapi/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureResponsesChatAdaptor struct {
	openaichannel.Adaptor
	request   dto.GeneralOpenAIRequest
	decodeErr error
}

func (a *captureResponsesChatAdaptor) DoRequest(_ *gin.Context, _ *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	a.decodeErr = common.DecodeJson(requestBody, &a.request)
	body := strings.Join([]string{
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"claude-opus-5","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"}}]}`,
		"",
		`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"claude-opus-5","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11,"prompt_tokens_details":{"cached_tokens":8}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestResponsesViaChatCompletionsForcesStreamUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	savedForceStreamOption := appconstant.ForceStreamOption
	savedStreamingTimeout := appconstant.StreamingTimeout
	appconstant.ForceStreamOption = true
	appconstant.StreamingTimeout = 60
	t.Cleanup(func() {
		appconstant.ForceStreamOption = savedForceStreamOption
		appconstant.StreamingTimeout = savedStreamingTimeout
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	stream := true
	request := &dto.OpenAIResponsesRequest{
		Model:  "claude-opus-5",
		Input:  json.RawMessage(`"hello"`),
		Stream: &stream,
		StreamOptions: &dto.StreamOptions{
			IncludeObfuscation: true,
		},
	}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponses,
		IsStream:        true,
		OriginModelName: "claude-opus-5",
		RequestURLPath:  "/v1/responses",
		RelayFormat:     types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:          appconstant.ChannelTypeOpenAI,
			UpstreamModelName:    "claude-opus-5",
			SupportStreamOptions: true,
		},
	}
	info.SetEstimatePromptTokens(99)
	adaptor := &captureResponsesChatAdaptor{
		Adaptor: openaichannel.Adaptor{ChannelType: appconstant.ChannelTypeOpenAI},
	}

	usage, apiErr := responsesViaChatCompletions(ctx, info, adaptor, request)

	require.Nil(t, apiErr)
	require.NoError(t, adaptor.decodeErr)
	require.NotNil(t, adaptor.request.StreamOptions)
	assert.True(t, adaptor.request.StreamOptions.IncludeUsage)
	require.NotNil(t, usage)
	assert.Equal(t, 10, usage.PromptTokens)
	require.NotNil(t, usage.InputTokensDetails)
	assert.Equal(t, 8, usage.InputTokensDetails.CachedTokens)
	require.NotNil(t, usage.BillingUsage)
	require.NotNil(t, usage.BillingUsage.OpenAIUsage)
	assert.Equal(t, 8, usage.BillingUsage.OpenAIUsage.PromptTokensDetails.CachedTokens)
}
