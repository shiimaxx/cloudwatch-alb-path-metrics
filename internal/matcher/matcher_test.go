package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	routes := []Route{
		{Method: "GET", Pattern: "/api/v1/users/:id", Name: "GetUser"},
	}
	m := New(routes)

	name, found := m.Match("GET", "/api/v1/users/123")
	assert.Equal(t, "GetUser", name)
	assert.True(t, found)
}
