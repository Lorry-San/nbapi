package model

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// LightcodeAvatarMaxBytes caps an uploaded avatar; it is stored inline, so keep it small.
const LightcodeAvatarMaxBytes = 256 * 1024

// LightcodeDisplayNameMax matches the users.display_name validation limit.
const LightcodeDisplayNameMax = 20

var ErrDisplayNameInvalid = errors.New("显示名长度需在 1 到 20 个字符之间")

// UserAvatar stores one avatar per user apart from the users table, so the large blob never
// rides along with user caches or login payloads.
type UserAvatar struct {
	UserId      int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	ContentType string `json:"content_type" gorm:"type:varchar(32)"`
	Data        []byte `json:"-"`
	// Version changes on every upload so clients can cache by URL and still see new images.
	Version   int64 `json:"version" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

// SaveUserAvatar replaces the user's avatar.
func SaveUserAvatar(userId int, contentType string, data []byte) (int64, error) {
	if len(data) == 0 || len(data) > LightcodeAvatarMaxBytes {
		return 0, errors.New("头像大小需在 256KB 以内")
	}
	now := GetDBTimestamp()
	avatar := UserAvatar{UserId: userId, ContentType: contentType, Data: data, Version: now, UpdatedAt: now}
	if err := DB.Save(&avatar).Error; err != nil {
		return 0, err
	}
	return now, nil
}

// GetUserAvatar returns the stored avatar, or nil when the user has none.
func GetUserAvatar(userId int) (*UserAvatar, error) {
	var avatar UserAvatar
	result := DB.Where("user_id = ?", userId).Limit(1).Find(&avatar)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &avatar, nil
}

// UserAvatarVersion reports the avatar version without loading the image, or 0 for none.
func UserAvatarVersion(userId int) int64 {
	var versions []int64
	DB.Model(&UserAvatar{}).Where("user_id = ?", userId).Limit(1).Pluck("version", &versions)
	if len(versions) == 0 {
		return 0
	}
	return versions[0]
}

// DeleteUserAvatar removes the user's avatar.
func DeleteUserAvatar(userId int) error {
	return DB.Where("user_id = ?", userId).Delete(&UserAvatar{}).Error
}

// UpdateUserDisplayName changes only the display name and refreshes the user cache.
func UpdateUserDisplayName(userId int, name string) (string, error) {
	name = strings.TrimSpace(name)
	length := utf8.RuneCountInString(name)
	if length == 0 || length > LightcodeDisplayNameMax {
		return "", ErrDisplayNameInvalid
	}
	if err := DB.Model(&User{}).Where("id = ?", userId).Update("display_name", name).Error; err != nil {
		return "", err
	}
	_ = invalidateUserCache(userId)
	return name, nil
}
