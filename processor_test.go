package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLogLine_Match(t *testing.T) {
	rules, err := newPathRules(`[{"host":"api.example.com","path":"^/users/[0-9]+$","route":"/users/:id"}]`)
	require.NoError(t, err)

	line := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 192.168.1.100:57832 10.0.1.1:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/users/123 HTTP/1.1" "Mozilla/5.0" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`

	entry, route, ok := normalizeLogLine(line, rules)
	require.True(t, ok)
	require.NotNil(t, entry)
	assert.Equal(t, "/users/:id", route)
	assert.Equal(t, "GET", entry.method)
}

func TestNormalizeLogLine_NoMatch(t *testing.T) {
	rules, err := newPathRules(`[{"host":"api.example.com","path":"^/users/[0-9]+$","route":"/users/:id"}]`)
	require.NoError(t, err)

	line := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 192.168.1.100:57832 10.0.1.1:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/health HTTP/1.1" "Mozilla/5.0" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`

	entry, route, ok := normalizeLogLine(line, rules)
	assert.False(t, ok)
	assert.Nil(t, entry)
	assert.Equal(t, "", route)
}

func TestNormalizeLogLine_ParseError(t *testing.T) {
	rules, err := newPathRules(`[{"host":"api.example.com","path":"^/users/[0-9]+$","route":"/users/:id"}]`)
	require.NoError(t, err)

	entry, route, ok := normalizeLogLine("invalid", rules)
	assert.False(t, ok)
	assert.Nil(t, entry)
	assert.Equal(t, "", route)
}
