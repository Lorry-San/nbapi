package middleware

import (
	"errors"
	"slices"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/model"
	"github.com/Lorry-San/nbapi/service"
	"github.com/gin-gonic/gin"
)

func LightcodeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"type": "lightcode_error", "code": code, "message": message}})
}

func lightcodeAttestFail(c *gin.Context, e *service.LightcodeAttestError) {
	body := gin.H{"type": "lightcode_error", "code": e.Code, "message": e.Message}
	if e.ServerTime != 0 { body["server_time"] = e.ServerTime }
	c.AbortWithStatusJSON(e.Status, gin.H{"error": body})
}

// LightcodeClientGuard rejects non-Lightcode user agents and outdated versions before authentication.
func LightcodeClientGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.LightcodeEnforce { c.Next(); return }
		if _, e := service.LightcodeCheckUA(c.Request.UserAgent()); e != nil { lightcodeAttestFail(c, e); return }
		c.Next()
	}
}

// LightcodeRequestSignature runs after LightcodeAuth and verifies the LC1 device signature without reading the body.
func LightcodeRequestSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !service.LightcodeEnforce { c.Next(); return }
		session, ok := c.Get("lightcode_session")
		s, _ := session.(*model.LightcodeSession)
		if !ok || s == nil { LightcodeError(c, 401, "invalid_token", "Sign in to Lightcode"); return }
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if e := service.VerifyLightcodeRequest(s, c.Request.Method, c.Request.URL.RequestURI(), token, c.GetHeader("X-LC-Ts"), c.GetHeader("X-LC-Nonce"), c.GetHeader("X-LC-Sig")); e != nil { lightcodeAttestFail(c, e); return }
		if service.LightcodeRecordIP(s.Id, common.GetClientIP(c)) > service.LightcodeMaxSessionIPs {
			_ = service.LightcodeRevokeSession(s.Id)
			LightcodeError(c, 401, "invalid_token", "Lightcode login expired or was revoked"); return
		}
		c.Next()
	}
}

// LightcodeAuth admits only Lightcode access tokens, never API keys or dashboard tokens.
func LightcodeAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") { LightcodeError(c, 401, "invalid_token", "Sign in to Lightcode"); return }
		session, err := service.ValidateLightcodeAccess(strings.TrimPrefix(header, "Bearer "))
		if errors.Is(err, model.ErrLightcodeGrant) { LightcodeError(c, 401, "invalid_token", "Lightcode login expired or was revoked"); return }
		if err != nil { LightcodeError(c, 503, "auth_unavailable", "认证服务暂时不可用，请稍后重试"); return }
		user, err := model.GetUserCache(session.UserId)
		if err != nil { LightcodeError(c, 503, "account_unavailable", "Account is temporarily unavailable"); return }
		user.WriteContext(c)
		c.Set("id", session.UserId)
		c.Set("lightcode_request", true)
		c.Set("lightcode_session_id", session.Id)
		c.Set("lightcode_session", session)
		c.Next()
	}
}

// LightcodeEntitlements validates model/effort before routing and pins billing to this subscription.
func LightcodeEntitlements() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct { Model string `json:"model"`; ReasoningEffort string `json:"reasoning_effort"`; Group string `json:"group"` }
		if err := common.UnmarshalBodyReusable(c, &request); err != nil || request.Model == "" {
			LightcodeError(c, 400, "invalid_request", "A model and valid JSON request are required"); return
		}
		subs, err := model.LightcodeSubscriptions(c.GetInt("id"))
		if err != nil { LightcodeError(c, 503, "subscription_unavailable", "Subscription is temporarily unavailable"); return }
		var active *model.UserSubscription
		for i := range subs { if subs[i].Status == "active" { active = &subs[i]; break } }
		if active == nil { LightcodeError(c, 402, "subscription_required", "需要有效的 Lightcode 订阅"); return }
		if request.Group != "" && request.Group != active.RoutingGroup { LightcodeError(c, 400, "group_forbidden", "路由分组由订阅决定，不能手动指定"); return }
		var policies []model.LightcodeGroupModel
		if err := model.DB.Where("routing_group = ? AND model = ?", active.RoutingGroup, request.Model).Find(&policies).Error; err != nil { LightcodeError(c, 503, "subscription_unavailable", "订阅服务暂时不可用，请稍后重试"); return }
		policy, reason := model.LightcodeModelEntitlement(active, request.Model, policies)
		if reason == "subscription_model_unsupported" { LightcodeError(c, 409, reason, "当前订阅不支持该模型"); return }
		if reason != "" { LightcodeError(c, 409, reason, "该模型暂未在 Lightcode 中开放"); return }
		if request.ReasoningEffort != "" && !slices.Contains(policy.Efforts, request.ReasoningEffort) { LightcodeError(c, 400, "reasoning_effort_unsupported", "Unsupported reasoning effort"); return }
		c.Set("lightcode_subscription_id", active.Id)
		c.Set("lightcode_multiplier", policy.EffectiveMultiplier())
		c.Set("token_name", "Lightcode")
		common.SetContextKey(c, constant.ContextKeyTokenUnlimited, true)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, active.RoutingGroup)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, active.RoutingGroup)
		// The existing relay dispatches its upstream protocol by canonical request path.
		c.Request.URL.Path = "/v1/chat/completions"
		c.Next()
	}
}
