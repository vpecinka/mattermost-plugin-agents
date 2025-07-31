// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package prompts

// Language represents a supported language
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// SupportedLanguages contains all languages supported by the prompts system
var SupportedLanguages = []Language{
	{Code: "en", Name: "English"},
	{Code: "cz", Name: "Česky (Czech)"},
}

// DefaultLanguage is the fallback language when no specific language is configured
const DefaultLanguage = "en"

// IsValidLanguage checks if the given language code is supported
func IsValidLanguage(code string) bool {
	for _, lang := range SupportedLanguages {
		if lang.Code == code {
			return true
		}
	}
	return false
}

// GetLanguageName returns the display name for a given language code
func GetLanguageName(code string) string {
	for _, lang := range SupportedLanguages {
		if lang.Code == code {
			return lang.Name
		}
	}
	return code // fallback to code if not found
}
