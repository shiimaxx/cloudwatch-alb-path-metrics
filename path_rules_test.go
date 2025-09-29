package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathRuleConfig_Unmarshal(t *testing.T) {
	payload := []byte(`{"host":"example.com","method":"GET","path":"^/users/[0-9]+$","route":"/users/:id"}`)

	var rule pathRuleConfig
	err := json.Unmarshal(payload, &rule)

	require.NoError(t, err)
	assert.Equal(t, "example.com", rule.Host)
	assert.Equal(t, "GET", rule.Method)
	assert.Equal(t, "^/users/[0-9]+$", rule.Path)
	assert.Equal(t, "/users/:id", rule.Route)
}
