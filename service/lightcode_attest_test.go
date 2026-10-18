package service

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Lorry-San/nbapi/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	vecSeed   = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	vecPub    = "A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg"
	vecToken  = "lc_at_example.token.value"
	vecSig    = "km-oSZwFg8anLWIuhekCuidn_qbB_wjj6QO4lVdVs8Xzb0PiwtqDBGSumm7nId6ugspCdKQrkJ-dbV2oCsUhAw"
	vecBind   = "QhuVXs4YrZK_j1G00GXWrbwRlY5G04_6uRbCttuLAUD5Iwd1PCByDjMhDeW1x5a3BVub6CUqvSqyKNfb33v5DQ"
	vecApp    = "3ce82ad07292823b6659005afa887dcb00558cdd36bf0ba975ae6735f2e173e0"
	vecNonce  = "AAAAAAAAAAAAAAAAAAAAAA"
	vecTs     = int64(1792300000000)
	vecUA     = "lightcode/0.2.1-alpha.5 (win32; x64)"
	vecTarget = "/lc/v1/chat/completions"
)

func vecKey(t *testing.T) ed25519.PrivateKey {
	seed, err := hex.DecodeString(vecSeed)
	require.NoError(t, err)
	return ed25519.NewKeyFromSeed(seed)
}

func attestSign(key ed25519.PrivateKey, msg string) string { return base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(msg))) }

func useAttestClock(t *testing.T, ms int64) {
	prev, prevSecret, prevEnforce := lightcodeNowMs, LightcodeAppSecret, LightcodeEnforce
	lightcodeNowMs, LightcodeAppSecret, LightcodeEnforce = func() int64 { return ms }, "TEST-SECRET", true
	t.Cleanup(func() { lightcodeNowMs, LightcodeAppSecret, LightcodeEnforce = prev, prevSecret, prevEnforce })
}

func TestLC1VectorReproduces(t *testing.T) {
	key := vecKey(t)
	assert.Equal(t, vecPub, base64.RawURLEncoding.EncodeToString(key.Public().(ed25519.PublicKey)))
	assert.Equal(t, "ad7a65b9ce5ef7c8893f43fed9539ad94d72a80ef75920f321a571d919d9866c", lightcodeHash(vecToken))
	assert.Equal(t, vecSig, attestSign(key, "LC1\nPOST\n"+vecTarget+"\n"+strconv.FormatInt(vecTs, 10)+"\n"+vecNonce+"\n"+lightcodeHash(vecToken)))
	assert.Equal(t, vecBind, attestSign(key, "LC1-BIND\nexample-authorization-code"))
	assert.Equal(t, vecApp, LightcodeAppProof("TEST-SECRET", "0.2.1-alpha.5", vecPub, "example-authorization-code"))

	useAttestClock(t, vecTs)
	s := &model.LightcodeSession{Id: "vec", DeviceKey: vecPub}
	require.Nil(t, VerifyLightcodeRequest(s, "post", vecTarget, vecToken, strconv.FormatInt(vecTs, 10), vecNonce, vecSig))
	att := &LightcodeAttestation{DeviceKey: vecPub, ClientVersion: "0.2.1-alpha.5", BindProof: vecBind, AppProof: vecApp, UAVersion: "0.2.1-alpha.5"}
	require.Nil(t, VerifyLightcodeAttestation(att, "example-authorization-code"))
	att.UAVersion = "0.2.2"
	assert.NotNil(t, VerifyLightcodeAttestation(att, "example-authorization-code"), "UA version must equal client_version")
}

