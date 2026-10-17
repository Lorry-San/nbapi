package middleware

import (
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

// LightcodeAuth admits only Lightcode access tokens, never API keys or dashboard tokens.
func LightcodeAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") { LightcodeError(c, 401, "invalid_token", "Sign in to Lightcode"); return }
		session, err := service.ValidateLightcodeAccess(strings.TrimPrefix(header, "Bearer "))
		if err != nil { LightcodeError(c, 401, "invalid_token", "Lightcode login expired or was revoked"); return }
		user, err := model.GetUserCache(session.UserId)
		if err != nil { LightcodeError(c, 503, "account_unavailable", "Account is temporarily unavailable"); return }
		user.WriteContext(c)
		c.Set("id", session.UserId)
		c.Set("lightcode_request", true)
		c.Set("lightcode_session_id", session.Id)
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
		if active == nil { LightcodeError(c, 403, "subscription_required", "An active Lightcode subscription is required"); return }
		if !slices.Contains(active.AllowedModels, request.Model) { LightcodeError(c, 403, "subscription_model_unsupported", "此订阅不支持该模型"); return }
		if request.Group != "" && request.Group != active.RoutingGroup { LightcodeError(c, 403, "group_forbidden", "The subscription determines the routing group"); return }
		var policy model.LightcodeGroupModel
		if err := model.DB.Where("routing_group = ? AND model = ?", active.RoutingGroup, request.Model).First(&policy).Error; err != nil { LightcodeError(c, 403, "model_unavailable", "This model is not configured for Lightcode"); return }
		if request.ReasoningEffort != "" && !slices.Contains(policy.Efforts, request.ReasoningEffort) { LightcodeError(c, 400, "reasoning_effort_unsupported", "Unsupported reasoning effort"); return }
		c.Set("lightcode_subscription_id", active.Id)
		c.Set("token_name", "Lightcode")
		common.SetContextKey(c, constant.ContextKeyTokenUnlimited, true)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, active.RoutingGroup)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, active.RoutingGroup)
		common.SetContextKey(c, constant.ContextKeyUserGroup, active.RoutingGroup)
		// The existing relay dispatches its upstream protocol by canonical request path.
		c.Request.URL.Path = "/v1/chat/completions"
		c.Next()
	}
}
