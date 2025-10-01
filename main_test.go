package main

import (
	"strings"
	"testing"
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateCloudWatchNamespace(tc.ns)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for namespace %q", tc.ns)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for namespace %q: %v", tc.ns, err)
			}
		})
	}
}
