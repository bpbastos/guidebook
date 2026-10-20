// Package markdown concentra a configuração do goldmark usada pelo
// guidebook (docs/spec.md, seção 3.2): CommonMark com tabela e tachado
// do GFM, sem WithUnsafe.
package markdown

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// New devolve o conversor com as extensões da spec. Sem WithUnsafe, o
// HTML cru que escapar das regras de escrita sai omitido na renderização.
func New() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough))
}

// Parse devolve a árvore do documento.
func Parse(source []byte) ast.Node {
	return New().Parser().Parse(text.NewReader(source))
}

// Lines converte posição no arquivo em número de linha.
type Lines []int

// NewLines indexa as quebras de linha de source.
func NewLines(source []byte) Lines {
	var l Lines
	for i, b := range source {
		if b == '\n' {
			l = append(l, i)
		}
	}
	return l
}

// Line devolve a linha (a partir de 1) da posição offset.
func (l Lines) Line(offset int) int {
	return sort.SearchInts(l, offset) + 1
}

// FirstOffset devolve a posição do primeiro texto dentro de n, ou def.
func FirstOffset(n ast.Node, def int) int {
	off := def
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			off = t.Segment.Start
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return off
}

// BlockOffset devolve a posição da primeira linha de um bloco, ou da
// primeira linha do primeiro bloco filho; -1 se não houver.
func BlockOffset(n ast.Node) int {
	for ; n != nil; n = n.FirstChild() {
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			return n.Lines().At(0).Start
		}
	}
	return -1
}

// PlainText devolve o texto de n sem marcação, incluindo código em
// linha.
func PlainText(n ast.Node, source []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

var schemeRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.-]*):`)

// Scheme devolve o esquema de dest em minúsculas ("https", "javascript")
// ou "" se dest for relativo.
func Scheme(dest string) string {
	if m := schemeRe.FindStringSubmatch(dest); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// ExternalScheme diz se o esquema é aceito como link externo
// (docs/spec.md, seção 3.3).
func ExternalScheme(scheme string) bool {
	return scheme == "http" || scheme == "https" || scheme == "mailto"
}