func TestLC1RequestRules(t *testing.T) {
	key := vecKey(t)
	useAttestClock(t, vecTs)
	ts := strconv.FormatInt(vecTs, 10)
	sign := func(target, nonce, ts string) string { return attestSign(key, "LC1\nPOST\n"+target+"\n"+ts+"\n"+nonce+"\n"+lightcodeHash(vecToken)) }
	cases := []struct {
		name, sessionKey, target, nonce, ts string
		status                                int
		code                                  string
	}{
		{"ok", vecPub, vecTarget, "n-ok-000000000000000000", ts, 0, ""},
		{"stale ts", vecPub, vecTarget, "n-stale-0000000000000", strconv.FormatInt(vecTs-300001, 10), 400, "clock_skew"},
		{"future ts", vecPub, vecTarget, "n-future-000000000000", strconv.FormatInt(vecTs+300001, 10), 400, "clock_skew"},
		{"tampered path", vecPub, vecTarget, "n-tamper-00000000000", ts, 401, "invalid_signature"},
		{"legacy session", "", vecTarget, "n-legacy-00000000000", ts, 426, "client_upgrade_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &model.LightcodeSession{Id: "rules", DeviceKey: tc.sessionKey}
			signedTarget := tc.target
			if tc.name == "tampered path" { signedTarget = "/lc/v1/catalog" }
			e := VerifyLightcodeRequest(s, "POST", tc.target, vecToken, tc.ts, tc.nonce, sign(signedTarget, tc.nonce, tc.ts))
			if tc.status == 0 { require.Nil(t, e); return }
			require.NotNil(t, e)
			assert.Equal(t, tc.status, e.Status)
			assert.Equal(t, tc.code, e.Code)
			if tc.code == "clock_skew" { assert.Equal(t, vecTs, e.ServerTime) }
		})
	}
	s := &model.LightcodeSession{Id: "replay", DeviceKey: vecPub}
	require.Nil(t, VerifyLightcodeRequest(s, "POST", vecTarget, vecToken, ts, "n-replay-0000000000000", sign(vecTarget, "n-replay-0000000000000", ts)))
	e := VerifyLightcodeRequest(s, "POST", vecTarget, vecToken, ts, "n-replay-0000000000000", sign(vecTarget, "n-replay-0000000000000", ts))
	require.NotNil(t, e)
	assert.Equal(t, 401, e.Status)
	assert.Equal(t, "invalid_signature", e.Code)
}

func TestLC1UserAgentAndVersions(t *testing.T) {
	prev := LightcodeMinVersion
	t.Cleanup(func() { LightcodeMinVersion = prev })
	LightcodeMinVersion = ""
	for _, ua := range []string{"", "curl/8", "lightcode/0.2.1 (win32; x86)", "lightcode/0.2 (win32; x64)", "Mozilla lightcode/0.2.1 (win32; x64)", "lightcode/0.2.1 (win32; x64) extra"} {
		_, e := LightcodeCheckUA(ua)
		require.NotNil(t, e, ua)
		assert.Equal(t, 426, e.Status)
	}
	v, e := LightcodeCheckUA(vecUA)
	require.Nil(t, e)
	assert.Equal(t, "0.2.1-alpha.5", v)
	assert.Equal(t, []int64{0, 2, 1, 5, 20261019, 4}, LightcodeVersionTokens("0.2.1-alpha.5.20261019.4"))
	LightcodeMinVersion = "0.2.1-alpha.6"
	_, e = LightcodeCheckUA(vecUA)
	require.NotNil(t, e, "below the minimum version")
	assert.Equal(t, "client_upgrade_required", e.Code)
	for _, tc := range []struct{ a, b string; want int }{
		{"0.2.1", "0.2.1", 0}, {"0.2.1-alpha.5", "0.2.1-alpha.6", -1}, {"0.2.10", "0.2.9", 1},
		{"0.3.0", "0.2.99", 1}, {"0.2.1-alpha.5.20261019.4", "0.2.1-alpha.5", 1}, {"1.0.0", "0.9.9.9", 1},
	} {
		assert.Equal(t, tc.want, LightcodeVersionCompare(tc.a, tc.b), tc.a+" vs "+tc.b)
	}
}

func TestLC1ReplayCacheIsBounded(t *testing.T) {
	c := newLightcodeReplayCache(3, time.Minute)
	for i, k := range []string{"a", "b", "c", "d"} { assert.False(t, c.Seen(k, int64(i))) }
	assert.LessOrEqual(t, len(c.seen), 3)
	assert.False(t, c.Seen("a", 10), "oldest entry was evicted")
	assert.True(t, c.Seen("d", 11))
	assert.False(t, c.Seen("d", 11+time.Minute.Milliseconds()+1), "expired entries are forgotten")
}

func setupLC1Session(t *testing.T) (userId int, refresh string) {
	user := setupAuthSessionTestDB(t)
	useTestSessionSecret(t)
	require.NoError(t, model.DB.AutoMigrate(&model.LightcodeAuthCode{}, &model.LightcodeSession{}, &model.LightcodeRefreshToken{}))
	return user.Id, ""
}

