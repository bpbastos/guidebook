package content

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/bpbastos/guidebook/internal/changelog"
	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/diag"
	"github.com/bpbastos/guidebook/internal/markdown"
)

// Guide é um guia pronto para gravar.
type Guide struct {
	Pages    []*Page
	Media    []string // arquivos usados, relativos ao diretório do guidebook.yaml
	Releases []changelog.Release
	Sections []GuideSection // só no guia do desenvolvedor
}

// GuideSection é uma seção do guia do desenvolvedor, na ordem da spec.
type GuideSection struct {
	Key   string
	Title string
	Pages []*Page // vazio em "changes", que mostra Releases
}

// Heading é um título de seção de uma página.
type Heading struct {
	Level  int
	Text   string
	Anchor string
}

// Tipos de link no HTML gerado (atributo data-gb).
const (
	linkNav    = "nav"
	linkDialog = "dialog"
)

// parsed é uma página lida, entre a primeira passada (títulos e
// âncoras) e a segunda (links e renderização).
type parsed struct {
	page    *Page
	source  []byte
	doc     ast.Node
	lines   markdown.Lines
	display string
	anchors map[string]bool
	blocked map[string]bool

	shots     map[ast.Node]*shot // parágrafo da captura → captura
	shotCount int
}

type loader struct {
	c        *config.Config
	dev      bool            // guia do desenvolvedor: links e Mermaid diferentes
	changes  map[string]bool // fontes da seção "changes" (só no guia de dev)
	byFile   map[string]*parsed
	media    map[string]bool
	problems diag.List
}

// Load lê e confere todas as páginas do guia do usuário: estrutura,
// referências do guidebook.yaml, títulos, links, imagens e CHANGELOG.
func Load(c *config.Config) (*Guide, diag.List) {
	g := &Guide{}
	scanned, problems := Scan(c)
	problems = append(problems, CheckReferences(c, scanned)...)
	l := &loader{c: c, byFile: map[string]*parsed{}, media: map[string]bool{}, problems: problems}

	all := l.parseAll(scanned, g)
	byID := map[string]*Page{}
	for _, p := range g.Pages {
		byID[p.ID] = p
	}
	l.videos(byID)
	for _, p := range all {
		l.resolve(p)
	}

	if c.User != nil {
		file := l.abs(c.User.Changelog)
		if source, err := os.ReadFile(file); err == nil {
			releases, problems := changelog.Parse(file, source)
			g.Releases = releases
			l.problems = append(l.problems, problems...)
		}
	}

	for m := range l.media {
		g.Media = append(g.Media, m)
	}
	sort.Strings(g.Media)
	return g, l.problems
}

// parseAll lê as páginas (primeira passada) e as registra no guia.
func (l *loader) parseAll(pages []Page, g *Guide) []*parsed {
	var all []*parsed
	for i := range pages {
		if p := l.parse(&pages[i]); p != nil {
			all = append(all, p)
			l.byFile[p.page.File] = p
			g.Pages = append(g.Pages, p.page)
		}
	}
	return all
}

