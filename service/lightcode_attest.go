package service

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
)

// LC1 client attestation, see LC1-protocol.md. Settings are read from the environment once at startup.
var (
	LightcodeEnforce       = os.Getenv("LIGHTCODE_ENFORCE_CLIENT") != "0"
	LightcodeAppSecret     = os.Getenv("LIGHTCODE_APP_SECRET")
	LightcodeMinVersion    = strings.TrimSpace(os.Getenv("LIGHTCODE_MIN_CLIENT_VERSION"))
	LightcodeMaxDevices    = lightcodeEnvInt("LIGHTCODE_MAX_DEVICES", 5)
	LightcodeMaxSessionIPs = lightcodeEnvInt("LIGHTCODE_MAX_SESSION_IPS_PER_HOUR", 8)
	lightcodeNowMs         = func() int64 { return time.Now().UnixMilli() }
	lightcodeUARe          = regexp.MustCompile(`^lightcode/(\d+\.\d+\.\d+(?:[-.][0-9A-Za-z.]+)*) \((win32|darwin|linux); (x64|arm64)\)$`)
	lightcodeDigits        = regexp.MustCompile(`\d+`)
	lightcodeReplay        = newLightcodeReplayCache(200000, 10*time.Minute)
)

const lightcodeSkewMs = 300000

func init() {
	if LightcodeEnforce && LightcodeAppSecret == "" {
		common.SysLog("WARNING: LIGHTCODE_APP_SECRET is not set; Lightcode code exchange will fail closed")
	}
}

func lightcodeEnvInt(name string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); err == nil && n > 0 { return n }
	return def
}

type LightcodeAttestError struct {
	Status     int
	Code       string
	Message    string
	ServerTime int64
}

func (e *LightcodeAttestError) Error() string { return e.Code }

func attestErr(status int, code, message string) *LightcodeAttestError { return &LightcodeAttestError{Status: status, Code: code, Message: message} }
func upgradeErr() *LightcodeAttestError { return attestErr(426, "client_upgrade_required", "请升级到最新版 Lightcode 客户端并重新登录") }
func signatureErr() *LightcodeAttestError { return attestErr(401, "invalid_signature", "请求签名无效") }

type LightcodeAttestation struct{ DeviceKey, ClientVersion, BindProof, AppProof, UAVersion string }
type LightcodeRefreshProof struct{ Ts, Nonce, Proof string }

// LightcodeVersionTokens turns "0.2.1-alpha.5" into its digit runs [0 2 1 5].
func LightcodeVersionTokens(v string) []int64 {
	var out []int64
	for _, d := range lightcodeDigits.FindAllString(v, -1) {
		n, err := strconv.ParseInt(d, 10, 64)
		if err != nil { n = math.MaxInt64 }
		out = append(out, n)
	}
	return out
}

// LightcodeVersionCompare compares digit tokens, treating missing tokens as zero.
func LightcodeVersionCompare(a, b string) int {
	x, y := LightcodeVersionTokens(a), LightcodeVersionTokens(b)
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int64
		if i < len(x) { p = x[i] }; if i < len(y) { q = y[i] }
		if p != q { if p < q { return -1 }; return 1 }
	}
	return 0
}

// LightcodeCheckUA validates the User-Agent shape and the configured minimum version, returning the client version.
func LightcodeCheckUA(ua string) (string, *LightcodeAttestError) {
	m := lightcodeUARe.FindStringSubmatch(ua)
	if m == nil { return "", upgradeErr() }
	if LightcodeMinVersion != "" && LightcodeVersionCompare(m[1], LightcodeMinVersion) < 0 { return "", upgradeErr() }
	return m[1], nil
}

func lightcodeVerifyEd(keyB64, sigB64, msg string) bool {
	key, err := base64.RawURLEncoding.DecodeString(keyB64)
	if err != nil || len(key) != ed25519.PublicKeySize { return false }
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil || len(sig) != ed25519.SignatureSize { return false }
	return ed25519.Verify(ed25519.PublicKey(key), []byte(msg), sig)
}

