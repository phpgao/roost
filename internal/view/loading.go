package view

import (
	"fmt"
	"strings"
)

// DefaultLoadingRenderer renders the loading/spinner screen.
type DefaultLoadingRenderer struct {
	styles Styles
}

// NewDefaultLoadingRenderer creates a new DefaultLoadingRenderer.
func NewDefaultLoadingRenderer(styles Styles) *DefaultLoadingRenderer {
	return &DefaultLoadingRenderer{styles: styles}
}

// Render renders the loading screen with logo and spinner.
func (r *DefaultLoadingRenderer) Render(data LoadingViewModel) string {
	var sb strings.Builder
	sb.WriteString(RenderLogo(r.styles, data.Width))
	fmt.Fprintf(&sb, "  %s %s",
		r.styles.Subtle.Render(data.SpinnerFrame),
		r.styles.Indicator.Render("scanning sessions..."))
	return sb.String()
}

// DefaultErrorRenderer renders the error screen.
type DefaultErrorRenderer struct {
	styles Styles
}

// NewDefaultErrorRenderer creates a new DefaultErrorRenderer.
func NewDefaultErrorRenderer(styles Styles) *DefaultErrorRenderer {
	return &DefaultErrorRenderer{styles: styles}
}

// Render renders the error screen.
func (r *DefaultErrorRenderer) Render(data ErrorViewModel) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "  error: %s", data.Error)
	if data.ErrorOutput != "" {
		sb.WriteString("\n\n" + indentBlock(data.ErrorOutput, "  "))
	}
	sb.WriteString("\n\n  press Esc to continue, q to quit")
	return sb.String()
}

func indentBlock(s, prefix string) string {
	if s == "" {
		return ""
	}
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