// LoadDeveloper lê e confere as fontes do guia do desenvolvedor
// (docs/spec.md, seção 8). Cada arquivo de seção vira uma página
// "<seção>/<arquivo>"; as fontes de "changes" são lidas como CHANGELOG.
func LoadDeveloper(c *config.Config) (*Guide, diag.List) {
	g := &Guide{}
	l := &loader{c: c, dev: true, changes: map[string]bool{}, byFile: map[string]*parsed{}, media: map[string]bool{}}
	var pages []Page
	var owner []int // índice da seção de cada página
	for _, s := range c.Developer.Sections.SectionList() {
		g.Sections = append(g.Sections, GuideSection{Key: s.Key, Title: s.DisplayTitle()})
		if s.Key == "changes" {
			for _, f := range s.Files {
				l.changes[f] = true
				if source, err := os.ReadFile(l.abs(f)); err != nil {
					l.problems.Addf(l.abs(f), 0, "", "não foi possível ler o arquivo: %v", err)
				} else {
					releases, problems := changelog.Parse(l.abs(f), source)
					g.Releases = append(g.Releases, releases...)
					l.problems = append(l.problems, problems...)
				}
			}
			continue
		}
		used := map[string]bool{}
		for _, f := range s.Files {
			name := Slug(strings.TrimSuffix(path.Base(f), ".md"))
			id := s.Key + "/" + name
			for i := 2; used[id]; i++ {
				id = fmt.Sprintf("%s/%s-%d", s.Key, name, i)
			}
			used[id] = true
			pages = append(pages, Page{ID: id, Kind: Kind(s.Key), File: f})
			owner = append(owner, len(g.Sections)-1)
		}
	}

	all := l.parseAll(pages, g)
	for _, p := range all {
		l.resolve(p)
	}
	for i, p := range pages {
		for _, parsed := range g.Pages {
			if parsed.File == p.File {
				g.Sections[owner[i]].Pages = append(g.Sections[owner[i]].Pages, parsed)
			}
		}
	}
	for m := range l.media {
		g.Media = append(g.Media, m)
	}
	sort.Strings(g.Media)
	return g, l.problems
}

func (l *loader) abs(rel string) string {
	return filepath.Join(l.c.Dir, filepath.FromSlash(rel))
}

// parse lê a página e numera os títulos.
func (l *loader) parse(page *Page) *parsed {
	display := l.abs(page.File)
	source, err := os.ReadFile(display)
	if err != nil {
		l.problems.Addf(display, 0, "", "não foi possível ler o arquivo: %v", err)
		return nil
	}
	p := &parsed{
		page: page, source: source, doc: markdown.Parse(source), lines: markdown.NewLines(source),
		display: display, anchors: map[string]bool{}, blocked: map[string]bool{}, shots: map[ast.Node]*shot{},
	}
	ast.Walk(p.doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		h, ok := n.(*ast.Heading)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		text := markdown.PlainText(h, source)
		anchor := Slug(text)
		for i := 2; p.anchors[anchor]; i++ {
			anchor = fmt.Sprintf("%s-%d", Slug(text), i)
		}
		p.anchors[anchor] = true
		h.SetAttributeString("id", "/"+page.ID+"/"+anchor)
		heading := Heading{Level: h.Level, Text: text, Anchor: anchor}
		page.Headings = append(page.Headings, heading)

		switch {
		case h.Level == 1 && page.Title == "":
			page.Title = text
		case h.Level == 1:
			l.problems.Addf(display, p.lines.Line(markdown.BlockOffset(h)), "", "segundo título de nível 1; a página só pode ter um")
		case h.Level == 2 && !l.dev && strings.HasPrefix(text, l.c.User.BlockedPrefix):
			page.Blocked = append(page.Blocked, heading)
			p.blocked[anchor] = true
		}
		return ast.WalkSkipChildren, nil
	})
	if page.Title == "" {
		l.problems.Addf(display, 1, "", "página sem título; comece com \"# Título\"")
	}
	page.Sections = sections(p)
	return p
}

// sections corta a página em trechos para a busca: a introdução e cada
// seção "## ", com o texto de tudo o que vem dentro dela.
func sections(p *parsed) []Section {
	var out []Section
	cur := Section{Title: p.page.Title}
	var parts []string
	flush := func() {
		cur.Text = strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
		out = append(out, cur)
		parts = nil
	}
	for n := p.doc.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok && h.Level <= 2 {
			if h.Level == 2 {
				flush()
				id, _ := h.AttributeString("id")
				anchor, _ := id.(string)
				cur = Section{Anchor: strings.TrimPrefix(anchor, "/"+p.page.ID+"/"), Title: markdown.PlainText(h, p.source)}
			}
			continue
		}
		parts = append(parts, blockText(n, p.source)...)
	}
	flush()
	return out
}

