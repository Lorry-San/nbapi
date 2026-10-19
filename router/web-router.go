package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/controller"
	"github.com/Lorry-San/nbapi/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// WebAssets holds the embedded dashboard frontend assets.
type WebAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets WebAssets) {
	// A missing site page only disables /lightcode; the dashboard still serves.
	siteIndex, _ := assets.BuildFS.ReadFile("web/dist/lightcode/index.html")
	setWebRouter(router, common.EmbedFolder(assets.BuildFS, "web/dist"), siteIndex, assets.IndexPage)
}

func setWebRouter(router *gin.Engine, frontendFS static.ServeFileSystem, siteIndex, indexPage []byte) {
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	// The product site shares its name with an embedded directory. Registered after static.Serve, the
	// directory would be redirected and its page cached for a week, so these routes precede it.
	if len(siteIndex) > 0 {
		serveSite := func(c *gin.Context) {
			c.Set(middleware.RouteTagKey, "web")
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", siteIndex)
		}
		// The product site is also the homepage.
		for _, path := range []string{"/", "/lightcode", "/lightcode/"} {
			router.GET(path, serveSite)
			router.HEAD(path, serveSite)
		}
	}
	router.Use(static.Serve("/", frontendFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
