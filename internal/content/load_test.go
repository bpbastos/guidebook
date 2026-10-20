package content

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/config"
)

func loadExample(t *testing.T) *Guide {
	t.Helper()
	cfg, problems := config.Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	g, problems := Load(cfg)
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados:\n%s", problems)
	}
	return g
}

func page(t *testing.T, g *Guide, id string) *Page {
	t.Helper()
	for _, p := range g.Pages {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("página %q não encontrada", id)
	return nil
}

func TestLoadExample(t *testing.T) {
	g := loadExample(t)
	if len(g.Pages) != 7 {
		t.Fatalf("páginas = %d; esperava 7", len(g.Pages))
	}
	task := page(t, g, "tasks/register-order")
	if task.Title != "Registrar um pedido" {
		t.Errorf("título = %q", task.Title)
	}
	wantBlocked := []Heading{{Level: 2, Text: "O sistema não me deixa salvar o pedido", Anchor: "o-sistema-nao-me-deixa-salvar-o-pedido"}}
	if !reflect.DeepEqual(task.Blocked, wantBlocked) {
		t.Errorf("bloqueios = %+v", task.Blocked)
	}
	for _, want := range []string{
		`<h1 id="/tasks/register-order/registrar-um-pedido">Registrar um pedido</h1>`,
		`<h2 id="/tasks/register-order/registre-o-pedido">`,
		`<a href="#/reference/order-status" data-gb="dialog">estados do pedido</a>`,
		`<figure class="gb-shot" id="gb-shot-tasks-register-order-1">`,
		`<ol data-gb-steps="">`,
		`<a class="gb-video-link" href="media/docs/user/media/videos/register-order.webm" data-poster="media/docs/user/media/videos/register-order.png">Ver em vídeo (1 min)</a>`,
	} {
		if !strings.Contains(task.HTML, want) {
			t.Errorf("HTML da tarefa sem %q:\n%s", want, task.HTML)
		}
	}
	tutorial := page(t, g, "tutorials/first-order")
	for _, want := range []string{
		`<a href="#/glossary/pedido" data-gb="dialog">pedido</a>`,
		`<a href="#/tasks/register-order" data-gb="nav">Registrar um pedido</a>`,
	} {
		if !strings.Contains(tutorial.HTML, want) {
			t.Errorf("HTML do tutorial sem %q:\n%s", want, tutorial.HTML)
		}
	}
	if want := []string{"docs/user/media/register-order-1.png", "docs/user/media/videos/register-order.png", "docs/user/media/videos/register-order.webm"}; !reflect.DeepEqual(g.Media, want) {
		t.Errorf("mídia = %v", g.Media)
	}
	if len(g.Releases) != 3 {
		t.Errorf("marcos = %d; esperava 3", len(g.Releases))
	}
}

func TestSlug(t *testing.T) {
	tests := map[string]string{
		"Registre o pedido":        "registre-o-pedido",
		"Ação: Não É Fácil!":       "acao-nao-e-facil",
		"  -- 2 vezes --  ":        "2-vezes",
		"Configuração do `go.mod`": "configuracao-do-go-mod",
		"???":                      "secao",
		"Über çà":                  "uber-ca",
	}
	for in, want := range tests {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q; esperava %q", in, got, want)
		}
	}
}

const linkConfig = "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n"

// linkProject cria um projeto com páginas fixas e tasks/a.md com body.
func linkProject(t *testing.T, body string) (*Guide, []string) {
	t.Helper()
	cfg := projectFiles(t, linkConfig, map[string]string{
		"docs/user/tasks/a.md":      "# A\n\n" + body + "\n\n## Seção local\n\nTexto.\n",
		"docs/user/tasks/b.md":      "# B\n\n## O sistema não me deixa salvar\n\nPorque sim.\n",
		"docs/user/reference/r.md":  "# R\n",
		"docs/user/glossary.md":     "# Glossário\n\n## Pedido\n\nLista de itens.\n",
		"docs/user/media/tela.png":  "png",
		"docs/user/media/notas.txt": "x",
	})
	g, problems := Load(cfg)
	var out []string
	for _, p := range problems.Sorted() {
		out = append(out, strings.TrimPrefix(filepath.ToSlash(p.String()), filepath.ToSlash(cfg.Dir)+"/"))
	}
	return g, out
}

