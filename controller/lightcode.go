package controller

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/Lorry-San/nbapi/service"
	"github.com/gin-gonic/gin"
)

func LightcodeAuthorizePage(c *gin.Context) {
	if err := service.ValidateLightcodeRedirect(c.Query("client_id"), c.Query("redirect_uri")); err != nil { c.JSON(400, gin.H{"error": "invalid_request"}); return }
	c.Redirect(http.StatusFound, "/lightcode/authorize?"+c.Request.URL.RawQuery)
}

func LightcodeAuthorize(c *gin.Context) {
	var req struct { ClientId string `json:"client_id"`; RedirectURI string `json:"redirect_uri"`; Challenge string `json:"code_challenge"`; Method string `json:"code_challenge_method"`; State string `json:"state"`; ResponseType string `json:"response_type"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.State == "" || len(req.State) > 512 || req.ResponseType != "code" { c.JSON(400, gin.H{"error": "invalid_request"}); return }
	code, err := service.AuthorizeLightcode(c.GetInt("id"), req.ClientId, req.RedirectURI, req.Challenge, req.Method)
	if err != nil { c.JSON(400, gin.H{"error": "invalid_request", "error_description": err.Error()}); return }
	u, _ := url.Parse(req.RedirectURI)
	q := u.Query(); q.Set("code", code); q.Set("state", req.State); u.RawQuery = q.Encode()
	c.JSON(200, gin.H{"redirect_uri": u.String()})
}

func LightcodeToken(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	var bundle *service.LightcodeTokenBundle
	var err error
	switch c.PostForm("grant_type") {
	case "authorization_code": bundle, err = service.ExchangeLightcodeGrant(c.PostForm("code"), c.PostForm("client_id"), c.PostForm("redirect_uri"), c.PostForm("code_verifier"), common.GetClientIP(c), c.Request.UserAgent())
	case "refresh_token": bundle, err = service.RefreshLightcodeGrant(c.PostForm("refresh_token"), c.PostForm("client_id"))
	default: c.JSON(400, gin.H{"error": "unsupported_grant_type"}); return
	}
	if err != nil { c.JSON(400, gin.H{"error": "invalid_grant"}); return }
	c.JSON(200, bundle)
}

func LightcodeRevoke(c *gin.Context) {
	if err := service.RevokeLightcodeToken(c.PostForm("token")); err != nil { c.JSON(503, gin.H{"error": "temporarily_unavailable"}); return }
	c.Status(200)
}

func LightcodeAccount(c *gin.Context) {
	subs, err := model.LightcodeSubscriptions(c.GetInt("id"))
	if err != nil { c.JSON(503, gin.H{"error": "subscription_unavailable"}); return }
	user, err := model.GetUserCache(c.GetInt("id"))
	if err != nil { c.JSON(503, gin.H{"error": "account_unavailable"}); return }
	views := []gin.H{}
	for _, sub := range subs {
		var plan model.SubscriptionPlan
		_ = common.UnmarshalJsonStr(sub.PlanSnapshot, &plan)
		remaining := ""
		if sub.AmountTotal > 0 && common.QuotaPerUnit > 0 {
			remaining = strconv.FormatFloat(float64(max(sub.AmountTotal-sub.AmountUsed, 0))/common.QuotaPerUnit, 'f', 2, 64)
		}
		views = append(views, gin.H{"id": sub.Id, "status": sub.Status, "plan": plan.Title, "start_time": sub.StartTime, "end_time": sub.EndTime, "next_reset_time": sub.NextResetTime, "remaining_usd": remaining, "unlimited": sub.AmountTotal == 0})
	}
	c.JSON(200, gin.H{"user": gin.H{"id": user.Id, "name": user.Username}, "subscriptions": views})
}

func LightcodeCatalog(c *gin.Context) {
	subs, err := model.LightcodeSubscriptions(c.GetInt("id"))
	if err != nil { c.JSON(503, gin.H{"error": "subscription_unavailable"}); return }
	models := []gin.H{}
	for _, sub := range subs {
		if sub.Status != "active" { continue }
		var policies []model.LightcodeGroupModel
		if err := model.DB.Where("routing_group = ?", sub.RoutingGroup).Order("id asc").Find(&policies).Error; err != nil { c.JSON(503, gin.H{"error": "catalog_unavailable"}); return }
		var names []string
		if err := model.DB.Model(&model.Ability{}).Distinct("model").Where("enabled = ?", true).Where(map[string]interface{}{"group": sub.RoutingGroup}).Pluck("model", &names).Error; err != nil { c.JSON(503, gin.H{"error": "catalog_unavailable"}); return }
		for _, name := range names {
			policy := model.LightcodeGroupModel{Model: name, DisplayName: name, Efforts: []string{}}
			configured := false
			for _, p := range policies { if p.Model == name { policy, configured = p, true; break } }
			available := configured && slices.Contains(sub.AllowedModels, name)
			reason := ""; if !available { reason = "subscription_model_unsupported" }
			models = append(models, gin.H{"id": name, "display_name": policy.DisplayName, "efforts": policy.Efforts, "default_effort": policy.DefaultEffort, "available": available, "unavailable_reason": reason, "context_window": policy.ContextWindow, "max_output_tokens": policy.MaxOutputTokens, "vision": policy.Vision})
		}
		break
	}
	c.JSON(200, gin.H{"models": models})
}

func AdminLightcodeModels(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		var models []model.LightcodeGroupModel
		if err := model.DB.Order("routing_group asc, id asc").Find(&models).Error; err != nil { common.ApiError(c, err); return }
		common.ApiSuccess(c, models); return
	}
	var req struct { Group string `json:"group"`; Models []model.LightcodeGroupModel `json:"models"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Group == "" || req.Group == "auto" || len(req.Models) > 500 { common.ApiErrorMsg(c, "Invalid Lightcode model group"); return }
	seen := map[string]bool{}
	for i := range req.Models {
		p := &req.Models[i]
		if p.Model == "" || len(p.Model) > 191 || seen[p.Model] || p.ContextWindow <= 0 || p.MaxOutputTokens <= 0 || p.MaxOutputTokens > p.ContextWindow { common.ApiErrorMsg(c, "Invalid model or token limits"); return }
		seen[p.Model] = true
		for _, effort := range p.Efforts { if !slices.Contains([]string{"minimal", "low", "medium", "high", "xhigh", "max"}, effort) { common.ApiErrorMsg(c, "Invalid reasoning effort"); return } }
		if p.DefaultEffort != "" && !slices.Contains(p.Efforts, p.DefaultEffort) { common.ApiErrorMsg(c, "Default reasoning effort must be supported"); return }
		if len(p.Efforts) > 0 && p.DefaultEffort == "" { common.ApiErrorMsg(c, "Select a default reasoning effort"); return }
		p.Id, p.RoutingGroup = 0, req.Group
		if p.DisplayName == "" { p.DisplayName = p.Model }
	}
	if err := model.ReplaceLightcodeGroupModels(req.Group, req.Models); err != nil { common.ApiError(c, err); return }
	common.ApiSuccess(c, nil)
}

func LightcodeDevices(c *gin.Context) {
	if c.Request.Method == http.MethodDelete {
		if err := model.DB.Model(&model.LightcodeSession{}).Where("id = ? AND user_id = ?", c.Param("id"), c.GetInt("id")).Update("revoked_at", time.Now().Unix()).Error; err != nil { common.ApiError(c, err); return }
		common.ApiSuccess(c, nil); return
	}
	var sessions []model.LightcodeSession
	if err := model.DB.Where("user_id = ?", c.GetInt("id")).Order("created_at desc").Limit(100).Find(&sessions).Error; err != nil { common.ApiError(c, err); return }
	common.ApiSuccess(c, sessions)
}