func TestLC1RefreshRequiresProofBeforeRotating(t *testing.T) {
	key := vecKey(t)
	useAttestClock(t, vecTs)
	userId, _ := setupLC1Session(t)
	verifier := "lightcode-verifier-that-is-long-enough-for-rfc7636-0123456789"
	sum := sha256.Sum256([]byte(verifier))
	code, err := AuthorizeLightcode(userId, LightcodeClientID, lightcodeTestRedirect, base64.RawURLEncoding.EncodeToString(sum[:]), "S256")
	require.NoError(t, err)
	att := &LightcodeAttestation{DeviceKey: vecPub, ClientVersion: "0.2.1-alpha.5", BindProof: attestSign(key, "LC1-BIND\n"+code), AppProof: LightcodeAppProof("TEST-SECRET", "0.2.1-alpha.5", vecPub, code)}
	_, err = ExchangeLightcodeAttested(code, LightcodeClientID, lightcodeTestRedirect, verifier, "1.1.1.1", "curl/8", att)
	var aerr *LightcodeAttestError
	require.True(t, errors.As(err, &aerr), "bad UA is rejected before the code is consumed")
	assert.Equal(t, 426, aerr.Status)
	bundle, err := ExchangeLightcodeAttested(code, LightcodeClientID, lightcodeTestRedirect, verifier, "1.1.1.1", vecUA, att)
	require.NoError(t, err)
	var stored model.LightcodeSession
	require.NoError(t, model.DB.First(&stored).Error)
	assert.Equal(t, vecPub, stored.DeviceKey)
	assert.Equal(t, "0.2.1-alpha.5", stored.ClientVersion)

	ts := strconv.FormatInt(vecTs, 10)
	_, err = RefreshLightcodeAttested(bundle.RefreshToken, LightcodeClientID, nil)
	require.True(t, errors.As(err, &aerr))
	assert.Equal(t, 426, aerr.Status)
	nonce := "n-refresh-0000000000000"
	_, err = RefreshLightcodeAttested(bundle.RefreshToken, LightcodeClientID, &LightcodeRefreshProof{Ts: ts, Nonce: nonce, Proof: attestSign(key, "LC1-REFRESH\nwrong\n"+ts+"\n"+nonce)})
	require.True(t, errors.As(err, &aerr))
	assert.Equal(t, 401, aerr.Status)
	var tokens int64
	model.DB.Model(&model.LightcodeRefreshToken{}).Count(&tokens)
	assert.EqualValues(t, 1, tokens, "no rotation happened")
	_, err = ValidateLightcodeAccess(bundle.AccessToken)
	require.NoError(t, err, "session was not revoked")

	proof := &LightcodeRefreshProof{Ts: ts, Nonce: nonce, Proof: attestSign(key, "LC1-REFRESH\n"+lightcodeHash(bundle.RefreshToken)+"\n"+ts+"\n"+nonce)}
	next, err := RefreshLightcodeAttested(bundle.RefreshToken, LightcodeClientID, proof)
	require.NoError(t, err)
	assert.NotEqual(t, bundle.RefreshToken, next.RefreshToken)
	_, err = RefreshLightcodeAttested(bundle.RefreshToken, LightcodeClientID, proof)
	require.True(t, errors.As(err, &aerr), "replayed nonce is rejected")
	assert.Equal(t, 401, aerr.Status)

	require.NoError(t, model.DB.Model(&model.LightcodeSession{}).Where("1 = 1").Update("device_key", "").Error)
	_, err = RefreshLightcodeAttested(next.RefreshToken, LightcodeClientID, proof)
	require.True(t, errors.As(err, &aerr), "legacy sessions must sign in again")
	assert.Equal(t, 426, aerr.Status)
}

func TestLC1DeviceCapRevokesLeastRecentlyActive(t *testing.T) {
	userId, _ := setupLC1Session(t)
	now := time.Now().Unix()
	for i := 0; i < 4; i++ {
		require.NoError(t, model.DB.Create(&model.LightcodeSession{Id: "s" + strconv.Itoa(i), UserId: userId, CreatedAt: now, LastActiveAt: now + int64(i), ExpiresAt: now + 1000}).Error)
	}
	ids, err := model.EnforceLightcodeDeviceCap(userId, 2)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"s0", "s1"}, ids)
	var live int64
	model.DB.Model(&model.LightcodeSession{}).Where("revoked_at = 0").Count(&live)
	assert.EqualValues(t, 2, live)
}
