package artgo

import (
	"fmt"
	"net/http"
	"strings"
)

type router struct {
	roots map[string]*node
}

func newRouter() *router {
	return &router{
		roots: make(map[string]*node),
	}
}

// parsePattern validates a route pattern and returns its non-empty segments.
// Trailing slashes are normalized so /posts/ and /posts register the same route.
func parsePattern(pattern string) []string {
	if pattern == "" || pattern[0] != '/' {
		panic("route pattern must begin with '/'")
	}
	if pattern == "/" {
		return nil
	}
	pattern = strings.TrimRight(pattern, "/")
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0, len(vs)-1)
	for index, item := range vs {
		if index == 0 {
			continue
		}
		if item == "" {
			panic(fmt.Sprintf("route pattern %q contains an empty segment", pattern+"/"))
		}
		if item[0] == ':' || item[0] == '*' {
			if len(item) == 1 {
				panic(fmt.Sprintf("route pattern %q contains an unnamed wildcard", pattern))
			}
		}
		if item[0] == '*' {
			if index != len(vs)-1 {
				panic(fmt.Sprintf("catch-all route segment must be the final segment in %q", pattern))
			}
			parts = append(parts, item)
			break
		}
		parts = append(parts, item)
	}
	return parts
}

func parseRequestPath(path string) []string {
	segments := strings.Split(path, "/")
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			parts = append(parts, segment)
		}
	}
	return parts
}

func (r *router) handle(c *Context, prefixHandlers []HandlerFunc) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		c.Params = params
		c.handlers = n.handlers
	} else {
		notFound := func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		}
		c.handlers = make([]HandlerFunc, 0, len(prefixHandlers)+1)
		c.handlers = append(c.handlers, prefixHandlers...)
		c.handlers = append(c.handlers, notFound)
	}
	c.Next()
}

func (r *router) addRoute(method string, pattern string, handlers ...HandlerFunc) {
	if method == "" {
		panic("route method cannot be empty")
	}
	if len(handlers) == 0 {
		panic(fmt.Sprintf("route %s %s requires a handler", method, pattern))
	}
	for _, handler := range handlers {
		if handler == nil {
			panic(fmt.Sprintf("route %s %s requires a non-nil handler", method, pattern))
		}
	}

	parts := parsePattern(pattern)
	params := make([]routeParam, 0, len(parts))
	for index, part := range parts {
		switch part[0] {
		case ':':
			params = append(params, routeParam{name: part[1:], index: index})
		case '*':
			params = append(params, routeParam{name: part[1:], index: index, catchAll: true})
		}
	}

	root, exists := r.roots[method]
	if !exists {
		root = &node{}
		r.roots[method] = root
	}
	root.insert(pattern, parts, params, handlers, 0)
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	root, exists := r.roots[method]
	if !exists {
		return nil, nil
	}

	n := root.search(parseRequestPath(path), 0)
	if n == nil {
		return nil, nil
	}

	var params map[string]string
	if len(n.params) > 0 {
		searchParts := parseRequestPath(path)
		params = make(map[string]string, len(n.params))
		for _, param := range n.params {
			value := ""
			if param.catchAll {
				if param.index < len(searchParts) {
					value = strings.Join(searchParts[param.index:], "/")
				}
			} else if param.index < len(searchParts) {
				value = searchParts[param.index]
			}
			params[param.name] = value
		}
	}
	return n, params
}
