package artgo_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/convee/artgo"
	"github.com/stretchr/testify/assert"
)

func TestJSONRenderingReportsMarshalErrorsBeforeWritingBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	context := &artgo.Context{Writer: recorder}

	context.JSON(http.StatusOK, make(chan int))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "unsupported type")
	assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
}

func TestHTMLRenderingDoesNotWritePartialTemplateOutput(t *testing.T) {
	recorder := httptest.NewRecorder()
	engine := artgo.New()
	templateDir := t.TempDir()
	templatePath := filepath.Join(templateDir, "index.tmpl")
	assert.Nil(t, os.WriteFile(templatePath, []byte("{{define \"index\"}}hello{{end}}"), 0o600))
	engine.LoadHTMLGlob(templatePath)

	engine.GET("/html", func(c *artgo.Context) {
		c.HTML(http.StatusOK, "missing", nil)
	})
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/html", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"missing" is undefined`)
}

func TestPostBodyCanReadAndBindMoreThanOnce(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"artgo"}`))
	context := &artgo.Context{Req: request}

	assert.Equal(t, `{"name":"artgo"}`, string(context.PostBody()))
	var payload struct {
		Name string `json:"name"`
	}
	assert.Nil(t, context.BindJson(&payload))
	assert.Equal(t, "artgo", payload.Name)
}

func TestBindFormParsesURLEncodedBody(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/?ignored=value",
		strings.NewReader("name=artgo&number=7"),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	context := &artgo.Context{Req: request}

	var form struct {
		Name   string `json:"name"`
		Number int    `json:"number"`
	}
	assert.Nil(t, context.BindForm(&form))
	assert.Equal(t, "artgo", form.Name)
	assert.Equal(t, 7, form.Number)
}

func TestBindFormParsesMultipartValues(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	assert.Nil(t, writer.WriteField("name", "multipart"))
	assert.Nil(t, writer.WriteField("number", "12"))
	assert.Nil(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context := &artgo.Context{Req: request}

	var form struct {
		Name   string `json:"name"`
		Number int    `json:"number"`
	}
	assert.Nil(t, context.BindForm(&form))
	assert.Equal(t, "multipart", form.Name)
	assert.Equal(t, 12, form.Number)
}

func TestRepeatedStatusCallsDoNotWriteHeaderTwice(t *testing.T) {
	recorder := httptest.NewRecorder()
	context := &artgo.Context{Writer: recorder}

	context.Status(http.StatusOK)
	context.Status(http.StatusInternalServerError)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestDataWithoutExplicitStatusTracksImplicitOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	context := &artgo.Context{Writer: recorder}

	context.Data(0, []byte("ok"))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, http.StatusOK, context.StatusCode)
	assert.Equal(t, "ok", recorder.Body.String())
}

func TestErrorAfterCommittedResponseKeepsActualStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	context := &artgo.Context{Writer: recorder}

	context.String(http.StatusCreated, "created")
	context.Error(http.StatusInternalServerError, "late error")

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, http.StatusCreated, context.StatusCode)
}

func TestRecoveryReturnsServerErrorInsteadOfCrashingServer(t *testing.T) {
	engine := artgo.Default()
	engine.GET("/panic", func(c *artgo.Context) {
		panic(errors.New("expected test panic"))
	})
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestStaticFileServingUpdatesContextStatus(t *testing.T) {
	root := t.TempDir()
	assert.Nil(t, os.WriteFile(filepath.Join(root, "app.js"), []byte("export const ok = true;\n"), 0o600))

	engine := artgo.New()
	var observedStatus int
	engine.Use(func(c *artgo.Context) {
		c.Next()
		observedStatus = c.StatusCode
	})
	engine.Static("/static", root)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, http.StatusOK, observedStatus)
	assert.Contains(t, recorder.Body.String(), "export const ok")
}

func TestContextStorePassesValuesBetweenMiddlewareAndHandler(t *testing.T) {
	router := artgo.New()
	router.Use(func(c *artgo.Context) {
		c.Set("user", "alice")
		c.Next()
	})
	router.GET("/me", func(c *artgo.Context) {
		value, exists := c.Get("user")
		assert.True(t, exists)
		c.String(http.StatusOK, "user=%v store=%s", value, c.GetString("user"))
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/me", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "user=alice store=alice", recorder.Body.String())
}

func TestContextStoreIsolatesRequests(t *testing.T) {
	router := artgo.New()
	router.GET("/first", func(c *artgo.Context) {
		c.Set("leak", "value")
		c.String(http.StatusOK, "ok")
	})
	router.GET("/second", func(c *artgo.Context) {
		_, exists := c.Get("leak")
		assert.False(t, exists, "value leaked across requests")
		c.String(http.StatusOK, "ok")
	})

	for _, target := range []string{"/first", "/second"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		assert.Equal(t, http.StatusOK, recorder.Code)
	}
}

func TestContextGetOnMissingKey(t *testing.T) {
	context := &artgo.Context{}

	value, exists := context.Get("absent")
	assert.False(t, exists)
	assert.Nil(t, value)
	assert.Equal(t, "", context.GetString("absent"))
	assert.Panics(t, func() { context.MustGet("absent") })
}
