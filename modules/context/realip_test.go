package context

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseRealIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "192.168.0.10")

	resp := httptest.NewRecorder()
	baseCtx, closeFn := NewBaseContext(resp, req)
	defer closeFn()

	assert.Equal(t, "192.168.0.10", baseCtx.RealIP())
}

func TestBaseRealIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "172.16.0.5, 10.0.0.1")

	resp := httptest.NewRecorder()
	baseCtx, closeFn := NewBaseContext(resp, req)
	defer closeFn()

	assert.Equal(t, "172.16.0.5", baseCtx.RealIP())
}

func TestBaseRealIP_Fallback(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	resp := httptest.NewRecorder()
	baseCtx, closeFn := NewBaseContext(resp, req)
	defer closeFn()

	assert.Equal(t, "10.0.0.1:1234", baseCtx.RealIP())
}
