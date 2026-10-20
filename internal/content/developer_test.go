package content

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/config"
)

func TestLoadDeveloperExample(t *testing.T) {
	cfg, problems := config.Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	g, problems := LoadDeveloper(cfg)
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados:\n%s", problems)
	}
	var keys, titles []string
	for _, s := range g.Sections {
		keys = append(keys, s.Key)
		titles = append(titles, s.Title)
	}
	if want := []string{"overview", "modules", "business_rules", "data_model", "decisions", "development", "changes"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("seções = %v; esperava %v", keys, want)
	}
	if want := []string{"Visão geral", "Módulos", "Regras de negócio", "Modelo de dados", "Decisões de arquitetura", "Como manter e evoluir", "Novidades técnicas"}; !reflect.DeepEqual(titles, want) {
		t.Errorf("títulos = %v; esperava %v", titles, want)
	}
	var ids []string
	for _, p := range g.Pages {
		ids = append(ids, p.ID)
	}
	if want := []string{"overview/overview", "modules/orders", "business_rules/orders", "data_model/data-model", "decisions/0001-single-database", "development/development"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("páginas = %v; esperava %v", ids, want)
	}
	if len(g.Releases) != 3 || !g.Releases[0].Unreleased {
		t.Errorf("o guia do desenvolvedor mostra todos os marcos, com o não publicado: %+v", g.Releases)
	}
	overview := page(t, g, "overview/overview")
	if !strings.Contains(overview.HTML, "<pre class=\"mermaid\">C4Context\n  Person(attendant, &#34;Atendente&#34;)") {
		t.Errorf("o bloco mermaid deveria virar <pre class=\"mermaid\"> com o texto escapado:\n%s", overview.HTML)
	}
	if strings.Contains(overview.HTML, "language-mermaid") {
		t.Errorf("o bloco mermaid não deveria sair como código comum")
	}
}

const devConfig = `version: 1
product:
  name: P
developer:
  sections:
    overview:
      sources: [docs/a.md]
    decisions:
      sources: [docs/adr/*.md]
    changes:
      sources: [CHANGELOG.md]
`

func devProject(t *testing.T, yaml, body string) (*Guide, []string) {
	t.Helper()
	cfg := projectFiles(t, yaml, map[string]string{
		"docs/a.md":            "# A\n\n" + body + "\n",
		"docs/adr/0001-x.md":   "# 0001. X\n\n## Contexto\n\nTexto.\n",
		"CHANGELOG.md":         "# Changelog\n\n## [Não publicado]\n\n### Adicionado\n\n- Algo.\n",
		"go.mod":               "module x\n",
		"docs/user/tasks/t.md": "# T\n",
	})
	g, problems := LoadDeveloper(cfg)
	var out []string
	for _, p := range problems.Sorted() {
		out = append(out, strings.TrimPrefix(filepath.ToSlash(p.String()), filepath.ToSlash(cfg.Dir)+"/"))
	}
	return g, out
}

func TestDeveloperLinks(t *testing.T) {
	withRepo := strings.Replace(devConfig, "developer:\n", "developer:\n  repository_url: https://github.com/o/r/blob/main\n", 1)
	tests := []struct {
		name, yaml, link, html string
	}{
		{"fonte do guia", devConfig, "[x](adr/0001-x.md)", `<a href="#/decisions/0001-x" data-gb="nav">x</a>`},
		{"âncora de fonte", devConfig, "[x](adr/0001-x.md#contexto)", `<a href="#/decisions/0001-x/contexto" data-gb="nav">x</a>`},
		{"changelog", devConfig, "[x](../CHANGELOG.md)", `<a href="#/changes" data-gb="nav">x</a>`},
		{"arquivo do repositório", withRepo, "[x](../go.mod)", `<a href="https://github.com/o/r/blob/main/go.mod" target="_blank" rel="noopener noreferrer">x</a>`},
		{"página do guia do usuário", withRepo, "[x](user/tasks/t.md)", `<a href="https://github.com/o/r/blob/main/docs/user/tasks/t.md" target="_blank" rel="noopener noreferrer">x</a>`},
		{"externo", devConfig, "[x](https://exemplo.com)", `<a href="https://exemplo.com" target="_blank" rel="noopener noreferrer">x</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, problems := devProject(t, tt.yaml, tt.link)
			if len(problems) > 0 {
				t.Fatalf("problemas inesperados: %v", problems)
			}
			if a := page(t, g, "overview/a"); !strings.Contains(a.HTML, tt.html) {
				t.Errorf("HTML sem %q:\n%s", tt.html, a.HTML)
			}
		})
	}
}

func TestDeveloperLinkProblems(t *testing.T) {
	tests := []struct {
		name, link, want string
	}{
		{"fora do guia sem repository_url", "[x](../go.mod)", `docs/a.md:3: link para "go.mod", que não é fonte do guia; defina developer.repository_url para apontar para o repositório`},
		{"inexistente", "[x](falta.md)", `docs/a.md:3: link para "docs/falta.md", que não existe`},
		{"âncora inexistente", "[x](adr/0001-x.md#nada)", `docs/a.md:3: âncora "#nada" não existe em docs/adr/0001-x.md`},
		{"esquema proibido", "[x](javascript:alert(1))", `docs/a.md:3: esquema "javascript:" não é permitido`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := devProject(t, devConfig, tt.link)
			for _, p := range problems {
				if strings.HasPrefix(p, tt.want) {
					return
				}
			}
			t.Errorf("esperava %q; veio:\n%s", tt.want, strings.Join(problems, "\n"))
		})
	}
}

func TestDeveloperTitleRequired(t *testing.T) {
	cfg := projectFiles(t, devConfig, map[string]string{
		"docs/a.md":          "Sem título.\n",
		"docs/adr/0001-x.md": "# X\n",
		"CHANGELOG.md":       "# Changelog\n",
	})
	_, problems := LoadDeveloper(cfg)
	if !strings.Contains(problems.String(), "a.md:1: página sem título") {
		t.Errorf("veio:\n%s", problems)
	}
}

func TestUserGuideKeepsMermaidAsCode(t *testing.T) {
	g, problems := linkProject(t, "```mermaid\ngraph TD\n```")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if html := page(t, g, "tasks/a").HTML; !strings.Contains(html, `<pre><code class="language-mermaid">`) {
		t.Errorf("no guia do usuário, mermaid continua código comum:\n%s", html)
	}
}
