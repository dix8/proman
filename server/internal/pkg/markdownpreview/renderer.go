package markdownpreview

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type Renderer struct {
	markdown goldmark.Markdown
	policy   *bluemonday.Policy
}

func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowURLSchemes("http", "https")
	policy.AllowAttrs("src", "alt", "title").OnElements("img")
	policy.AllowElements("img")
	policy.RequireParseableURLs(true)

	return &Renderer{
		markdown: md,
		policy:   policy,
	}
}

func (r *Renderer) Render(content string) (string, error) {
	var buf bytes.Buffer
	if err := r.markdown.Convert([]byte(content), &buf); err != nil {
		return "", err
	}

	safeHTML := r.policy.Sanitize(buf.String())
	return safeHTML, nil
}
