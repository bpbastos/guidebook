package content

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/config"
)

func project(t *testing.T, yaml string, files ...string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, "/") {
			return
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("guidebook.yaml", yaml)
	write("docs/user/", "")
	for _, f := range files {
		write(f, "# T\n")
	}
	cfg, problems := config.Load(filepath.Join(dir, "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatalf("configuração inválida:\n%s", problems)
	}
	return cfg
}

const base = "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n"

func TestScanExample(t *testing.T) {
	cfg, problems := config.Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	pages, problems := Scan(cfg)
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados:\n%s", problems)
	}
	var ids []string
	for _, p := range pages {
		ids = append(ids, p.ID)
	}
	want := []string{
		"index", "glossary",
		"tutorials/first-order",
		"tasks/register-order", "tasks/close-day",
		"reference/order-status",
		"explanation/how-billing-works",
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("ids = %v\nesperava %v", ids, want)
	}
	if got := pages[3]; got.Kind != KindTask || got.File != "docs/user/tasks/01-register-order.md" {
		t.Errorf("página = %+v", got)
	}
	if problems := CheckReferences(cfg, pages); len(problems) > 0 {
		t.Errorf("referências do exemplo deveriam valer:\n%s", problems)
	}
}

func TestScanOrderAndPrefix(t *testing.T) {
	cfg := project(t, base,
		"docs/user/tasks/10-b.md", "docs/user/tasks/02-c.md", "docs/user/tasks/a.md",
		"docs/user/.DS_Store", "docs/user/tasks/.notas.md")
	pages, problems := Scan(cfg)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	var ids []string
	for _, p := range pages {
		ids = append(ids, p.ID)
	}
	// Ordem de bytes do nome do arquivo: 02-c, 10-b, a.
	if want := []string{"tasks/c", "tasks/b", "tasks/a"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v; esperava %v", ids, want)
	}
}

func TestScanProblems(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		file  string
		msg   string
	}{
		{"arquivo que não é md", []string{"docs/user/tasks/x.txt"}, "docs/user/tasks/x.txt", "só arquivos .md"},
		{"subdiretório", []string{"docs/user/tasks/sub/x.md"}, "docs/user/tasks/sub", "subdiretório"},
		{"entrada estranha na raiz", []string{"docs/user/extra.md"}, "docs/user/extra.md", "não é esperado"},
		{"diretório estranho na raiz", []string{"docs/user/howto/x.md"}, "docs/user/howto", "não é esperado"},
		{"id repetido", []string{"docs/user/tasks/01-a.md", "docs/user/tasks/02-a.md"}, "docs/user/tasks/02-a.md", `"tasks/a" repetido`},
		{"nome vazio depois do prefixo", []string{"docs/user/tasks/01-.md"}, "docs/user/tasks/01-.md", "nome vazio"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := project(t, base, tt.files...)
			_, problems := Scan(cfg)
			for _, p := range problems {
				if strings.HasSuffix(filepath.ToSlash(p.File), tt.file) && strings.Contains(p.Msg, tt.msg) {
					return
				}
			}
			t.Errorf("esperava %s com %q; veio:\n%s", tt.file, tt.msg, problems)
		})
	}
}

func TestCheckReferences(t *testing.T) {
	yaml := base + `  profiles:
    - id: a
      name: A
      start_here: [tasks/existe, tasks/falta]
  videos:
    manifest: m.json
    tasks: [tasks/existe, reference/r, tasks/sumiu]
`
	cfg := project(t, yaml, "docs/user/tasks/existe.md", "docs/user/reference/r.md", "m.json")
	pages, problems := Scan(cfg)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	got := map[string]string{}
	for _, p := range CheckReferences(cfg, pages) {
		got[p.Field] = p.Msg
	}
	want := map[string]string{
		"user.profiles[0].start_here[1]": `página "tasks/falta" não existe`,
		"user.videos.tasks[1]":           `"reference/r" não é uma tarefa (tasks/)`,
		"user.videos.tasks[2]":           `página "tasks/sumiu" não existe`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("problemas = %v\nesperava %v", got, want)
	}
}
