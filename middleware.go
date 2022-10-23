package artgo

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				c.Status(http.StatusInternalServerError)
				log.Printf("panic:method[%s];err[%v];stack[%v]", c.Method, err, string(debug.Stack()))
			}
		}()
		c.Next()
	}
}

func Logger() HandlerFunc {
	return func(c *Context) {
		t := time.Now()
		c.Next()
		log.Printf("[%d] %s %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