func TestLinks(t *testing.T) {
	tests := []struct {
		name, link, html string
	}{
		{"tarefa navega", "[x](b.md)", `<a href="#/tasks/b" data-gb="nav">x</a>`},
		{"referência em diálogo", "[x](../reference/r.md)", `<a href="#/reference/r" data-gb="dialog">x</a>`},
		{"termo do glossário em diálogo", "[x](../glossary.md#pedido)", `<a href="#/glossary/pedido" data-gb="dialog">x</a>`},
		{"glossário sem âncora navega", "[x](../glossary.md)", `<a href="#/glossary" data-gb="nav">x</a>`},
		{"bloqueio em diálogo", "[x](b.md#o-sistema-nao-me-deixa-salvar)", `<a href="#/tasks/b/o-sistema-nao-me-deixa-salvar" data-gb="dialog">x</a>`},
		{"âncora local", "[x](#secao-local)", `<a href="#/tasks/a/secao-local" data-gb="nav">x</a>`},
		{"externo", "[x](https://exemplo.com/a)", `<a href="https://exemplo.com/a" target="_blank" rel="noopener noreferrer">x</a>`},
		{"e-mail", "[x](mailto:a@exemplo.com)", `<a href="mailto:a@exemplo.com" target="_blank" rel="noopener noreferrer">x</a>`},
		{"imagem", "![Tela de pedido](../media/tela.png)", `<img src="media/docs/user/media/tela.png" alt="Tela de pedido">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, problems := linkProject(t, tt.link)
			if len(problems) > 0 {
				t.Fatalf("problemas inesperados: %v", problems)
			}
			if a := page(t, g, "tasks/a"); !strings.Contains(a.HTML, tt.html) {
				t.Errorf("HTML sem %q:\n%s", tt.html, a.HTML)
			}
		})
	}
}

func TestImageMedia(t *testing.T) {
	g, problems := linkProject(t, "![Tela](../media/tela.png) e ![Tela de novo](../media/tela.png)")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if want := []string{"docs/user/media/tela.png"}; !reflect.DeepEqual(g.Media, want) {
		t.Errorf("mídia = %v; esperava %v", g.Media, want)
	}
}

func TestLinkProblems(t *testing.T) {
	tests := []struct {
		name, link, want string
	}{
		{"javascript", "[x](javascript:alert(1))", `docs/user/tasks/a.md:3: esquema "javascript:" não é permitido; use http, https ou mailto`},
		{"data", "[x](data:text/html,oi)", `docs/user/tasks/a.md:3: esquema "data:" não é permitido; use http, https ou mailto`},
		{"vbscript", "[x](vbscript:msgbox)", `docs/user/tasks/a.md:3: esquema "vbscript:" não é permitido`},
		{"file", "[x](file:///etc/passwd)", `docs/user/tasks/a.md:3: esquema "file:" não é permitido`},
		{"autolink javascript", "<javascript:alert(1)>", `docs/user/tasks/a.md:3: esquema "javascript:" não é permitido`},
		{"sem esquema", "[x](//exemplo.com)", `docs/user/tasks/a.md:3: link sem esquema; use http:// ou https://`},
		{"arquivo inexistente", "[x](c.md)", `docs/user/tasks/a.md:3: link para "docs/user/tasks/c.md", que não existe`},
		{"arquivo fora do guia", "[x](../../../guidebook.yaml)", `docs/user/tasks/a.md:3: link para "guidebook.yaml", que não é página do guia`},
		{"mídia não é página", "[x](../media/notas.txt)", `docs/user/tasks/a.md:3: link para "docs/user/media/notas.txt", que não é página do guia`},
		{"sai do projeto", "[x](../../../../fora.md)", `docs/user/tasks/a.md:3: link sai do diretório do guidebook.yaml`},
		{"âncora inexistente", "[x](b.md#nada)", `docs/user/tasks/a.md:3: âncora "#nada" não existe em docs/user/tasks/b.md`},
		{"âncora local inexistente", "[x](#nada)", `docs/user/tasks/a.md:3: âncora "#nada" não existe em docs/user/tasks/a.md`},
		{"imagem sem alternativo", "![](../media/tela.png)", `docs/user/tasks/a.md:3: imagem sem texto alternativo`},
		{"imagem inexistente", "![x](../media/falta.png)", `docs/user/tasks/a.md:3: imagem "docs/user/media/falta.png" não existe`},
		{"imagem externa", "![x](https://exemplo.com/a.png)", `docs/user/tasks/a.md:3: imagem externa não é permitida; o guia abre sem internet`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := linkProject(t, tt.link)
			for _, p := range problems {
				if strings.HasPrefix(p, tt.want) {
					return
				}
			}
			t.Errorf("esperava %q; veio:\n%s", tt.want, strings.Join(problems, "\n"))
		})
	}
}

