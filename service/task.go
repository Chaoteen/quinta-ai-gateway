package service

import (
	"strings"

	"github.com/Chaoteen/quinta-ai-gateway/constant"
)

func CoverTaskActionToModelName(platform constant.TaskPlatform, action string) string {
	return strings.ToLower(string(platform)) + "_" + strings.ToLower(action)
}
