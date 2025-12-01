package telegram

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// Embedded telegram bot templates from pkg/templates/telegram/
//
//go:embed templates/**/*.tmpl
var embeddedTemplatesFS embed.FS

// TemplateRegistry manages message templates for telegram bot
type TemplateRegistry struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	fs        fs.FS
	basePath  string
}

// NewTemplateRegistry creates a new template registry from filesystem
func NewTemplateRegistry(filesystem fs.FS, basePath string) (*TemplateRegistry, error) {
	r := &TemplateRegistry{
		templates: make(map[string]*template.Template),
		fs:        filesystem,
		basePath:  basePath,
	}

	// Load all templates
	if err := r.loadAll(); err != nil {
		return nil, err
	}

	return r, nil
}

// NewEmbeddedTemplateRegistry creates registry from embedded filesystem
func NewEmbeddedTemplateRegistry(embeddedFS embed.FS, subPath string) (*TemplateRegistry, error) {
	subFS, err := fs.Sub(embeddedFS, subPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub filesystem: %w", err)
	}

	return NewTemplateRegistry(subFS, subPath)
}

// NewDefaultTemplateRegistry creates registry with embedded telegram templates
// This is the default way to initialize templates for telegram bots
func NewDefaultTemplateRegistry() (*TemplateRegistry, error) {
	subFS, err := fs.Sub(embeddedTemplatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("failed to access embedded templates: %w", err)
	}

	return NewTemplateRegistry(subFS, "templates")
}

// Register manually registers a template
func (r *TemplateRegistry) Register(name, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl, err := template.New(name).Funcs(r.getFuncMap()).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	r.templates[name] = tmpl
	return nil
}

