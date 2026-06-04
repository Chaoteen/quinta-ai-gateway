package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseModelQuotaSnapshotEmptyIsUnrestricted(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		parsed, err := ParseModelQuotaSnapshot(raw)
		require.NoError(t, err)
		require.True(t, parsed.Unrestricted)
		require.Empty(t, parsed.Allow)
	}
}

func TestParseModelQuotaSnapshotAllowlist(t *testing.T) {
	parsed, err := ParseModelQuotaSnapshot(`{"allow":["gpt-4o"," gpt-4o-mini ","gpt-4o",""]}`)
	require.NoError(t, err)
	require.False(t, parsed.Unrestricted)
	require.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, parsed.Allow)
}

func TestParseModelQuotaSnapshotInvalidJSON(t *testing.T) {
	_, err := ParseModelQuotaSnapshot(`{"allow":`)
	require.Error(t, err)
}
