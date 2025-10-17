package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLogLine_Match(t *testing.T) {
	rules, err := NewPathRules(`[{"host":"api.example.com","pattern":"^/users/[0-9]+$","name":"/users/:id"}]`)
	require.NoError(t, err)
	processor := &MetricsProcessor{rules: rules}

	line := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57832 203.0.113.10:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/users/123 HTTP/1.1" "Mozilla/5.0" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`

	entry, name, ok := processor.normalizeLogLine(line)

	assert.NotNil(t, entry)
	assert.Equal(t, "GET", entry.method)
	assert.Equal(t, "/users/:id", name)
	assert.True(t, ok)
}

func TestNormalizeLogLine_NoMatch(t *testing.T) {
	rules, err := NewPathRules(`[{"host":"api.example.com","pattern":"^/users/[0-9]+$","name":"/users/:id"}]`)
	require.NoError(t, err)
	processor := &MetricsProcessor{rules: rules}

	line := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57832 203.0.113.10:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/health HTTP/1.1" "Mozilla/5.0" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`

	entry, name, ok := processor.normalizeLogLine(line)

	assert.Nil(t, entry)
	assert.Equal(t, "", name)
	assert.False(t, ok)
}

func TestNormalizeLogLine_ParseError(t *testing.T) {
	rules, err := NewPathRules(`[{"host":"api.example.com","pattern":"^/users/[0-9]+$","name":"/users/:id"}]`)
	require.NoError(t, err)
	processor := &MetricsProcessor{rules: rules}

	entry, name, ok := processor.normalizeLogLine("invalid")

	assert.Nil(t, entry)
	assert.Equal(t, "", name)
	assert.False(t, ok)
}
