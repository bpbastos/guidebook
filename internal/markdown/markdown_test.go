package markdown

import "testing"

func TestScheme(t *testing.T) {
	tests := map[string]string{
		"https://exemplo.com":   "https",
		"JavaScript:alert(1)":   "javascript",
		"data:text/html,oi":     "data",
		"mailto:a@exemplo.com":  "mailto",
		"../tasks/a.md":         "",
		"#secao":                "",
		"a.md#secao:com-dois":   "",
		"//exemplo.com/caminho": "",
	}
	for dest, want := range tests {
		if got := Scheme(dest); got != want {
			t.Errorf("Scheme(%q) = %q; esperava %q", dest, got, want)
		}
	}
	for _, s := range []string{"http", "https", "mailto"} {
		if !ExternalScheme(s) {
			t.Errorf("%s deveria ser aceito", s)
		}
	}
	for _, s := range []string{"javascript", "data", "vbscript", "file", ""} {
		if ExternalScheme(s) {
			t.Errorf("%q deveria ser recusado", s)
		}
	}
}

func TestPlainTextAndLines(t *testing.T) {
	src := []byte("# Título com `código` e *ênfase*\n\nTexto\nem duas linhas.\n")
	doc := Parse(src)
	if got := PlainText(doc.FirstChild(), src); got != "Título com código e ênfase" {
		t.Errorf("PlainText = %q", got)
	}
	para := doc.FirstChild().NextSibling()
	if got := PlainText(para, src); got != "Texto em duas linhas." {
		t.Errorf("PlainText do parágrafo = %q", got)
	}
	lines := NewLines(src)
	if got := lines.Line(BlockOffset(para)); got != 3 {
		t.Errorf("linha do parágrafo = %d; esperava 3", got)
	}
	if got := lines.Line(FirstOffset(para, -1)); got != 3 {
		t.Errorf("linha do primeiro texto = %d; esperava 3", got)
	}
}
