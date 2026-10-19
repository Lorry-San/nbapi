package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dirFS mirrors the embedded frontend: directories exist, so static.Serve would claim them.
type dirFS struct{ http.FileSystem }

func (d dirFS) Exists(_ string, path string) bool {
	file, err := d.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func TestLightcodeSiteIsServedBeforeTheStaticDirectoryOfTheSameName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	site := []byte("<html>lightcode site</html>")
	spa := []byte("<html>dashboard</html>")
	engine := gin.New()
	setWebRouter(engine, dirFS{http.FS(fstest.MapFS{
		"lightcode/index.html":    {Data: site},
		"lightcode/favicon.png":   {Data: []byte("png")},
		"assets/app.js":           {Data: []byte("js")},
		"favicon.ico":             {Data: []byte("ico")},
		"lightcode/nested/x.html": {Data: []byte("x")},
	})}, site, spa)

	do := func(method, path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
		return recorder
	}

	got := do(http.MethodGet, "/")
	require.Equal(t, http.StatusOK, got.Code, "the homepage must not redirect")
	assert.Equal(t, string(site), got.Body.String())
	assert.Equal(t, "no-cache", got.Header().Get("Cache-Control"), "the page must not be cached for a week")
	assert.Equal(t, http.StatusOK, do(http.MethodHead, "/").Code)
	for _, path := range []string{"/lightcode", "/lightcode/", "/lightcode/index.html"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			got := do(method, path+"?lang=en")
			assert.Equal(t, http.StatusPermanentRedirect, got.Code)
			assert.Equal(t, "/?lang=en", got.Header().Get("Location"))
		}
	}

	asset := do(http.MethodGet, "/lightcode/favicon.png")
	assert.Equal(t, http.StatusOK, asset.Code)
	assert.Equal(t, "png", asset.Body.String(), "site assets still come from the static files")

	for _, path := range []string{"/lightcode/authorize", "/lightcode/models", "/pricing"} {
		got := do(http.MethodGet, path)
		assert.Equal(t, http.StatusOK, got.Code)
		assert.Equal(t, string(spa), got.Body.String(), "%s stays a dashboard route", path)
	}
}

func TestLightcodeSiteRoutesAreAbsentWithoutASitePage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spa := []byte("<html>dashboard</html>")
	engine := gin.New()
	setWebRouter(engine, dirFS{http.FS(fstest.MapFS{"favicon.ico": {Data: []byte("ico")}})}, nil, spa)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/lightcode", nil))
	assert.Equal(t, string(spa), recorder.Body.String(), "without a site page /lightcode falls back to the dashboard")
}

func TestLightcodeInstallerAcceptsHeadAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes["GET /download/lightcode/:name"])
	assert.True(t, routes["HEAD /download/lightcode/:name"])
	assert.True(t, routes["GET /api/lightcode/download"])
}

func TestLightcodeProfileRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	SetRelayRouter(engine)
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	assert.True(t, routes["GET /api/lightcode/profile"])
	assert.True(t, routes["GET /lc/v1/profile"])
}
