package normalizer

import (
	"regexp"
	"strings"
)

type PathNormalizer struct {
	rules []NormalizationRule
}

type NormalizationRule struct {
	Pattern     *regexp.Regexp
	Replacement string
}

func NewPathNormalizer() *PathNormalizer {
	rules := []NormalizationRule{
		{
			Pattern:     regexp.MustCompile(`/\d+`),
			Replacement: "/:id",
		},
		{
			Pattern:     regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
			Replacement: "/:uuid",
		},
		{
			Pattern:     regexp.MustCompile(`/[a-zA-Z0-9_-]+\.(jpg|jpeg|png|gif|css|js|ico|svg|woff|woff2|ttf|eot)`),
			Replacement: "/:file",
		},
		{
			Pattern:     regexp.MustCompile(`/[a-zA-Z0-9_-]{32,}`),
			Replacement: "/:hash",
		},
	}
	
	return &PathNormalizer{rules: rules}
}

func (n *PathNormalizer) Normalize(path string) string {
	if path == "" {
		return "/"
	}
	
	normalizedPath := strings.TrimSpace(path)
	
	normalizedPath = n.removeQueryString(normalizedPath)
	
	normalizedPath = n.normalizePath(normalizedPath)
	
	for _, rule := range n.rules {
		normalizedPath = rule.Pattern.ReplaceAllString(normalizedPath, rule.Replacement)
	}
	
	if normalizedPath == "" {
		normalizedPath = "/"
	}
	
	return normalizedPath
}

func (n *PathNormalizer) removeQueryString(path string) string {
	if idx := strings.Index(path, "?"); idx != -1 {
		return path[:idx]
	}
	return path
}

func (n *PathNormalizer) normalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	
	parts := strings.Split(path, "/")
	var normalizedParts []string
	
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(normalizedParts) > 0 {
				normalizedParts = normalizedParts[:len(normalizedParts)-1]
			}
			continue
		}
		normalizedParts = append(normalizedParts, part)
	}
	
	if len(normalizedParts) == 0 {
		return "/"
	}
	
	return "/" + strings.Join(normalizedParts, "/")
}

func (n *PathNormalizer) AddRule(pattern string, replacement string) error {
	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	rule := NormalizationRule{
		Pattern:     compiledPattern,
		Replacement: replacement,
	}
	
	n.rules = append(n.rules, rule)
	return nil
}
