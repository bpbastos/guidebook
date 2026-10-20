// Package site grava os guias em HTML (docs/spec.md, seção 9).
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bpbastos/guidebook/internal/changelog"
	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/content"
	"github.com/bpbastos/guidebook/internal/media"
)

//go:embed assets
var assets embed.FS

var guideTemplate = template.Must(template.New("guide.html.tmpl").Funcs(template.FuncMap{"hash": hash}).ParseFS(assets, "assets/guide.html.tmpl"))

// hash monta o endereço interno de uma página ou seção. Os
// identificadores vêm do build, não do texto do projeto; o html/template
// ainda codificaria a "/" se o valor entrasse como texto.
func hash(parts ...string) template.URL {
	return template.URL("#/" + strings.Join(parts, "/"))
}

// Artigos gerados pelo guidebook. Os identificadores não colidem com os
// das páginas, que são index, glossary ou tipo/nome.
const (
	kindStartHere     = "start-here"
	kindBlocked       = "blocked"
	kindWhatsNew      = "whats-new"
	kindBusinessRules = "business_rules"
	kindChanges       = "changes"
)

type article struct {
	ID, Kind string
	HTML     template.HTML
	HasVideo bool
}

type navItem struct {
	ID, Label string
	HasVideo  bool
	Children  []navItem
}

type profileView struct {
	ID, Name string
	Pages    []pageRef
}

type pageRef struct{ ID, Title string }

type blockedEntry struct {
	Page, Anchor, Text, PageTitle string
}

type releaseView struct {
	Version, Date string
	Groups        []groupView
}

type groupView struct {
	Label string
	HTML  template.HTML
}

type rulesModule struct {
	ID, Title string
	HTML      template.HTML
}

// guideView é o que o template recebe; os campos de um guia só ficam
// vazios no outro.
type guideView struct {
	Product, GuideName, Home, Logo string
	UI                             map[string]string
	Nav                            []navItem
	Articles                       []article
	DataFile                       string
	User                           bool // apresentação e vídeo só existem no guia do usuário
	Mermaid                        bool

	// Guia do usuário.
	Profiles []profileView
	Blocked  []blockedEntry

	// "O que há de novo" ou "Novidades técnicas".
	ReleasesTitle string
	Releases      []releaseView

	// Guia do desenvolvedor.
	RulesTitle   string
	RulesModules []rulesModule
}

// guideData vai para assets/<guia>-data.js.
type guideData struct {
	Guide      string      `json:"guide"`
	Home       string      `json:"home"`
	Pages      []pageData  `json:"pages"`
	Search     []searchDoc `json:"search"`
	StorageKey string      `json:"storageKey"`
}

// searchDoc é um documento da busca: a introdução ou uma seção "## " de
// uma página. O índice é montado no navegador (ADR 0002).
type searchDoc struct {
	ID      string `json:"id"`
	Page    string `json:"page"`
	Title   string `json:"title"`
	Section string `json:"section"`
	Text    string `json:"text"`
	Kind    string `json:"kind"`
}

type pageData struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

// uiKeys são os textos da interface que o template usa.
var uiKeys = []string{
	"user_guide", "developer_guide", "back_to_where_i_was", "close", "start_here", "profile", "blocked_index", "whats_new",
	"search", "search_no_results",
	"present", "slide_list", "slide_of", "previous", "next", "fullscreen", "exit_presentation",
	"has_video", "rules_filter", "rules_module", "rules_all", "rules_none",
	"tutorials", "tasks", "reference", "explanation", "glossary",
}

func newView(c *config.Config, guideKey string) guideView {
	v := guideView{Product: c.Product.Name, GuideName: c.UIText(guideKey), UI: map[string]string{}}
	for _, k := range uiKeys {
		v.UI[k] = c.UIText(k)
	}
	if c.Product.Logo != "" {
		v.Logo = "media/" + c.Product.Logo
	}
	return v
}

