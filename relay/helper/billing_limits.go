package helper

import (
	"fmt"

	"github.com/Chaoteen/quinta-ai-gateway/dto"
)

const maxRequestOutputTokens = 1 << 24

// Validate before provider conversion or pre-consume casts uint values to int.
// Nil and explicit zero are left intact for the provider's own semantics.
func validateRelayBudget(request dto.Request) error {
	check := func(name string, value *uint) error {
		if value != nil && *value > maxRequestOutputTokens {
			return fmt.Errorf("%s exceeds maximum output budget", name)
		}
		return nil
	}
	switch r := request.(type) {
	case *dto.GeneralOpenAIRequest:
		if err := check("max_tokens", r.MaxTokens); err != nil {
			return err
		}
		return check("max_completion_tokens", r.MaxCompletionTokens)
	case *dto.ClaudeRequest:
		if err := check("max_tokens", r.MaxTokens); err != nil {
			return err
		}
		return check("max_tokens_to_sample", r.MaxTokensToSample)
	case *dto.OpenAIResponsesRequest:
		return check("max_output_tokens", r.MaxOutputTokens)
	case *dto.GeminiChatRequest:
		return check("maxOutputTokens", r.GenerationConfig.MaxOutputTokens)
	}
	return nil
}
