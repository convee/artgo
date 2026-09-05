package artgo

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"sync"
	"time"
)

type HandlerFunc func(*Context)

type Engine struct {
	*RouterGroup
	router        *router
	htmlTemplates *template.Template
	funcMap       template.FuncMap

	mu     sync.Mutex
	server *http.Server
}

// anyMethods 是 Any 注册的方法集合，也是框架承诺支持的动词全集
var anyMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	http.MethodPatch, http.MethodHead, http.MethodOptions,
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

// Handle 以任意 HTTP 方法注册路由，是所有动词方法的公共入口
func (g *RouterGroup) Handle(method string, pattern string, handler HandlerFunc) {
	g.addRoute(method, pattern, handler)
}

// GET 将 GET 路由加载到内存
func (g *RouterGroup) GET(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodGet, pattern, handler)
}

// POST 将 POST 路由加载到内存
func (g *RouterGroup) POST(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodPost, pattern, handler)
}

// PUT 将 PUT 路由加载到内存
func (g *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodPut, pattern, handler)
}

// DELETE 将 DELETE 路由加载到内存
func (g *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodDelete, pattern, handler)
}

// PATCH 将 PATCH 路由加载到内存
func (g *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodPatch, pattern, handler)
}

// HEAD 将 HEAD 路由加载到内存
func (g *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodHead, pattern, handler)
}

// OPTIONS 将 OPTIONS 路由加载到内存
func (g *RouterGroup) OPTIONS(pattern string, handler HandlerFunc) {
	g.addRoute(http.MethodOptions, pattern, handler)
}

// Any 为 anyMethods 中的每个方法注册同一个 handler
func (g *RouterGroup) Any(pattern string, handler HandlerFunc) {
	for _, method := range anyMethods {
		g.addRoute(method, pattern, handler)
	}
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
	g.HEAD(urlPattern, handler)
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
	e.router.handle(c)
}

// Run 启动自定义 http 服务器
func (e *Engine) Run(addr string) (err error) {
	server := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}
	e.mu.Lock()
	e.server = server
	e.mu.Unlock()
	return server.ListenAndServe()
}

// Shutdown 优雅关闭由 Run 启动的服务器：停止接受新连接并等待在途请求完成。
// 未调用过 Run 时返回 nil。Run 会随之返回 http.ErrServerClosed。
func (e *Engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	server := e.server
	e.mu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

