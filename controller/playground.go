package controller

import (
	"errors"
	"fmt"

	"github.com/Lorry-San/nbapi/middleware"
	"github.com/Lorry-San/nbapi/model"
	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/Lorry-San/nbapi/types"

	"github.com/gin-gonic/gin"
)

func Playground(c *gin.Context) {
	var nbapiError *types.NBAPIError

	defer func() {
		if nbapiError != nil {
			c.JSON(nbapiError.StatusCode, gin.H{
				"error": nbapiError.ToOpenAIError(),
			})
		}
	}()

	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		nbapiError = types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, nil, nil)
	if err != nil {
		nbapiError = types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		return
	}

	userId := c.GetInt("id")

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		nbapiError = types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
		return
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)

	Relay(c, types.RelayFormatOpenAI)
}
