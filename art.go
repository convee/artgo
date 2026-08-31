package artgo

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"time"
)

type HandlerFunc func(*Context)

type Engine struct {
	*RouterGroup
	router        *router
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

type RouterGroup struct {
	name        string
	parent      *RouterGroup
	middlewares []HandlerFunc
	engine      *Engine
}

func New() *Engine {
	e := &Engine{
		router: newRouter(),
	}
	e.RouterGroup = &RouterGroup{engine: e}
	return e
}

func Default() *Engine {
	e := New()
	e.Use(Logger(), Recovery())
	return e
}

func (g *RouterGroup) Group(name string) *RouterGroup {
	return &RouterGroup{
		name:   path.Join(g.name, name),
		parent: g,
		engine: g.engine,
	}
}

func (g *RouterGroup) Use(middlewares ...HandlerFunc) {
	for _, middleware := range middlewares {
		if middleware == nil {
			panic(fmt.Sprintf("router group %q requires non-nil middleware", g.name))
		}
	}
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	g.engine.router.addRoute(method, path.Join(g.name, comp), g.handlers(handler)...)
}

func (g *RouterGroup) handlers(handler HandlerFunc) []HandlerFunc {
	length := 1
	for group := g; group != nil; group = group.parent {
		length += len(group.middlewares)
	}

	chain := make([]HandlerFunc, 0, length)
	g.collectHandlers(&chain)
	return append(chain, handler)
}

func (g *RouterGroup) collectHandlers(chain *[]HandlerFunc) {
	if g.parent != nil {
		g.parent.collectHandlers(chain)
	}
	*chain = append(*chain, g.middlewares...)
}

// GET 将 GET 路由加载到内存
func (g *RouterGroup) GET(pattern string, handler HandlerFunc) {
	g.addRoute("GET", pattern, handler)
}

// POST 将 POST 路由加载到内存
func (g *RouterGroup) POST(pattern string, handler HandlerFunc) {
	g.addRoute("POST", pattern, handler)
}

// createStaticHandler 创建静态文件 handler
func (g *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(g.name, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		opened, err := fs.Open(file)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		_ = opened.Close()

		statusWriter := &responseStatusWriter{ResponseWriter: c.Writer}
		fileServer.ServeHTTP(statusWriter, c.Req)
		if statusWriter.status != 0 {
			c.StatusCode = statusWriter.status
			c.wroteHeader = true
		}
	}
}

type responseStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseStatusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

// Static 静态文件服务
func (g *RouterGroup) Static(relativePath string, root string) {
	handler := g.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	// Register GET handlers
	g.GET(urlPattern, handler)
}

// SetFuncMap 渲染自定义模板
func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

// LoadHTMLGlob 加载模板到内存
func (e *Engine) LoadHTMLGlob(pattern string) {
	e.htmlTemplates = template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
}

// ServeHTTP 实现 http.Handler 接口
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	c.engine = e
	e.router.handle(c, e.RouterGroup.middlewaresCopy())
}

// Run 启动自定义 http 服务器
func (e *Engine) Run(addr string) (err error) {
	server := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return server.ListenAndServe()
}

func (g *RouterGroup) middlewaresCopy() []HandlerFunc {
	if len(g.middlewares) == 0 {
		return nil
	}
	return append([]HandlerFunc(nil), g.middlewares...)
}
