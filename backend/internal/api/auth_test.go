package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/config"
)

// newAuthAPIOf 构造一个只用于鉴权测试的 API（不需要 DB/Worker）。
func newAuthAPI(pw, token, secret string) *API {
	return &API{Cfg: &config.Config{Password: pw, APIToken: token, SessionSecret: secret}}
}

// okHandler 恒返回 200 的下游 handler。
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
}

// TestExemptPath 校验免鉴权路径判定。
func TestExemptPath(t *testing.T) {
	yes := []string{"/api/v1/health", "/api/v1/login", "/api/v1/session"}
	for _, p := range yes {
		if !exemptPath(p) {
			t.Fatalf("%s 应免鉴权", p)
		}
	}
	if exemptPath("/api/v1/tasks") {
		t.Fatalf("/api/v1/tasks 不应免鉴权")
	}
}

// TestSessionValueHMAC 校验 sessionValue 是 HMAC-SHA256 结果（同 secret 稳定、不同 secret 不同、64 hex）。
func TestSessionValueHMAC(t *testing.T) {
	a1 := newAuthAPI("pw", "", "secret-a")
	v1 := a1.sessionValue()
	if len(v1) != 64 {
		t.Fatalf("sessionValue 应为 64 位 hex，实际 %d: %s", len(v1), v1)
	}
	if a1.sessionValue() != v1 {
		t.Fatalf("同 secret 应稳定")
	}
	a2 := newAuthAPI("pw", "", "secret-b")
	if a2.sessionValue() == v1 {
		t.Fatalf("不同 secret 应产生不同签名")
	}
}

// TestAuth_TokenHeader 校验 X-API-Token 鉴权通过、错误 token 被拒。
func TestAuth_TokenHeader(t *testing.T) {
	a := newAuthAPI("", "s3cret", "x")
	h := a.withAuth(okHandler())

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req.Header.Set("X-API-Token", "s3cret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("正确 token 应放行，实际 %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req2.Header.Set("X-API-Token", "wrong")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 401 {
		t.Fatalf("错误 token 应 401，实际 %d", rec2.Code)
	}
}

// TestAuth_BearerAndQuery 校验 Bearer 头与 ?token= 两种携带方式。
func TestAuth_BearerAndQuery(t *testing.T) {
	a := newAuthAPI("", "s3cret", "x")
	h := a.withAuth(okHandler())

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("Bearer 应放行，实际 %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/tasks?token=s3cret", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("?token= 应放行，实际 %d", rec2.Code)
	}
}

// TestAuth_SessionCookie 校验正确 session cookie 放行、伪造 cookie 被拒。
func TestAuth_SessionCookie(t *testing.T) {
	a := newAuthAPI("pw", "", "secret-a")
	h := a.withAuth(okHandler())

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: a.sessionValue()})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("正确 cookie 应放行，实际 %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	req2.AddCookie(&http.Cookie{Name: "session", Value: "forged"})
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 401 {
		t.Fatalf("伪造 cookie 应 401，实际 %d", rec2.Code)
	}

	req3 := httptest.NewRequest("GET", "/api/v1/tasks", nil) // 无 cookie
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != 401 {
		t.Fatalf("无 cookie 应 401，实际 %d", rec3.Code)
	}
}

// TestAuth_Open 校验 token 与 password 皆空时开放访问。
func TestAuth_Open(t *testing.T) {
	a := newAuthAPI("", "", "")
	h := a.withAuth(okHandler())
	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("开放模式应放行，实际 %d", rec.Code)
	}
}

// TestAuth_ExemptPathsBypass 校验免鉴权路径即使启用密码也放行。
func TestAuth_ExemptPathsBypass(t *testing.T) {
	a := newAuthAPI("pw", "", "secret-a")
	h := a.withAuth(okHandler())
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("health 应免鉴权放行，实际 %d", rec.Code)
	}
}

// TestAuth_NonAPI 校验非 /api/ 前缀请求直接放行。
func TestAuth_NonAPI(t *testing.T) {
	a := newAuthAPI("pw", "tok", "secret-a")
	h := a.withAuth(okHandler())
	req := httptest.NewRequest("GET", "/index.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("非 API 路径应放行，实际 %d", rec.Code)
	}
}
