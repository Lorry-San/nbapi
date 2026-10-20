package controller

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/gin-gonic/gin"
)

// lightcodeReply writes one result in the envelope each surface expects: the dashboard wraps
// replies in ApiSuccess, while /lc/v1 clients read raw JSON with an error object.
func lightcodeReply(c *gin.Context, raw bool, status int, data gin.H, err error) {
	if err != nil {
		if raw {
			c.JSON(status, gin.H{"error": gin.H{"type": "lightcode_error", "code": "request_failed", "message": err.Error()}})
			return
		}
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if raw {
		c.JSON(http.StatusOK, data)
		return
	}
	common.ApiSuccess(c, data)
}

func useResetCard(c *gin.Context, raw bool) {
	cardId, _ := strconv.Atoi(c.Param("id"))
	if cardId <= 0 {
		lightcodeReply(c, raw, http.StatusBadRequest, nil, model.ErrResetCardNotFound)
		return
	}
	card, err := model.UseLightcodeResetCard(c.GetInt("id"), cardId)
	if err != nil {
		lightcodeReply(c, raw, http.StatusConflict, nil, err)
		return
	}
	lightcodeReply(c, raw, http.StatusOK, gin.H{"kind": card.Kind, "used_at": card.UsedAt}, nil)
}

// LightcodeUseResetCard spends a reset card from the dashboard.
func LightcodeUseResetCard(c *gin.Context) { useResetCard(c, false) }

// LightcodeClientUseResetCard spends a reset card from the desktop client.
func LightcodeClientUseResetCard(c *gin.Context) { useResetCard(c, true) }

func updateDisplayName(c *gin.Context, raw bool) {
	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		lightcodeReply(c, raw, http.StatusBadRequest, nil, model.ErrDisplayNameInvalid)
		return
	}
	name, err := model.UpdateUserDisplayName(c.GetInt("id"), req.DisplayName)
	if err != nil {
		lightcodeReply(c, raw, http.StatusBadRequest, nil, err)
		return
	}
	lightcodeReply(c, raw, http.StatusOK, gin.H{"display_name": name}, nil)
}

// LightcodeUpdateDisplayName edits the display name from the dashboard.
func LightcodeUpdateDisplayName(c *gin.Context) { updateDisplayName(c, false) }

// LightcodeClientUpdateDisplayName edits the display name from the desktop client.
func LightcodeClientUpdateDisplayName(c *gin.Context) { updateDisplayName(c, true) }

var errAvatarType = errors.New("头像仅支持 PNG、JPEG、WebP 或 GIF")

// avatarContentType sniffs the bytes instead of trusting the client-declared type, so a
// renamed file cannot be served under an image content type.
func avatarContentType(data []byte) (string, error) {
	switch http.DetectContentType(data) {
	case "image/png":
		return "image/png", nil
	case "image/jpeg":
		return "image/jpeg", nil
	case "image/gif":
		return "image/gif", nil
	case "image/webp":
		return "image/webp", nil
	}
	return "", errAvatarType
}

func uploadAvatar(c *gin.Context, raw bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, model.LightcodeAvatarMaxBytes+64*1024)
	var data []byte
	if file, _, err := c.Request.FormFile("avatar"); err == nil {
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, model.LightcodeAvatarMaxBytes+1))
		if err != nil {
			lightcodeReply(c, raw, http.StatusBadRequest, nil, err)
			return
		}
	} else {
		// The desktop client sends the image as the raw request body.
		body, readErr := io.ReadAll(io.LimitReader(c.Request.Body, model.LightcodeAvatarMaxBytes+1))
		if readErr != nil {
			lightcodeReply(c, raw, http.StatusBadRequest, nil, readErr)
			return
		}
		data = body
	}
	contentType, err := avatarContentType(data)
	if err != nil {
		lightcodeReply(c, raw, http.StatusBadRequest, nil, err)
		return
	}
	userId := c.GetInt("id")
	if _, err := model.SaveUserAvatar(userId, contentType, data); err != nil {
		lightcodeReply(c, raw, http.StatusBadRequest, nil, err)
		return
	}
	lightcodeReply(c, raw, http.StatusOK, gin.H{"avatar_url": lightcodeAvatarURL(userId)}, nil)
}

// LightcodeUploadAvatar stores an avatar uploaded from the dashboard.
func LightcodeUploadAvatar(c *gin.Context) { uploadAvatar(c, false) }

// LightcodeClientUploadAvatar stores an avatar uploaded from the desktop client.
func LightcodeClientUploadAvatar(c *gin.Context) { uploadAvatar(c, true) }

// LightcodeDeleteAvatar removes the caller's avatar.
func LightcodeDeleteAvatar(c *gin.Context) {
	if err := model.DeleteUserAvatar(c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// LightcodeAvatar serves a user's avatar. Avatars are profile pictures, so they are public by
// user id the way any profile image is; the version query only busts caches.
func LightcodeAvatar(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		c.Status(http.StatusNotFound)
		return
	}
	avatar, err := model.GetUserAvatar(userId)
	if err != nil || avatar == nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, int64(len(avatar.Data)), avatar.ContentType, bytes.NewReader(avatar.Data), nil)
}

// AdminGiftResetCards gives cards to every subscriber the target reaches.
func AdminGiftResetCards(c *gin.Context) {
	var req struct {
		TargetType string `json:"target_type"`
		TargetId   int    `json:"target_id"`
		Kind       string `json:"kind"`
		Count      int    `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}
	users, err := model.GiftLightcodeResetCards(req.TargetType, req.TargetId, req.Kind, req.Count)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.reset_card_gift", map[string]interface{}{
		"target_type": req.TargetType, "target_id": req.TargetId, "kind": req.Kind, "count": req.Count, "user_count": users,
	})
	common.ApiSuccess(c, gin.H{"user_count": users, "card_count": users * req.Count})
}

// AdminScheduleReset records a reset to run at a future time.
func AdminScheduleReset(c *gin.Context) {
	var req struct {
		TargetType       string `json:"target_type"`
		TargetId         int    `json:"target_id"`
		RunAt            int64  `json:"run_at"`
		AdvanceResetTime *bool  `json:"advance_reset_time"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	schedule, err := model.ScheduleLightcodeReset(req.TargetType, req.TargetId, req.RunAt, resolveAdvanceResetTime(req.AdvanceResetTime))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.reset_schedule", map[string]interface{}{
		"target_type": schedule.TargetType, "target_id": schedule.TargetId, "run_at": schedule.RunAt,
	})
	common.ApiSuccess(c, schedule)
}

// AdminListScheduledResets lists recent schedules for the reset dialog.
func AdminListScheduledResets(c *gin.Context) {
	schedules, err := model.ListLightcodeScheduledResets(50)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, schedules)
}

// AdminCancelScheduledReset withdraws a schedule that has not run.
func AdminCancelScheduledReset(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	if err := model.CancelLightcodeScheduledReset(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "subscription.reset_schedule_cancel", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}
