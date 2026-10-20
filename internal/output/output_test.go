package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range list {
		names = append(names, e.Name())
	}
	return names
}

func TestPrepareCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out", "guides")
	if err := Prepare(dir); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); len(got) != 1 || got[0] != Marker {
		t.Errorf("conteúdo = %v; esperava só %s", got, Marker)
	}
}

func TestPrepareEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := Prepare(dir); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); len(got) != 1 || got[0] != Marker {
		t.Errorf("conteúdo = %v", got)
	}
}

func TestPrepareClearsPreviousOutput(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{Marker, "user-guide.html", "assets/guidebook.js"} {
		p := filepath.Join(dir, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
	}
	if err := Prepare(dir); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); len(got) != 1 || got[0] != Marker {
		t.Errorf("conteúdo = %v; a saída anterior deveria ter sido apagada", got)
	}
}

func TestPrepareRefusesForeignDir(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "guidebook.yaml")
	os.WriteFile(keep, []byte("version: 1\n"), 0o644)
	err := Prepare(dir)
	if err == nil || !strings.Contains(err.Error(), Marker) {
		t.Fatalf("erro = %v; esperava recusa citando %s", err, Marker)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("o arquivo do projeto sumiu: %v", err)
	}
}

func TestPrepareRefusesFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "out")
	os.WriteFile(file, []byte("x"), 0o644)
	if err := Prepare(file); err == nil || !strings.Contains(err.Error(), "não é um diretório") {
		t.Fatalf("erro = %v", err)
	}
}
