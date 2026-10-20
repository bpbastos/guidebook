package site

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/content"
)

// maxUserGuideBytes é o teto do guia do usuário do projeto de exemplo,
// sem media/ (ADR 0006).
const maxUserGuideBytes = 160 * 1024

func exampleConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, problems := config.Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	return cfg
}

func build(t *testing.T, cfg *config.Config) (string, map[string][]byte) {
	t.Helper()
	g, problems := content.Load(cfg)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	out := t.TempDir()
	if _, err := WriteUser(cfg, g, out, nil); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(out, p)
			files[filepath.ToSlash(rel)], _ = os.ReadFile(p)
		}
		return nil
	})
	return out, files
}

func names(files map[string][]byte) []string {
	var n []string
	for k := range files {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

func TestWriteUserFiles(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	want := []string{
		"assets/guidebook.css",
		"assets/guidebook.js",
		"assets/minisearch.LICENSE.txt",
		"assets/minisearch.js",
		"assets/theme.css",
		"assets/user-data.js",
		"media/docs/assets/logo.svg",
		"media/docs/user/media/register-order-1.png",
		"media/docs/user/media/videos/register-order.png",
		"media/docs/user/media/videos/register-order.webm",
		"user-guide.html",
	}
	if got := names(files); !reflect.DeepEqual(got, want) {
		t.Errorf("arquivos = %v\nesperava %v", got, want)
	}
}

func TestUserGuideHTML(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	html := string(files["user-guide.html"])
	for _, want := range []string{
		`<html lang="pt-BR">`,
		`<title>Guia do usuário — Pedido Fácil</title>`,
		`<img src="media/docs/assets/logo.svg" alt="" class="gb-logo">`,
		`<article id="/index" class="gb-page" data-kind="index">`,
		`<article id="/tasks/register-order" class="gb-page" data-kind="tasks" data-has-video="">`,
		`<article id="/tasks/close-day" class="gb-page" data-kind="tasks">`,
		`<a href="#/tasks/register-order">Registrar um pedido <span class="gb-has-video" title="Tem vídeo"><span class="gb-visually-hidden">Tem vídeo</span></span></a>`,
		`<a href="#/tasks/close-day">Fechar o dia</a>`,
		`<article id="/start-here" class="gb-page" data-kind="start-here">`,
		`<option value="attendant">Atendente</option>`,
		`<section class="gb-profile" data-profile="manager">`,
		`<li><a href="#/tasks/close-day" data-gb="nav">Fechar o dia</a></li>`,
		`<article id="/blocked" class="gb-page" data-kind="blocked">`,
		`<a href="#/tasks/register-order/o-sistema-n%c3%a3o-me-deixa-salvar-o-pedido" data-gb="nav">O sistema não me deixa salvar o pedido</a>`,
		`<article id="/whats-new" class="gb-page" data-kind="whats-new">`,
		`<h2>1.1.0 <span class="gb-date">15/09/2026</span></h2>`,
		`<h3>Corrigido</h3>`,
		`<button type="button" class="gb-back" hidden>Voltar para onde eu estava</button>`,
		`<button type="button" class="gb-present" hidden>Apresentar</button>`,
		`<div class="gb-presentation" hidden>`,
		`<div class="gb-stage" tabindex="-1">`,
		`<p class="gb-slide-count" aria-live="polite" data-slide-of="Slide {n} de {total}"></p>`,
		`<button type="button" class="gb-prev">Anterior</button>`,
		`<button type="button" class="gb-next">Próximo</button>`,
		`<button type="button" class="gb-fullscreen">Tela cheia</button>`,
		`<button type="button" class="gb-exit">Sair da apresentação</button>`,
		`<button type="button" class="gb-slide-list-toggle" aria-expanded="false" aria-controls="gb-slide-list">Lista de slides</button>`,
		`<ol id="gb-slide-list" class="gb-slide-list" data-has-video="Tem vídeo" hidden></ol>`,
		`<dialog class="gb-video-dialog">`,
		`<div class="gb-video-body"></div>`,
		`<button type="button" class="gb-video-close">Fechar</button>`,
		`<dialog class="gb-dialog">`,
		`<a class="gb-open-full" href="#/index" data-gb="nav" hidden>Abrir a página inteira</a>`,
		`<div class="gb-search" hidden>`,
		`<label for="gb-search-input" class="gb-visually-hidden">Buscar</label>`,
		`<input type="search" id="gb-search-input" placeholder="Buscar" autocomplete="off" aria-controls="gb-search-results" data-no-results="Nada encontrado">`,
		`<ul id="gb-search-results" class="gb-search-results" hidden></ul>`,
		`<script src="assets/minisearch.js"></script>`,
		`<script src="assets/user-data.js"></script>`,
		`<script src="assets/guidebook.js"></script>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML sem %q", want)
		}
	}
	if !strings.HasSuffix(html, "</html>\n") || strings.HasSuffix(html, "\n\n") {
		t.Errorf("o HTML deveria terminar em um só \"</html>\\n\": %q", html[len(html)-20:])
	}
	if strings.Contains(html, "Não publicado") || strings.Contains(html, "Relatório de vendas") {
		t.Errorf("o marco não publicado não deveria aparecer no guia do usuário")
	}
	// Ordem de leitura sem JavaScript.
	order := []string{`id="/index"`, `id="/start-here"`, `id="/tutorials/first-order"`, `id="/tasks/register-order"`,
		`id="/reference/order-status"`, `id="/explanation/how-billing-works"`, `id="/glossary"`, `id="/blocked"`, `id="/whats-new"`}
	last := -1
	for _, o := range order {
		i := strings.Index(html, `<article `+o)
		if i < last {
			t.Errorf("%s fora de ordem", o)
		}
		last = i
	}
	if strings.Index(html, "1.1.0") > strings.Index(html, "1.0.0") {
		t.Errorf("O que há de novo deveria vir do mais novo para o mais antigo")
	}
}

func TestNoInlineScript(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	for _, m := range regexp.MustCompile(`<script[^>]*>`).FindAllString(string(files["user-guide.html"]), -1) {
		if !strings.Contains(m, ` src="assets/`) {
			t.Errorf("script em linha: %s", m)
		}
	}
}

func TestDataJS(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	js := string(files["assets/user-data.js"])
	const prefix, suffix = "window.GUIDEBOOK_DATA = ", ";\n"
	if !strings.HasPrefix(js, prefix) || !strings.HasSuffix(js, suffix) {
		t.Fatalf("formato de user-data.js: %q", js)
	}
	var data struct {
		Guide string `json:"guide"`
		Home  string `json:"home"`
		Pages []struct {
			ID, Kind, Title string
		} `json:"pages"`
		StorageKey string `json:"storageKey"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimPrefix(js, prefix), suffix)), &data); err != nil {
		t.Fatal(err)
	}
	if data.Guide != "user" || data.Home != "index" || len(data.Pages) != 7 || data.StorageKey != "guidebook:Pedido Fácil" {
		t.Errorf("dados = %+v", data)
	}
}

func TestEscaping(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	write("guidebook.yaml", "version: 1\nproduct:\n  name: '<b>P</b>'\nuser:\n  dir: docs/user\n")
	write("docs/user/tasks/a.md", "# Fim `</script>`\n")
	cfg, problems := config.Load(filepath.Join(dir, "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	_, files := build(t, cfg)
	html, data := string(files["user-guide.html"]), string(files["assets/user-data.js"])
	if strings.Contains(html, "<b>P</b>") || !strings.Contains(html, "&lt;b&gt;P&lt;/b&gt;") {
		t.Errorf("nome do produto sem escape no HTML")
	}
	if strings.Contains(data, "</script>") || !strings.Contains(data, `\u003c/script\u003e`) {
		t.Errorf("título sem escape em user-data.js: %s", data)
	}
}

func TestDeterministic(t *testing.T) {
	cfg := exampleConfig(t)
	_, a := build(t, cfg)
	_, b := build(t, cfg)
	if !reflect.DeepEqual(names(a), names(b)) {
		t.Fatalf("arquivos diferentes entre builds")
	}
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s mudou entre dois builds iguais", name)
		}
	}
}

func TestOffline(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	for name, body := range files {
		s := string(body)
		switch {
		case name == "assets/minisearch.js":
			// Vendorizado: conferido por TestVendorOffline.
		case strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css"):
			for _, bad := range []string{"http://", "https://", "fetch(", "XMLHttpRequest", "import(", "@import"} {
				if strings.Contains(s, bad) {
					t.Errorf("%s contém %q", name, bad)
				}
			}
		case strings.HasSuffix(name, ".html"):
			// URL externa só em link de conteúdo, nunca em src.
			if regexp.MustCompile(`src="(https?:)?//`).MatchString(s) {
				t.Errorf("%s carrega recurso externo", name)
			}
		}
	}
}

