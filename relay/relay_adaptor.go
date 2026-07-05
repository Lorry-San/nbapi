package relay

import (
	"strconv"

	"github.com/Lorry-San/nbapi/constant"
	"github.com/Lorry-San/nbapi/relay/channel"
	"github.com/Lorry-San/nbapi/relay/channel/advancedcustom"
	"github.com/Lorry-San/nbapi/relay/channel/ali"
	"github.com/Lorry-San/nbapi/relay/channel/aws"
	"github.com/Lorry-San/nbapi/relay/channel/baidu"
	"github.com/Lorry-San/nbapi/relay/channel/baidu_v2"
	"github.com/Lorry-San/nbapi/relay/channel/claude"
	"github.com/Lorry-San/nbapi/relay/channel/cloudflare"
	"github.com/Lorry-San/nbapi/relay/channel/codex"
	"github.com/Lorry-San/nbapi/relay/channel/cohere"
	"github.com/Lorry-San/nbapi/relay/channel/coze"
	"github.com/Lorry-San/nbapi/relay/channel/deepseek"
	"github.com/Lorry-San/nbapi/relay/channel/dify"
	"github.com/Lorry-San/nbapi/relay/channel/gemini"
	"github.com/Lorry-San/nbapi/relay/channel/jimeng"
	"github.com/Lorry-San/nbapi/relay/channel/jina"
	"github.com/Lorry-San/nbapi/relay/channel/minimax"
	"github.com/Lorry-San/nbapi/relay/channel/mistral"
	"github.com/Lorry-San/nbapi/relay/channel/mokaai"
	"github.com/Lorry-San/nbapi/relay/channel/moonshot"
	"github.com/Lorry-San/nbapi/relay/channel/ollama"
	"github.com/Lorry-San/nbapi/relay/channel/openai"
	"github.com/Lorry-San/nbapi/relay/channel/palm"
	"github.com/Lorry-San/nbapi/relay/channel/perplexity"
	"github.com/Lorry-San/nbapi/relay/channel/replicate"
	"github.com/Lorry-San/nbapi/relay/channel/siliconflow"
	"github.com/Lorry-San/nbapi/relay/channel/submodel"
	taskali "github.com/Lorry-San/nbapi/relay/channel/task/ali"
	taskdoubao "github.com/Lorry-San/nbapi/relay/channel/task/doubao"
	taskGemini "github.com/Lorry-San/nbapi/relay/channel/task/gemini"
	"github.com/Lorry-San/nbapi/relay/channel/task/hailuo"
	taskjimeng "github.com/Lorry-San/nbapi/relay/channel/task/jimeng"
	"github.com/Lorry-San/nbapi/relay/channel/task/kling"
	tasksora "github.com/Lorry-San/nbapi/relay/channel/task/sora"
	"github.com/Lorry-San/nbapi/relay/channel/task/suno"
	taskvertex "github.com/Lorry-San/nbapi/relay/channel/task/vertex"
	taskVidu "github.com/Lorry-San/nbapi/relay/channel/task/vidu"
	"github.com/Lorry-San/nbapi/relay/channel/tencent"
	"github.com/Lorry-San/nbapi/relay/channel/vertex"
	"github.com/Lorry-San/nbapi/relay/channel/volcengine"
	"github.com/Lorry-San/nbapi/relay/channel/xai"
	"github.com/Lorry-San/nbapi/relay/channel/xunfei"
	"github.com/Lorry-San/nbapi/relay/channel/zhipu"
	"github.com/Lorry-San/nbapi/relay/channel/zhipu_4v"
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
	case constant.APITypeAdvancedCustom:
		return &advancedcustom.Adaptor{}
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
