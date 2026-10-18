package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const LightcodeIssuer = "https://lightcode.lcapi.online"
const LightcodeClientID = "lightcode-desktop"

type LightcodeClaims struct {
	SessionId string `json:"sid"`
	jwt.RegisteredClaims
}

type LightcodeTokenBundle struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int64 `json:"expires_in"`
	Scope string `json:"scope"`
}

func lightcodeHash(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }

func lightcodeRandom() string { return base64.RawURLEncoding.EncodeToString(randBytes()) }
func randBytes() []byte { data := make([]byte, 32); if _, err := rand.Read(data); err != nil { panic(err) }; return data }

func ValidateLightcodeRedirect(clientId, redirect string) error {
	u, err := url.Parse(redirect)
	if err != nil || clientId != LightcodeClientID || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" && u.Hostname() != "localhost") || u.Port() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/oauth/lightcode/callback" { return errors.New("invalid Lightcode redirect URI") }
	port, err := strconv.Atoi(u.Port()); if err != nil || port < 1 || port > 65535 { return errors.New("invalid loopback port") }
	return nil
}

func AuthorizeLightcode(userId int, clientId, redirect, challenge, method string) (string, error) {
	if err := ValidateLightcodeRedirect(clientId, redirect); err != nil { return "", err }
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(decoded) != 32 || method != "S256" { return "", errors.New("PKCE S256 is required") }
	code := lightcodeRandom()
	err = model.DB.Create(&model.LightcodeAuthCode{Hash: lightcodeHash(code), UserId: userId, ClientId: clientId, RedirectURI: redirect, Challenge: challenge, ExpiresAt: time.Now().Add(2*time.Minute).Unix()}).Error
	return code, err
}

func issueLightcodeBundle(session *model.LightcodeSession, refresh string, issuedAt int64) (*LightcodeTokenBundle, error) {
	claims := LightcodeClaims{SessionId: session.Id, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: LightcodeIssuer, Subject: strconv.Itoa(session.UserId), Audience: jwt.ClaimStrings{"lightcode"},
		IssuedAt: jwt.NewNumericDate(time.Unix(issuedAt, 0)), ExpiresAt: jwt.NewNumericDate(time.Unix(issuedAt+1800, 0)),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "lightcode-at+jwt"
	access, err := token.SignedString(authSigningKey("lightcode-access"))
	if err != nil { return nil, err }
	return &LightcodeTokenBundle{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: 1800, Scope: "lightcode"}, nil
}

func ExchangeLightcodeGrant(code, clientId, redirect, verifier, ip, userAgent string) (*LightcodeTokenBundle, error) {
	return exchangeLightcode(code, clientId, redirect, verifier, ip, userAgent, nil)
}

// ExchangeLightcodeAttested verifies the LC1 sign-in proofs (unless enforcement is off) and binds the device key to the new session.
func ExchangeLightcodeAttested(code, clientId, redirect, verifier, ip, userAgent string, att *LightcodeAttestation) (*LightcodeTokenBundle, error) {
	if LightcodeEnforce {
		if att == nil { return nil, upgradeErr() }
		version, aerr := LightcodeCheckUA(userAgent)
		if aerr != nil { return nil, aerr }
		att.UAVersion = version
		if aerr := VerifyLightcodeAttestation(att, code); aerr != nil { return nil, aerr }
	} else if att != nil && (att.DeviceKey == "" || att.BindProof == "") { att = nil }
	return exchangeLightcode(code, clientId, redirect, verifier, ip, userAgent, att)
}

func exchangeLightcode(code, clientId, redirect, verifier, ip, userAgent string, att *LightcodeAttestation) (*LightcodeTokenBundle, error) {
	if err := ValidateLightcodeRedirect(clientId, redirect); err != nil { return nil, model.ErrLightcodeGrant }
	if len(verifier) < 43 || len(verifier) > 128 || strings.ContainsAny(verifier, " \r\n\t") { return nil, model.ErrLightcodeGrant }
	sum := sha256.Sum256([]byte(verifier))
	now := time.Now().Unix()
	refresh := "lc_rt_"+lightcodeRandom()
	session := &model.LightcodeSession{Id: lightcodeRandom(), ClientId: clientId, IP: truncateAuthMetadata(ip, 64), UserAgent: truncateAuthMetadata(userAgent, 512), CreatedAt: now, LastActiveAt: now, ExpiresAt: now+60*86400}
	if att != nil { session.DeviceKey, session.ClientVersion = att.DeviceKey, att.ClientVersion }
	if err := model.ExchangeLightcodeCode(lightcodeHash(code), clientId, redirect, base64.RawURLEncoding.EncodeToString(sum[:]), lightcodeHash(refresh), session); err != nil { return nil, err }
	if LightcodeEnforce {
		if ids, err := model.EnforceLightcodeDeviceCap(session.UserId, LightcodeMaxDevices); err == nil {
			lightcodeIPs.Lock(); for _, id := range ids { delete(lightcodeIPs.m, id) }; lightcodeIPs.Unlock()
		}
	}
	return issueLightcodeBundle(session, refresh, now)
}

// RefreshLightcodeAttested verifies the device signature BEFORE rotating, so a stolen refresh token cannot rotate or trigger replay revocation.
func RefreshLightcodeAttested(refresh, clientId string, proof *LightcodeRefreshProof) (*LightcodeTokenBundle, error) {
	if LightcodeEnforce && clientId == LightcodeClientID && strings.HasPrefix(refresh, "lc_rt_") && len(refresh) <= 256 {
		if session, err := model.LightcodeSessionByRefresh(lightcodeHash(refresh)); err == nil {
			if aerr := VerifyLightcodeRefresh(session, refresh, proof); aerr != nil { return nil, aerr }
		}
	}
	return RefreshLightcodeGrant(refresh, clientId)
}

func RefreshLightcodeGrant(refresh, clientId string) (*LightcodeTokenBundle, error) {
	if clientId != LightcodeClientID || !strings.HasPrefix(refresh, "lc_rt_") || len(refresh) > 256 { return nil, model.ErrLightcodeGrant }
	mac := hmac.New(sha256.New, authSigningKey("lightcode-refresh-rotation"))
	mac.Write([]byte(refresh))
	next := "lc_rt_"+base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	session, issuedAt, err := model.RotateLightcodeRefresh(lightcodeHash(refresh), lightcodeHash(next), clientId)
	if err != nil { return nil, err }
	return issueLightcodeBundle(session, next, issuedAt)
}

func ValidateLightcodeAccess(raw string) (*model.LightcodeSession, error) {
	claims := &LightcodeClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Header["typ"] != "lightcode-at+jwt" { return nil, model.ErrLightcodeGrant }
		return authSigningKey("lightcode-access"), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(LightcodeIssuer), jwt.WithAudience("lightcode"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid { return nil, model.ErrLightcodeGrant }
	var session model.LightcodeSession
	if err := model.DB.First(&session, "id = ?", claims.SessionId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, model.ErrLightcodeGrant }
		return nil, err
	}
	if session.RevokedAt != 0 || session.ExpiresAt <= time.Now().Unix() || claims.Subject != strconv.Itoa(session.UserId) { return nil, model.ErrLightcodeGrant }
	user, err := model.GetUserCache(session.UserId)
	if err != nil { return nil, err }
	if user.Status != common.UserStatusEnabled || user.AuthVersion != session.AuthVersion { return nil, model.ErrLightcodeGrant }
	return &session, nil
}

func RevokeLightcodeToken(raw string) error { return model.RevokeLightcodeRefresh(lightcodeHash(raw)) }