func TestSizeCeiling(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	total := 0
	for name, body := range files {
		if !strings.HasPrefix(name, "media/") {
			total += len(body)
		}
	}
	t.Logf("guia do usuário do exemplo, sem media/: %d bytes", total)
	if total > maxUserGuideBytes {
		t.Errorf("guia com %d bytes; o teto é %d (ADR 0006)", total, maxUserGuideBytes)
	}
}

func TestTextOn(t *testing.T) {
	tests := map[string]string{"#1f6feb": "#ffffff", "#ffff00": "#111111", "#000000": "#ffffff", "#ffffff": "#111111"}
	for bg, want := range tests {
		if got := textOn(bg); got != want {
			t.Errorf("textOn(%s) = %s; esperava %s", bg, got, want)
		}
	}
}

func TestOptionalSections(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "docs", "user", "tasks"), 0o755)
	os.WriteFile(filepath.Join(dir, "guidebook.yaml"), []byte("version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "user", "tasks", "a.md"), []byte("# A\n"), 0o644)
	cfg, _ := config.Load(filepath.Join(dir, "guidebook.yaml"))
	_, files := build(t, cfg)
	html := string(files["user-guide.html"])
	for _, absent := range []string{`id="/start-here"`, `id="/blocked"`, `id="/whats-new"`, `id="/glossary"`, `class="gb-logo"`} {
		if strings.Contains(html, absent) {
			t.Errorf("%s deveria faltar sem conteúdo", absent)
		}
	}
	if !strings.Contains(string(files["assets/user-data.js"]), `"home":"tasks/a"`) {
		t.Errorf("sem index.md, a página inicial deveria ser a primeira página")
	}
}

func TestScriptOrder(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	html := string(files["user-guide.html"])
	a, b, c := strings.Index(html, "assets/minisearch.js"), strings.Index(html, "assets/user-data.js"), strings.Index(html, "assets/guidebook.js")
	if a < 0 || a > b || b > c {
		t.Errorf("ordem dos scripts: minisearch %d, dados %d, guidebook %d", a, b, c)
	}
}

func TestSearchDocs(t *testing.T) {
	_, files := build(t, exampleConfig(t))
	js := strings.TrimSuffix(strings.TrimPrefix(string(files["assets/user-data.js"]), "window.GUIDEBOOK_DATA = "), ";\n")
	var data struct {
		Search []struct {
			ID      string `json:"id"`
			Page    string `json:"page"`
			Title   string `json:"title"`
			Section string `json:"section"`
			Text    string `json:"text"`
			Kind    string `json:"kind"`
		} `json:"search"`
	}
	if err := json.Unmarshal([]byte(js), &data); err != nil {
		t.Fatal(err)
	}
	byID := map[string]int{}
	for i, d := range data.Search {
		if _, dup := byID[d.ID]; dup {
			t.Errorf("documento repetido: %s", d.ID)
		}
		byID[d.ID] = i
	}
	i, ok := byID["tasks/register-order/o-sistema-não-me-deixa-salvar-o-pedido"]
	if !ok {
		t.Fatalf("seção de bloqueio fora da busca; documentos: %v", byID)
	}
	d := data.Search[i]
	if d.Page != "tasks/register-order" || d.Title != "Registrar um pedido" || d.Section != "O sistema não me deixa salvar o pedido" ||
		d.Kind != "tasks" || !strings.Contains(d.Text, "botão Salvar fica desligado") {
		t.Errorf("documento = %+v", d)
	}
	if _, ok := byID["tasks/register-order"]; !ok {
		t.Errorf("a introdução da página deveria ser um documento")
	}
	if _, ok := byID["glossary/pedido"]; !ok {
		t.Errorf("os termos do glossário deveriam entrar na busca")
	}
}

// maxDeveloperGuideBytes é o teto do guia do desenvolvedor do projeto de
// exemplo, sem media/ (ADR 0010).
const maxDeveloperGuideBytes = 6 * 1024 * 1024

func buildDeveloper(t *testing.T, cfg *config.Config) map[string][]byte {
	t.Helper()
	g, problems := content.LoadDeveloper(cfg)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	out := t.TempDir()
	if _, err := WriteDeveloper(cfg, g, out, nil); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(out, p)
			files[filepath.ToSlash(rel)], _ = os.ReadFile(p)
		}
		return nil
	})
	return files
}

