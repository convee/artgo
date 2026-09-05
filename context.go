package artgo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	engine      *Engine
	Writer      http.ResponseWriter
	Req         *http.Request
	Path        string
	Method      string
	Params      map[string]string
	StatusCode  int
	keys        map[string]any
	handlers    []HandlerFunc
	index       int
	aborted     bool
	body        []byte
	bodyErr     error
	bodyRead    bool
	wroteHeader bool
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

func (c *Context) Next() {
	if c.index >= len(c.handlers) {
		return
	}
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Abort prevents remaining handlers in the current chain from running.
func (c *Context) Abort() {
	c.index = len(c.handlers)
	c.aborted = true
}

// IsAborted reports whether the current handler chain has been aborted.
func (c *Context) IsAborted() bool {
	return c.aborted
}

// Set 在当前请求上下文中存入键值，供后续中间件与 handler 读取。
// 仅在单个请求的处理链内有效，不跨请求共享。
func (c *Context) Set(key string, value any) {
	if c.keys == nil {
		c.keys = make(map[string]any)
	}
	c.keys[key] = value
}

// Get 读取 Set 存入的值，第二个返回值表示键是否存在
func (c *Context) Get(key string) (any, bool) {
	value, exists := c.keys[key]
	return value, exists
}

// MustGet 读取 Set 存入的值，键不存在时 panic
func (c *Context) MustGet(key string) any {
	value, exists := c.Get(key)
	if !exists {
		panic(fmt.Sprintf("key %q does not exist in context", key))
	}
	return value
}

// GetString 读取字符串值，键不存在或类型不符时返回零值
func (c *Context) GetString(key string) string {
	value, _ := c.Get(key)
	s, _ := value.(string)
	return s
}

// Param 获取路由参数
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// PostForm 获取 POST 参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// PostBody 读取 Body
func (c *Context) PostBody() []byte {
	body, _ := c.postBody()
	return append([]byte(nil), body...)
}

func (c *Context) postBody() ([]byte, error) {
	if !c.bodyRead {
		c.body, c.bodyErr = io.ReadAll(c.Req.Body)
		c.bodyRead = true
	}
	return c.body, c.bodyErr
}

// Query 获取 GET 参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// Status 设置响应状态码
func (c *Context) Status(code int) {
	if c.wroteHeader {
		return
	}
	c.StatusCode = code
	c.wroteHeader = true
	c.Writer.WriteHeader(code)
}

// SetHeader 设置响应头
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// String 返回格式化字符串
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", ContentTypeTextPlain)
	c.Status(code)
	_, _ = c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON 返回 json 数据
func (c *Context) JSON(code int, obj interface{}) {
	data, err := JSON.Marshal(obj)
	if err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.SetHeader("Content-Type", ContentTypeJson)
	c.Status(code)
	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.Write([]byte("\n"))
}

// Data 返回文本数据
func (c *Context) Data(code int, data []byte) {
	if code == 0 {
		code = http.StatusOK
	}
	c.Status(code)
	_, _ = c.Writer.Write(data)
}

// HTML 输出 html
func (c *Context) HTML(code int, name string, data interface{}) {
	if c.engine == nil || c.engine.htmlTemplates == nil {
		c.Error(http.StatusInternalServerError, "html templates are not loaded")
		return
	}
	var output bytes.Buffer
	if err := c.engine.htmlTemplates.ExecuteTemplate(&output, name, data); err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
		return
	}
	c.SetHeader("Content-Type", ContentTypeHtml)
	c.Status(code)
	_, _ = c.Writer.Write(output.Bytes())
}

// Redirect 重定向
func (c *Context) Redirect(code int, location string) {
	if c.wroteHeader {
		return
	}
	c.StatusCode = code
	http.Redirect(c.Writer, c.Req, location, code)
	c.wroteHeader = true
}

// Error 返回错误状态
func (c *Context) Error(code int, err string) {
	c.SetHeader("Content-Type", ContentTypeTextPlain)
	c.Status(code)
	_, _ = c.Writer.Write([]byte(err))
	_, _ = c.Writer.Write([]byte("\n"))
}

// SetCookie 设置 cookie
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

func (c *Context) Bind(binding Binding, out interface{}) error {
	return binding.Bind(c, out)
}

func (c *Context) BindJson(out interface{}) error {
	return BindJson.Bind(c, out)
}

func (c *Context) BindProtobuf(out interface{}) error {
	return BindProtoBuf.Bind(c, out)
}

// BindQuery use json tag
func (c *Context) BindQuery(out interface{}) error {
	return BindQuery.Bind(c, out)
}

// BindForm use json tag
func (c *Context) BindForm(out interface{}) error {
	return BindForm.Bind(c, out)
}

func (c *Context) Render(render Render, code int, in interface{}) error {
	return render.Render(c, code, in)
}

func (c *Context) RenderJson(code int, in interface{}) error {
	return RenderJson.Render(c, code, in)
}

func (c *Context) RenderProtoBuf(code int, in interface{}) error {
	return RenderProtoBuf.Render(c, code, in)
}
