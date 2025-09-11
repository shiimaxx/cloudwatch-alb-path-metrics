package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterEvaluate(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		entry  albLogEntry
		want   bool
	}{
		{
			name:   "filter by method",
			filter: `method == "GET"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/users/123", status: 200},
			want:   true,
		},
		{
			name:   "filter by method - should fail",
			filter: `method == "POST"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/users/123", status: 200},
			want:   false,
		},
		{
			name:   "filter by host",
			filter: `host == "api.example.com"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/users/123", status: 200},
			want:   true,
		},
		{
			name:   "filter by path",
			filter: `path == "/users/123"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/users/123", status: 200},
			want:   true,
		},
		{
			name:   "exclude specific path",
			filter: `path != "/health"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/health", status: 200},
			want:   false,
		},
		{
			name:   "complex filter - AND condition",
			filter: `method == "POST" && host == "api.example.com"`,
			entry:  albLogEntry{method: "POST", host: "api.example.com", path: "/api/orders", status: 201},
			want:   true,
		},
		{
			name:   "complex filter - should fail partial match",
			filter: `method == "POST" && host == "api.example.com"`,
			entry:  albLogEntry{method: "GET", host: "api.example.com", path: "/api/orders", status: 200},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := newFilter(tt.filter)
			if err != nil {
				t.Fatalf("failed to create filter: %v", err)
			}

			got, err := filter.evaluate(tt.entry)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterEvaluate_EmptyFilter(t *testing.T) {
	entry := albLogEntry{method: "GET", host: "api.example.com", path: "/users/123", status: 200}

	filter, err := newFilter("")
	if err != nil {
		t.Fatalf("failed to create filter: %v", err)
	}

	got, err := filter.evaluate(entry)

	assert.NoError(t, err)
	assert.True(t, got)
}
