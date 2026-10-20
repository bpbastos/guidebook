package diag

import "testing"

func TestString(t *testing.T) {
	var l List
	l.Addf("b.md", 3, "", "frase com %d palavras", 31)
	l.Addf("guidebook.yaml", 0, "user.dir", "obrigatório")
	l.Addf("a.md", 0, "", "não é esperado aqui")
	l.Addf("b.md", 1, "", "HTML cru")
	want := "a.md: não é esperado aqui\n" +
		"b.md:1: HTML cru\n" +
		"b.md:3: frase com 31 palavras\n" +
		"guidebook.yaml: user.dir: obrigatório\n"
	if got := l.String(); got != want {
		t.Errorf("String() =\n%s\nesperava\n%s", got, want)
	}
}
