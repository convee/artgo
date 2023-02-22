package artgo

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	engine     *Engine
	Writer     http.ResponseWriter
	Req        *http.Request
	Path       string
	Method     string
	Params     map[string]string
	StatusCode int
	handlers   []HandlerFunc
	index      int
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
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Param 获取路由参数
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// PostForm 获取 POST 参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// Query 获取 GET 参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// Status 设置响应状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 设置响应头
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// String 返回格式化字符串
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	_, _ = c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON 返回 json 数据
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

// Data 返回文本数据
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	_, _ = c.Writer.Write(data)
}

// HTML 输出 html
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Error(http.StatusInternalServerError, err.Error())
	}
}

// Redirect 重定向
func (c *Context) Redirect(code int, location string) {
	http.Redirect(c.Writer, c.Req, location, code)
}

// Error 返回错误状态
func (c *Context) Error(code int, err string) {
	http.Error(c.Writer, err, code)
}

// SetCookie 设置 cookie
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// Render 渲染模板
func (c *Context) Render(data interface{}, filenames ...string) error {
	t, err := template.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	err = t.Execute(c.Writer, data)
	if err != nil {
		return err
	}
	return nil
}
