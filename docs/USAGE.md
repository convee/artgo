# 使用指南

以下示例均假设模块中引入 `github.com/convee/artgo`。可直接运行的完整项目见 `examples/basic`。

## 创建引擎

```go
e := artgo.New()     // 不带默认中间件
d := artgo.Default() // Logger + Recovery
```

`Engine` 实现了 `http.Handler`，可以被标准库或其它 HTTP 服务器复用。

## 路由

```go
e := artgo.New()

e.GET("/", func(c *artgo.Context) {
	c.String(http.StatusOK, "index\n")
})

e.GET("/users/:id", func(c *artgo.Context) {
	c.String(http.StatusOK, "user: %s\n", c.Param("id"))
})

e.GET("/assets/*filepath", func(c *artgo.Context) {
	c.String(http.StatusOK, "asset: %s\n", c.Param("filepath"))
})

e.POST("/users", func(c *artgo.Context) {
	c.String(http.StatusCreated, "created\n")
})
```

公开注册方法为 `GET`、`POST`、`PUT`、`DELETE`、`PATCH`、`HEAD`、`OPTIONS`。
`Handle` 可注册任意方法（含 WebDAV 等自定义动词），`Any` 一次注册上述全部标准方法：

```go
e.PUT("/users/:id", updateUser)
e.DELETE("/users/:id", deleteUser)
e.Handle("REPORT", "/docs", reportDocs)
e.Any("/ping", pong)
```

路径已注册但请求方法不匹配时返回 `405 Method Not Allowed`，并按 RFC 9110 要求带上
`Allow` 响应头（值已排序，跨请求稳定）；路径本身不存在才返回 404。

## 分组与中间件

```go
e := artgo.Default()
api := e.Group("/api")
api.Use(func(c *artgo.Context) {
	c.SetHeader("X-Example", "artgo")
	c.Next()
})

api.GET("/health", func(c *artgo.Context) {
	c.RenderJson(http.StatusOK, artgo.H{"status": "ok"})
})
```

中间件在路由注册时绑定到该组后续注册的路由；请先调用 `Use`，再注册路由。嵌套分组的中间件顺序为根组、父组、当前组、endpoint。`/api` 分组的中间件不会误作用于 `/api-other`。

需要提前终止后续 handler 时：

```go
e.Use(func(c *artgo.Context) {
	if !isAllowed(c) {
		c.Abort()
		c.String(http.StatusForbidden, "forbidden\n")
		return
	}
	c.Next()
})
```

## JSON 绑定与校验

```go
artgo.EnableValidate()

type CreateUser struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"gte=0,lte=150"`
}

e.POST("/users", func(c *artgo.Context) {
	var input CreateUser
	if err := c.BindJson(&input); err != nil {
		c.RenderJson(http.StatusBadRequest, artgo.H{"error": err.Error()})
		return
	}
	c.RenderJson(http.StatusCreated, artgo.H{"name": input.Name})
})
```

`EnableValidate()` 是包级全局开关，会影响同一进程内所有 `BindJson` 调用，而不是只影响一个 `Engine`。

## Query 绑定

`BindQuery` 使用 `json` tag 或字段名，并把名称转为小写后匹配：

```go
type ListUsers struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

e.GET("/users", func(c *artgo.Context) {
	var input ListUsers
	if err := c.BindQuery(&input); err != nil {
		c.RenderJson(http.StatusBadRequest, artgo.H{"error": err.Error()})
		return
	}
	c.RenderJson(http.StatusOK, artgo.H{"page": input.Page, "size": input.Size})
})
```

## Form 绑定

`BindForm` 会按需解析 URL encoded 和 multipart 表单。它只绑定普通值，不处理文件字段：

```go
type Search struct {
	Keyword string `json:"keyword"`
}

e.POST("/search", func(c *artgo.Context) {
	var input Search
	if err := c.BindForm(&input); err != nil {
		c.RenderJson(http.StatusBadRequest, artgo.H{"error": err.Error()})
		return
	}
	c.RenderJson(http.StatusOK, artgo.H{"keyword": input.Keyword})
})
```

`BindForm` 的合并顺序是 multipart value、URL encoded PostForm、query；同名值最终以 query 为准。multipart 解析的内存阈值为 32 MiB，超出部分写入临时文件。

## 渲染

```go
// RenderJson 使用 jsoniter
c.RenderJson(http.StatusOK, artgo.H{"message": "hello"})

// JSON 与 RenderJson 一样使用共享的 jsoniter 配置
c.JSON(http.StatusOK, artgo.H{"message": "hello"})

c.String(http.StatusOK, "hello %s\n", "artgo")
c.Data(http.StatusOK, []byte("binary"))
```

Protobuf 相关 API 优先接受 `google.golang.org/protobuf/proto.Message`，同时兼容旧版 `github.com/golang/protobuf/proto.Message`。`RenderProtobuf` 在请求 `Content-Type` 媒体类型为 `application/json` 时转走 JSONPB 渲染；这是按请求头而不是 `Accept` 头判断。

## 模板与静态文件

必须先 `SetFuncMap`，再 `LoadHTMLGlob`：

```go
e.SetFuncMap(template.FuncMap{
	"upper": strings.ToUpper,
})
e.LoadHTMLGlob("templates/*.tmpl")

e.GET("/template/:name", func(c *artgo.Context) {
	c.HTML(http.StatusOK, "hello.tmpl", c.Param("name"))
})

e.Static("/static", "./static")
```

## 请求级键值存储

中间件与 handler 之间通过 `Context` 传值，作用域仅限单个请求，不跨请求共享：

```go
e.Use(func(c *artgo.Context) {
	c.Set("userID", authenticate(c))
	c.Next()
})

e.GET("/me", func(c *artgo.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized\n")
		return
	}
	c.RenderJson(http.StatusOK, artgo.H{"user": userID})
})
```

`MustGet` 在键不存在时 panic，`GetString` 返回字符串值（键不存在或类型不符时返回零值）。

## 优雅关闭

`Run` 启动的服务器可以用 `Shutdown` 停止接受新连接并等待在途请求完成，
`Run` 随之返回 `http.ErrServerClosed`：

```go
e := artgo.Default()
e.GET("/health", func(c *artgo.Context) { c.String(http.StatusOK, "ok") })

go func() {
	if err := e.Run(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}()

stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
<-stop

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := e.Shutdown(ctx); err != nil {
	log.Printf("shutdown: %v", err)
}
```

未调用过 `Run` 时 `Shutdown` 返回 nil。若自行组合 `http.Server`（见下节），请直接使用该 server 的 `Shutdown`。

`LoadHTMLGlob` 和 `Static` 的路径相对于进程当前工作目录。`examples/basic` 明确要求先进入示例目录执行 `go run .`，避免从仓库根目录运行时找不到模板和静态资源。

## 组合标准库 HTTP 服务

```go
func main() {
	e := artgo.Default()
	e.GET("/health", func(c *artgo.Context) {
		c.RenderJson(http.StatusOK, artgo.H{"status": "ok"})
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
```

## 运行内置示例

```shell
cd examples/basic
go run .
```

示例默认监听 `:8080`，也支持传入地址：

```shell
go run . 127.0.0.1:18080
```

验证：

```shell
curl -i http://127.0.0.1:8080/api/health
curl -i http://127.0.0.1:8080/users/alice
curl -i http://127.0.0.1:8080/template/alice
curl -i -X POST http://127.0.0.1:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","age":30}'
```
