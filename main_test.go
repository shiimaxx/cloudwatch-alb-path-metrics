package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCloudWatchNamespace(t *testing.T) {
	tests := []struct {
		name    string
		ns      string
		wantErr bool
	}{
		{name: "valid", ns: "MyApp/Metrics", wantErr: false},
		{name: "valid with space", ns: "Custom Namespace", wantErr: false},
		{name: "empty", ns: "", wantErr: true},
		{name: "too long", ns: strings.Repeat("a", 256), wantErr: true},
		{name: "reserved prefix", ns: "AWS/Custom", wantErr: true},
		{name: "non ascii", ns: "メトリクス", wantErr: true},
		{name: "control char", ns: "metric\nname", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCloudWatchNamespace(tc.ns)
			if tc.wantErr {
				assert.Error(t, err, "expected error for namespace %q", tc.ns)
				return
			}
			assert.NoError(t, err, "unexpected error for namespace %q", tc.ns)
		})
	}
}