func TestTitleProblems(t *testing.T) {
	cfg := projectFiles(t, linkConfig, map[string]string{
		"docs/user/tasks/sem.md":  "Texto sem título.\n",
		"docs/user/tasks/dois.md": "# Um\n\nTexto.\n\n# Dois\n",
	})
	_, problems := Load(cfg)
	got := problems.String()
	for _, want := range []string{
		"dois.md:5: segundo título de nível 1; a página só pode ter um",
		"sem.md:1: página sem título; comece com \"# Título\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("esperava %q; veio:\n%s", want, got)
		}
	}
}

func TestLoadReportsChangelogAndScan(t *testing.T) {
	cfg := projectFiles(t, linkConfig, map[string]string{
		"docs/user/tasks/a.md": "# A\n",
		"docs/user/extra.md":   "# X\n",
		"CHANGELOG.md":         "# Changelog\n\n## [1.0] - ontem\n",
	})
	_, problems := Load(cfg)
	got := problems.String()
	for _, want := range []string{"CHANGELOG.md:3: marco fora do formato", "extra.md: não é esperado aqui"} {
		if !strings.Contains(got, want) {
			t.Errorf("esperava %q; veio:\n%s", want, got)
		}
	}
}

func TestDuplicateAnchors(t *testing.T) {
	g, problems := linkProject(t, "## Passo\n\nUm.\n\n## Passo\n\nDois.")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	a := page(t, g, "tasks/a")
	if !strings.Contains(a.HTML, `id="/tasks/a/passo"`) || !strings.Contains(a.HTML, `id="/tasks/a/passo-2"`) {
		t.Errorf("âncoras repetidas deveriam ganhar -2:\n%s", a.HTML)
	}
}

// projectFiles cria um projeto com o guidebook.yaml e os arquivos dados.
func projectFiles(t *testing.T, yaml string, files map[string]string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	files["guidebook.yaml"] = yaml
	files["docs/user/.keep"] = ""
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, problems := config.Load(filepath.Join(dir, "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatalf("configuração inválida:\n%s", problems)
	}
	return cfg
}

func TestSections(t *testing.T) {
	g := loadExample(t)
	got := page(t, g, "tasks/register-order").Sections
	want := []Section{
		{Anchor: "", Title: "Registrar um pedido", Text: ""},
		{Anchor: "registre-o-pedido", Title: "Registre o pedido", Text: "Tela de novo pedido Escolha o cliente. Adicione os itens. Clique em Salvar. Veja os estados do pedido."},
		{Anchor: "o-sistema-nao-me-deixa-salvar-o-pedido", Title: "O sistema não me deixa salvar o pedido", Text: "O botão Salvar fica desligado enquanto falta o cliente. Escolha o cliente e o botão liga."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("seções =\n%+v\nesperava\n%+v", got, want)
	}
}

func TestSectionsIncludeCodeAndSubheadings(t *testing.T) {
	g, problems := linkProject(t, "Rode o comando:\n\n```sh\nguidebook check\n```\n\n### Detalhe\n\n| a | b |\n|---|---|\n| um | dois |")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	got := page(t, g, "tasks/a").Sections
	if len(got) != 2 || got[0].Text != "Rode o comando: guidebook check Detalhe a b um dois" || got[1].Title != "Seção local" {
		t.Errorf("seções = %+v", got)
	}
}

func TestSlides(t *testing.T) {
	g := loadExample(t)
	html := page(t, g, "tasks/register-order").HTML
	if n := strings.Count(html, `<div class="gb-slide">`); n != 3 {
		t.Errorf("slides = %d; esperava 3 (título e duas seções \"## \")\n%s", n, html)
	}
	if !strings.HasPrefix(html, `<div class="gb-slide">`+"\n"+`<h1 `) {
		t.Errorf("o primeiro slide deveria começar pelo título:\n%s", html)
	}
}

func TestSlidesCutAtThematicBreak(t *testing.T) {
	g, problems := linkProject(t, "Intro.\n\n---\n\nDepois do corte.\n\n### Sub\n\nNão corta.")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	html := page(t, g, "tasks/a").HTML
	slides := strings.Split(html, `<div class="gb-slide">`)[1:]
	if len(slides) != 3 {
		t.Fatalf("slides = %d; esperava 3:\n%s", len(slides), html)
	}
	if !strings.HasPrefix(slides[1], "\n<hr>") || !strings.Contains(slides[1], "Não corta.") {
		t.Errorf("o segundo slide deveria começar no --- e levar o ###:\n%s", slides[1])
	}
	if !strings.Contains(slides[2], "Seção local") {
		t.Errorf("o terceiro slide deveria ser a seção local:\n%s", slides[2])
	}
}
