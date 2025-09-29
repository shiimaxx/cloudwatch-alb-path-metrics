package main

// pathRuleConfig represents the JSON shape used to configure route normalization rules.
type pathRuleConfig struct {
	Host   string `json:"host"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Route  string `json:"route"`
}
