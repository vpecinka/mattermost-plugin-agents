// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"errors"
)

type Prompts struct {
	templates       map[string]*template.Template // language -> templates mapping
	defaultLanguage string
}

const PromptExtension = "tmpl"

func NewPrompts(input fs.FS) (*Prompts, error) {
	// Create a map to hold templates for each language
	templatesMap := make(map[string]*template.Template)

	// Scan for language directories
	entries, err := fs.ReadDir(input, ".")
	if err != nil {
		return nil, fmt.Errorf("unable to read prompts directory: %w", err)
	}

	defaultLang := "en"
	for _, entry := range entries {
		if entry.IsDir() {
			langCode := entry.Name()

			// Load templates for this language
			pattern := fmt.Sprintf("%s/*.tmpl", langCode)
			templates, err := template.ParseFS(input, pattern)
			if err != nil {
				return nil, fmt.Errorf("unable to parse prompt templates for language %s: %w", langCode, err)
			}
			templatesMap[langCode] = templates
		}
	}

	// Fallback: if no language directories found, try to load from root (backward compatibility)
	if len(templatesMap) == 0 {
		templates, err := template.ParseFS(input, "*.tmpl")
		if err != nil {
			return nil, fmt.Errorf("unable to parse prompt templates: %w", err)
		}
		templatesMap[defaultLang] = templates
	}

	return &Prompts{
		templates:       templatesMap,
		defaultLanguage: defaultLang,
	}, nil
}

func withPromptExtension(filename string) string {
	return filename + "." + PromptExtension
}

func (p *Prompts) FormatString(templateCode string, context *Context) (string, error) {
	// Use the language from context, fallback to default
	lang := p.getLanguageFromContext(context)
	templates := p.getTemplatesForLanguage(lang)

	template, err := templates.Clone()
	if err != nil {
		return "", err
	}

	template, err = template.Parse(templateCode)
	if err != nil {
		return "", err
	}

	out := &strings.Builder{}
	if err := template.Execute(out, context); err != nil {
		return "", fmt.Errorf("unable to execute template: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

func (p *Prompts) Format(templateName string, context *Context) (string, error) {
	lang := p.getLanguageFromContext(context)
	templates := p.getTemplatesForLanguage(lang)

	tmpl := templates.Lookup(withPromptExtension(templateName))
	if tmpl == nil {
		return "", errors.New("template not found")
	}

	return p.execute(tmpl, context)
}

// getLanguageFromContext extracts the language preference from the context
func (p *Prompts) getLanguageFromContext(context *Context) string {
	// Try to get language from bot configuration first
	if context != nil && context.BotLanguage != "" {
		return context.BotLanguage
	}
	// Fallback to default
	return p.defaultLanguage
}

// getTemplatesForLanguage returns templates for the specified language, with fallback
func (p *Prompts) getTemplatesForLanguage(lang string) *template.Template {
	if templates, exists := p.templates[lang]; exists {
		return templates
	}
	// Fallback to default language
	if templates, exists := p.templates[p.defaultLanguage]; exists {
		return templates
	}
	// This should not happen if properly initialized, but return the first available
	for _, templates := range p.templates {
		return templates
	}
	return nil
}

func (p *Prompts) execute(template *template.Template, data *Context) (string, error) {
	out := &strings.Builder{}
	if err := template.Execute(out, data); err != nil {
		return "", fmt.Errorf("unable to execute template: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}