func TestWriteDeveloperFiles(t *testing.T) {
	files := buildDeveloper(t, exampleConfig(t))
	want := []string{
		"assets/developer-data.js",
		"assets/guidebook.css",
		"assets/guidebook.js",
		"assets/mermaid.LICENSE.txt",
		"assets/mermaid.min.js",
		"assets/minisearch.LICENSE.txt",
		"assets/minisearch.js",
		"assets/theme.css",
		"developer-guide.html",
		"media/docs/assets/logo.svg",
	}
	if got := names(files); !reflect.DeepEqual(got, want) {
		t.Errorf("arquivos = %v\nesperava %v", got, want)
	}
}

func TestDeveloperGuideHTML(t *testing.T) {
	html := string(buildDeveloper(t, exampleConfig(t))["developer-guide.html"])
	for _, want := range []string{
		`<title>Guia do desenvolvedor — Pedido Fácil</title>`,
		`<a href="#/overview/overview">Visão geral</a>`,
		`<span class="gb-nav-group">Decisões de arquitetura</span>`,
		`<li><a href="#/decisions/0001-single-database">0001. Um só banco de dados</a></li>`,
		`<a href="#/business_rules">Regras de negócio</a>`,
		`<a href="#/changes">Novidades técnicas</a>`,
		`<article id="/overview/overview" class="gb-page" data-kind="overview">`,
		`<pre class="mermaid">C4Context`,
		`<article id="/business_rules" class="gb-page" data-kind="business_rules">`,
		`<h1>Regras de negócio</h1>`,
		`<span class="gb-nav-group">Módulos</span>`,
		`<p class="gb-rules-filter" hidden><label for="gb-rules-text">Filtrar regras</label> <input type="search" id="gb-rules-text" autocomplete="off"> <label for="gb-rules-module">Módulo</label> <select id="gb-rules-module"><option value="">Todos os módulos</option><option value="Pedidos">Pedidos</option></select></p>`,
		`<p class="gb-rules-none" hidden>Nenhuma regra encontrada</p>`,
		`<section class="gb-rules-module" id="/business_rules/orders" data-module="Pedidos">`,
		`<article id="/changes" class="gb-page" data-kind="changes">`,
		`<h1>Novidades técnicas</h1>`,
		`<h2>Não publicado <span class="gb-date"></span></h2>`,
		`<h2>1.1.0 <span class="gb-date">15/09/2026</span></h2>`,
		`<script src="assets/minisearch.js"></script>`,
		`<script src="assets/developer-data.js"></script>`,
		`<script src="assets/mermaid.min.js"></script>`,
		`<script src="assets/guidebook.js"></script>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML sem %q", want)
		}
	}
	for _, absent := range []string{"gb-present", "gb-presentation", "gb-video-dialog", "gb-open-full", "user-data.js"} {
		if strings.Contains(html, absent) {
			t.Errorf("o guia do desenvolvedor não tem %s", absent)
		}
	}
	m, d, g := strings.Index(html, "mermaid.min.js"), strings.Index(html, "developer-data.js"), strings.Index(html, "guidebook.js")
	if !(d < m && m < g) {
		t.Errorf("ordem dos scripts: dados %d, mermaid %d, guidebook %d", d, m, g)
	}
}

func TestDeveloperData(t *testing.T) {
	js := string(buildDeveloper(t, exampleConfig(t))["assets/developer-data.js"])
	for _, want := range []string{`"guide":"developer"`, `"home":"overview/overview"`, `"id":"business_rules/orders/pedido-precisa-de-cliente"`} {
		if !strings.Contains(js, want) {
			t.Errorf("developer-data.js sem %s", want)
		}
	}
}

// TestMermaidConfig: o guidebook inicializa o Mermaid com
// securityLevel strict e sem início automático.
func TestMermaidConfig(t *testing.T) {
	js := string(buildDeveloper(t, exampleConfig(t))["assets/guidebook.js"])
	for _, want := range []string{`securityLevel: "strict"`, `startOnLoad: false`, `window.mermaid.initialize(`, `window.mermaid.run(`} {
		if !strings.Contains(js, want) {
			t.Errorf("guidebook.js sem %s", want)
		}
	}
	for _, bad := range []string{"startOnLoad: true", `securityLevel: "loose"`, `securityLevel: "antiscript"`, "mermaid.init("} {
		if strings.Contains(js, bad) {
			t.Errorf("guidebook.js com %s", bad)
		}
	}
}

func TestDeveloperDeterministicAndCeiling(t *testing.T) {
	cfg := exampleConfig(t)
	a, b := buildDeveloper(t, cfg), buildDeveloper(t, cfg)
	total := 0
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s mudou entre dois builds iguais", name)
		}
		if !strings.HasPrefix(name, "media/") {
			total += len(a[name])
		}
	}
	t.Logf("guia do desenvolvedor do exemplo, sem media/: %d bytes", total)
	if total > maxDeveloperGuideBytes {
		t.Errorf("guia com %d bytes; o teto é %d (ADR 0010)", total, maxDeveloperGuideBytes)
	}
}

// TestEmptyReleaseHidden: marco sem itens não aparece em nenhum dos dois
// guias; só com marcos vazios, a seção inteira some.
func TestEmptyReleaseHidden(t *testing.T) {
	project := func(changelog string) *config.Config {
		t.Helper()
		dir := t.TempDir()
		files := map[string]string{
			"guidebook.yaml": "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n" +
				"developer:\n  sections:\n    changes:\n      sources: [CHANGELOG.md]\n",
			"docs/user/tasks/a.md": "# A\n",
			"CHANGELOG.md":         changelog,
		}
		for name, body := range files {
			p := filepath.Join(dir, filepath.FromSlash(name))
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.WriteFile(p, []byte(body), 0o644)
		}
		cfg, problems := config.Load(filepath.Join(dir, "guidebook.yaml"))
		if len(problems) > 0 {
			t.Fatal(problems)
		}
		return cfg
	}

	cfg := project("# Changelog\n\n## [Não publicado]\n\n## [0.2.0] - 2026-10-21\n\n### Adicionado\n\n## [0.1.0] - 2026-10-20\n\n### Adicionado\n\n- Primeira versão.\n")
	_, files := build(t, cfg)
	dev := buildDeveloper(t, cfg)
	for name, html := range map[string]string{"usuário": string(files["user-guide.html"]), "desenvolvedor": string(dev["developer-guide.html"])} {
		if !strings.Contains(html, "<h2>0.1.0 ") {
			t.Errorf("guia do %s: o marco com itens deveria aparecer", name)
		}
		for _, empty := range []string{"<h2>Não publicado", "<h2>0.2.0"} {
			if strings.Contains(html, empty) {
				t.Errorf("guia do %s: marco sem itens não deveria aparecer: %s", name, empty)
			}
		}
	}

	cfg = project("# Changelog\n\n## [Não publicado]\n\n## [0.1.0] - 2026-10-20\n\n### Adicionado\n")
	_, files = build(t, cfg)
	if strings.Contains(string(files["user-guide.html"]), `id="/whats-new"`) {
		t.Errorf("só com marcos vazios, \"O que há de novo\" deveria faltar")
	}
	if dev := buildDeveloper(t, cfg); strings.Contains(string(dev["developer-guide.html"]), `id="/changes"`) {
		t.Errorf("só com marcos vazios, \"Novidades técnicas\" deveria faltar")
	}
}

// TestMediaOptimized: o PNG de mídia sai com paleta, menor e com as
// mesmas dimensões; o logo e o vídeo saem como vieram (docs/spec.md,
// seção 9.4).
func TestMediaOptimized(t *testing.T) {
	cfg := exampleConfig(t)
	_, files := build(t, cfg)
	read := func(rel string) []byte {
		body, err := os.ReadFile(filepath.Join(cfg.Dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	for _, rel := range []string{"docs/user/media/register-order-1.png", "docs/user/media/videos/register-order.png"} {
		in, out := read(rel), files["media/"+rel]
		if len(out) >= len(in) {
			t.Errorf("%s: %d bytes → %d; esperava menor", rel, len(in), len(out))
			continue
		}
		before, _ := png.Decode(bytes.NewReader(in))
		after, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if _, ok := after.(*image.Paletted); !ok {
			t.Errorf("%s: saiu %T; esperava PNG com paleta", rel, after)
		}
		if before.Bounds() != after.Bounds() {
			t.Errorf("%s: dimensões %v → %v", rel, before.Bounds(), after.Bounds())
		}
	}
	for _, rel := range []string{"docs/assets/logo.svg", "docs/user/media/videos/register-order.webm"} {
		if !bytes.Equal(files["media/"+rel], read(rel)) {
			t.Errorf("%s: deveria sair como veio", rel)
		}
	}
}
