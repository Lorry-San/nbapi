package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/Lorry-San/nbapi/relay/common"
	"github.com/Lorry-San/nbapi/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleGroupRatioAppliesLightcodeMultiplierOnlyToLightcode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	base := ratio_setting.GetGroupRatio("default")

	newCtx := func() *gin.Context {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
		ctx.Set("lightcode_multiplier", 2.5)
		return ctx
	}

	lc := HandleGroupRatio(newCtx(), &relaycommon.RelayInfo{UserGroup: "default", UsingGroup: "default", IsLightcode: true})
	require.InDelta(t, base*2.5, lc.GroupRatio, 1e-9)

	api := HandleGroupRatio(newCtx(), &relaycommon.RelayInfo{UserGroup: "default", UsingGroup: "default"})
	require.InDelta(t, base, api.GroupRatio, 1e-9)
}
