package jj

import (
	"embed"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var templatesFS embed.FS

// Templates holds the loaded template strings passed as the -T argument for jj commands.
type Templates struct {
	templates map[string]string
}

// NewTemplates loads templates from the embedded filesystem and returns a Templates instance.
func NewTemplates() *Templates {
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		panic("failed to read templates: " + err.Error())
	}

	templates := make(map[string]string, len(entries))

	for _, entry := range entries {
		data, err := templatesFS.ReadFile("templates/" + entry.Name())
		if err != nil {
			panic("failed to read template file: " + err.Error())
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		templates[name] = strings.TrimSpace(string(data))
	}

	return &Templates{templates: templates}
}

// Get returns the template string for the given name, or panics if not found.
func (t *Templates) Get(name string) string {
	if tmpl, ok := t.templates[name]; ok {
		return tmpl
	}

	panic("template not found: " + name)
}
