package service

import (
	"github.com/Chaoteen/quinta-ai-gateway/setting/system_setting"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWorkerRejectsProductionAndMalformedTargets(t *testing.T) {
	old := system_setting.WorkerUrl
	system_setting.WorkerUrl = "https://worker.example.invalid"
	t.Cleanup(func() { system_setting.WorkerUrl = old })
	t.Setenv("APP_ENV", "production")
	r, err := DoWorkerRequest(&WorkerRequest{URL: "https://public.example.invalid/image"})
	require.Error(t, err)
	require.Nil(t, r)
	t.Setenv("APP_ENV", "development")
	for _, target := range []string{"https-evil://example.invalid", "file:///etc/passwd", "https://user:pass@example.invalid/image", "https:///missing-host"} {
		r, err := DoWorkerRequest(&WorkerRequest{URL: target})
		require.Error(t, err)
		require.Nil(t, r)
	}
	r, err = DoWorkerRequest(nil)
	require.Error(t, err)
	require.Nil(t, r)
}
