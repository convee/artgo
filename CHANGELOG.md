# Changelog

## Unreleased

### Changed

- 路由处理器链在注册期绑定，中间件不再按请求路径字符串前缀误匹配。
- 静态路由优先于参数路由；注册期检测非法、冲突和重复路由。
- 新增 `Context.Abort` 和 `Context.IsAborted`，支持中间件短路。
- JSON、模板和静态资源渲染路径减少半响应、重复读取请求体和资源句柄泄漏。
- `Engine.Run` 增加默认 `ReadHeaderTimeout`。
- Protobuf 迁移到新版 `google.golang.org/protobuf` API，并兼容旧版 MessageV1。
- 依赖升级到安全维护版本，最低 Go 版本提升到 1.25。
- 命中路由的请求不再拷贝全局中间件切片，该拷贝此前只有未命中分支用得到。
- `getRoute` 对同一路径只切分一次，此前每个带参请求会重复切分并多分配一个切片。
- 移除 `node.children` 死字段（只写不读）。

### Added

- 补齐 HTTP 动词：`PUT`、`DELETE`、`PATCH`、`HEAD`、`OPTIONS`，以及 `Handle`（任意方法）和 `Any`。
- 路径已注册但方法不匹配时返回 405 并带排序稳定的 `Allow` 响应头，与 404 区分开。
- `Context.Set`/`Get`/`MustGet`/`GetString` 请求级键值存储，作用域限单个请求。
- `Engine.Shutdown` 优雅关闭，等待在途请求完成。
- `Static` 同时注册 HEAD，静态资源可被 HEAD 探测。
- 带全局中间件的性能基准，覆盖此前未被测量到的请求路径。
- 路由、上下文、渲染、绑定、静态资源和恢复中间件回归测试。
- 可运行示例、性能基准、架构/使用/测试文档、Makefile 和 CI。
- MIT License、贡献指南、安全策略、支持范围和行为准则。

## v1.1.0 - 2023-03-04

- 提供参数绑定和校验能力。
- 更新项目说明文档。

## v1.0.0 - 2023-02-22

- 初始路由、中间件、上下文、模板渲染和 HTTP 服务能力。
