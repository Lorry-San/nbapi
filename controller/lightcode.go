package controller

import (
	"errors"
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
	if err := service.ValidateLightcodeRedirect(c.Query("client_id"), c.Query("redirect_uri")); err != nil {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}
	c.Redirect(http.StatusFound, "/lightcode/authorize?"+c.Request.URL.RawQuery)
}

func LightcodeAuthorize(c *gin.Context) {
	var req struct {
		ClientId     string `json:"client_id"`
		RedirectURI  string `json:"redirect_uri"`
		Challenge    string `json:"code_challenge"`
		Method       string `json:"code_challenge_method"`
		State        string `json:"state"`
		ResponseType string `json:"response_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.State == "" || len(req.State) > 512 || req.ResponseType != "code" {
		c.JSON(400, gin.H{"error": "invalid_request"})
		return
	}
	code, err := service.AuthorizeLightcode(c.GetInt("id"), req.ClientId, req.RedirectURI, req.Challenge, req.Method)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	u, _ := url.Parse(req.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", req.State)
	u.RawQuery = q.Encode()
	c.JSON(200, gin.H{"redirect_uri": u.String()})
}

func LightcodeToken(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	var bundle *service.LightcodeTokenBundle
	var err error
	if service.LightcodeEnforce {
		if _, aerr := service.LightcodeCheckUA(c.Request.UserAgent()); aerr != nil {
			lightcodeTokenError(c, aerr)
			return
		}
	}
	switch c.PostForm("grant_type") {
	case "authorization_code":
		att := &service.LightcodeAttestation{DeviceKey: c.PostForm("device_key"), ClientVersion: c.PostForm("client_version"), BindProof: c.PostForm("bind_proof"), AppProof: c.PostForm("app_proof")}
		bundle, err = service.ExchangeLightcodeAttested(c.PostForm("code"), c.PostForm("client_id"), c.PostForm("redirect_uri"), c.PostForm("code_verifier"), common.GetClientIP(c), c.Request.UserAgent(), att)
	case "refresh_token":
		bundle, err = service.RefreshLightcodeAttested(c.PostForm("refresh_token"), c.PostForm("client_id"), &service.LightcodeRefreshProof{Ts: c.PostForm("ts"), Nonce: c.PostForm("nonce"), Proof: c.PostForm("refresh_proof")})
	default:
		c.JSON(400, gin.H{"error": "unsupported_grant_type"})
		return
	}
	var aerr *service.LightcodeAttestError
	if errors.As(err, &aerr) {
		lightcodeTokenError(c, aerr)
		return
	}
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid_grant"})
		return
	}
	c.JSON(200, bundle)
}

func lightcodeTokenError(c *gin.Context, e *service.LightcodeAttestError) {
	body := gin.H{"error": e.Code, "error_description": e.Message}
	if e.ServerTime != 0 {
		body["server_time"] = e.ServerTime
	}
	c.JSON(e.Status, body)
}

func LightcodeRevoke(c *gin.Context) {
	if err := service.RevokeLightcodeToken(c.PostForm("token")); err != nil {
		c.JSON(503, gin.H{"error": "temporarily_unavailable"})
		return
	}
	c.Status(200)
}

func LightcodeAccount(c *gin.Context) {
	subs, err := model.LightcodeSubscriptions(c.GetInt("id"))
	if err != nil {
		c.JSON(503, gin.H{"error": "subscription_unavailable"})
		return
	}
	user, err := model.GetUserCache(c.GetInt("id"))
	if err != nil {
		c.JSON(503, gin.H{"error": "account_unavailable"})
		return
	}
	views := []gin.H{}
	for _, sub := range subs {
		var plan model.SubscriptionPlan
		_ = common.UnmarshalJsonStr(sub.PlanSnapshot, &plan)
		short, long := model.LightcodeWindowStatus(&sub, time.Now().Unix())
		views = append(views, gin.H{"id": sub.Id, "status": sub.Status, "plan": plan.Title, "start_time": sub.StartTime, "end_time": sub.EndTime, "next_reset_time": sub.NextResetTime,
			"window_short": lightcodeWindowJSON(short, plan.WindowShortSeconds), "window_long": lightcodeWindowJSON(long, plan.WindowLongSeconds)})
	}
	c.JSON(200, gin.H{"user": gin.H{"id": user.Id, "name": user.Username}, "subscriptions": views})
}

// lightcodeWindowJSON exposes utilization without disclosing the monetary quota.
func lightcodeWindowJSON(view *model.LightcodeWindowView, seconds int64) gin.H {
	if view == nil {
		return nil
	}
	return gin.H{"seconds": seconds, "used_percent": view.UsedPercent, "reset_at": view.ResetAt}
}

// LightcodeUsage returns the caller's Lightcode token usage and window
// percentages. It never exposes quota, prices or multipliers.
func LightcodeUsage(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	tzOffset := lightcodeTzOffset(c)
	userId := c.GetInt("id")
	now := time.Now().Unix()
	usage, err := model.GetLightcodeUsage(userId, days, tzOffset, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	subscription, err := lightcodeSubscriptionJSON(userId, now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"days": usage.Days, "model_days": usage.ModelDays, "models": usage.Models, "totals": usage.Totals, "subscription": subscription})
}

// lightcodeTzOffset parses tz_offset_minutes (minutes east of UTC), defaulting
// to the server zone when absent or outside [-840, 840].
func lightcodeTzOffset(c *gin.Context) int {
	_, offset := time.Now().Zone()
	tzOffset := offset / 60
	if v := c.Query("tz_offset_minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= -14*60 && n <= 14*60 {
			tzOffset = n
		}
	}
	return tzOffset
}

// lightcodeSubscriptionJSON returns the first active Lightcode subscription as
// plan/end_time/window percentages, or nil when there is none.
func lightcodeSubscriptionJSON(userId int, now int64) (gin.H, error) {
	subs, err := model.LightcodeSubscriptions(userId)
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		if sub.Status != "active" {
			continue
		}
		var plan model.SubscriptionPlan
		_ = common.UnmarshalJsonStr(sub.PlanSnapshot, &plan)
		short, long := model.LightcodeWindowStatus(&sub, now)
		view := gin.H{"plan": plan.Title, "end_time": sub.EndTime,
			"window_short": lightcodeWindowJSON(short, plan.WindowShortSeconds), "window_long": lightcodeWindowJSON(long, plan.WindowLongSeconds),
			"scheduled_reset": nil}
		// A pending operator reset is shown to the user so the cleared windows are expected.
		if pending, err := model.LightcodePendingResetFor(&sub); err == nil && pending != nil {
			view["scheduled_reset"] = gin.H{"run_at": pending.RunAt}
		}
		return view, nil
	}
	return nil, nil
}

// lightcodeResetCardsJSON lists the user's spendable cards without internal ids beyond the
// one needed to spend a card.
func lightcodeResetCardsJSON(userId int) []gin.H {
	cards, err := model.LightcodeUsableResetCards(userId)
	views := make([]gin.H, 0, len(cards))
	if err != nil {
		return views
	}
	for _, card := range cards {
		views = append(views, gin.H{"id": card.Id, "kind": card.Kind, "expires_at": card.ExpiresAt})
	}
	return views
}

// lightcodeAvatarURL is the public, cache-busting avatar path, or empty without an avatar.
func lightcodeAvatarURL(userId int) string {
	version := model.UserAvatarVersion(userId)
	if version == 0 {
		return ""
	}
	return "/api/lightcode/avatar/" + strconv.Itoa(userId) + "?v=" + strconv.FormatInt(version, 10)
}

func buildLightcodeProfile(c *gin.Context) (gin.H, error) {
	userId := c.GetInt("id")
	now := time.Now().Unix()
	user, err := model.GetUserCache(userId)
	if err != nil {
		return nil, err
	}
	displayName, err := model.GetUserDisplayName(userId)
	if err != nil {
		return nil, err
	}
	if displayName == "" {
		displayName = user.Username
	}
	activity, err := model.GetLightcodeActivity(userId, lightcodeTzOffset(c), now)
	if err != nil {
		return nil, err
	}
	subscription, err := lightcodeSubscriptionJSON(userId, now)
	if err != nil {
		return nil, err
	}
	return gin.H{
		"user":         gin.H{"id": user.Id, "username": user.Username, "display_name": displayName, "avatar_url": lightcodeAvatarURL(userId)},
		"subscription": subscription,
		"reset_cards":  lightcodeResetCardsJSON(userId),
		"stats":        activity.Stats,
		"days":         activity.Days,
		"models":       activity.Models,
		"truncated":    activity.Truncated,
	}, nil
}

// LightcodeProfile serves the dashboard activity page (ApiSuccess wrapper).
func LightcodeProfile(c *gin.Context) {
	data, err := buildLightcodeProfile(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

// LightcodeClientProfile serves the same data raw under /lc/v1.
func LightcodeClientProfile(c *gin.Context) {
	data, err := buildLightcodeProfile(c)
	if err != nil {
		c.JSON(503, gin.H{"error": "profile_unavailable"})
		return
	}
	c.JSON(200, data)
}

// orderCatalogNames returns the routable models in the operator's row order, with any routed
// model that lacks a policy row appended so it still appears.
func orderCatalogNames(policies []model.LightcodeGroupModel, routed []string) []string {
	ordered := make([]string, 0, len(routed))
	remaining := slices.Clone(routed)
	for _, policy := range policies {
		if !slices.Contains(remaining, policy.Model) {
			continue
		}
		ordered = append(ordered, policy.Model)
		remaining = slices.DeleteFunc(remaining, func(name string) bool { return name == policy.Model })
	}
	return append(ordered, remaining...)
}

func LightcodeCatalog(c *gin.Context) {
	subs, err := model.LightcodeSubscriptions(c.GetInt("id"))
	if err != nil {
		c.JSON(503, gin.H{"error": "subscription_unavailable"})
		return
	}
	models := []gin.H{}
	for _, sub := range subs {
		if sub.Status != "active" {
			continue
		}
		var policies []model.LightcodeGroupModel
		if err := model.DB.Where("routing_group = ?", sub.RoutingGroup).Order("id asc").Find(&policies).Error; err != nil {
			c.JSON(503, gin.H{"error": "catalog_unavailable"})
			return
		}
		var routed []string
		if err := model.DB.Model(&model.Ability{}).Distinct("model").Where("enabled = ?", true).Where(map[string]interface{}{"group": sub.RoutingGroup}).Pluck("model", &routed).Error; err != nil {
			c.JSON(503, gin.H{"error": "catalog_unavailable"})
			return
		}
		// The operator's row order in Lightcode models is the selector order. Abilities only
		// decide whether a model is routable, so a channel-side reordering never reshuffles
		// what users see. A routed model without a policy row is still listed at the end.
		names := orderCatalogNames(policies, routed)
		for _, name := range names {
			policy := model.LightcodeGroupModel{Model: name, DisplayName: name, Efforts: []string{}}
			for _, p := range policies {
				if p.Model == name {
					policy = p
					break
				}
			}
			_, reason := model.LightcodeModelEntitlement(&sub, name, policies)
			available := reason == ""
			models = append(models, gin.H{"id": name, "display_name": policy.DisplayName, "efforts": policy.Efforts, "default_effort": policy.DefaultEffort, "available": available, "unavailable_reason": reason, "context_window": policy.ContextWindow, "max_output_tokens": policy.MaxOutputTokens, "vision": policy.Vision, "hidden": policy.Hidden})
		}
		break
	}
	c.JSON(200, gin.H{"models": models})
}

func AdminLightcodeModels(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		var models []model.LightcodeGroupModel
		if err := model.DB.Order("routing_group asc, id asc").Find(&models).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, models)
		return
	}
	var req struct {
		Group  string                      `json:"group"`
		Models []model.LightcodeGroupModel `json:"models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Group == "" || req.Group == "auto" || len(req.Models) > 500 {
		common.ApiErrorMsg(c, "Invalid Lightcode model group")
		return
	}
	seen := map[string]bool{}
	for i := range req.Models {
		p := &req.Models[i]
		if p.Model == "" || len(p.Model) > 191 || seen[p.Model] || p.ContextWindow <= 0 || p.MaxOutputTokens <= 0 || p.MaxOutputTokens > p.ContextWindow {
			common.ApiErrorMsg(c, "Invalid model or token limits")
			return
		}
		seen[p.Model] = true
		for _, effort := range p.Efforts {
			if !slices.Contains([]string{"minimal", "low", "medium", "high", "xhigh", "max"}, effort) {
				common.ApiErrorMsg(c, "Invalid reasoning effort")
				return
			}
		}
		if p.DefaultEffort != "" && !slices.Contains(p.Efforts, p.DefaultEffort) {
			common.ApiErrorMsg(c, "Default reasoning effort must be supported")
			return
		}
		if len(p.Efforts) > 0 && p.DefaultEffort == "" {
			common.ApiErrorMsg(c, "Select a default reasoning effort")
			return
		}
		if p.Multiplier == 0 {
			p.Multiplier = 1
		}
		if p.Multiplier < 0.01 || p.Multiplier > 100 {
			common.ApiErrorMsg(c, "Consumption multiplier must be between 0.01 and 100")
			return
		}
		p.Id, p.RoutingGroup = 0, req.Group
		if p.DisplayName == "" {
			p.DisplayName = p.Model
		}
	}
	if err := model.ReplaceLightcodeGroupModels(req.Group, req.Models); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func LightcodeDevices(c *gin.Context) {
	if c.Request.Method == http.MethodDelete {
		if err := model.DB.Model(&model.LightcodeSession{}).Where("id = ? AND user_id = ?", c.Param("id"), c.GetInt("id")).Update("revoked_at", time.Now().Unix()).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, nil)
		return
	}
	var sessions []model.LightcodeSession
	if err := model.DB.Where("user_id = ?", c.GetInt("id")).Order("created_at desc").Limit(100).Find(&sessions).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, sessions)
}
