package builder

import (
	"slices"
	"strings"
)

// LLM represents an LLM provider and model
// <provider>::<model>
// in the convention of the provider
type LLM string
type LLMValidateError string

const (
	InvalidFormatError   LLMValidateError = "Invalid format. Expected <provider>::<model> got %s"
	InvalidProviderError LLMValidateError = "Invalid provider. Expected one of %s got %s"
)

var validProviders = []string{"openai", "deepseek", "togetherai", "openrouter"}

func fromString(model string) LLM {
	return LLM(model)
}

func (l LLM) provider() string {
	segments := strings.Split(string(l), "::")
	if len(segments) != 2 {
		return string(InvalidFormatError) + string(l)
	}
	provider := segments[0]
	if !slices.Contains(validProviders, provider) {
		return string(InvalidProviderError) + provider
	}
	return provider
}

func (l LLM) model() string {
	segments := strings.Split(string(l), "::")
	if len(segments) != 2 {
		return string(InvalidFormatError) + string(l)
	}
	return segments[1]
}
