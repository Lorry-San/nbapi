package controller

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pngHeader = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 13, 'I', 'H', 'D', 'R'}

// The stored content type comes from the bytes, so a script renamed to .png is refused
// instead of being served back as an image.
func TestAvatarContentTypeAcceptsOnlyImages(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want string
	}{
		"png":  {pngHeader, "image/png"},
		"jpeg": {[]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F'}, "image/jpeg"},
		"gif":  {[]byte("GIF89a\x01\x00\x01\x00"), "image/gif"},
		"webp": {[]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
	}
	for name, tc := range cases {
		got, err := avatarContentType(tc.data)
		require.NoError(t, err, name)
		assert.Equal(t, tc.want, got, name)
	}
}

func TestAvatarContentTypeRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"html":  []byte("<html><script>alert(1)</script></html>"),
		"svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`),
		"text":  []byte("plain text"),
		"empty": {},
	} {
		_, err := avatarContentType(data)
		assert.ErrorIs(t, err, errAvatarType, name)
	}
}

func replyRecorder(raw bool, status int, data gin.H, err error) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
	lightcodeReply(c, raw, status, data, err)
	return recorder
}

// The desktop client reads raw JSON with a lightcode_error object, while the dashboard reads
// the ApiSuccess envelope; each surface must get the shape its parser expects.
func TestLightcodeReplyUsesTheEnvelopeOfEachSurface(t *testing.T) {
	raw := replyRecorder(true, http.StatusOK, gin.H{"kind": "full"}, nil)
	assert.Equal(t, http.StatusOK, raw.Code)
	assert.JSONEq(t, `{"kind":"full"}`, raw.Body.String())

	wrapped := replyRecorder(false, http.StatusOK, gin.H{"kind": "full"}, nil)
	assert.Contains(t, wrapped.Body.String(), `"success":true`)
	assert.Contains(t, wrapped.Body.String(), `"kind":"full"`)
}

func TestLightcodeReplyReportsErrorsPerSurface(t *testing.T) {
	failure := errors.New("重置卡已使用或已过期")

	raw := replyRecorder(true, http.StatusConflict, nil, failure)
	assert.Equal(t, http.StatusConflict, raw.Code)
	assert.Contains(t, raw.Body.String(), `"type":"lightcode_error"`)
	assert.Contains(t, raw.Body.String(), "重置卡已使用或已过期")

	wrapped := replyRecorder(false, http.StatusConflict, nil, failure)
	assert.Contains(t, wrapped.Body.String(), `"success":false`)
}
