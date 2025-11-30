package templates

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"prometheus/pkg/errors"
)

//go:embed prompts/**/*.tmpl
var promptsFS embed.FS

//go:embed telegram/**/*.tmpl
var telegramFS embed.FS

//go:embed notifications/*.tmpl
var notificationsFS embed.FS

// Template represents a parsed template loaded from disk.
type Template struct {
	ID      string
	Path    string
	Content string

	parsed *template.Template
}

// Render executes the template with the provided data and returns the result.
func (t *Template) Render(data any) (string, error) {
	var buf bytes.Buffer
	if err := t.parsed.Execute(&buf, data); err != nil {
		return "", errors.Wrapf(err, "render template %s", t.ID)
	}

	return buf.String(), nil
}

// Registry holds loaded templates and resolves them by ID.
type Registry struct {
	basePath  string
	fs        fs.FS
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewRegistry loads all templates from the provided base path.
func NewRegistry(basePath string) (*Registry, error) {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return nil, errors.Wrap(err, "resolve template base path")
	}

	return NewRegistryFromFS(os.DirFS(absBase), absBase)
}

// NewRegistryFromFS constructs a registry from an arbitrary filesystem.
// rootPath is used for deriving template IDs when walking the filesystem.
func NewRegistryFromFS(filesystem fs.FS, rootPath string) (*Registry, error) {
	r := &Registry{
		basePath:  rootPath,
		fs:        filesystem,
		templates: map[string]*Template{},
	}

	if err := r.loadAll(); err != nil {
		return nil, err
	}

	return r, nil
}

// Get returns a lazily initialized default registry rooted at embedded prompts.
func Get() *Registry {
	defaultOnce.Do(func() {
		defaultRegistry, defaultErr = newEmbeddedRegistry()
	})

	if defaultErr != nil {
		panic(defaultErr)
	}

	return defaultRegistry
}

// GetTemplate retrieves a template by its ID.
func (r *Registry) GetTemplate(id string) (*Template, error) {
	r.mu.RLock()
	tmpl, ok := r.templates[id]
	r.mu.RUnlock()

	if ok {
		return tmpl, nil
	}

	// Attempt lazy load in case the template was added after initialization.
	path := filepath.Join(filepath.FromSlash(id) + ".tmpl")
	if _, err := fs.Stat(r.fs, path); err == nil {
		if err := r.loadTemplate(path); err != nil {
			return nil, err
		}
		r.mu.RLock()
		tmpl = r.templates[id]
		r.mu.RUnlock()
		if tmpl != nil {
			return tmpl, nil
		}
	}

	return nil, errors.Wrapf(errors.ErrInternal, "template not found: %s", id)
}

// Render executes a template by ID using the provided data.
func (r *Registry) Render(id string, data any) (string, error) {
	tmpl, err := r.GetTemplate(id)
	if err != nil {
		return "", err
	}

	return tmpl.Render(data)
}

// List returns all known template IDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.templates))
	for id := range r.templates {
		ids = append(ids, id)
	}

	return ids
}

func (r *Registry) loadAll() error {
	return fs.WalkDir(r.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".tmpl" {
			return nil
		}

		return r.loadTemplate(path)
	})
}

func (r *Registry) loadTemplate(path string) error {
	id := r.pathToID(path)
	content, err := fs.ReadFile(r.fs, path)
	if err != nil {
		return errors.Wrapf(err, "read template %s", id)
	}

	// Create template with helper functions
	parsed, err := template.New(id).Funcs(template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"mul":   func(a, b float64) float64 { return a * b },
		"add":   func(a, b float64) float64 { return a + b },
		"sub":   func(a, b float64) float64 { return a - b },
		"div": func(a, b float64) float64 {
			if b != 0 {
				return a / b
			}
			return 0
		},
		"float64": func(v interface{}) float64 {
			switch val := v.(type) {
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case float64:
				return val
			case float32:
				return float64(val)
			default:
				return 0
			}
		},
		// Telegram Markdown helpers
		"escape":     EscapeMarkdown,       // Escape special Markdown characters
		"escapeV2":   EscapeMarkdownV2,     // Escape special MarkdownV2 characters
		"escapeCode": EscapeMarkdownV2Code, // Escape for code blocks
		"escapeLink": EscapeMarkdownV2Link, // Escape for link URLs
		"safeText":   SafeText,             // Sanitize UTF-8 + escape Markdown (use for user input)
		"safeTextV2": SafeTextV2,           // Sanitize UTF-8 + escape MarkdownV2
	}).Parse(string(content))
	if err != nil {
		return errors.Wrapf(err, "parse template %s", id)
	}

	r.mu.Lock()
	r.templates[id] = &Template{
		ID:      id,
		Path:    path,
		Content: string(content),
		parsed:  parsed,
	}
	r.mu.Unlock()

	return nil
}

func (r *Registry) pathToID(rel string) string {
	normalized := filepath.ToSlash(rel)
	normalized = strings.TrimPrefix(normalized, "/")
	return strings.TrimSuffix(normalized, filepath.Ext(normalized))
}

func newEmbeddedRegistry() (*Registry, error) {
	r := &Registry{
		basePath:  "embedded",
		templates: map[string]*Template{},
	}

	// Load prompts (agents, tools, workflows, etc.)
	promptsSubFS, err := fs.Sub(promptsFS, "prompts")
	if err != nil {
		return nil, errors.Wrap(err, "prepare prompts templates")
	}
	r.fs = promptsSubFS
	if err := r.loadAll(); err != nil {
		return nil, errors.Wrap(err, "load prompts templates")
	}

	// Load telegram templates
	r.fs = telegramFS
	if err := r.loadAll(); err != nil {
		return nil, errors.Wrap(err, "load telegram templates")
	}

	// Load notification templates
	r.fs = notificationsFS
	if err := r.loadAll(); err != nil {
		return nil, errors.Wrap(err, "load notification templates")
	}

	return r, nil
}

var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
	defaultErr      error
)