func (v *guideView) addPage(p *content.Page) {
	v.Articles = append(v.Articles, article{ID: p.ID, Kind: string(p.Kind), HTML: template.HTML(p.HTML), HasVideo: p.Video != nil})
}

// setReleases monta os marcos; categoria e marco sem itens ficam de fora.
func (v *guideView) setReleases(title string, releases []changelog.Release) {
	v.ReleasesTitle = title
	for _, r := range releases {
		rv := releaseView{Version: r.Version, Date: brazilianDate(r.Date)}
		for _, gr := range r.Groups {
			if strings.TrimSpace(gr.HTML) != "" {
				rv.Groups = append(rv.Groups, groupView{Label: gr.Label, HTML: template.HTML(gr.HTML)})
			}
		}
		if len(rv.Groups) > 0 {
			v.Releases = append(v.Releases, rv)
		}
	}
}

// WriteUser grava o guia do usuário em out, que já foi preparado. Os PNG
// de mídia passam por cache; sem ele, cada um é otimizado na hora.
func WriteUser(c *config.Config, g *content.Guide, out string, cache *media.Cache) error {
	v := newView(c, "user_guide")
	v.User = true
	byID := map[string]*content.Page{}
	byKind := map[content.Kind][]*content.Page{}
	for _, p := range g.Pages {
		byID[p.ID] = p
		byKind[p.Kind] = append(byKind[p.Kind], p)
	}

	for _, p := range byKind[content.KindIndex] {
		v.addPage(p)
		v.Nav = append(v.Nav, navItem{ID: p.ID, Label: p.Title})
	}
	if len(c.User.Profiles) > 0 {
		for _, prof := range c.User.Profiles {
			pv := profileView{ID: prof.ID, Name: prof.Name}
			for _, id := range prof.StartHere {
				pv.Pages = append(pv.Pages, pageRef{ID: id, Title: byID[id].Title})
			}
			v.Profiles = append(v.Profiles, pv)
		}
		v.Articles = append(v.Articles, article{ID: kindStartHere, Kind: kindStartHere})
		v.Nav = append(v.Nav, navItem{ID: kindStartHere, Label: v.UI["start_here"]})
	}
	for _, kind := range []content.Kind{content.KindTutorial, content.KindTask, content.KindReference, content.KindExplanation} {
		if len(byKind[kind]) == 0 {
			continue
		}
		group := navItem{Label: v.UI[string(kind)]}
		for _, p := range byKind[kind] {
			v.addPage(p)
			group.Children = append(group.Children, navItem{ID: p.ID, Label: p.Title, HasVideo: p.Video != nil})
		}
		v.Nav = append(v.Nav, group)
	}
	for _, p := range byKind[content.KindGlossary] {
		v.addPage(p)
		v.Nav = append(v.Nav, navItem{ID: p.ID, Label: v.UI["glossary"]})
	}

	for _, p := range g.Pages {
		for _, h := range p.Blocked {
			v.Blocked = append(v.Blocked, blockedEntry{Page: p.ID, Anchor: h.Anchor, Text: h.Text, PageTitle: p.Title})
		}
	}
	sort.SliceStable(v.Blocked, func(i, j int) bool {
		return strings.ToLower(v.Blocked[i].Text) < strings.ToLower(v.Blocked[j].Text)
	})
	if len(v.Blocked) > 0 {
		v.Articles = append(v.Articles, article{ID: kindBlocked, Kind: kindBlocked})
		v.Nav = append(v.Nav, navItem{ID: kindBlocked, Label: v.UI["blocked_index"]})
	}

	v.setReleases(v.UI["whats_new"], changelog.Published(g.Releases))
	if len(v.Releases) > 0 {
		v.Articles = append(v.Articles, article{ID: kindWhatsNew, Kind: kindWhatsNew})
		v.Nav = append(v.Nav, navItem{ID: kindWhatsNew, Label: v.UI["whats_new"]})
	}
	return writeGuide(c, v, g, out, "user", nil, cache)
}

