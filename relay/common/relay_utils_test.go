package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lorry-San/nbapi/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestValidateMultipartDirectNormalizesImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := strings.NewReader(`{"model":"wan2.7-i2v","prompt":"animate","image":" https://example.com/first.png "}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	info := &RelayInfo{
		TaskRelayInfo: &TaskRelayInfo{},
	}

	taskErr := ValidateMultipartDirect(context, info)

	require.Nil(t, taskErr)
	storedReq, err := GetTaskRequest(context)
	require.NoError(t, err)
	require.Equal(t, []string{"https://example.com/first.png"}, storedReq.Images)
	require.Equal(t, constant.TaskActionGenerate, info.Action)
}

func TestGetFullRequestURLUsesVersionedBasePathForResponses(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL("https://upstream.example/v3", "/v1/responses", constant.ChannelTypeOpenAI)

	require.Equal(t, "https://upstream.example/v3/responses", got)
}

func TestGetFullRequestURLUsesVersionedBasePathWithTrailingSlash(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL("https://upstream.example/api/v3/", "/v1/chat/completions", constant.ChannelTypeOpenAI)

	require.Equal(t, "https://upstream.example/api/v3/chat/completions", got)
}

func TestGetFullRequestURLKeepsDefaultVersionForNonVersionedBasePath(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL("https://upstream.example/api", "/v1/responses", constant.ChannelTypeOpenAI)

	require.Equal(t, "https://upstream.example/api/v1/responses", got)
}

func TestGetFullRequestURLDoesNotTreatHostAsVersionPath(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL("https://v3.example.com", "/v1/responses", constant.ChannelTypeOpenAI)

	require.Equal(t, "https://v3.example.com/v1/responses", got)
}
