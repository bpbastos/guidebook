// Package changelog lê um CHANGELOG no formato Keep a Changelog
// (docs/spec.md, seção 3.6).
package changelog

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark/ast"

	"github.com/bpbastos/guidebook/internal/diag"
	"github.com/bpbastos/guidebook/internal/markdown"
)

// Release é um marco do CHANGELOG.
type Release struct {
	Version    string // como escrito entre colchetes
	Date       string // AAAA-MM-DD; vazio no marco não publicado
	Unreleased bool
	Groups     []Group
}

// Group é uma categoria dentro do marco (Adicionado, Corrigido...).
type Group struct {
	Label string // sempre em pt-BR
	HTML  string // conteúdo renderizado
}

// categories aceita os nomes do Keep a Changelog em inglês e em pt-BR.
var categories = map[string]string{
	"added": "Adicionado", "adicionado": "Adicionado",
	"changed": "Modificado", "modificado": "Modificado",
	"deprecated": "Obsoleto", "obsoleto": "Obsoleto",
	"removed": "Removido", "removido": "Removido",
	"fixed": "Corrigido", "corrigido": "Corrigido",
	"security": "Segurança", "segurança": "Segurança",
}

var (
	releaseRe = regexp.MustCompile(`^\[([^\]]+)\](?:\s+-\s+(.+))?$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Parse lê o CHANGELOG; file é o nome exibido nos problemas.
func Parse(file string, source []byte) ([]Release, diag.List) {
	var (
		releases []Release
		problems diag.List
		group    *Group
		html     bytes.Buffer
		lines    = markdown.NewLines(source)
		renderer = markdown.New().Renderer()
	)
	report := func(offset int, format string, args ...any) {
		problems.Addf(file, lines.Line(offset), "", format, args...)
	}
	closeGroup := func() {
		if group != nil {
			group.HTML = html.String()
			releases[len(releases)-1].Groups = append(releases[len(releases)-1].Groups, *group)
			group = nil
		}
		html.Reset()
	}

	doc := markdown.Parse(source)
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		offset := markdown.BlockOffset(n)
		h, isHeading := n.(*ast.Heading)
		switch {
		case isHeading && h.Level == 1:
			continue
		case isHeading && h.Level == 2:
			closeGroup()
			releases = append(releases, parseRelease(rawLine(h, source), offset, report))
			continue
		case len(releases) == 0:
			continue // introdução antes do primeiro marco
		case isHeading && h.Level == 3:
			closeGroup()
			name := rawLine(h, source)
			label, ok := categories[strings.ToLower(name)]
			if !ok {
				report(offset, "categoria %q desconhecida; use Adicionado, Modificado, Obsoleto, Removido, Corrigido ou Segurança (ou os nomes em inglês)", name)
				label = name
			}
			group = &Group{Label: label}
			continue
		case group == nil:
			report(offset, "conteúdo fora de uma categoria; abra uma categoria com ### antes")
			continue
		}
		checkLinks(n, source, offset, report)
		_ = renderer.Render(&html, source, n)
	}
	closeGroup()
	return releases, problems
}

func parseRelease(text string, offset int, report func(int, string, ...any)) Release {
	m := releaseRe.FindStringSubmatch(text)
	if m == nil {
		report(offset, "marco fora do formato; use ## [versão] - AAAA-MM-DD")
		return Release{Version: text}
	}
	r := Release{Version: m[1], Date: m[2]}
	switch v := strings.ToLower(r.Version); {
	case v == "unreleased" || v == "não publicado":
		r.Unreleased = true
		if r.Date != "" {
			report(offset, "o marco não publicado não leva data")
		}
	case r.Date == "":
		report(offset, "marco sem data; use ## [%s] - AAAA-MM-DD", r.Version)
	default:
		if _, err := time.Parse("2006-01-02", r.Date); !dateRe.MatchString(r.Date) || err != nil {
			report(offset, "marco fora do formato; a data precisa ser AAAA-MM-DD")
		}
	}
	return r
}

// rawLine devolve o texto do título como está no arquivo, sem os "#".
// O texto renderizado perderia os colchetes quando a versão é link.
func rawLine(h *ast.Heading, source []byte) string {
	if h.Lines().Len() == 0 {
		return ""
	}
	line := h.Lines().At(0)
	return strings.TrimSpace(string(line.Value(source)))
}

// checkLinks recusa link relativo e esquema fora de http, https e
// mailto: o CHANGELOG não é página do guia.
func checkLinks(n ast.Node, source []byte, blockOffset int, report func(int, string, ...any)) {
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest string
		switch l := c.(type) {
		case *ast.Link:
			dest = string(l.Destination)
		case *ast.Image:
			dest = string(l.Destination)
		case *ast.AutoLink:
			dest = string(l.URL(source))
		default:
			return ast.WalkContinue, nil
		}
		off := markdown.FirstOffset(c, blockOffset)
		switch scheme := markdown.Scheme(dest); {
		case scheme == "":
			report(off, "link relativo não é aceito no CHANGELOG: %q", dest)
		case !markdown.ExternalScheme(scheme):
			report(off, "esquema %q não é permitido; use http, https ou mailto", scheme+":")
		}
		return ast.WalkContinue, nil
	})
}

// Published devolve só os marcos publicados, do mais novo para o mais
// antigo.
func Published(releases []Release) []Release {
	var out []Release
	for _, r := range releases {
		if !r.Unreleased {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}
