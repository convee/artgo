package performance

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	artgo "github.com/convee/artgo"
)

func newEngine() *artgo.Engine {
	e := artgo.New()

	e.GET("/health", func(c *artgo.Context) {
		c.String(http.StatusOK, "ok")
	})
	e.GET("/users/:id", func(c *artgo.Context) {
		c.String(http.StatusOK, "%s", c.Param("id"))
	})
	e.POST("/users", func(c *artgo.Context) {
		var input struct {
			Name string `json:"name"`
		}
		if err := c.BindJson(&input); err != nil {
			c.String(http.StatusBadRequest, "%s", err.Error())
			return
		}
		c.RenderJson(http.StatusCreated, artgo.H{"name": input.Name})
	})

	return e
}

func serve(b *testing.B, e *artgo.Engine, newRequest func() *http.Request) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		e.ServeHTTP(w, newRequest())
		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			b.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
		}
	}
}

func BenchmarkServeHTTP_StaticRoute(b *testing.B) {
	e := newEngine()
	serve(b, e, func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/health", nil)
	})
}

func BenchmarkServeHTTP_ParamRoute(b *testing.B) {
	e := newEngine()
	serve(b, e, func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/users/123", nil)
	})
}

func BenchmarkServeHTTP_JSONPost(b *testing.B) {
	e := newEngine()
	serve(b, e, func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/users", nil)
		req.Body = io.NopCloser(bytes.NewBufferString(`{"name":"alice"}`))
		req.Header.Set("Content-Type", "application/json")
		return req
	})
}

// newEngineWithGlobalMiddleware 反映真实用法：生产服务几乎都会挂全局中间件。
// 这一路径此前每个请求都要拷贝一次全局中间件切片，即使请求命中了路由。
func newEngineWithGlobalMiddleware() *artgo.Engine {
	e := newEngine()
	e.Use(func(c *artgo.Context) { c.Next() })
	e.Use(func(c *artgo.Context) { c.Next() })
	e.Use(func(c *artgo.Context) { c.Next() })
	return e
}

func BenchmarkServeHTTP_StaticRouteWithGlobalMiddleware(b *testing.B) {
	e := newEngineWithGlobalMiddleware()
	serve(b, e, func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/health", nil)
	})
}

func BenchmarkServeHTTP_ParamRouteWithGlobalMiddleware(b *testing.B) {
	e := newEngineWithGlobalMiddleware()
	serve(b, e, func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/users/123", nil)
	})
}
