package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/config"
)

// words devolve n vezes a palavra "palavra", separadas por espaço.
func words(n int) string {
	w := make([]string, n)
	for i := range w {
		w[i] = "palavra"
	}
	return strings.Join(w, " ")
}

func check(t *testing.T, limit int, patterns []config.ForbiddenPattern, src string) []string {
	t.Helper()
	c := New(config.Rules{MaxSentenceWords: &limit, ForbiddenPatterns: patterns})
	var out []string
	for _, p := range c.Check("doc.md", []byte(src)).Sorted() {
		out = append(out, p.String())
	}
	return out
}

func TestSentenceLength(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"no limite", words(5) + ".\n", nil},
		{"acima do limite", words(6) + ".\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"linha do início da frase", "# T\n\nCurta.\n" + words(3) + "\n" + words(3) + ".\n", []string{`doc.md:4: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"ponto separa frases", words(5) + ". " + words(5) + "! " + words(5) + "? " + words(5) + ": " + words(5) + "\n", nil},
		{"ponto sem espaço não separa", "veja exemplo.com.br agora mesmo.\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "veja exemplo.com.br agora mesmo"`}},
		{"pontuação seguida de parêntese", "(" + words(5) + ".) " + words(5) + ".\n", nil},
		{"fim do bloco termina a frase", words(5) + "\n\n" + words(5) + "\n", nil},
		{"hífen e apóstrofo", "o guarda-chuva d'água fica aqui.\n", nil},
		{"número com separadores é uma palavra", "pagou 1.234,56 e 3.5 hoje.\n", nil},
		{"números contam um cada", "1.234,56 7,5 8.0 9 10 11.\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "1.234,56 7,5 8.0 9 10 11"`}},
		{"vírgula com espaço separa", "1, 2, 3, 4, 5, 6.\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "1, 2, 3, 4, 5, 6"`}},
		{"código em linha fora", words(4) + " `um dois três quatro`.\n", nil},
		{"bloco de código fora", "```\n" + words(30) + "\n```\n", nil},
		{"código indentado fora", "    " + words(30) + "\n", nil},
		{"url fora", "veja https://exemplo.com/um/dois/tres/quatro agora.\n", nil},
		{"autolink fora", "veja <https://exemplo.com/um/dois/tres/quatro> agora.\n", nil},
		{"texto do link conta", "[" + words(6) + "](https://exemplo.com).\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"destino do link não conta", "[um](https://exemplo.com/a/b/c/d/e/f).\n", nil},
		{"título", "# " + words(6) + "\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"item de lista", "- " + words(3) + "\n- " + words(6) + "\n", []string{`doc.md:2: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"célula de tabela", "| a | b |\n|---|---|\n| " + words(6) + " | x |\n", []string{`doc.md:3: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"texto alternativo", "![" + words(6) + "](a.png)\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"alternativo separado do parágrafo", words(3) + " ![" + words(3) + "](a.png) fim.\n", nil},
		{"citação em bloco", "> " + words(6) + "\n", []string{`doc.md:1: frase com 6 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra"`}},
		{"trecho com pontuação e código", "Use `make test`, depois veja: isto, aquilo e mais.\n", nil},
		{"trecho original", "Rode `go test`, " + words(3) + "\n" + words(3) + ".\n", []string{"doc.md:1: frase com 7 palavras (limite: 5): \"Rode `go test`, palavra palavra palavra palavra palavra palavra\""}},
		{"trecho encurtado", words(12) + "\n", []string{`doc.md:1: frase com 12 palavras (limite: 5): "palavra palavra palavra palavra palavra palavra palavra palavra…"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := check(t, 5, nil, tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("violações =\n%q\nesperava\n%q", got, tt.want)
			}
		})
	}
}

func TestForbiddenPatterns(t *testing.T) {
	patterns := []config.ForbiddenPattern{
		{Pattern: `\b[A-Z]{2,}-\d+\b`, Message: "não cite chamado interno"},
		{Pattern: `intranet\.example`, Message: "não cite a intranet"},
	}
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"no texto", "Veja o ABC-123 hoje.\n", []string{`doc.md:1: não cite chamado interno: "ABC-123"`}},
		{"em código em linha", "Rode `ABC-9`.\n", []string{`doc.md:1: não cite chamado interno: "ABC-9"`}},
		{"no destino do link", "\n[aqui](https://intranet.example/x).\n", []string{`doc.md:2: não cite a intranet: "intranet.example"`}},
		{"em url solta", "Veja https://intranet.example/x.\n", []string{`doc.md:1: não cite a intranet: "intranet.example"`}},
		{"no título", "# Sobre XY-1\n", []string{`doc.md:1: não cite chamado interno: "XY-1"`}},
		{"bloco de código fora", "```\nABC-123\n```\n", nil},
		{"várias ocorrências", "AB-1 e\nCD-2.\n", []string{`doc.md:1: não cite chamado interno: "AB-1"`, `doc.md:2: não cite chamado interno: "CD-2"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := check(t, 25, patterns, tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("violações =\n%q\nesperava\n%q", got, tt.want)
			}
		})
	}
}

func TestRawHTML(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"bloco", "# T\n\n<div>\noi\n</div>\n", []string{"doc.md:3: HTML cru não é permitido no Markdown"}},
		{"comentário", "<!-- nota -->\n", []string{"doc.md:1: HTML cru não é permitido no Markdown"}},
		{"em linha", "Linha<br>quebrada.\n", []string{"doc.md:1: HTML cru não é permitido no Markdown"}},
		{"autolink não conta", "Veja <https://exemplo.com>.\n", nil},
		{"dentro de código não conta", "Use `<br>` e\n\n```\n<div></div>\n```\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := check(t, 25, nil, tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("violações =\n%q\nesperava\n%q", got, tt.want)
			}
		})
	}
}

func TestAllViolationsSorted(t *testing.T) {
	src := "# T\n\n" + words(30) + ".\n\n<b>x</b> ZZ-1.\n"
	limit := 25
	c := New(config.Rules{MaxSentenceWords: &limit, ForbiddenPatterns: []config.ForbiddenPattern{{Pattern: `ZZ-\d`, Message: "proibido"}}})
	got := c.Check("doc.md", []byte(src)).Sorted()
	if len(got) != 3 || got[0].Line != 3 || got[1].Line != 5 || got[2].Line != 5 {
		t.Errorf("violações = %v", got)
	}
}