func LightcodeAppProof(secret, version, deviceKey, code string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("LC1-APP\n" + version + "\n" + deviceKey + "\n" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyLightcodeAttestation checks the sign-in proofs before the authorization code is consumed.
func VerifyLightcodeAttestation(a *LightcodeAttestation, code string) *LightcodeAttestError {
	if LightcodeAppSecret == "" { return attestErr(503, "attestation_unavailable", "服务暂时不可用，请稍后重试") }
	if a == nil || a.DeviceKey == "" || a.ClientVersion == "" || a.BindProof == "" || a.AppProof == "" || len(a.ClientVersion) > 64 || a.ClientVersion != a.UAVersion { return upgradeErr() }
	if key, err := base64.RawURLEncoding.DecodeString(a.DeviceKey); err != nil || len(key) != ed25519.PublicKeySize { return signatureErr() }
	if !lightcodeVerifyEd(a.DeviceKey, a.BindProof, "LC1-BIND\n"+code) { return signatureErr() }
	got, err := hex.DecodeString(a.AppProof)
	want, _ := hex.DecodeString(LightcodeAppProof(LightcodeAppSecret, a.ClientVersion, a.DeviceKey, code))
	if err != nil || !hmac.Equal(got, want) { return signatureErr() }
	return nil
}

// verifyLightcodeSigned applies freshness, signature and replay rules shared by requests and refreshes.
func verifyLightcodeSigned(s *model.LightcodeSession, msg, ts, nonce, sig string) *LightcodeAttestError {
	if s.DeviceKey == "" || ts == "" || nonce == "" || sig == "" { return upgradeErr() }
	sent, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || len(nonce) > 64 { return signatureErr() }
	now := lightcodeNowMs()
	if sent > now+lightcodeSkewMs || sent < now-lightcodeSkewMs {
		e := attestErr(400, "clock_skew", "设备时间与服务器相差过大，请校准系统时间")
		e.ServerTime = now
		return e
	}
	if !lightcodeVerifyEd(s.DeviceKey, sig, msg) { return signatureErr() }
	if lightcodeReplay.Seen(s.Id+"\n"+nonce, now) { return signatureErr() }
	return nil
}

func VerifyLightcodeRequest(s *model.LightcodeSession, method, target, token, ts, nonce, sig string) *LightcodeAttestError {
	return verifyLightcodeSigned(s, "LC1\n"+strings.ToUpper(method)+"\n"+target+"\n"+ts+"\n"+nonce+"\n"+lightcodeHash(token), ts, nonce, sig)
}

func VerifyLightcodeRefresh(s *model.LightcodeSession, refresh string, p *LightcodeRefreshProof) *LightcodeAttestError {
	if p == nil { p = &LightcodeRefreshProof{} }
	return verifyLightcodeSigned(s, "LC1-REFRESH\n"+lightcodeHash(refresh)+"\n"+p.Ts+"\n"+p.Nonce, p.Ts, p.Nonce, p.Proof)
}

type lightcodeReplayEntry struct { key string; exp int64 }

// lightcodeReplayCache remembers (session, nonce) pairs in memory only; it resets on restart.
type lightcodeReplayCache struct {
	mu    sync.Mutex
	seen  map[string]int64
	order []lightcodeReplayEntry
	head  int
	max   int
	ttl   int64
}

func newLightcodeReplayCache(max int, ttl time.Duration) *lightcodeReplayCache {
	return &lightcodeReplayCache{seen: map[string]int64{}, max: max, ttl: ttl.Milliseconds()}
}

// Seen reports whether key was already used; otherwise it records it. Expired entries go first, then the oldest.
func (r *lightcodeReplayCache) Seen(key string, now int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if exp, ok := r.seen[key]; ok && exp > now { return true }
	for r.head < len(r.order) {
		e := r.order[r.head]
		if e.exp > now && len(r.seen) < r.max { break }
		if r.seen[e.key] == e.exp { delete(r.seen, e.key) }
		r.head++
	}
	if r.head > 4096 && r.head*2 > len(r.order) { r.order = append([]lightcodeReplayEntry(nil), r.order[r.head:]...); r.head = 0 }
	exp := now + r.ttl
	r.seen[key] = exp
	r.order = append(r.order, lightcodeReplayEntry{key, exp})
	return false
}

// Per-session IP tracking is in memory and resets on restart.
var lightcodeIPs = struct { sync.Mutex; m map[string]map[string]int64 }{m: map[string]map[string]int64{}}

// LightcodeRecordIP notes the session's source IP and returns its distinct IP count over the last hour.
func LightcodeRecordIP(sessionId, ip string) int {
	now := lightcodeNowMs()
	cutoff := now - 3600000
	lightcodeIPs.Lock()
	defer lightcodeIPs.Unlock()
	if len(lightcodeIPs.m) > 50000 {
		for id, ips := range lightcodeIPs.m {
			live := false
			for _, seen := range ips { if seen > cutoff { live = true; break } }
			if !live { delete(lightcodeIPs.m, id) }
		}
	}
	ips := lightcodeIPs.m[sessionId]
	if ips == nil { ips = map[string]int64{}; lightcodeIPs.m[sessionId] = ips }
	ips[ip] = now
	for k, seen := range ips { if seen <= cutoff { delete(ips, k) } }
	return len(ips)
}

func LightcodeRevokeSession(sessionId string) error {
	lightcodeIPs.Lock(); delete(lightcodeIPs.m, sessionId); lightcodeIPs.Unlock()
	return model.RevokeLightcodeSession(sessionId)
}
