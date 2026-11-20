package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	goldmarktext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.yaml.in/yaml/v3"

	"github.com/muchobien/lyra/internal/store"
	"github.com/muchobien/lyra/internal/webui"
)

var markdown = goldmark.New(goldmark.WithExtensions(
	extension.GFM,
	highlighting.NewHighlighting(highlighting.WithFormatOptions(chromahtml.WithClasses(true))),
	videos{},
))

var kindVideo = ast.NewNodeKind("Video")

type video struct {
	ast.BaseInline
	destination []byte
}

func (node *video) Kind() ast.NodeKind {
	return kindVideo
}

func (node *video) Dump(source []byte, level int) {
	ast.DumpHelper(node, source, level, map[string]string{"Destination": string(node.destination)}, nil)
}

type videos struct{}

func (extension videos) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(extension, 999)))
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(extension, 999)))
}

func (videos) Transform(document *ast.Document, _ goldmarktext.Reader, _ parser.Context) {
	found := []*ast.Image{}
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if image, isImage := node.(*ast.Image); entering && isImage {
			address, err := url.Parse(string(image.Destination))
			extension := ""
			if err == nil {
				extension = strings.ToLower(path.Ext(address.Path))
			}
			if slices.Contains([]string{".mp4", ".webm", ".mov"}, extension) {
				found = append(found, image)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, image := range found {
		image.Parent().ReplaceChild(image.Parent(), image, &video{destination: image.Destination})
	}
}

func (extension videos) RegisterFuncs(registry renderer.NodeRendererFuncRegisterer) {
	registry.Register(kindVideo, extension.render)
}

func (videos) render(writer util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	destination := node.(*video).destination
	if !entering || goldmarkhtml.IsDangerousURL(destination) {
		return ast.WalkSkipChildren, nil
	}
	escaped := util.EscapeHTML(util.URLEscape(destination, true))
	_, err := fmt.Fprintf(writer, `<video controls src="%s"></video>`, escaped)
	return ast.WalkSkipChildren, err
}

var codeFormatter = chromahtml.New(
	chromahtml.WithClasses(true),
	chromahtml.WithLineNumbers(true),
	chromahtml.LineNumbersInTable(true),
)

var patchFormatter = chromahtml.New(chromahtml.WithClasses(true))

func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func preformatted(source string) template.HTML {
	return template.HTML("<pre>" + template.HTMLEscapeString(source) + "</pre>")
}

func renderMarkdown(source string) template.HTML {
	var rendered bytes.Buffer
	if markdown.Convert([]byte(source), &rendered) != nil {
		return preformatted(source)
	}
	return template.HTML(rendered.String())
}

func renderMarkdownFile(content []byte) (string, error) {
	var rendered bytes.Buffer
	rendered.WriteString(`<article class="markdown">`)
	normalized := append(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")), '\n')
	if rest, opened := bytes.CutPrefix(normalized, []byte("---\n")); opened {
		if matter, body, closed := bytes.Cut(append([]byte("\n"), rest...), []byte("\n---\n")); closed {
			var document yaml.Node
			if yaml.Unmarshal(matter, &document) != nil ||
				len(document.Content) > 0 && document.Content[0].Kind != yaml.MappingNode {
				rendered.WriteString(string(preformatted(string(bytes.TrimPrefix(matter, []byte("\n"))))))
			} else if len(document.Content) > 0 {
				rendered.WriteString(frontmatterHTML(document.Content[0]))
			}
			content = body
		}
	}
	if err := markdown.Convert(content, &rendered); err != nil {
		return "", err
	}
	rendered.WriteString(`</article>`)
	return rendered.String(), nil
}

func frontmatterHTML(node *yaml.Node) string {
	switch node.Kind {
	case yaml.MappingNode:
		var head, row strings.Builder
		for index := 0; index+1 < len(node.Content); index += 2 {
			head.WriteString("<th>" + frontmatterHTML(node.Content[index]) + "</th>")
			row.WriteString("<td>" + frontmatterHTML(node.Content[index+1]) + "</td>")
		}
		return "<table><thead><tr>" + head.String() + "</tr></thead>" +
			"<tbody><tr>" + row.String() + "</tr></tbody></table>"
	case yaml.SequenceNode:
		var row strings.Builder
		for _, item := range node.Content {
			row.WriteString("<td>" + frontmatterHTML(item) + "</td>")
		}
		return "<table><tbody><tr>" + row.String() + "</tr></tbody></table>"
	case yaml.AliasNode:
		return template.HTMLEscapeString("*" + node.Value)
	}
	return template.HTMLEscapeString(node.Value)
}

func highlight(formatter *chromahtml.Formatter, lexer chroma.Lexer, source string) template.HTML {
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, source)
	if err != nil {
		return preformatted(source)
	}
	var output strings.Builder
	if err := formatter.Format(&output, styles.Get("github"), iterator); err != nil {
		return preformatted(source)
	}
	return template.HTML(output.String())
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	webui.Serve(w, r, status, message)
}

func (s *Server) uiLookup(r *http.Request) (store.Repo, error) {
	name := r.PathValue("owner") + "/" + r.PathValue("repo")
	if !identityFrom(r).can(scopeRead, name) {
		return store.Repo{}, forbidden("This token cannot read " + name + ".")
	}
	repo, err := s.store.Repo(r.Context(), name)
	return repo, missing(err, "Repository "+name+" not found.")
}
