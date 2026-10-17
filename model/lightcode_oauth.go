package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrLightcodeGrant = errors.New("invalid_grant")

type LightcodeAuthCode struct {
	Hash string `gorm:"primaryKey;type:varchar(64)"`
	UserId int
	ClientId string `gorm:"type:varchar(64)"`
	RedirectURI string `gorm:"type:text"`
	Challenge string `gorm:"type:varchar(128)"`
	ExpiresAt int64
	UsedAt int64
}

type LightcodeSession struct {
	Id string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId int `json:"-" gorm:"index"`
	ClientId string `json:"client_id" gorm:"type:varchar(64)"`
	AuthVersion int64 `json:"-"`
	UserAgent string `json:"device" gorm:"type:varchar(512)"`
	IP string `json:"ip" gorm:"type:varchar(64)"`
	CreatedAt int64 `json:"created_at"`
	LastActiveAt int64 `json:"last_active_at"`
	ExpiresAt int64 `json:"expires_at"`
	RevokedAt int64 `json:"revoked_at"`
}

type LightcodeRefreshToken struct {
	Hash string `gorm:"primaryKey;type:varchar(64)"`
	SessionId string `gorm:"type:varchar(64);index"`
	CreatedAt int64
	UsedAt int64
}

// ExchangeLightcodeCode consumes a code and persists its device session atomically.
func ExchangeLightcodeCode(hash, clientId, redirect, challenge, refreshHash string, session *LightcodeSession) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var code LightcodeAuthCode
		if err := lockForUpdate(tx).First(&code, "hash = ?", hash).Error; err != nil { return ErrLightcodeGrant }
		now := time.Now().Unix()
		if code.UsedAt != 0 || code.ExpiresAt <= now || code.ClientId != clientId || code.RedirectURI != redirect || code.Challenge != challenge { return ErrLightcodeGrant }
		res := tx.Model(&LightcodeAuthCode{}).Where("hash = ? AND used_at = 0", hash).Update("used_at", now)
		if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrLightcodeGrant }
		var user User
		if err := tx.First(&user, code.UserId).Error; err != nil { return err }
		if user.Status != 1 { return ErrLightcodeGrant }
		session.UserId, session.AuthVersion = user.Id, user.AuthVersion
		if err := tx.Create(session).Error; err != nil { return err }
		return tx.Create(&LightcodeRefreshToken{Hash: refreshHash, SessionId: session.Id, CreatedAt: now}).Error
	})
}

// RotateLightcodeRefresh commits revocation even when replay is rejected.
func RotateLightcodeRefresh(hash, nextHash, clientId string) (*LightcodeSession, int64, error) {
	var session LightcodeSession
	var issuedAt int64
	var rejected bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var token LightcodeRefreshToken
		if err := tx.First(&token, "hash = ?", hash).Error; err != nil { return ErrLightcodeGrant }
		if err := tx.Model(&LightcodeSession{}).Where("id = ?", token.SessionId).UpdateColumn("last_active_at", gorm.Expr("last_active_at")).Error; err != nil { return err }
		if err := lockForUpdate(tx).First(&session, "id = ?", token.SessionId).Error; err != nil { return ErrLightcodeGrant }
		if err := tx.First(&token, "hash = ?", hash).Error; err != nil { return err }
		now := time.Now().Unix()
		if session.ClientId != clientId || session.RevokedAt != 0 || session.ExpiresAt <= now || session.LastActiveAt <= now-30*86400 { return ErrLightcodeGrant }
		var user User
		if err := tx.First(&user, session.UserId).Error; err != nil { return err }
		if user.Status != 1 || user.AuthVersion != session.AuthVersion { return ErrLightcodeGrant }
		if token.UsedAt != 0 {
			var next LightcodeRefreshToken
			if now-token.UsedAt <= 10 && tx.First(&next, "hash = ?", nextHash).Error == nil && next.UsedAt == 0 {
				issuedAt = token.UsedAt
				return nil
			}
			rejected = true
			return tx.Model(&session).Update("revoked_at", now).Error
		}
		issuedAt = now
		if err := tx.Model(&token).Update("used_at", now).Error; err != nil { return err }
		if err := tx.Create(&LightcodeRefreshToken{Hash: nextHash, SessionId: session.Id, CreatedAt: now}).Error; err != nil { return err }
		return tx.Model(&session).Update("last_active_at", now).Error
	})
	if err != nil { return nil, 0, err }; if rejected { return nil, 0, ErrLightcodeGrant }
	return &session, issuedAt, nil
}

func RevokeLightcodeRefresh(hash string) error {
	var token LightcodeRefreshToken
	if err := DB.First(&token, "hash = ?", hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil }; return err
	}
	return DB.Model(&LightcodeSession{}).Where("id = ?", token.SessionId).Update("revoked_at", time.Now().Unix()).Error
}