// blockText devolve o texto de cada bloco folha dentro de n, incluindo
// código; HTML fica fora.
func blockText(n ast.Node, source []byte) []string {
	var parts []string
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var text string
		switch c.(type) {
		case *ast.Paragraph, *ast.TextBlock, *ast.Heading, *east.TableCell:
			text = markdown.PlainText(c, source)
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			text = strings.TrimSpace(string(c.Lines().Value(source)))
		case *ast.HTMLBlock:
		default:
			return ast.WalkContinue, nil
		}
		if text != "" {
			parts = append(parts, text)
		}
		return ast.WalkSkipChildren, nil
	})
	return parts
}

// resolve confere e reescreve links e imagens e renderiza a página.
func (l *loader) resolve(p *parsed) {
	blockOffset := 0
	report := func(n ast.Node, format string, args ...any) {
		l.problems.Addf(p.display, p.lines.Line(markdown.FirstOffset(n, blockOffset)), "", format, args...)
	}
	ast.Walk(p.doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			blockOffset = n.Lines().At(0).Start
		}
		switch n := n.(type) {
		case *ast.Link:
			href, kind, problem := l.link(p, string(n.Destination))
			if problem != "" {
				report(n, "%s", problem)
				return ast.WalkContinue, nil
			}
			n.Destination = []byte(href)
			setLinkKind(n, kind)
		case *ast.AutoLink:
			if scheme := markdown.Scheme(string(n.URL(p.source))); !markdown.ExternalScheme(scheme) {
				report(n, "esquema %q não é permitido; use http, https ou mailto", scheme+":")
				return ast.WalkContinue, nil
			}
			setLinkKind(n, "external")
		case *ast.Image:
			if markdown.PlainText(n, p.source) == "" {
				report(n, "imagem sem texto alternativo")
			}
			if src, problem := l.image(p, string(n.Destination)); problem != "" {
				report(n, "%s", problem)
			} else {
				n.Destination = []byte(src)
				// No check -no-media, a imagem não é lida.
				if rel := strings.TrimPrefix(src, "media/"); !l.c.NoMedia {
					if s := l.shot(p, n, rel, src, report); s != nil {
						p.shots[n.Parent()] = s
					} else {
						l.imageAttributes(n, rel)
					}
				}
			}
		}
		return ast.WalkContinue, nil
	})

	p.page.HTML = l.render(p)
}

// render gera o HTML da página já cortado em slides (docs/spec.md, seção
// 9.3): um novo slide começa em cada "## " e em cada "---". No modo
// leitura, os cortes não aparecem.
func (l *loader) render(p *parsed) string {
	var b bytes.Buffer
	renderer := markdown.New().Renderer()
	open := false
	videoDone := p.page.Video == nil
	for n := p.doc.FirstChild(); n != nil; n = n.NextSibling() {
		h, isHeading := n.(*ast.Heading)
		_, isBreak := n.(*ast.ThematicBreak)
		if !open || isBreak || (isHeading && h.Level == 2) {
			if open {
				b.WriteString("</div>\n")
			}
			b.WriteString(`<div class="gb-slide">` + "\n")
			open = true
		}
		if fb, ok := n.(*ast.FencedCodeBlock); ok && l.dev && string(fb.Language(p.source)) == "mermaid" {
			// Desenhado no navegador pelo Mermaid; o texto vai escapado.
			b.WriteString(`<pre class="mermaid">` + html.EscapeString(strings.TrimRight(string(fb.Lines().Value(p.source)), "\n")) + "</pre>\n")
		} else if s := p.shots[n]; s != nil {
			l.renderShot(&b, s)
		} else if err := renderer.Render(&b, p.source, n); err != nil {
			l.problems.Addf(p.display, 0, "", "não foi possível renderizar: %v", err)
		}
		// O vídeo da tarefa vem logo depois do título.
		if isHeading && h.Level == 1 && !videoDone {
			l.renderVideoLink(&b, p.page.Video)
			videoDone = true
		}
	}
	if open {
		b.WriteString("</div>\n")
	}
	return b.String()
}

func setLinkKind(n ast.Node, kind string) {
	if kind == "external" {
		n.SetAttributeString("target", "_blank")
		n.SetAttributeString("rel", "noopener noreferrer")
		return
	}
	n.SetAttributeString("data-gb", kind)
}

