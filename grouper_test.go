package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrouperGroupPath(t *testing.T) {
	tests := []struct {
		name    string
		regexes string
		path    string
		want    string
	}{
		{
			name:    "match user ID pattern",
			regexes: `/api/users/\d+`,
			path:    "/api/users/123",
			want:    `/api/users/\d+`,
		},
		{
			name:    "match order ID pattern",
			regexes: `/api/orders/\d+`,
			path:    "/api/orders/456",
			want:    `/api/orders/\d+`,
		},
		{
			name:    "match product slug pattern",
			regexes: `/api/products/[^/]+`,
			path:    "/api/products/abc-def",
			want:    `/api/products/[^/]+`,
		},
		{
			name:    "no match - return original path",
			regexes: `/api/users/\d+`,
			path:    "/api/health",
			want:    "/api/health",
		},
		{
			name:    "multiple patterns - first match wins",
			regexes: `/api/users/\d+,/api/orders/\d+,/api/products/[^/]+`,
			path:    "/api/users/123",
			want:    `/api/users/\d+`,
		},
		{
			name:    "multiple patterns - second pattern matches",
			regexes: `/api/users/\d+,/api/orders/\d+,/api/products/[^/]+`,
			path:    "/api/orders/456",
			want:    `/api/orders/\d+`,
		},
		{
			name:    "multiple patterns - third pattern matches",
			regexes: `/api/users/\d+,/api/orders/\d+,/api/products/[^/]+`,
			path:    "/api/products/abc-def-ghi",
			want:    `/api/products/[^/]+`,
		},
		{
			name:    "multiple patterns - no match",
			regexes: `/api/users/\d+,/api/orders/\d+,/api/products/[^/]+`,
			path:    "/api/health",
			want:    "/api/health",
		},
		{
			name:    "complex path with query parameters",
			regexes: `/api/users/\d+`,
			path:    "/api/users/123?include=profile",
			want:    `/api/users/\d+`,
		},
		{
			name:    "nested resource pattern",
			regexes: `/api/users/\d+/orders/\d+`,
			path:    "/api/users/123/orders/456",
			want:    `/api/users/\d+/orders/\d+`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grouper, err := newGrouper(tt.regexes)
			if err != nil {
				t.Fatalf("failed to create grouper: %v", err)
			}

			got := grouper.groupPath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGrouperGroupPath_EmptyRegexes(t *testing.T) {
	path := "/api/users/123"

	grouper, err := newGrouper("")
	if err != nil {
		t.Fatalf("failed to create grouper: %v", err)
	}

	got := grouper.groupPath(path)
	assert.Equal(t, path, got)
}

func TestNewGrouper_InvalidRegex(t *testing.T) {
	tests := []struct {
		name    string
		regexes string
	}{
		{
			name:    "invalid regex - unclosed bracket",
			regexes: `/api/users/[`,
		},
		{
			name:    "invalid regex - unclosed parentheses",
			regexes: `/api/users/(`,
		},
		{
			name:    "invalid regex - invalid escape sequence",
			regexes: `/api/users/\k`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newGrouper(tt.regexes)
			assert.Error(t, err)
		})
	}
}
