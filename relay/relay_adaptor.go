package relay

import (
	"strconv"

	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/ali"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/aws"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/baidu"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/baidu_v2"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/claude"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/cloudflare"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/codex"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/cohere"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/coze"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/deepseek"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/dify"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/gemini"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/jimeng"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/jina"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/minimax"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/mistral"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/mokaai"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/moonshot"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/ollama"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/openai"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/palm"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/perplexity"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/replicate"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/siliconflow"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/submodel"
	taskali "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/ali"
	taskdoubao "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/doubao"
	taskGemini "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/gemini"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/hailuo"
	taskjimeng "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/jimeng"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/kling"
	tasksora "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/sora"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/suno"
	taskvertex "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/vertex"
	taskVidu "github.com/Chaoteen/quinta-ai-gateway/relay/channel/task/vidu"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/tencent"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/vertex"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/volcengine"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/xai"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/xunfei"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/zhipu"
	"github.com/Chaoteen/quinta-ai-gateway/relay/channel/zhipu_4v"
	"github.com/gin-gonic/gin"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeBaidu:
		return &baidu.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipu:
		return &zhipu.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypePerplexity:
		return &perplexity.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeMokaAI:
		return &mokaai.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeBaiduV2:
		return &baidu_v2.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXinference:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
	case constant.APITypeCoze:
		return &coze.Adaptor{}
	case constant.APITypeJimeng:
		return &jimeng.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{} // Moonshot uses Claude API
	case constant.APITypeSubmodel:
		return &submodel.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	case constant.APITypeReplicate:
		return &replicate.Adaptor{}
	case constant.APITypeCodex:
		return &codex.Adaptor{}
	}
	return nil
}

func GetTaskPlatform(c *gin.Context) constant.TaskPlatform {
	channelType := c.GetInt("channel_type")
	if channelType > 0 {
		return constant.TaskPlatform(strconv.Itoa(channelType))
	}
	return constant.TaskPlatform(c.GetString("platform"))
}

func GetTaskAdaptor(platform constant.TaskPlatform) channel.TaskAdaptor {
	switch platform {
	//case constant.APITypeAIProxyLibrary:
	//	return &aiproxy.Adaptor{}
	case constant.TaskPlatformSuno:
		return &suno.TaskAdaptor{}
	}
	if channelType, err := strconv.ParseInt(string(platform), 10, 64); err == nil {
		switch channelType {
		case constant.ChannelTypeAli:
			return &taskali.TaskAdaptor{}
		case constant.ChannelTypeKling:
			return &kling.TaskAdaptor{}
		case constant.ChannelTypeJimeng:
			return &taskjimeng.TaskAdaptor{}
		case constant.ChannelTypeVertexAi:
			return &taskvertex.TaskAdaptor{}
		case constant.ChannelTypeVidu:
			return &taskVidu.TaskAdaptor{}
		case constant.ChannelTypeDoubaoVideo, constant.ChannelTypeVolcEngine:
			return &taskdoubao.TaskAdaptor{}
		case constant.ChannelTypeSora, constant.ChannelTypeOpenAI:
			return &tasksora.TaskAdaptor{}
		case constant.ChannelTypeGemini:
			return &taskGemini.TaskAdaptor{}
		case constant.ChannelTypeMiniMax:
			return &hailuo.TaskAdaptor{}
		}
	}
	return nil
}