// Render renders a template by name with data
func (r *TemplateRegistry) Render(name string, data interface{}) (string, error) {
	r.mu.RLock()
	tmpl, exists := r.templates[name]
	r.mu.RUnlock()

	if !exists {
		// Try lazy loading
		if err := r.loadTemplate(name + ".tmpl"); err != nil {
			return "", fmt.Errorf("template not found: %s", name)
		}
		r.mu.RLock()
		tmpl = r.templates[name]
		r.mu.RUnlock()
	}

	if tmpl == nil {
		return "", fmt.Errorf("template not found: %s", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// MustRender renders template or returns error message
func (r *TemplateRegistry) MustRender(name string, data interface{}) string {
	result, err := r.Render(name, data)
	if err != nil {
		return fmt.Sprintf("❌ Template error: %v", err)
	}
	return result
}

// Exists checks if template exists
func (r *TemplateRegistry) Exists(name string) bool {
	r.mu.RLock()
	_, exists := r.templates[name]
	r.mu.RUnlock()
	return exists
}

// List returns all registered template names
func (r *TemplateRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// Reload reloads all templates from filesystem
func (r *TemplateRegistry) Reload() error {
	r.mu.Lock()
	r.templates = make(map[string]*template.Template)
	r.mu.Unlock()

	return r.loadAll()
}

// loadAll loads all .tmpl files from filesystem
func (r *TemplateRegistry) loadAll() error {
	if r.fs == nil {
		return nil // No filesystem, templates registered manually
	}

	return fs.WalkDir(r.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".tmpl" {
			return nil
		}

		return r.loadTemplate(path)
	})
}

// loadTemplate loads a single template file
func (r *TemplateRegistry) loadTemplate(path string) error {
	content, err := fs.ReadFile(r.fs, path)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", path, err)
	}

	// Convert path to template name (remove .tmpl extension)
	name := r.pathToName(path)

	tmpl, err := template.New(name).Funcs(r.getFuncMap()).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	r.mu.Lock()
	r.templates[name] = tmpl
	r.mu.Unlock()

	return nil
}

// pathToName converts file path to template name
func (r *TemplateRegistry) pathToName(path string) string {
	// Normalize path separators
	normalized := filepath.ToSlash(path)
	// Remove leading slash
	normalized = strings.TrimPrefix(normalized, "/")
	// Remove .tmpl extension
	return strings.TrimSuffix(normalized, ".tmpl")
}

// getFuncMap returns template function map with telegram-specific helpers
func (r *TemplateRegistry) getFuncMap() template.FuncMap {
	return template.FuncMap{
		// String operations
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"title":   strings.Title,
		"trim":    strings.TrimSpace,
		"replace": strings.ReplaceAll,
		"join":    strings.Join,
		"split":   strings.Split,

		// Math operations
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b != 0 {
				return a / b
			}
			return 0
		},
		"mod": func(a, b int) int {
			if b != 0 {
				return a % b
			}
			return 0
		},

		// Type conversions
		"int": func(v interface{}) int {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case float32:
				return int(val)
			case string:
				var i int
				fmt.Sscanf(val, "%d", &i)
				return i
			default:
				return 0
			}
		},
		"float": func(v interface{}) float64 {
			switch val := v.(type) {
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case float64:
				return val
			case float32:
				return float64(val)
			case string:
				var f float64
				fmt.Sscanf(val, "%f", &f)
				return f
			default:
				return 0
			}
		},
		"string": func(v interface{}) string {
			return fmt.Sprintf("%v", v)
		},

		// Formatting
		"printf":  fmt.Sprintf,
		"money":   func(amount float64) string { return fmt.Sprintf("$%.2f", amount) },
		"percent": func(value float64) string { return fmt.Sprintf("%.1f%%", value*100) },
		"duration": func(d time.Duration) string {
			if d < time.Minute {
				return fmt.Sprintf("%ds", int(d.Seconds()))
			}
			if d < time.Hour {
				return fmt.Sprintf("%dm", int(d.Minutes()))
			}
			return fmt.Sprintf("%.1fh", d.Hours())
		},
		"date":     func(t time.Time) string { return t.Format("2006-01-02") },
		"datetime": func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },
		"time":     func(t time.Time) string { return t.Format("15:04:05") },

		// Telegram Markdown formatting
		"bold":   func(s string) string { return fmt.Sprintf("*%s*", s) },
		"italic": func(s string) string { return fmt.Sprintf("_%s_", s) },
		"code":   func(s string) string { return fmt.Sprintf("`%s`", s) },
		"pre":    func(s string) string { return fmt.Sprintf("```\n%s\n```", s) },
		"link": func(text, url string) string {
			return fmt.Sprintf("[%s](%s)", text, url)
		},

		// Markdown escaping
		"escape":     EscapeMarkdown,
		"escapeV2":   EscapeMarkdownV2,
		"escapeCode": EscapeMarkdownV2Code,
		"escapeLink": EscapeMarkdownV2Link,
		"safeText":   SafeText,
		"safeTextV2": SafeTextV2,

		// Emojis (common shortcuts)
		"checkmark": func() string { return "✅" },
		"cross":     func() string { return "❌" },
		"warning":   func() string { return "⚠️" },
		"info":      func() string { return "ℹ️" },
		"rocket":    func() string { return "🚀" },
		"chart":     func() string { return "📊" },
		"fire":      func() string { return "🔥" },
		"clock":     func() string { return "⏰" },
		"hourglass": func() string { return "⏳" },

		// Logic helpers
		"eq":  func(a, b interface{}) bool { return a == b },
		"ne":  func(a, b interface{}) bool { return a != b },
		"lt":  func(a, b float64) bool { return a < b },
		"le":  func(a, b float64) bool { return a <= b },
		"gt":  func(a, b float64) bool { return a > b },
		"ge":  func(a, b float64) bool { return a >= b },
		"and": func(a, b bool) bool { return a && b },
		"or":  func(a, b bool) bool { return a || b },
		"not": func(a bool) bool { return !a },

		// Collection helpers
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case string:
				return len(val)
			case []interface{}:
				return len(val)
			case map[string]interface{}:
				return len(val)
			default:
				return 0
			}
		},
		"empty": func(v interface{}) bool {
			if v == nil {
				return true
			}
			switch val := v.(type) {
			case string:
				return val == ""
			case []interface{}:
				return len(val) == 0
			case map[string]interface{}:
				return len(val) == 0
			default:
				return false
			}
		},
		"first": func(v []interface{}) interface{} {
			if len(v) > 0 {
				return v[0]
			}
			return nil
		},
		"last": func(v []interface{}) interface{} {
			if len(v) > 0 {
				return v[len(v)-1]
			}
			return nil
		},

		// Default value helper
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
	}
}

// Markdown escaping functions (telegram-specific)

// EscapeMarkdown escapes special characters for Telegram Markdown format
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// EscapeMarkdownV2 escapes special characters for Telegram MarkdownV2 format
func EscapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", // Backslash must be escaped first
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// EscapeMarkdownV2Code escapes special characters inside code/pre blocks
func EscapeMarkdownV2Code(code string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
	)
	return replacer.Replace(code)
}

// EscapeMarkdownV2Link escapes special characters inside inline link URLs
func EscapeMarkdownV2Link(url string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		")", "\\)",
	)
	return replacer.Replace(url)
}

// SafeText sanitizes text for safe use in Telegram messages
func SafeText(text string) string {
	text = strings.ToValidUTF8(text, "")
	return EscapeMarkdown(text)
}

// SafeTextV2 sanitizes text for safe use in Telegram MarkdownV2 messages
func SafeTextV2(text string) string {
	text = strings.ToValidUTF8(text, "")
	return EscapeMarkdownV2(text)
}
