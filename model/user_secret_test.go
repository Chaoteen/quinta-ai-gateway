package model

import (
	"testing"

	"github.com/Chaoteen/quinta-ai-gateway/common"
	"github.com/stretchr/testify/require"
)

func TestSecurityUserSerializationExcludesManagementCredential(t *testing.T) {
	fixture := "synthetic-management-credential"
	data, err := common.Marshal(User{Id: 1, Username: "fixture", AccessToken: &fixture})
	require.NoError(t, err)
	require.NotContains(t, string(data), fixture)
	require.NotContains(t, string(data), "access_token")
}
