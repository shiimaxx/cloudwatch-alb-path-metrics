package main

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseLogEntry(t *testing.T, entry string) []string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(entry))
	reader.Comma = ' '
	fields, err := reader.Read()
	require.NoError(t, err, "failed to parse log line")
	return fields
}

func TestParseALBLogFields(t *testing.T) {
	getLogEntry := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57832 203.0.113.10:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/users/123 HTTP/1.1" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`
	postLogEntry := `http 2024-01-15T10:00:01.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57833 203.0.113.10:80 0.000 0.002 0.000 201 201 345 1024 "POST http://api.example.com/api/orders HTTP/1.1" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8f api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:01.000000Z forward - - - - - - -`

	tests := []struct {
		name   string
		fields []string
		want   albLogEntry
	}{
		{
			name:   "basic GET request",
			fields: parseLogEntry(t, getLogEntry),
			want: albLogEntry{
				timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				method:    "GET",
				host:      "api.example.com",
				path:      "/users/123",
				status:    200,
				duration:  0.001,
			},
		},
		{
			name:   "POST request with different path",
			fields: parseLogEntry(t, postLogEntry),
			want: albLogEntry{
				timestamp: time.Date(2024, 1, 15, 10, 0, 1, 0, time.UTC),
				method:    "POST",
				host:      "api.example.com",
				path:      "/api/orders",
				status:    201,
				duration:  0.002,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseALBLogFields(tt.fields)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.timestamp, got.timestamp)
			assert.Equal(t, tt.want.method, got.method)
			assert.Equal(t, tt.want.host, got.host)
			assert.Equal(t, tt.want.path, got.path)
			assert.Equal(t, tt.want.status, got.status)
			assert.Equal(t, tt.want.duration, got.duration)
		})
	}
}

func TestParseALBLogLine(t *testing.T) {
	line := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57832 203.0.113.10:80 0.000 0.001 0.000 200 200 218 587 "GET http://api.example.com/users/123 HTTP/1.1" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`

	entry, err := parseALBLogLine(line)
	assert.NoError(t, err)

	assert.Equal(t, "GET", entry.method)
	assert.Equal(t, "api.example.com", entry.host)
	assert.Equal(t, "/users/123", entry.path)
	assert.Equal(t, 200, entry.status)
	assert.InEpsilon(t, 0.001, entry.duration, 1e-9)
}
