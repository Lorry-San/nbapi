package middleware

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Lorry-San/nbapi/model"
	"github.com/Lorry-San/nbapi/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lcPub   = "A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg"
	lcToken = "lc_at_example.token.value"
	lcUA    = "lightcode/0.2.1-alpha.5 (win32; x64)"
)

func timeNowMs() int64 { return time.Now().UnixMilli() }

func lcKey(t *testing.T) ed25519.PrivateKey {
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	require.NoError(t, err)
	return ed25519.NewKeyFromSeed(seed)
}

func lcEngine(deviceKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(LightcodeClientGuard(), func(c *gin.Context) { c.Set("lightcode_session", &model.LightcodeSession{Id: "mw-session", DeviceKey: deviceKey}) }, LightcodeRequestSignature())
	e.POST("/lc/v1/chat/completions", func(c *gin.Context) { c.Status(200) })
	return e
}

func lcSend(t *testing.T, e *gin.Engine, ua, target string, signedTarget, ts, nonce string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Authorization", "Bearer "+lcToken)
	req.Header.Set("X-LC-Ts", ts)
	req.Header.Set("X-LC-Nonce", nonce)
	req.Header.Set("X-LC-Sig", base64.RawURLEncoding.EncodeToString(ed25519.Sign(lcKey(t), []byte("LC1\nPOST\n"+signedTarget+"\n"+ts+"\n"+nonce+"\nad7a65b9ce5ef7c8893f43fed9539ad94d72a80ef75920f321a571d919d9866c"))))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestLightcodeClientGuardAndSignatureMiddleware(t *testing.T) {
	prev := service.LightcodeEnforce
	t.Cleanup(func() { service.LightcodeEnforce = prev })
	service.LightcodeEnforce = true
	now := strconv.FormatInt(timeNowMs(), 10)
	stale := strconv.FormatInt(timeNowMs()-400000, 10)
	path := "/lc/v1/chat/completions?x=1"

	rec := lcSend(t, lcEngine(lcPub), lcUA, path, path, now, "mw-nonce-ok-00000000000")
	assert.Equal(t, 200, rec.Code, rec.Body.String())

	rec = lcSend(t, lcEngine(lcPub), "python-requests/2", path, path, now, "mw-nonce-ua-0000000000")
	assert.Equal(t, 426, rec.Code)
	assert.Contains(t, rec.Body.String(), "client_upgrade_required")

	rec = lcSend(t, lcEngine(lcPub), lcUA, path, path, stale, "mw-nonce-stale-00000000")
	assert.Equal(t, 400, rec.Code)
	assert.Contains(t, rec.Body.String(), "clock_skew")
	assert.Contains(t, rec.Body.String(), "server_time")

	e := lcEngine(lcPub)
	require.Equal(t, 200, lcSend(t, e, lcUA, path, path, now, "mw-nonce-replay-0000000").Code)
	rec = lcSend(t, e, lcUA, path, path, now, "mw-nonce-replay-0000000")
	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_signature")

	rec = lcSend(t, lcEngine(lcPub), lcUA, path, "/lc/v1/chat/completions?x=2", now, "mw-nonce-tamper-000000")
	assert.Equal(t, 401, rec.Code)

	rec = lcSend(t, lcEngine(""), lcUA, path, path, now, "mw-nonce-legacy-000000")
	assert.Equal(t, 426, rec.Code)
	assert.Contains(t, rec.Body.String(), "client_upgrade_required")

	service.LightcodeEnforce = false
	assert.Equal(t, 200, lcSend(t, lcEngine(""), "curl/8", path, path, "0", "x").Code, "emergency switch disables the checks")
}
