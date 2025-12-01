package telegram

import (
	"prometheus/pkg/telegram"
	"prometheus/pkg/templates"
)

// TemplateRendererAdapter adapts templates.Registry to telegram.TemplateRenderer interface
type TemplateRendererAdapter struct {
	registry *templates.Registry
}

// NewTemplateRendererAdapter creates adapter wrapping templates.Registry
func NewTemplateRendererAdapter(registry *templates.Registry) *TemplateRendererAdapter {
	return &TemplateRendererAdapter{
		registry: registry,
	}
}

// Render renders a template with data (accepts any type - struct or map)
func (a *TemplateRendererAdapter) Render(templatePath string, data interface{}) (string, error) {
	return a.registry.Render(templatePath, data)
}

// Verify adapter implements interface at compile time
var _ telegram.TemplateRenderer = (*TemplateRendererAdapter)(nil)
