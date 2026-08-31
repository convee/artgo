# 测试与性能验证

## 回归检查

在仓库根目录执行：

```shell
go test ./...
go test -race ./...
go vet ./...
```

或使用 Makefile：

```shell
make check
```

`make check` 依次执行：

1. `go test ./...`：覆盖根包现有绑定测试，并编译所有示例。
2. `go test -race ./...`：捕捉数据竞争。
3. `go vet ./...`：执行标准静态检查。

### 示例冒烟检查

```shell
make smoke
```

该目标会：

1. 构建 `examples/basic`。
2. 在 `127.0.0.1` 的临时空闲端口启动示例。
3. 等待健康检查成功。
4. 验证首页、参数路由、模板、静态文件和 JSON POST。
5. 通过 trap 清理后台进程。

如需前台启动示例：

```shell
make example
```

`make example` 默认端口是 `8080`。可覆盖：

```shell
make example HOST=127.0.0.1 EXAMPLE_PORT=18080
```

## 性能基准

`examples/performance` 提供 `Engine.ServeHTTP` 端到端基准，覆盖：

- 静态 GET 路由
- 带路径参数的 GET 路由
- JSON 绑定与 JSON 响应的 POST 路由

运行：

```shell
go test -run '^$' -bench . -benchmem ./examples/performance
```

或：

```shell
make bench
```

CI 中只执行短路径确认基准可编译可运行：

```shell
go test -run '^$' -bench . -benchtime=1x -benchmem ./examples/performance
```

对应：

```shell
make bench-smoke
```

## 对比方法

性能结论不能只看单次运行。建议：

1. 固定同一台机器、同一电源模式和相近系统负载。
2. 固定 Go 版本，并记录 `go version`。
3. 修改前后各运行基准至少 3 次。
4. 使用 `ns/op`、`B/op`、`allocs/op` 三个指标比较，不直接比较 CI 或跨机器结果。
5. 涉及路由或 Context 分配的改动，重点观察参数 GET 与 JSON POST 两个场景。

示例记录格式：

```text
go version go1.25.4 darwin/arm64
commit: <commit-sha>

BenchmarkServeHTTP_StaticRoute-10   <iterations>   <ns/op>   <B/op>   <allocs/op>
BenchmarkServeHTTP_ParamRoute-10    <iterations>   <ns/op>   <B/op>   <allocs/op>
BenchmarkServeHTTP_JSONPost-10      <iterations>   <ns/op>   <B/op>   <allocs/op>
```

这些基准包含 `httptest` 请求和响应对象创建，用于相对回归比较，不代表线上吞吐上限。

## 基准不覆盖的范围

当前基准不能替代以下验证：

- 真实网络下的吞吐、延迟分布和长连接行为
- 大请求体、慢客户端和超时场景
- 并发中间件链的语义验证
- 数据库、模板和静态文件磁盘 IO
- Protobuf 与模板渲染路径

相关改动仍需补充针对性测试或手动验收，并在 PR 中列出环境、命令和输出。
