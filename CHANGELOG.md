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

### Added

- 路由、上下文、渲染、绑定、静态资源和恢复中间件回归测试。
- 可运行示例、性能基准、架构/使用/测试文档、Makefile 和 CI。
- MIT License、贡献指南、安全策略、支持范围和行为准则。

## v1.1.0 - 2023-03-04

- 提供参数绑定和校验能力。
- 更新项目说明文档。

## v1.0.0 - 2023-02-22

- 初始路由、中间件、上下文、模板渲染和 HTTP 服务能力。