// WriteDeveloper grava o guia do desenvolvedor em out, que já foi
// preparado (docs/spec.md, seção 8): uma entrada de menu por seção, as
// regras de negócio num artigo só, com filtro, e as novidades técnicas
// com o marco não publicado.
func WriteDeveloper(c *config.Config, g *content.Guide, out string, cache *media.Cache) error {
	v := newView(c, "developer_guide")
	v.Mermaid = true
	for _, s := range g.Sections {
		switch {
		case s.Key == kindChanges:
			v.setReleases(s.Title, g.Releases)
			if len(v.Releases) > 0 {
				v.Articles = append(v.Articles, article{ID: kindChanges, Kind: kindChanges})
				v.Nav = append(v.Nav, navItem{ID: kindChanges, Label: s.Title})
			}
		case s.Key == kindBusinessRules:
			v.RulesTitle = s.Title
			for _, p := range s.Pages {
				v.RulesModules = append(v.RulesModules, rulesModule{ID: p.ID, Title: p.Title, HTML: template.HTML(p.HTML)})
			}
			v.Articles = append(v.Articles, article{ID: kindBusinessRules, Kind: kindBusinessRules})
			v.Nav = append(v.Nav, navItem{ID: kindBusinessRules, Label: s.Title})
		// Módulos e decisões têm um item por arquivo; as outras seções,
		// com um arquivo só, viram um item direto.
		case len(s.Pages) == 1 && s.Key != "modules" && s.Key != "decisions":
			v.addPage(s.Pages[0])
			v.Nav = append(v.Nav, navItem{ID: s.Pages[0].ID, Label: s.Title})
		default:
			group := navItem{Label: s.Title}
			for _, p := range s.Pages {
				v.addPage(p)
				group.Children = append(group.Children, navItem{ID: p.ID, Label: p.Title})
			}
			v.Nav = append(v.Nav, group)
		}
	}
	return writeGuide(c, v, g, out, "developer", [][2]string{
		{"third_party/mermaid/mermaid.min.js", "assets/mermaid.min.js"},
		{"third_party/mermaid/LICENSE", "assets/mermaid.LICENSE.txt"},
	}, cache)
}

// writeGuide grava o HTML, os dados, o tema, os recursos embutidos e a
// mídia de um guia. guide é "user" ou "developer".
func writeGuide(c *config.Config, v guideView, g *content.Guide, out, guide string, vendor [][2]string, cache *media.Cache) error {
	if len(v.Articles) > 0 {
		v.Home = v.Articles[0].ID
	}
	v.DataFile = guide + "-data.js"
	data := guideData{Guide: guide, Home: v.Home, StorageKey: "guidebook:" + c.Product.Name}
	for _, p := range g.Pages {
		data.Pages = append(data.Pages, pageData{ID: p.ID, Kind: string(p.Kind), Title: p.Title})
		for _, sec := range p.Sections {
			doc := searchDoc{ID: p.ID, Page: p.ID, Title: p.Title, Text: sec.Text, Kind: string(p.Kind)}
			if sec.Anchor != "" {
				doc.ID, doc.Section = p.ID+"/"+sec.Anchor, sec.Title
			}
			data.Search = append(data.Search, doc)
		}
	}

	var html bytes.Buffer
	if err := guideTemplate.Execute(&html, v); err != nil {
		return fmt.Errorf("não foi possível montar o HTML: %v", err)
	}
	dataJS, err := dataScript(data)
	if err != nil {
		return err
	}
	theme := fmt.Sprintf(":root {\n  --gb-primary: %s;\n  --gb-on-primary: %s;\n  --gb-accent: %s;\n}\n",
		c.Product.Colors.Primary, textOn(c.Product.Colors.Primary), c.Product.Colors.Accent)

	files := []struct {
		name string
		body []byte
	}{
		{guide + "-guide.html", html.Bytes()},
		{"assets/" + v.DataFile, dataJS},
		{"assets/theme.css", []byte(theme)},
	}
	// Recursos embutidos: origem em assets/ do pacote → destino em out/.
	embedded := append([][2]string{
		{"guidebook.css", "assets/guidebook.css"},
		{"guidebook.js", "assets/guidebook.js"},
		{"third_party/minisearch/minisearch.js", "assets/minisearch.js"},
		{"third_party/minisearch/LICENSE.txt", "assets/minisearch.LICENSE.txt"},
	}, vendor...)
	for _, f := range embedded {
		body, err := assets.ReadFile("assets/" + f[0])
		if err != nil {
			return err
		}
		files = append(files, struct {
			name string
			body []byte
		}{f[1], body})
	}
	for _, f := range files {
		if err := writeFile(filepath.Join(out, filepath.FromSlash(f.name)), f.body); err != nil {
			return err
		}
	}
	used := append([]string(nil), g.Media...)
	if c.Product.Logo != "" {
		used = append(used, c.Product.Logo)
	}
	if cache == nil {
		cache = media.NewCache()
	}
	return copyMedia(c.Dir, out, used, c.Product.Logo, cache)
}

