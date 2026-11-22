package service

import (
	"context"
	"net/http"
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

func TestProductionTransportsRejectProxyBypass(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8080")
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	InitHttpClient()
	tr := GetHttpClient().Transport.(*http.Transport)
	require.Nil(t, tr.Proxy)
	d := NewOutboundWebsocketDialer()
	require.Nil(t, d.Proxy)
	conn, err := d.NetDialContext(context.Background(), "tcp", "169.254.169.254:80")
	require.Error(t, err)
	require.Nil(t, conn)
}
