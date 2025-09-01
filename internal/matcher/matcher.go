package matcher

import (
	"strings"
)

type Route struct {
	Method  string `json:"method"`
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
}

type node struct {
	children  map[string]*node
	isParam   bool
	paramName string
	name      string
	isEnd     bool
}

type Matcher struct {
	roots map[string]*node
}

func New(routes []Route) *Matcher {
	m := &Matcher{
		roots: make(map[string]*node),
	}
	for _, r := range routes {
		m.addRoute(r)
	}
	return m
}

func (pm *Matcher) addRoute(config Route) {
	if pm.roots[config.Method] == nil {
		pm.roots[config.Method] = &node{
			children: make(map[string]*node),
		}
	}

	root := pm.roots[config.Method]
	parts := strings.Split(strings.Trim(config.Pattern, "/"), "/")
	current := root

	for _, part := range parts {
		if part == "" {
			continue
		}

		var key string
		isParam := false
		paramName := ""

		if strings.HasPrefix(part, ":") {
			key = ":param"
			isParam = true
			paramName = part[1:]
		} else {
			key = part
		}

		if current.children[key] == nil {
			current.children[key] = &node{
				children:  make(map[string]*node),
				isParam:   isParam,
				paramName: paramName,
			}
		}

		current = current.children[key]
	}

	current.isEnd = true
	current.name = config.Name
}

func (pm *Matcher) Match(method, path string) (string, bool) {
	root := pm.roots[method]
	if root == nil {
		return "", false
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := root

	for _, part := range parts {
		if part == "" {
			continue
		}

		if current.children[part] != nil {
			current = current.children[part]
		} else if current.children[":param"] != nil {
			current = current.children[":param"]
		} else {
			return "", false
		}
	}

	if current.isEnd {
		return current.name, true
	}

	return "", false
}
