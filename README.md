## 简介

Artgo 简单的Web框架

## 主要内容

* 基础路由
* Http 服务
* 中间件
* 错误捕获
* 上下文
* 分组路由
* 多种请求方式
* 模板
* 参数绑定和校验

## 目录结构

```shell
.
├── README.md
├── art.go #框架核心
├── binding.go #参数绑定
├── binding_test.go
├── context.go #上下文
├── go.mod
├── go.sum
├── middleware.go # 中间件
├── render.go #数据渲染
├── router.go #路由
├── trie.go #前缀树路由
├── validator.go #参数校验
└── vars.go #变量定义


```