// link resolve o destino de um link (docs/spec.md, seção 3.3) e devolve
// o href reescrito e o tipo, ou a descrição do problema.
func (l *loader) link(from *parsed, dest string) (href, kind, problem string) {
	if strings.HasPrefix(dest, "//") {
		return "", "", "link sem esquema; use http:// ou https://"
	}
	if scheme := markdown.Scheme(dest); scheme != "" {
		if markdown.ExternalScheme(scheme) {
			return dest, "external", ""
		}
		return "", "", fmt.Sprintf("esquema %q não é permitido; use http, https ou mailto", scheme+":")
	}
	pathPart, frag, _ := strings.Cut(dest, "#")
	target := from
	if pathPart != "" {
		rel, problem := l.relative(from, pathPart)
		if problem != "" {
			return "", "", problem
		}
		if target = l.byFile[rel]; target == nil {
			_, err := os.Stat(l.abs(rel))
			switch {
			case l.dev && l.changes[rel]:
				return "#/changes", linkNav, ""
			case err != nil:
				return "", "", fmt.Sprintf("link para %q, que não existe", rel)
			case !l.dev:
				return "", "", fmt.Sprintf("link para %q, que não é página do guia", rel)
			case l.c.Developer.RepositoryURL == "":
				return "", "", fmt.Sprintf("link para %q, que não é fonte do guia; defina developer.repository_url para apontar para o repositório", rel)
			}
			href = l.c.Developer.RepositoryURL + "/" + rel
			if frag != "" {
				href += "#" + frag
			}
			return href, "external", ""
		}
	}
	if frag != "" && !target.anchors[frag] {
		return "", "", fmt.Sprintf("âncora %q não existe em %s", "#"+frag, target.page.File)
	}

	href = "#/" + target.page.ID
	if frag != "" {
		href += "/" + frag
	}
	switch k := target.page.Kind; {
	case l.dev:
		return href, linkNav, ""
	case k == KindGlossary && frag != "", k == KindReference, k == KindExplanation, target.blocked[frag]:
		return href, linkDialog, ""
	default:
		return href, linkNav, ""
	}
}

// image confere a imagem e devolve o src reescrito para media/.
func (l *loader) image(from *parsed, dest string) (string, string) {
	if scheme := markdown.Scheme(dest); scheme != "" || strings.HasPrefix(dest, "//") {
		if scheme == "" || markdown.ExternalScheme(scheme) {
			return "", "imagem externa não é permitida; o guia abre sem internet"
		}
		return "", fmt.Sprintf("esquema %q não é permitido em imagem", scheme+":")
	}
	rel, problem := l.relative(from, dest)
	if problem != "" {
		return "", problem
	}
	if l.c.NoMedia {
		return "media/" + rel, ""
	}
	if info, err := os.Stat(l.abs(rel)); err != nil || info.IsDir() {
		return "", fmt.Sprintf("imagem %q não existe", rel)
	}
	l.media[rel] = true
	return "media/" + rel, ""
}

// relative resolve um caminho do Markdown a partir da página, relativo
// ao diretório do guidebook.yaml.
func (l *loader) relative(from *parsed, p string) (string, string) {
	unescaped, err := url.PathUnescape(p)
	if err != nil {
		return "", fmt.Sprintf("caminho inválido: %q", p)
	}
	if strings.HasPrefix(unescaped, "/") {
		return "", "use caminho relativo à página, sem \"/\" no começo"
	}
	rel := path.Clean(path.Join(path.Dir(from.page.File), unescaped))
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "link sai do diretório do guidebook.yaml"
	}
	return rel, ""
}

var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n", "ý", "y", "ÿ", "y",
)

// Slug gera a âncora de um título (docs/spec.md, seção 3.2).
func Slug(text string) string {
	var b strings.Builder
	dash := false
	for _, r := range accents.Replace(strings.ToLower(text)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		} else {
			dash = true
		}
	}
	if b.Len() == 0 {
		return "secao"
	}
	return b.String()
}
