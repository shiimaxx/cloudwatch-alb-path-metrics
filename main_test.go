package main

import (
	"strings"
	"testing"
)

func TestValidateCloudWatchNamespace(t *testing.T) {
	tests := []struct {
		name string
		ns   string
		want bool
	}{
		{name: "valid", ns: "MyApp/Metrics", want: false},
		{name: "valid with space", ns: "Custom Namespace", want: false},
		{name: "empty", ns: "", want: true},
		{name: "too long", ns: strings.Repeat("a", 256), want: true},
		{name: "reserved prefix", ns: "AWS/Custom", want: true},
		{name: "non ascii", ns: "メトリクス", want: true},
		{name: "control char", ns: "metric\nname", want: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateCloudWatchNamespace(tc.ns)
			if tc.want {
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
