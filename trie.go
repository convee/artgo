package artgo

import (
	"fmt"
	"strings"
)

type node struct {
	pattern        string
	part           string
	handlers       []HandlerFunc
	params         []routeParam
	staticChildren map[string]*node
	wildcardChild  *node
	isWild         bool
}

type routeParam struct {
	name     string
	index    int
	catchAll bool
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

func (n *node) insert(pattern string, parts []string, params []routeParam, handlers []HandlerFunc, height int) {
	if len(parts) == height {
		if n.pattern != "" {
			panic(fmt.Sprintf("route %s conflicts with existing route %s", pattern, n.pattern))
		}
		n.pattern = pattern
		n.params = params
		n.handlers = handlers
		return
	}

	part := parts[height]
	child, err := n.childFor(part)
	if err != nil {
		panic(err)
	}
	child.insert(pattern, parts, params, handlers, height+1)
}

func (n *node) childFor(part string) (*node, error) {
	if part == "" {
		return nil, fmt.Errorf("empty route segment")
	}
	if part[0] != ':' && part[0] != '*' {
		if n.staticChildren == nil {
			n.staticChildren = make(map[string]*node)
		}
		if child, exists := n.staticChildren[part]; exists {
			return child, nil
		}
		child := &node{part: part}
		n.staticChildren[part] = child
		return child, nil
	}

	wildcardName := part[1:]
	if wildcardName == "" {
		return nil, fmt.Errorf("wildcard route segment %q must have a name", part)
	}
	if n.wildcardChild == nil {
		n.wildcardChild = &node{part: part, isWild: true}
		return n.wildcardChild, nil
	}
	if n.wildcardChild.part != part {
		return nil, fmt.Errorf("route segment %q conflicts with existing wildcard segment %q", part, n.wildcardChild.part)
	}
	return n.wildcardChild, nil
}

func (n *node) search(parts []string, height int) *node {
	if strings.HasPrefix(n.part, "*") && n.pattern != "" {
		return n
	}
	if len(parts) == height {
		if n.pattern != "" {
			return n
		}
		if n.wildcardChild != nil && strings.HasPrefix(n.wildcardChild.part, "*") && n.wildcardChild.pattern != "" {
			return n.wildcardChild
		}
		return nil
	}

	if child, exists := n.staticChildren[parts[height]]; exists {
		if result := child.search(parts, height+1); result != nil {
			return result
		}
	}
	if n.wildcardChild != nil {
		return n.wildcardChild.search(parts, height+1)
	}
	return nil
}
