// Package rules aplica as regras de escrita (docs/spec.md, seção 5):
// frase longa, citação proibida e HTML cru.
package rules

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/diag"
	"github.com/bpbastos/guidebook/internal/markdown"
)

// snippetWords é quantas palavras do começo da frase longa aparecem na
// mensagem, para achar a frase no arquivo.
const snippetWords = 8

var (
	wordRe = regexp.MustCompile(`[\p{L}\p{N}]+(?:['’-][\p{L}\p{N}]+)*`)
	// Fim de frase: pontuação, talvez fechamento de parêntese ou aspas, e
	// espaço. O fim do bloco também fecha a frase.
	sentenceEndRe = regexp.MustCompile(`[.!?:…][)\]"'»”’*_]*\s`)
	urlRe         = regexp.MustCompile(`(?:https?|mailto):[^\s]*[^\s.,;:!?)\]]`)
)

type pattern struct {
	re  *regexp.Regexp
	msg string
}

// Checker aplica um conjunto de regras já compilado.
type Checker struct {
	limit    int
	patterns []pattern
}

// New compila as regras. Os padrões já foram validados pelo config; sem
// limite definido, vale o padrão da spec.
func New(r config.Rules) *Checker {
	c := &Checker{limit: config.DefaultMaxSentenceWords}
	if r.MaxSentenceWords != nil {
		c.limit = *r.MaxSentenceWords
	}
	for _, fp := range r.ForbiddenPatterns {
		c.patterns = append(c.patterns, pattern{re: regexp.MustCompile(fp.Pattern), msg: fp.Message})
	}
	return c
}

// Check devolve as violações de source; file é o nome exibido.
func (c *Checker) Check(file string, source []byte) diag.List {
	r := &run{c: c, file: file, source: source, lines: markdown.NewLines(source), htmlLines: map[int]bool{}}
	ast.Walk(markdown.Parse(source), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			if n.Lines().Len() > 0 {
				r.rawHTML(n.Lines().At(0).Start)
			}
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph, *ast.TextBlock, *ast.Heading, *east.TableCell:
			t := &text{}
			r.inline(n, t)
			r.block(t)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return r.problems
}

// run guarda o estado de uma checagem.
type run struct {
	c         *Checker
	file      string
	source    []byte
	lines     markdown.Lines
	htmlLines map[int]bool
	problems  diag.List
}

func (r *run) line(offset int) int { return r.lines.Line(offset) }

// text é o texto de um bloco, com a posição no arquivo de cada byte.
type text struct {
	b    []byte
	off  []int
	last int
}

func (t *text) add(s []byte, start int) {
	for i := range s {
		t.b = append(t.b, s[i])
		t.off = append(t.off, start+i)
	}
	if len(s) > 0 {
		t.last = start + len(s) - 1
	}
}

// gap separa palavras sem acrescentar nenhuma (código, imagem, quebra).
func (t *text) gap() { t.add([]byte(" "), t.last) }

// inline junta o texto dos filhos de n em t. Código em linha, destino
// de link e texto alternativo de imagem não entram em t: código e
// destino só passam pelos padrões proibidos; o texto alternativo vira
// um bloco próprio.
func (r *run) inline(n ast.Node, t *text) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			t.add(c.Segment.Value(r.source), c.Segment.Start)
			if c.SoftLineBreak() || c.HardLineBreak() {
				t.gap()
			}
		case *ast.String:
			t.add(c.Value, t.last)
		case *ast.CodeSpan:
			code := &text{last: t.last}
			r.inline(c, code)
			r.forbidden(code)
			t.gap()
		case *ast.RawHTML:
			if c.Segments.Len() > 0 {
				r.rawHTML(c.Segments.At(0).Start)
			}
			t.gap()
		case *ast.AutoLink:
			url := &text{}
			url.add(c.URL(r.source), markdown.FirstOffset(c, t.last))
			r.forbidden(url)
			t.gap()
		case *ast.Link:
			dest := &text{}
			dest.add(c.Destination, markdown.FirstOffset(c, t.last))
			r.forbidden(dest)
			r.inline(c, t)
		case *ast.Image:
			alt := &text{last: t.last}
			r.inline(c, alt)
			r.block(alt)
			dest := &text{}
			dest.add(c.Destination, markdown.FirstOffset(c, t.last))
			r.forbidden(dest)
			t.gap()
		default:
			r.inline(c, t)
		}
	}
}

func (r *run) rawHTML(offset int) {
	line := r.line(offset)
	if r.htmlLines[line] {
		return // uma violação por linha basta
	}
	r.htmlLines[line] = true
	r.problems.Addf(r.file, line, "", "HTML cru não é permitido no Markdown")
}

// block aplica as regras de texto a um bloco.
func (r *run) block(t *text) {
	r.forbidden(t)
	r.sentences(t)
}

func (r *run) forbidden(t *text) {
	for _, p := range r.c.patterns {
		for _, m := range p.re.FindAllIndex(t.b, -1) {
			r.problems.Addf(r.file, r.line(t.off[m[0]]), "", "%s: %q", p.msg, t.b[m[0]:m[1]])
		}
	}
}

func (r *run) sentences(t *text) {
	// URLs viram espaço, sem mudar as posições.
	b := urlRe.ReplaceAllFunc(t.b, func(m []byte) []byte { return bytes.Repeat([]byte(" "), len(m)) })

	var words [][]int
	flush := func() {
		if len(words) > r.c.limit {
			// Trecho tirado do arquivo, com pontuação e código, do começo da
			// primeira palavra ao fim da oitava.
			last := words[min(len(words), snippetWords)-1]
			snippet := strings.Join(strings.Fields(string(r.source[t.off[words[0][0]]:t.off[last[1]-1]+1])), " ")
			if len(words) > snippetWords {
				snippet += "…"
			}
			r.problems.Addf(r.file, r.line(t.off[words[0][0]]), "",
				"frase com %d palavras (limite: %d): %q", len(words), r.c.limit, snippet)
		}
		words = words[:0]
	}
	prevEnd := 0
	for _, w := range wordRe.FindAllIndex(b, -1) {
		if len(words) > 0 && sentenceEndRe.Match(b[prevEnd:w[0]]) {
			flush()
		}
		words = append(words, w)
		prevEnd = w[1]
	}
	flush()
}
