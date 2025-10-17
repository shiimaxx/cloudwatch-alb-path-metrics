package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseALBLogLine(t *testing.T) {
	getLogEntry := `http 2024-01-15T10:00:00.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57832 203.0.113.10:80 0.002 0.003 0.004 200 200 218 587 "GET http://api.example.com/users/123 HTTP/1.1" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8e api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:00.000000Z forward - - - - - - -`
	postLogEntry := `http 2024-01-15T10:00:01.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57833 203.0.113.10:80 0.001 0.005 0.002 201 201 345 1024 "POST http://api.example.com/api/orders HTTP/1.1" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d8f api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:01.000000Z forward - - - - - - -`
	timeoutLogEntry := `http 2024-01-15T10:00:02.000000Z app/my-loadbalancer/50dc6c495c0c9188 198.51.100.100:57834 203.0.113.10:80 -1.0 -1.0 0.002 504 504 512 256 "GET http://api.example.com/api/timeout HTTP/1.1" "curl/7.64.1" - - arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/73e2d6bc24d8a067 Root=1-65a5b7e0-4f2d8c9a7b1e3f4a5b6c7d90 api.example.com arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 0 2024-01-15T10:00:02.000000Z forward - - - - - - -`

	tests := []struct {
		name  string
		entry string
		want  albLogEntry
	}{
		{
			name:  "basic GET request",
			entry: getLogEntry,
			want: albLogEntry{
				timestamp: parseTime(t, "2024-01-15T10:00:00Z"),
				method:    "GET",
				host:      "api.example.com",
				path:      "/users/123",
				status:    200,
				targetProcessingTime: 0.003,
			},
		},
		{
			name:  "POST request with different path",
			entry: postLogEntry,
			want: albLogEntry{
				timestamp: parseTime(t, "2024-01-15T10:00:01Z"),
				method:    "POST",
				host:      "api.example.com",
				path:      "/api/orders",
				status:    201,
				targetProcessingTime: 0.005,
			},
		},
		{
			name:  "negative processing times remain unchanged",
			entry: timeoutLogEntry,
			want: albLogEntry{
				timestamp: parseTime(t, "2024-01-15T10:00:02Z"),
				method:    "GET",
				host:      "api.example.com",
				path:      "/api/timeout",
				status:    504,
				targetProcessingTime: -1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseALBLogLine(tt.entry)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.timestamp, got.timestamp)
			assert.Equal(t, tt.want.method, got.method)
			assert.Equal(t, tt.want.host, got.host)
			assert.Equal(t, tt.want.path, got.path)
			assert.Equal(t, tt.want.status, got.status)
			assert.InDelta(t, tt.want.targetProcessingTime, got.targetProcessingTime, 1e-9)
		})
	}
}
