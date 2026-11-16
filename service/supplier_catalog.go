package service

import (
	"errors"
	"github.com/Chaoteen/quinta-ai-gateway/constant"
	"github.com/Chaoteen/quinta-ai-gateway/model"
	"net/url"
	"strings"
)

type SupplierProvider struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Docs        string   `json:"docs"`
	Partnership string   `json:"partnership"`
	Endpoint    string   `json:"endpoint"`
	ChannelType int      `json:"channel_type"`
	Guidance    string   `json:"guidance"`
	Hosts       []string `json:"-"`
}

var SupplierProviders = []SupplierProvider{
	{"volcengine", "火山引擎 / 豆包 / Seedance", "https://docs.volcengine.com/docs/ark/api-key", "https://partner.volcengine.com/cooperate/RETAIL", "https://ark.cn-beijing.volces.com", constant.ChannelTypeVolcEngine, "项目空间、API Key资源ID、Model ID或推理接入点须匹配；订阅套餐不可自动用于转售，分销需产品线及合同确认。", []string{"ark.cn-beijing.volces.com"}},
	{"aliyun", "阿里云百炼 / Qwen", "https://www.alibabacloud.com/help/en/model-studio/what-is-model-studio", "https://partner.aliyun.com/programs/reseller_P", "https://dashscope.aliyuncs.com", constant.ChannelTypeAli, "按控制台选择地域与Workspace；填写真实Key资源ID，不填写Key明文；伙伴资质不等于所有模型均获转售许可。", []string{"dashscope.aliyuncs.com", "dashscope-intl.aliyuncs.com", "dashscope-us.aliyuncs.com", "*.maas.aliyuncs.com"}},
	{"minimax", "MiniMax", "https://platform.minimax.io/docs/api-reference/text-openai-api.md", "https://platform.minimaxi.com/zh/cooperate", "https://api.minimax.io", constant.ChannelTypeOpenAI, "国际官方文档提供OpenAI兼容API；中国站域名/地域与账号权限须按控制台另行确认，不能混用订阅与API凭据。", []string{"api.minimax.io"}},
	{"moonshot", "Moonshot / Kimi", "https://www.kimi.com/help/kimi-api/api-troubleshooting", "https://platform.kimi.com/contact-sales", "https://api.moonshot.cn", constant.ChannelTypeMoonshot, "中国与国际站账号、余额和Key独立；Kimi Code、会员与按量API不是同一产品；转售范围需商务书面确认。", []string{"api.moonshot.cn", "api.moonshot.ai"}},
	{"deepseek", "DeepSeek", "https://api-docs.deepseek.com/", "https://api-docs.deepseek.com/", "https://api.deepseek.com", constant.ChannelTypeDeepSeek, "官方提供API和余额查询；本次未确认公开分销开户API，使用官网商务联系入口确认合同范围，不自动认定允许转售。", []string{"api.deepseek.com"}},
}

func supplierProvider(id string) (SupplierProvider, error) {
	for _, p := range SupplierProviders {
		if p.Id == id {
			return p, nil
		}
	}
	return SupplierProvider{}, errors.New("unknown supplier provider")
}
func validateSupplierChannel(provider string, channel *model.Channel) error {
	p, err := supplierProvider(provider)
	if err != nil {
		return err
	}
	if channel.Type != p.ChannelType && !(provider == "volcengine" && channel.Type == constant.ChannelTypeDoubaoVideo) {
		return errors.New("channel protocol does not match provider")
	}
	raw := constant.ChannelBaseURLs[channel.Type]
	if channel.BaseURL != nil && *channel.BaseURL != "" {
		raw = *channel.BaseURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("use an official HTTPS endpoint without credentials or query")
	}
	host := strings.ToLower(u.Hostname())
	allowed := false
	for _, h := range p.Hosts {
		if host == h || (strings.HasPrefix(h, "*.") && strings.HasSuffix(host, h[1:]) && host != h[2:]) {
			allowed = true
		}
	}
	if !allowed {
		return errors.New("channel endpoint has not been verified as an official supplier endpoint")
	}
	path := strings.ToLower(u.Path)
	if strings.Contains(path, "coding") || strings.Contains(path, "/plan") {
		return errors.New("subscription endpoints cannot be bound to this API procurement account")
	}
	if u.Path != "" && u.Path != "/" {
		return errors.New("channel base must be the HTTPS origin; the adapter appends API paths")
	}
	return nil
}