// dataScript serializa os dados com json.Marshal, que escapa <, >, &,
// U+2028 e U+2029: nenhum texto do projeto entra cru em <script>.
func dataScript(v any) ([]byte, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("não foi possível serializar os dados: %v", err)
	}
	return []byte("window.GUIDEBOOK_DATA = " + string(j) + ";\n"), nil
}

func brazilianDate(iso string) string {
	if len(iso) != len("2006-01-02") {
		return iso
	}
	return iso[8:10] + "/" + iso[5:7] + "/" + iso[0:4]
}

func writeFile(name string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return fmt.Errorf("não foi possível criar %s: %v", filepath.Dir(name), err)
	}
	if err := os.WriteFile(name, body, 0o644); err != nil {
		return fmt.Errorf("não foi possível gravar %s: %v", name, err)
	}
	return nil
}

// copyMedia copia os arquivos do projeto para out/media/ (docs/spec.md,
// seção 9.4). Os PNG, fora o logo, são otimizados em paralelo; a
// gravação segue a ordem dos caminhos.
func copyMedia(projectDir, out string, files []string, logo string, cache *media.Cache) error {
	sort.Strings(files)
	files = slices.Compact(files)
	type result struct {
		body []byte
		err  error
	}
	results := make([]chan result, len(files))
	work := make(chan int)
	for range min(runtime.GOMAXPROCS(0), len(files)) {
		go func() {
			for i := range work {
				rel := files[i]
				body, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(rel)))
				if err == nil && rel != logo && strings.EqualFold(path.Ext(rel), ".png") {
					body = cache.Optimize(body)
				}
				results[i] <- result{body, err}
			}
		}()
	}
	for i := range results {
		results[i] = make(chan result, 1)
	}
	go func() {
		for i := range files {
			work <- i
		}
		close(work)
	}()
	var first error
	for i, rel := range files {
		r := <-results[i]
		if first != nil {
			continue // espera os outros para não deixar goroutine presa
		}
		if r.err != nil {
			first = fmt.Errorf("não foi possível ler %s: %v", rel, r.err)
			continue
		}
		first = writeFile(filepath.Join(out, "media", filepath.FromSlash(path.Clean(rel))), r.body)
	}
	return first
}

// textOn escolhe a cor de texto com mais contraste sobre bg (#rrggbb),
// pela fórmula de contraste da WCAG.
func textOn(bg string) string {
	const light, dark = "#ffffff", "#111111"
	l := luminance(bg)
	if (1.05)/(l+0.05) >= (l+0.05)/(luminance(dark)+0.05) {
		return light
	}
	return dark
}

func luminance(hex string) float64 {
	var rgb [3]float64
	for i := range rgb {
		v, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		c := float64(v) / 255
		if c <= 0.03928 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}
