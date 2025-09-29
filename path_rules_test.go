package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathRuleConfig_Unmarshal(t *testing.T) {
	payload := `{"host":"example.com","path":"^/users/[0-9]+$","route":"/users/:id"}`

	var rule pathRuleConfig
	err := json.Unmarshal([]byte(payload), &rule)

	require.NoError(t, err)
	assert.Equal(t, "example.com", rule.Host)
	assert.Equal(t, "^/users/[0-9]+$", rule.Path)
	assert.Equal(t, "/users/:id", rule.Route)
}

func TestNewPathRules_Success(t *testing.T) {
	raw := `[
		{"host":"example.com","path":"^/users/[0-9]+$","route":"/users/:id"},
		{"host":"example.com","path":"^/articles/[a-z0-9-]+$","route":"/articles/:slug"}
	]`

	rules, err := newPathRules(raw)

	require.NoError(t, err)
	require.NotNil(t, rules)
	assert.True(t, rules.enabled)
	require.Len(t, rules.rules, 2)

	first := rules.rules[0]
	assert.Equal(t, "example.com", first.host)
	assert.Equal(t, "/users/:id", first.route)
	assert.True(t, first.regex.MatchString("/users/42"))
	assert.False(t, first.regex.MatchString("/articles/next-gen"))

	second := rules.rules[1]
	assert.Equal(t, "example.com", second.host)
	assert.Equal(t, "/articles/:slug", second.route)
	assert.True(t, second.regex.MatchString("/articles/next-gen"))
}

func TestNewPathRules_EmptyString(t *testing.T) {
	rules, err := newPathRules("")

	require.NoError(t, err)
	require.NotNil(t, rules)
	assert.False(t, rules.enabled)
	assert.Empty(t, rules.rules)
}

func TestNewPathRules_InvalidJSON(t *testing.T) {
	_, err := newPathRules("not-json")
	assert.Error(t, err)
}

func TestNewPathRules_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "missing host", json: `[{"path":"^/users/[0-9]+$","route":"/users/:id"}]`},
		{name: "missing path", json: `[{"host":"example.com","route":"/users/:id"}]`},
		{name: "missing route", json: `[{"host":"example.com","path":"^/users/[0-9]+$"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newPathRules(tt.json)
			assert.Error(t, err)
		})
	}
}

func TestNewPathRules_InvalidRegex(t *testing.T) {
	_, err := newPathRules(`[{"host":"example.com","path":"^/users/[","route":"/users/:id"}]`)
	assert.Error(t, err)
}
