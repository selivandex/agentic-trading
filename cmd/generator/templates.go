package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// generateFile creates a file from template
func generateFile(filePath string, tmpl string, data interface{}, dryRun bool) error {
	// Parse template
	t, err := template.New("file").Funcs(templateFuncs()).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if dryRun {
		fmt.Printf("Would create: %s\n", filePath)
		return nil
	}

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// templateFuncs returns template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toLower":     strings.ToLower,
		"toUpper":     strings.ToUpper,
		"toTitle":     toTitle,
		"toCamel":     toCamelCase,
		"toPascal":    toPascalCase,
		"toSnake":     toSnakeCase,
		"toKebab":     toKebabCase,
		"pluralize":   pluralize,
		"singularize": singularize,
		"join":        strings.Join,
		"replace":     strings.ReplaceAll,
		"trimSuffix":  strings.TrimSuffix,
		"trimPrefix":  strings.TrimPrefix,
	}
}

// String conversion helpers
func toCamelCase(s string) string {
	// Handle PascalCase input (Agent -> agent)
	if !strings.Contains(s, "_") && len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' {
		return strings.ToLower(s[:1]) + s[1:]
	}

	// Handle snake_case input (user_name -> userName)
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(s), "_", "-")
}

func pluralize(s string) string {
	// Simple pluralization rules
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func singularize(s string) string {
	// Simple singularization rules
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
