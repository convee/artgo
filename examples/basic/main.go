package main

import (
	"net/http"
	"os"
	"strings"
	"text/template"

	artgo "github.com/convee/artgo"
)

type CreateUserRequest struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"gte=0,lte=150"`
}

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	artgo.EnableValidate()
	e := artgo.Default()

	e.SetFuncMap(template.FuncMap{
		"upper": strings.ToUpper,
	})
	e.LoadHTMLGlob("templates/*.tmpl")
	e.Static("/static", "./static")

	e.GET("/", func(c *artgo.Context) {
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})
	e.GET("/users/:name", func(c *artgo.Context) {
		c.RenderJson(http.StatusOK, artgo.H{"name": c.Param("name")})
	})
	e.GET("/template/:name", func(c *artgo.Context) {
		c.HTML(http.StatusOK, "hello.tmpl", c.Param("name"))
	})

	api := e.Group("/api")
	api.Use(func(c *artgo.Context) {
		c.SetHeader("X-Artgo-Example", "basic")
		c.Next()
	})
	api.GET("/health", func(c *artgo.Context) {
		c.RenderJson(http.StatusOK, artgo.H{"status": "ok"})
	})
	api.POST("/users", func(c *artgo.Context) {
		var req CreateUserRequest
		if err := c.BindJson(&req); err != nil {
			c.RenderJson(http.StatusBadRequest, artgo.H{"error": err.Error()})
			return
		}
		c.RenderJson(http.StatusCreated, artgo.H{"name": req.Name, "age": req.Age})
	})

	if err := e.Run(addr); err != nil {
		panic(err)
	}
}
