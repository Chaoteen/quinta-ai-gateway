package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityProductionOutboundBlocksPrivateDestinations(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	for _, address := range []string{"127.0.0.1:80", "169.254.169.254:80", "10.0.0.1:443", "[::1]:443", "[::ffff:127.0.0.1]:443"} {
		conn, err := dialOutbound(context.Background(), "tcp", address)
		require.Error(t, err, address)
		require.Nil(t, conn)
	}
	_, err := NewProxyHttpClient("http://127.0.0.1:8080")
	require.Error(t, err)
}
