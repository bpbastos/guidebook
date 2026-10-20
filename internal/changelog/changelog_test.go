package changelog

import (
	"reflect"
	"strings"
	"testing"
)

const sample = `# Changelog

Mudanças relevantes.

## [Não publicado]

### Adicionado

- Relatório novo.

## [1.0.0] - 2026-08-01

### Added

- Registro de pedidos.

## [1.1.0] - 2026-09-15

### Corrigido

- Total com desconto.

### Security

- Senha mais forte.

[1.1.0]: https://exemplo.com/compare/v1.0.0...v1.1.0
`

func TestParse(t *testing.T) {
	releases, problems := Parse("CHANGELOG.md", []byte(sample))
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados:\n%s", problems)
	}
	if len(releases) != 3 {
		t.Fatalf("marcos = %d; esperava 3", len(releases))
	}
	r := releases[2]
	if r.Version != "1.1.0" || r.Date != "2026-09-15" || r.Unreleased {
		t.Errorf("marco = %+v", r)
	}
	var labels []string
	for _, g := range r.Groups {
		labels = append(labels, g.Label)
	}
	if want := []string{"Corrigido", "Segurança"}; !reflect.DeepEqual(labels, want) {
		t.Errorf("categorias = %v; esperava %v", labels, want)
	}
	if !strings.Contains(r.Groups[0].HTML, "<li>Total com desconto.</li>") {
		t.Errorf("HTML = %q", r.Groups[0].HTML)
	}
	if !releases[0].Unreleased || releases[0].Date != "" {
		t.Errorf("não publicado = %+v", releases[0])
	}
	if releases[1].Groups[0].Label != "Adicionado" {
		t.Errorf("Added deveria virar Adicionado: %+v", releases[1].Groups)
	}
}

func TestPublishedNewestFirst(t *testing.T) {
	releases, _ := Parse("CHANGELOG.md", []byte(sample))
	var got []string
	for _, r := range Published(releases) {
		got = append(got, r.Version)
	}
	if want := []string{"1.1.0", "1.0.0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("publicados = %v; esperava %v", got, want)
	}
}

func TestParseProblems(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"marco sem data", "## [1.0.0]\n\n### Adicionado\n\n- x\n", "CHANGELOG.md:1: marco sem data"},
		{"data inválida", "## [1.0.0] - 01/08/2026\n", "CHANGELOG.md:1: marco fora do formato"},
		{"sem colchetes", "## 1.0.0 - 2026-08-01\n", "CHANGELOG.md:1: marco fora do formato"},
		{"categoria desconhecida", "## [1.0.0] - 2026-08-01\n\n### Novidades\n\n- x\n", `CHANGELOG.md:3: categoria "Novidades" desconhecida`},
		{"item fora de categoria", "## [1.0.0] - 2026-08-01\n\n- x\n", "CHANGELOG.md:3: conteúdo fora de uma categoria"},
		{"não publicado com data", "## [Unreleased] - 2026-08-01\n", "CHANGELOG.md:1: o marco não publicado não leva data"},
		{"link com esquema proibido", "## [1.0.0] - 2026-08-01\n\n### Adicionado\n\n- [x](javascript:alert(1))\n", `CHANGELOG.md:5: esquema "javascript:" não é permitido`},
		{"link relativo", "## [1.0.0] - 2026-08-01\n\n### Adicionado\n\n- [x](docs/a.md)\n", "CHANGELOG.md:5: link relativo não é aceito no CHANGELOG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := Parse("CHANGELOG.md", []byte(tt.src))
			for _, p := range problems {
				if strings.HasPrefix(p.String(), tt.want) {
					return
				}
			}
			t.Errorf("esperava %q; veio:\n%s", tt.want, problems)
		})
	}
}
