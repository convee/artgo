# artgo

artgo 是一个用 Go 编写的轻量 Web 框架，提供前缀树路由、路由分组、中间件、参数绑定与校验、JSON/Protobuf 渲染、模板和静态文件服务。它实现 `http.Handler`，可以嵌入任何标准库 HTTP 服务中。

当前代码面向 Go 1.25 及以上版本，仓库固定使用 Go 1.25.13 工具链。项目仍是轻量框架，适合学习和小型服务；直接用于公网生产服务前，请先阅读 [架构与边界](docs/ARCHITECTURE.md)。

## 功能

- GET/POST 路由与 `:param`、`*filepath` 通配参数
- 路由分组与前缀中间件
- 默认 Logger/Recovery 中间件
- JSON、Query、Form、Protobuf 绑定
- `go-playground/validator` 结构体校验
- JSON/Protobuf/模板/静态文件响应
- 实现 `http.Handler`，可组合标准库 `http.Server`

## 安装

```shell
go get github.com/convee/artgo@v1.1.0
```

## 快速开始

仓库内置一个可运行示例：

```shell
cd examples/basic
go run .
```

然后访问：

```text
http://127.0.0.1:8080/
http://127.0.0.1:8080/users/alice
http://127.0.0.1:8080/template/alice
```

或运行自动化冒烟检查：

```shell
make smoke
```

最小 API 示例：

```go
package main

import (
	"net/http"

	artgo "github.com/convee/artgo"
)

func main() {
	e := artgo.Default()

	e.GET("/health", func(c *artgo.Context) {
		c.RenderJson(http.StatusOK, artgo.H{"status": "ok"})
	})

	_ = e.Run(":8080")
}
```

## 文档

- [架构与使用边界](docs/ARCHITECTURE.md)：请求生命周期、组件职责、性能特征和当前限制
- [使用指南](docs/USAGE.md)：路由、中间件、绑定、校验、渲染、模板、静态文件和生产服务组合
- [测试与性能验证](docs/TESTING.md)：回归命令、基准测试和对比方法
- [贡献指南](CONTRIBUTING.md)、[安全策略](SECURITY.md) 和 [行为准则](CODE_OF_CONDUCT.md)

## 常用命令

```shell
make test       # go test ./...
make check      # test + race + vet
make bench      # 运行端到端路由基准
make smoke      # 构建、启动并验证 examples/basic
make example    # 前台启动 examples/basic
```

不想使用 Make 时，等价命令见 [测试与性能验证](docs/TESTING.md)。

## 目录结构

```text
.
├── README.md
├── Makefile
├── art.go               # Engine、RouterGroup、静态文件与模板入口
├── binding.go           # JSON/Query/Form/Protobuf 绑定
├── binding_test.go
├── context.go           # 请求上下文与响应方法
├── docs/
│   ├── ARCHITECTURE.md
│   ├── TESTING.md
│   └── USAGE.md
├── examples/
│   ├── basic/           # 可运行 API + 模板 + 静态资源示例
│   ├── port/            # 为 make smoke 查找本机空闲端口
│   └── performance/     # ServeHTTP 基准测试
├── go.mod
├── go.sum
├── middleware.go        # Logger/Recovery
├── render.go            # JSON/Protobuf 渲染
├── router.go            # 方法级路由分发
├── trie.go              # 前缀树路由
├── validator.go         # 结构体校验
└── vars.go              # 共享 JSON 序列化配置
```

## 贡献前检查

提交前至少运行：

```shell
make check
make bench-smoke
make smoke
```

CI 会在 Go 1.25 和当前稳定版上执行同等检查。

## 开源治理

artgo 按 MIT License 发布。缺陷、安全报告、支持范围和贡献流程见：

- [CHANGELOG.md](CHANGELOG.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
