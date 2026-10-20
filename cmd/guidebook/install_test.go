package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/bpbastos/guidebook"

// TestGoInstall instala o guidebook como o usuário instala, por
// go install a partir de um módulo de verdade, e roda o build do projeto
// de exemplo com o binário instalado. O zip de módulo descarta arquivos
// que o repositório tem, como os de subdiretórios de "vendor"; só este
// caminho pega isso. Sem rede: o módulo vem de um repositório git bare
// temporário com a árvore atual, e as dependências, do cache de módulos
// local.
func TestGoInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("instalação por go install fica fora do -short")
	}
	for _, tool := range []string{"git", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s ausente: %v", tool, err)
		}
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	modcache := strings.TrimSpace(cmdOutput(t, root, nil, "go", "env", "GOMODCACHE"))

	tmp := t.TempDir()
	gitconfig := filepath.Join(tmp, "gitconfig")
	bare := filepath.Join(tmp, "repo.git")
	err = os.WriteFile(gitconfig, []byte(`[url "file://`+bare+`"]
	insteadOf = https://`+modulePath+`
[protocol "file"]
	allow = always
[user]
	name = guidebook
	email = guidebook@example.invalid
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	env := []string{
		"GIT_CONFIG_GLOBAL=" + gitconfig,
		"GIT_CONFIG_NOSYSTEM=1",
		"GOPRIVATE=" + modulePath,
		"GOPROXY=file://" + filepath.ToSlash(filepath.Join(modcache, "cache", "download")),
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOFLAGS=-modcacherw",
		"GOPATH=" + filepath.Join(tmp, "gopath"),
		"GOMODCACHE=" + filepath.Join(tmp, "gopath", "pkg", "mod"),
		"GOBIN=" + filepath.Join(tmp, "bin"),
	}

	// A árvore atual, com o que ainda não foi commitado e sem o que o
	// .gitignore exclui, vira um commit no repositório bare.
	tree := filepath.Join(tmp, "tree")
	files := cmdOutput(t, root, nil, "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	for name := range strings.SplitSeq(strings.TrimRight(files, "\x00"), "\x00") {
		body, err := os.ReadFile(filepath.Join(root, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(tree, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmdOutput(t, tree, env, "git", "init", "-q", "-b", "main")
	cmdOutput(t, tree, env, "git", "add", "-A")
	cmdOutput(t, tree, env, "git", "commit", "-q", "-m", "teste")
	cmdOutput(t, tmp, env, "git", "clone", "-q", "--bare", tree, bare)
	commit := strings.TrimSpace(cmdOutput(t, tree, env, "git", "rev-parse", "HEAD"))

	cmdOutput(t, tmp, env, "go", "install", modulePath+"/cmd/guidebook@"+commit)

	bin := filepath.Join(tmp, "bin", "guidebook")
	cfg := filepath.Join(root, "testdata", "example", "guidebook.yaml")
	cmd := exec.Command(bin, "build", "-config", cfg, "-out", filepath.Join(tmp, "out"))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("build com o binário instalado: %v\n%s", err, out.String())
	}
}

// cmdOutput roda o comando em dir, com env somado ao ambiente, e
// devolve a saída padrão; falha o teste se o comando falhar.
func cmdOutput(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, errb.String())
	}
	return out.String()
}
