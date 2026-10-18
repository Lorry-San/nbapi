package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Lorry-San/nbapi/common"
	"github.com/Lorry-San/nbapi/model"
	"github.com/gin-gonic/gin"
)

// Installer file names are restricted to one safe path segment so the download
// route can never leave the download directory or inject header characters.
var (
	lightcodeInstallerName    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,120}-win-x64(?:-unsigned)?\.exe$`)
	lightcodeInstallerVersion = regexp.MustCompile(`(\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+?)?)-win-x64`)
)

// LightcodeInstaller describes the newest Windows installer in the download directory.
type LightcodeInstaller struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	Signed    bool   `json:"signed"`
	UpdatedAt int64  `json:"updated_at"`
	URL       string `json:"url"`
}

type lightcodeInstallerHash struct {
	size    int64
	modTime int64
	sum     string
}

// The digest of a large installer is computed once per (size, mtime) and reused.
var lightcodeInstallerHashes = struct {
	sync.Mutex
	byPath map[string]lightcodeInstallerHash
}{byPath: map[string]lightcodeInstallerHash{}}

func lightcodeDownloadDir() string {
	return common.GetEnvOrDefaultString("LIGHTCODE_DOWNLOAD_DIR", "downloads")
}

func lightcodeInstallerSum(path string, size, modTime int64) (string, error) {
	cache := &lightcodeInstallerHashes
	cache.Lock()
	defer cache.Unlock()
	if h, ok := cache.byPath[path]; ok && h.size == size && h.modTime == modTime {
		return h.sum, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(digest.Sum(nil))
	cache.byPath[path] = lightcodeInstallerHash{size: size, modTime: modTime, sum: sum}
	return sum, nil
}

// findLightcodeInstaller returns the most recently modified installer, or nil
// when the directory is missing or holds none.
func findLightcodeInstaller(dir string) (*LightcodeInstaller, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var best *LightcodeInstaller
	var bestInfo os.FileInfo
	for _, entry := range entries {
		if !lightcodeInstallerName.MatchString(entry.Name()) {
			continue
		}
		// DirEntry.Info does not follow links, so a symlink is not a regular file here.
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if bestInfo != nil && !info.ModTime().After(bestInfo.ModTime()) {
			continue
		}
		bestInfo = info
		best = &LightcodeInstaller{
			Name:      entry.Name(),
			Size:      info.Size(),
			Signed:    !strings.Contains(entry.Name(), "-unsigned"),
			UpdatedAt: info.ModTime().Unix(),
			URL:       "/download/lightcode/" + entry.Name(),
		}
		if m := lightcodeInstallerVersion.FindStringSubmatch(entry.Name()); m != nil {
			best.Version = m[1]
		}
	}
	if best == nil {
		return nil, nil
	}
	sum, err := lightcodeInstallerSum(filepath.Join(dir, best.Name), bestInfo.Size(), bestInfo.ModTime().UnixNano())
	if err != nil {
		return nil, err
	}
	best.SHA256 = sum
	return best, nil
}

// LightcodePublicPlan is the display-only view of an enabled Lightcode plan for visitors who are not signed in.
type LightcodePublicPlan struct {
	Id                 int      `json:"id"`
	Title              string   `json:"title"`
	Subtitle           string   `json:"subtitle"`
	PriceAmount        float64  `json:"price_amount"`
	DurationUnit       string   `json:"duration_unit"`
	DurationValue      int      `json:"duration_value"`
	WindowShortSeconds int64    `json:"window_short_seconds"`
	WindowLongSeconds  int64    `json:"window_long_seconds"`
	Models             []string `json:"models"`
}

// LightcodePublicPlans lists enabled Lightcode plans without payment identifiers, groups or routing details.
func LightcodePublicPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ? AND pool = ?", true, model.SubscriptionPoolLightcode).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]LightcodePublicPlan, 0, len(plans))
	for _, p := range plans {
		p.NormalizeDefaults()
		models := p.AllowedModels
		if models == nil {
			models = []string{}
		}
		result = append(result, LightcodePublicPlan{
			Id: p.Id, Title: p.Title, Subtitle: p.Subtitle, PriceAmount: p.PriceAmount,
			DurationUnit: p.DurationUnit, DurationValue: p.DurationValue,
			WindowShortSeconds: p.WindowShortSeconds,
			WindowLongSeconds:  p.WindowLongSeconds,
			Models:             models,
		})
	}
	common.ApiSuccess(c, result)
}

// lightcodeReleaseOwners lists the GitHub owners of the public releases repository. The repository moved to
// LightCode-AI; GitHub redirects the old owner, so links configured before the move remain valid.
var lightcodeReleaseOwners = []string{"LightCode-AI", "Lorry-San"}

func isLightcodeReleasePath(path string) bool {
	for _, owner := range lightcodeReleaseOwners {
		if strings.HasPrefix(path, "/"+owner+"/lightcode-releases/releases/") {
			return true
		}
	}
	return false
}

// LightcodeDownloadInfo reports the installer the introduction page offers; data is null when none is published.
func LightcodeDownloadInfo(c *gin.Context) {
	installer, err := findLightcodeInstaller(lightcodeDownloadDir())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if installer != nil {
		if configured := os.Getenv("LIGHTCODE_WINDOWS_DOWNLOAD_URL"); configured != "" {
			link, err := url.Parse(configured)
			if err != nil || link.Scheme != "https" || link.Host != "github.com" || link.User != nil ||
				!isLightcodeReleasePath(link.Path) {
				common.ApiErrorMsg(c, "Invalid Lightcode release download URL")
				return
			}
			installer.URL = link.String()
		}
	}
	common.ApiSuccess(c, installer)
}

// LightcodeDownloadFile streams one installer by name; anything that is not a listed installer is a 404.
func LightcodeDownloadFile(c *gin.Context) {
	name := c.Param("name")
	if !lightcodeInstallerName.MatchString(name) {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(lightcodeDownloadDir(), name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Cache-Control", "public, max-age=300")
	c.File(path)
}
