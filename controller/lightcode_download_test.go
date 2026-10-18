package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func writeInstallerFile(t *testing.T, dir, name, body string, age time.Duration) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	stamp := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(path, stamp, stamp))
}

func TestFindLightcodeInstallerPicksNewestMatchingFile(t *testing.T) {
	dir := t.TempDir()
	writeInstallerFile(t, dir, "Lightcode-0.2.0-win-x64.exe", "old", 2*time.Hour)
	writeInstallerFile(t, dir, "Lightcode-0.2.1-alpha.5-win-x64-unsigned.exe", "new", time.Hour)
	writeInstallerFile(t, dir, "notes.txt", "ignored", 0)
	writeInstallerFile(t, dir, "Lightcode-9.9.9-win-x64.exe.bak", "ignored", 0)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "Lightcode-8.8.8-win-x64.exe"), 0o700))

	installer, err := findLightcodeInstaller(dir)
	require.NoError(t, err)
	require.NotNil(t, installer)
	sum := sha256.Sum256([]byte("new"))
	assert.Equal(t, "Lightcode-0.2.1-alpha.5-win-x64-unsigned.exe", installer.Name)
	assert.Equal(t, "0.2.1-alpha.5", installer.Version)
	assert.False(t, installer.Signed)
	assert.EqualValues(t, 3, installer.Size)
	assert.Equal(t, hex.EncodeToString(sum[:]), installer.SHA256)
	assert.Equal(t, "/download/lightcode/Lightcode-0.2.1-alpha.5-win-x64-unsigned.exe", installer.URL)
}

func TestFindLightcodeInstallerReportsSignedStableRelease(t *testing.T) {
	dir := t.TempDir()
	writeInstallerFile(t, dir, "Lightcode-1.4.0-win-x64.exe", "x", 0)
	installer, err := findLightcodeInstaller(dir)
	require.NoError(t, err)
	assert.True(t, installer.Signed)
	assert.Equal(t, "1.4.0", installer.Version)
}

func TestFindLightcodeInstallerWithoutDirectoryOrFiles(t *testing.T) {
	installer, err := findLightcodeInstaller(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	assert.Nil(t, installer)

	installer, err = findLightcodeInstaller(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, installer)
}

func TestLightcodePublicPlansExposeOnlyEnabledLightcodePlansWithoutPaymentIds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=private"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}))
	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })

	for _, plan := range []model.SubscriptionPlan{
		{Title: "Lightcode GO", Pool: model.SubscriptionPoolLightcode, RoutingGroup: "secret-group", AllowedModels: []string{"m1"},
			StripePriceId: "price_secret", PriceAmount: 20, DurationUnit: "month", DurationValue: 1, WindowShortSeconds: 18000, WindowShortAmount: 10, Enabled: true},
		{Title: "Disabled", Pool: model.SubscriptionPoolLightcode, RoutingGroup: "g", AllowedModels: []string{"m1"}, DurationUnit: "month", DurationValue: 1},
		{Title: "API plan", Pool: model.SubscriptionPoolAPI, DurationUnit: "month", DurationValue: 1, Enabled: true},
	} {
		plan := plan
		enabled := plan.Enabled
		require.NoError(t, db.Create(&plan).Error)
		// The column default turns a false Enabled into true on insert, so set it explicitly.
		require.NoError(t, db.Model(&model.SubscriptionPlan{}).Where("id = ?", plan.Id).Update("enabled", enabled).Error)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/lightcode/plans", nil)
	LightcodePublicPlans(ctx)

	body := recorder.Body.String()
	assert.Contains(t, body, "Lightcode GO")
	assert.NotContains(t, body, "Disabled")
	assert.NotContains(t, body, "API plan")
	assert.NotContains(t, body, "price_secret", "payment identifiers stay private")
	assert.NotContains(t, body, "secret-group", "routing groups stay private")
	assert.NotContains(t, body, "window_short_amount", "public plans do not disclose window amounts")
	assert.NotContains(t, body, "window_long_amount")
}

func TestLightcodeWindowJSONExposesOnlyUtilizationAndReset(t *testing.T) {
	view := &model.LightcodeWindowView{Limit: 5000000, Used: 1350000, UsedPercent: 27, ResetAt: 1800000000}
	got := lightcodeWindowJSON(view, 18000)
	assert.Equal(t, gin.H{"seconds": int64(18000), "used_percent": 27, "reset_at": int64(1800000000)}, got)
	assert.Nil(t, lightcodeWindowJSON(nil, 18000))
}

func TestLightcodeDownloadInfoUsesConfiguredPublicRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("LIGHTCODE_DOWNLOAD_DIR", dir)
	writeInstallerFile(t, dir, "Lightcode-0.2.1-win-x64-unsigned.exe", "installer", 0)
	for _, owner := range []string{"LightCode-AI", "Lorry-San"} {
		link := "https://github.com/" + owner + "/lightcode-releases/releases/download/v0.2.1/Lightcode-0.2.1-win-x64-unsigned.exe"
		t.Setenv("LIGHTCODE_WINDOWS_DOWNLOAD_URL", link)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		LightcodeDownloadInfo(ctx)
		var response struct {
			Success bool               `json:"success"`
			Data    LightcodeInstaller `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, owner)
		assert.Equal(t, link, response.Data.URL)
		assert.EqualValues(t, 9, response.Data.Size)
	}
}

func TestLightcodeDownloadInfoRejectsOtherReleaseOwners(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("LIGHTCODE_DOWNLOAD_DIR", dir)
	writeInstallerFile(t, dir, "Lightcode-0.2.1-win-x64-unsigned.exe", "installer", 0)
	for _, link := range []string{
		"https://github.com/someone-else/lightcode-releases/releases/download/v0.2.1/x.exe",
		"https://github.com/LightCode-AI/other-repo/releases/download/v0.2.1/x.exe",
		"http://github.com/LightCode-AI/lightcode-releases/releases/download/v0.2.1/x.exe",
	} {
		t.Setenv("LIGHTCODE_WINDOWS_DOWNLOAD_URL", link)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		LightcodeDownloadInfo(ctx)
		var response struct {
			Success bool `json:"success"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response.Success, link)
	}
}

func TestLightcodeDownloadFileServesOnlyListedInstallers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("LIGHTCODE_DOWNLOAD_DIR", dir)
	writeInstallerFile(t, dir, "Lightcode-0.2.1-win-x64.exe", "installer-bytes", 0)
	writeInstallerFile(t, dir, "secret.txt", "secret", 0)

	router := gin.New()
	router.GET("/download/lightcode/:name", LightcodeDownloadFile)
	get := func(name string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/download/lightcode/"+name, nil))
		return recorder
	}

	ok := get("Lightcode-0.2.1-win-x64.exe")
	assert.Equal(t, http.StatusOK, ok.Code)
	assert.Equal(t, "installer-bytes", ok.Body.String())
	assert.Contains(t, ok.Header().Get("Content-Disposition"), `filename="Lightcode-0.2.1-win-x64.exe"`)

	assert.Equal(t, http.StatusNotFound, get("secret.txt").Code, "files outside the installer pattern are not served")
	assert.Equal(t, http.StatusNotFound, get("Lightcode-9.9.9-win-x64.exe").Code, "unknown installers are not found")
	assert.Equal(t, http.StatusNotFound, get("..%2Fsecret-win-x64.exe").Code, "path separators never reach the file system")
}
