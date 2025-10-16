package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
