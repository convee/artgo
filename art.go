package artgo

import (
	"html/template"
	"net/http"
	"path"
	"strings"
)

type HandlerFunc func(*Context)

type Engine struct {
	*RouterGroup
	router        *router
	routerGroups  []*RouterGroup
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

type RouterGroup struct {
	name        string
	middlewares []HandlerFunc
	engine      *Engine
}

func New() *Engine {
	e := &Engine{
		router: newRouter(),
	}
	e.RouterGroup = &RouterGroup{engine: e}
	e.routerGroups = []*RouterGroup{e.RouterGroup}
	return e
}

func Default() *Engine {
	e := New()
	e.Use(Logger(), Recovery())
	return e
}

func (g *RouterGroup) Group(name string) *RouterGroup {
	e := g.engine
	routeGroup := &RouterGroup{
		name:   g.name + name,
		engine: e,
	}
	e.routerGroups = append(e.routerGroups, routeGroup)
	return routeGroup
}

func (g *RouterGroup) Use(middlewares ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := g.name + comp
	g.engine.router.addRoute(method, pattern, handler)
}

//GET 将 GET 路由加载到内存
func (g *RouterGroup) GET(pattern string, handler HandlerFunc) {
	g.addRoute("GET", pattern, handler)
}

//POST 将 POST 路由加载到内存
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
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
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
	var middlewares []HandlerFunc
	for _, group := range e.routerGroups {
		if strings.HasPrefix(req.URL.Path, group.name) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = e
	e.router.handle(c)
}

// Run 启动自定义 http 服务器
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}
