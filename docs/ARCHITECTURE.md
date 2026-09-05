# 架构与使用边界

## 总体结构

artgo 是标准库 `http.Handler` 的轻量适配层。应用可以调用 `Engine.Run` 使用默认服务，也可以把 `*Engine` 交给自定义 `http.Server` 获得完整超时、TLS 和生命周期控制。

```text
应用启动
  |
  |-- Engine.Run(addr) ------------------> http.Server{Handler: engine}
  `-- http.Server{Handler: engine} ------> net/http 标准服务
                                                  |
                                                  v
                                     Engine.ServeHTTP(w, req)
                                                  |
                                              new Context
                                                  |
                                       router.handle(method, path)
                                                  |
                                       trie.search 解析路由参数
                                                  |
                                     执行注册期绑定的 handler 链
                                                  |
                                       中间件 -> endpoint -> 中间件后置逻辑
                                                  |
                                  String/JSON/HTML/Protobuf/Data 渲染
```

## 组件职责

| 组件 | 职责 | 不负责 |
| --- | --- | --- |
| `Engine` | 实现 `ServeHTTP`，持有模板、路由和根路由组，并通过 `Shutdown` 优雅关闭由 `Run` 启动的服务器 | 进程信号、TLS 配置 |
| `RouterGroup` | 保存路径前缀、父组和中间件，并把中间件链绑定到注册路由 | 独立监听端口、请求隔离 |
| `router` | 校验路由，按 HTTP 方法维护前缀树和 handler 链 | 权限、限流、重试 |
| `trie` | 用静态子节点 map 和唯一 wildcard 子节点匹配路径，并提取参数 | 权限、请求解析 |
| `Context` | 保存单次请求状态，提供绑定、渲染、`Abort`、请求级键值存储和响应方法 | 跨请求会话与共享状态 |
| middleware | 以函数链包裹 endpoint，可调用 `Abort` 短路 | 自动识别业务语义 |
| binding | 将 JSON/Query/Form/Protobuf 写入结构体 | 认证授权、请求大小限制 |
| validator | 包装 `go-playground/validator` | 每个引擎独立配置；当前是包级全局开关 |
| render | 预序列化对象并设置常用 Content-Type | 内容协商、压缩、流式渲染 |

## 请求生命周期

1. `net/http` 调用 `Engine.ServeHTTP`。
2. `Engine` 为请求创建 `Context`。
3. `router.handle` 按方法查找路由节点；未命中时使用根中间件链，并按同路径是否注册过其他方法决定走内置 405 还是 404 handler。
4. 命中路由后，`Context` 使用注册路由时已绑定的中间件链和 endpoint，避免按请求路径猜测分组。
5. `Context.Next` 执行函数链；中间件可调用 `Abort` 阻止后续 handler。
6. Handler 调用 `String`、`JSON`、`RenderJson`、`HTML`、`Data` 或 Protobuf 渲染方法写响应。

## 路由模型

- 公开注册方法为 `GET`、`POST`、`PUT`、`DELETE`、`PATCH`、`HEAD`、`OPTIONS`；`Handle` 支持任意方法，`Any` 一次注册全部标准方法。
- 路径已注册但方法不匹配时返回 405 并带排序稳定的 `Allow` 头；路径不存在才返回 404。该判定只在未命中时执行，不在命中路径上产生开销。
- `:name` 匹配单个路径分段。
- `*name` 匹配剩余路径，必须位于 pattern 末尾且必须有名字。
- 静态路由优先于参数路由；同一层级只允许一个 wildcard 名称。
- 非法 pattern、未命名 wildcard、非末尾 catch-all、重复路由和路由冲突在注册期 panic，尽早暴露配置错误。
- 路由分段由 `staticChildren` map 直接定位；wildcard 查询为 O(1)，整体复杂度主要由路径分段数决定。

## 中间件模型

- `Default()` 启用 `Logger()` 和 `Recovery()`。
- 中间件链在路由注册时按“根组 -> 父组 -> 当前组 -> endpoint”的顺序绑定。
- 中间件归属于路由注册点，而不是请求路径字符串前缀，因此 `/api` 的中间件不会误作用于 `/apiary`。
- 根组中间件也会覆盖内置 404/405 响应，便于统一日志、追踪和错误处理。命中路由的请求不再拷贝根中间件切片。
- `Abort` 将执行位置移动到链尾并保留状态，`IsAborted` 可供外层中间件观察。

## 渲染与绑定

- `JSON` 和 `RenderJson` 都先序列化，成功后再写响应，避免序列化失败时输出半截 JSON。
- 请求体在同一 `Context` 中缓存，JSON 绑定可以重复读取。
- `BindForm` 支持 URL encoded 和 multipart 输入；同名值最终以 query 覆盖。
- Protobuf JSON 判断使用解析后的媒体类型，忽略参数大小写与顺序。
- `HTML` 先渲染到缓冲区，模板执行失败时返回 500，不输出半截页面。

## 性能特征

- 每个请求创建一个 `Context`；路由命中且无参数时避免分配参数 map。
- 中间件链在注册期创建，请求期不再按路由组线性收集。
- 路由静态分段使用 map 查找，参数路径只保留一个 wildcard 分支。
- `examples/performance` 覆盖静态 GET、参数 GET 和 JSON POST 的 `ServeHTTP` 端到端路径。基准包含 `httptest` 对象开销，适合相对比较，不代表线上 QPS。

## 生产使用边界

`Engine.Run` 设置 10 秒 `ReadHeaderTimeout` 以降低慢请求头风险，并可配合 `Engine.Shutdown` 优雅关闭；它仍不处理 TLS、写超时或空闲超时。公网服务建议显式组合标准库：

```go
server := &http.Server{
	Addr:              ":8080",
	Handler:           e,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       10 * time.Second,
	WriteTimeout:      15 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```

不可信请求仍应由网关或应用限制请求体大小、认证、授权和速率。`Engine.Shutdown` 只覆盖 `Run` 启动的服务器；自行组合 `http.Server` 时请直接调用该 server 的 `Shutdown`。
