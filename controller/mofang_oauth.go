package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/i18n"
	"github.com/Lorry-San/nbapi/logger"
	"github.com/Lorry-San/nbapi/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

const mofangProviderName = "Mofang"

type mofangSessionRequest struct {
	JWT string `json:"jwt"`
}

type mofangUserInfo struct {
	Id          string
	Email       string
	Username    string
	DisplayName string
}

var mofangUsernamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func MofangOAuthSession(c *gin.Context) {
	if !common.MofangOAuthEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, providerParams(mofangProviderName))
		return
	}

	var req mofangSessionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	jwt := strings.TrimSpace(req.JWT)
	if jwt == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	mofangUser, err := fetchMofangUserInfo(c.Request.Context(), jwt)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("[MofangOAuth] fetch user info failed: %s", err.Error()))
		common.ApiErrorI18n(c, i18n.MsgOAuthGetUserErr)
		return
	}

	user, err := findOrCreateMofangUser(mofangUser, sessions.Default(c))
	if err != nil {
		switch err.(type) {
		case *OAuthRegistrationDisabledError:
			common.ApiErrorI18n(c, i18n.MsgUserRegisterDisabled)
		default:
			common.ApiError(c, err)
		}
		return
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthUserBanned)
		return
	}

	setupLogin(user, c)
}

func MofangOAuthBind(c *gin.Context) {
	if !common.MofangOAuthEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, providerParams(mofangProviderName))
		return
	}

	var req mofangSessionRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	jwt := strings.TrimSpace(req.JWT)
	if jwt == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	mofangUser, err := fetchMofangUserInfo(c.Request.Context(), jwt)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("[MofangOAuth] fetch user info for bind failed: %s", err.Error()))
		common.ApiErrorI18n(c, i18n.MsgOAuthGetUserErr)
		return
	}

	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorI18n(c, i18n.MsgAuthNotLoggedIn)
		return
	}

	user := model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return
	}

	existing := model.User{MofangId: mofangUser.Id}
	if err := existing.FillUserByMofangId(); err == nil {
		if existing.Id != user.Id {
			common.ApiErrorI18n(c, i18n.MsgOAuthAlreadyBound, providerParams(mofangProviderName))
			return
		}
		common.ApiSuccessI18n(c, i18n.MsgOAuthBindSuccess, gin.H{
			"action":    "bind",
			"mofang_id": mofangUser.Id,
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiError(c, err)
		return
	}

	user.MofangId = mofangUser.Id
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccessI18n(c, i18n.MsgOAuthBindSuccess, gin.H{
		"action":    "bind",
		"mofang_id": mofangUser.Id,
	})
}

func fetchMofangUserInfo(parent context.Context, jwt string) (*mofangUserInfo, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(common.MofangApiBase), "/")
	if apiBase == "" {
		return nil, errors.New("mofang api base is not configured")
	}
	if _, err := url.ParseRequestURI(apiBase); err != nil {
		return nil, fmt.Errorf("invalid mofang api base: %w", err)
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/v1/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "JWT "+jwt)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("mofang user api status %d", resp.StatusCode)
	}

	user := parseMofangUserInfo(string(body))
	if user.Email == "" {
		return nil, errors.New("mofang user email is empty")
	}
	if user.Id == "" {
		user.Id = user.Email
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Email
	}

	return user, nil
}

func parseMofangUserInfo(body string) *mofangUserInfo {
	return &mofangUserInfo{
		Id:          firstGJSONString(body, "data.client.id", "client.id", "data.user.id", "user.id", "data.id", "id", "uid", "user_id"),
		Email:       strings.ToLower(firstGJSONString(body, "data.client.email", "client.email", "data.user.email", "user.email", "data.email", "email")),
		Username:    firstGJSONString(body, "data.client.username", "client.username", "data.user.username", "user.username", "data.username", "username", "data.client.name", "client.name", "name"),
		DisplayName: firstGJSONString(body, "data.client.name", "client.name", "data.user.name", "user.name", "data.name", "name", "data.client.company", "client.company", "company"),
	}
}

func firstGJSONString(body string, paths ...string) string {
	for _, path := range paths {
		value := gjson.Get(body, path)
		if !value.Exists() || value.Type == gjson.Null {
			continue
		}
		if value.Type == gjson.Number {
			return strings.TrimSpace(value.Raw)
		}
		text := strings.TrimSpace(value.String())
		if text != "" {
			return text
		}
	}
	return ""
}

func findOrCreateMofangUser(mofangUser *mofangUserInfo, session sessions.Session) (*model.User, error) {
	user := &model.User{}
	created := false
	inviterId := 0

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("mofang_id = ?", mofangUser.Id).First(user).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Where("email = ?", mofangUser.Email).First(user).Error; err == nil {
			if user.MofangId != "" && user.MofangId != mofangUser.Id {
				return fmt.Errorf("email is already bound to another mofang account")
			}
			if user.MofangId == "" {
				if err := tx.Model(user).Update("mofang_id", mofangUser.Id).Error; err != nil {
					return err
				}
				user.MofangId = mofangUser.Id
				if err := model.InvalidateUserCache(user.Id); err != nil {
					common.SysLog(fmt.Sprintf("failed to invalidate mofang-bound user cache for user %d: %s", user.Id, err.Error()))
				}
			}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if !common.RegisterEnabled {
			return &OAuthRegistrationDisabledError{}
		}

		if affCode := session.Get("aff"); affCode != nil {
			if code, ok := affCode.(string); ok && code != "" {
				inviterId, _ = model.GetUserIdByAffCode(code)
			}
		}

		createdUser := model.User{
			Username:    buildMofangUsername(mofangUser),
			DisplayName: truncateMofangField(mofangUser.DisplayName, model.UserNameMaxLength),
			Email:       mofangUser.Email,
			MofangId:    mofangUser.Id,
			Role:        common.RoleCommonUser,
			Status:      common.UserStatusEnabled,
		}
		if createdUser.DisplayName == "" {
			createdUser.DisplayName = createdUser.Username
		}

		if err := createdUser.InsertWithTx(tx, inviterId); err != nil {
			return err
		}
		*user = createdUser
		created = true

		return nil
	})
	if err != nil {
		return nil, err
	}

	if user.Id == 0 {
		return nil, errors.New("failed to resolve mofang user")
	}

	if created {
		user.FinalizeOAuthUserCreation(inviterId)
	}

	return user, nil
}

func buildMofangUsername(user *mofangUserInfo) string {
	base := strings.TrimSpace(user.Username)
	if base == "" && user.Email != "" {
		base = strings.Split(user.Email, "@")[0]
	}
	base = strings.Trim(mofangUsernamePattern.ReplaceAllString(base, "_"), "_-")
	base = truncateMofangField(base, 12)
	if base == "" {
		base = "mofang"
	}

	for i := 0; i < 20; i++ {
		suffix := common.GetRandomString(6)
		candidate := truncateMofangField(base, model.UserNameMaxLength-len(suffix)-1) + "_" + suffix
		exists, err := model.CheckUserExistOrDeleted(candidate, "")
		if err == nil && !exists {
			return candidate
		}
	}

	return "mofang_" + strconv.Itoa(model.GetMaxUserId()+1)
}

func truncateMofangField(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if maxLen <= 0 || len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}
