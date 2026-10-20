package main

import (
	"bytes"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var example = filepath.Join("..", "..", "testdata", "example", "guidebook.yaml")

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "user", "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "docs", "user", "tasks", "a.md"), []byte("# A\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "dev.md"), []byte("# Dev\n"), 0o644)
	p := filepath.Join(dir, "guidebook.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const brokenRef = `version: 1
product:
  name: P
user:
  dir: docs/user
  profiles:
    - id: a
      name: A
      start_here: [tasks/falta]
developer:
  sections:
    overview:
      sources: [docs/dev.md]
`

func TestExitCodes(t *testing.T) {
	foreign := t.TempDir()
	os.WriteFile(filepath.Join(foreign, "notes.txt"), []byte("meu"), 0o644)
	broken := writeConfig(t, brokenRef)
	invalid := writeConfig(t, "version: 1\nproduct:\n  name: P\n  cor: x\nuser:\n  dir: docs/user\n")
	devOnly := writeConfig(t, "version: 1\nproduct:\n  name: P\ndeveloper:\n  sections:\n    overview:\n      sources: [docs/dev.md]\n")

	tests := []struct {
		name   string
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{"sem argumentos", nil, 2, "", "uso:"},
		{"ajuda", []string{"-h"}, 0, "uso:", ""},
		{"versão", []string{"version"}, 0, "guidebook ", ""},
		{"versão com argumento", []string{"version", "x"}, 2, "", "argumento inesperado"},
		{"subcomando desconhecido", []string{"publish"}, 2, "", `subcomando desconhecido: "publish"`},
		{"opção desconhecida", []string{"check", "-x"}, 2, "", "opção desconhecida: -x"},
		{"opção sem valor", []string{"check", "-config"}, 2, "", "a opção -config precisa de um valor"},
		{"argumento sobrando", []string{"check", "-config", example, "extra"}, 2, "", "argumento inesperado"},
		{"guia inválido", []string{"check", "-config", example, "-guide", "admin"}, 2, "", "-guide aceita all, user ou developer"},
		{"check do exemplo", []string{"check", "-config", example}, 0, "nenhum problema", ""},
		{"configuração inválida", []string{"check", "-config", invalid}, 2, "", "product.cor: chave desconhecida"},
		{"configuração ausente", []string{"check", "-config", filepath.Join(t.TempDir(), "guidebook.yaml")}, 2, "", "não existe"},
		{"referência quebrada", []string{"check", "-config", broken}, 1, "", `user.profiles[0].start_here[0]: página "tasks/falta" não existe`},
		{"só o guia do desenvolvedor", []string{"check", "-config", broken, "-guide", "developer"}, 0, "nenhum problema", ""},
		{"guia ausente na configuração", []string{"check", "-config", devOnly, "-guide", "user"}, 2, "", "não tem a seção user"},
		{"build em diretório alheio", []string{"build", "-config", example, "-out", foreign}, 2, "", ".guidebook-output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(tt.args...)
			if code != tt.code {
				t.Errorf("código = %d; esperava %d\nstdout: %s\nstderr: %s", code, tt.code, stdout, stderr)
			}
			if !strings.Contains(stdout, tt.stdout) {
				t.Errorf("stdout = %q; esperava conter %q", stdout, tt.stdout)
			}
			if !strings.Contains(stderr, tt.stderr) {
				t.Errorf("stderr = %q; esperava conter %q", stderr, tt.stderr)
			}
		})
	}
}

func TestBuildExample(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	code, stdout, stderr := runCLI("build", "-config", example, "-out", out)
	if code != 0 {
		t.Fatalf("código = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, f := range []string{".guidebook-output", "user-guide.html", "assets/guidebook.js", "assets/user-data.js", "media/docs/assets/logo.svg"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s ausente: %v", f, err)
		}
	}
	for _, f := range []string{"developer-guide.html", "assets/developer-data.js", "assets/mermaid.min.js"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s ausente: %v", f, err)
		}
	}
	if !strings.Contains(stdout, "user-guide.html") || !strings.Contains(stdout, "developer-guide.html") {
		t.Errorf("stdout deveria citar os dois guias gravados: %q", stdout)
	}
	// Segundo build no mesmo diretório: aceito, porque tem o marcador.
	if code, _, stderr := runCLI("build", "-config", example, "-out", out); code != 0 {
		t.Errorf("segundo build: código = %d\n%s", code, stderr)
	}
}

func TestBuildDoesNotTouchOutputOnProblems(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	broken := writeConfig(t, brokenRef)
	if code, _, _ := runCLI("build", "-config", broken, "-out", out); code != 1 {
		t.Fatalf("código = %d; esperava 1", code)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("build com problema não deveria criar a saída (stat: %v)", err)
	}
}

func TestWritingRules(t *testing.T) {
	dir := t.TempDir()
	long := strings.TrimSpace(strings.Repeat("palavra ", 30))
	files := map[string]string{
		"guidebook.yaml": `version: 1
product:
  name: P
rules:
  forbidden_patterns:
    - pattern: 'ZZ-\d+'
      message: não cite chamado
user:
  dir: docs/user
developer:
  rules:
    max_sentence_words: 40
  sections:
    overview:
      sources: [docs/dev.md]
    changes:
      sources: [CHANGELOG.md]
`,
		"docs/user/tasks/a.md": "# A\n\n" + long + ".\n\n<div>x</div>\n",
		"docs/dev.md":          "# Dev\n\n" + long + ".\n",
		"CHANGELOG.md":         "# Changelog\n\nVeja ZZ-12.\n",
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	cfg := filepath.Join(dir, "guidebook.yaml")
	code, _, stderr := runCLI("check", "-config", cfg)
	if code != 1 {
		t.Fatalf("código = %d; esperava 1\n%s", code, stderr)
	}
	user := filepath.Join(dir, "docs", "user", "tasks", "a.md")
	changelog := filepath.Join(dir, "CHANGELOG.md")
	want := changelog + `:3: não cite chamado: "ZZ-12"` + "\n" +
		user + `:3: frase com 30 palavras (limite: 25): "palavra palavra palavra palavra palavra palavra palavra palavra…"` + "\n" +
		user + ":5: HTML cru não é permitido no Markdown\n"
	// O guia do desenvolvedor aceita 40 palavras: docs/dev.md passa. O
	// CHANGELOG entra nos dois guias, mas a violação sai uma vez só.
	if stderr != want {
		t.Errorf("stderr =\n%s\nesperava\n%s", stderr, want)
	}

	if code, _, stderr := runCLI("check", "-config", cfg, "-guide", "developer"); code != 1 || strings.Contains(stderr, "a.md") {
		t.Errorf("-guide developer: código %d, stderr:\n%s", code, stderr)
	}
}

func TestCheckRejectsDangerousLink(t *testing.T) {
	p := writeConfig(t, "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n")
	os.WriteFile(filepath.Join(filepath.Dir(p), "docs", "user", "tasks", "a.md"), []byte("# A\n\n[x](javascript:alert(1))\n"), 0o644)
	code, _, stderr := runCLI("check", "-config", p)
	if code != 1 || !strings.Contains(stderr, `a.md:3: esquema "javascript:" não é permitido`) {
		t.Errorf("código %d, stderr:\n%s", code, stderr)
	}
}

func TestCheckDeveloperLinks(t *testing.T) {
	p := writeConfig(t, "version: 1\nproduct:\n  name: P\ndeveloper:\n  sections:\n    overview:\n      sources: [docs/dev.md]\n")
	os.WriteFile(filepath.Join(filepath.Dir(p), "docs", "dev.md"), []byte("# Dev\n\n[x](falta.md)\n"), 0o644)
	code, _, stderr := runCLI("check", "-config", p)
	if code != 1 || !strings.Contains(stderr, `dev.md:3: link para "docs/falta.md", que não existe`) {
		t.Errorf("código %d, stderr:\n%s", code, stderr)
	}
}

// developerOnly cria um projeto só com o guia do desenvolvedor.
func developerOnly(t *testing.T) string {
	t.Helper()
	return writeConfig(t, "version: 1\nproduct:\n  name: P\ndeveloper:\n  sections:\n    overview:\n      sources: [docs/dev.md]\n")
}

func TestServeUsage(t *testing.T) {
	tests := []struct {
		args []string
		msg  string
	}{
		{[]string{"serve", "-interval", "100ms"}, "-interval precisa ser de pelo menos 500ms"},
		{[]string{"serve", "-interval", "rápido"}, "opção inválida"},
		{[]string{"serve", "extra"}, "argumento inesperado"},
		{[]string{"serve", "-guide", "user"}, "o serve não tem -guide: ele serve só o guia do usuário"},
		{[]string{"serve", "-guide=developer"}, "o serve não tem -guide: ele serve só o guia do usuário"},
		{[]string{"serve", "-config", developerOnly(t)}, "o serve só serve o guia do usuário"},
		{[]string{"serve", "-addr", "999.0.0.1:x", "-config", example}, "não foi possível escutar em 999.0.0.1:x"},
	}
	for _, tt := range tests {
		code, _, stderr := runCLI(tt.args...)
		if code != 2 || !strings.Contains(stderr, tt.msg) {
			t.Errorf("%v: código %d, stderr:\n%s", tt.args, code, stderr)
		}
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// TestServeSIGTERM sobe o serve de verdade e manda SIGTERM ao próprio
// processo: o serve desliga limpo e sai com 0.
func TestServeSIGTERM(t *testing.T) {
	var stdout, stderr lockedBuffer
	done := make(chan int, 1)
	go func() {
		done <- run([]string{"serve", "-config", example, "-addr", "127.0.0.1:0", "-interval", "500ms"}, &stdout, &stderr)
	}()

	addr := regexp.MustCompile(`ouvindo em (http://\S+)`)
	var base string
	for i := 0; i < 100 && base == ""; i++ {
		if m := addr.FindStringSubmatch(stderr.String()); m != nil {
			base = m[1]
		}
		time.Sleep(50 * time.Millisecond)
	}
	if base == "" {
		t.Fatalf("serve não subiu:\n%s", stderr.String())
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("/healthz: %v %v\n%s", resp, err, stderr.String())
	}
	resp.Body.Close()
	resp, err = client.Get(base + "/user-guide.html")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("guia do usuário: %v %v", resp, err)
	}
	resp.Body.Close()
	resp, err = client.Get(base + "/developer-guide.html")
	if err != nil || resp.StatusCode != 404 {
		t.Fatalf("o guia do desenvolvedor nunca é servido: %v %v", resp, err)
	}
	resp.Body.Close()

	syscall.Kill(os.Getpid(), syscall.SIGTERM)
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("código %d depois do SIGTERM\n%s", code, stderr.String())
		}
	case <-time.After(12 * time.Second):
		t.Fatal("serve não terminou depois do SIGTERM")
	}
	if !strings.Contains(stderr.String(), "desligando") {
		t.Errorf("o log deveria registrar o desligamento:\n%s", stderr.String())
	}
}

func TestVersionString(t *testing.T) {
	info := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	tests := []struct {
		ldflags string
		info    *debug.BuildInfo
		want    string
	}{
		{"v0.1.0", info("v0.2.0"), "v0.1.0"}, // -ldflags manda
		{"dev", info("v0.1.0"), "v0.1.0"},    // go install @v0.1.0
		{"dev", info("(devel)"), "dev"},      // go build no clone
		{"dev", nil, "dev"},                  // sem informação de build
	}
	for _, tt := range tests {
		if got := versionString(tt.ldflags, tt.info); got != tt.want {
			t.Errorf("versionString(%q, %v) = %q; esperava %q", tt.ldflags, tt.info, got, tt.want)
		}
	}
}

// sizeLine acha a linha de tamanho de um guia e devolve os bytes.
var sizeLine = regexp.MustCompile(`(?m)^(build|check): guia (do usuário|do desenvolvedor) com ([\d.]+) bytes \((\d+,\d) MiB\)$`)

func sizes(t *testing.T, stdout string) map[string]int64 {
	t.Helper()
	got := map[string]int64{}
	for _, m := range sizeLine.FindAllStringSubmatch(stdout, -1) {
		n, err := strconv.ParseInt(strings.ReplaceAll(m[3], ".", ""), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		got[m[2]] = n
	}
	return got
}

// TestReportedSize: check e build informam o mesmo tamanho, que é a
// soma dos arquivos que o build grava para o guia (docs/spec.md, seção
// 2.1).
func TestReportedSize(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out")
	code, built, stderr := runCLI("build", "-config", example, "-out", out, "-guide", "user")
	if code != 0 {
		t.Fatalf("build: código %d\n%s", code, stderr)
	}
	code, checked, stderr := runCLI("check", "-config", example, "-guide", "user")
	if code != 0 {
		t.Fatalf("check: código %d\n%s", code, stderr)
	}
	b, c := sizes(t, built)["do usuário"], sizes(t, checked)["do usuário"]
	if b == 0 || b != c {
		t.Fatalf("build informou %d e check %d\nbuild: %s\ncheck: %s", b, c, built, checked)
	}
	var sum int64
	filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() != ".guidebook-output" {
			info, _ := d.Info()
			sum += info.Size()
		}
		return nil
	})
	if b != sum {
		t.Errorf("informou %d bytes; o out/ tem %d", b, sum)
	}
	if !strings.HasSuffix(strings.TrimSpace(checked), "check: nenhum problema encontrado") {
		t.Errorf("a linha final do check deveria vir por último: %q", checked)
	}

	// Com os dois guias, uma linha para cada.
	_, both, _ := runCLI("check", "-config", example)
	if s := sizes(t, both); len(s) != 2 || s["do usuário"] != b {
		t.Errorf("check com os dois guias: %v\n%s", s, both)
	}
}

func TestFormatSize(t *testing.T) {
	for n, want := range map[int64]string{
		0:        "0 bytes (0,0 MiB)",
		999:      "999 bytes (0,0 MiB)",
		1048576:  "1.048.576 bytes (1,0 MiB)",
		9876543:  "9.876.543 bytes (9,4 MiB)",
		12345678: "12.345.678 bytes (11,8 MiB)",
	} {
		if got := formatSize(n); got != want {
			t.Errorf("formatSize(%d) = %q; esperava %q", n, got, want)
		}
	}
}

// TestSizeCeiling: guia acima de max_size_mib é erro de conteúdo, com o
// tamanho e o limite; as linhas de tamanho saem mesmo assim.
func TestSizeCeiling(t *testing.T) {
	p := writeConfig(t, "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n  max_size_mib: 1\n")
	dir := filepath.Dir(p)
	big := make([]byte, 1100*1024)
	for i := range big {
		big[i] = byte(i * 7919 >> 3)
	}
	os.MkdirAll(filepath.Join(dir, "docs", "user", "media"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "user", "media", "foto.jpg"), big, 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "user", "tasks", "a.md"), []byte("# A\n\n![Foto](../media/foto.jpg)\n"), 0o644)

	for _, cmd := range [][]string{{"check"}, {"build", "-out", filepath.Join(t.TempDir(), "out")}} {
		code, stdout, stderr := runCLI(append(cmd, "-config", p)...)
		if code != 1 {
			t.Errorf("%s: código %d; esperava 1\n%s", cmd[0], code, stderr)
		}
		if !regexp.MustCompile(`guidebook\.yaml: user\.max_size_mib: guia do usuário com 1,\d MiB \(limite: 1 MiB\)`).MatchString(stderr) {
			t.Errorf("%s: stderr sem o teto:\n%s", cmd[0], stderr)
		}
		if len(sizes(t, stdout)) != 1 {
			t.Errorf("%s: stdout sem a linha de tamanho:\n%s", cmd[0], stdout)
		}
	}
}

// project grava os arquivos (caminho relativo → conteúdo) e devolve o
// caminho do guidebook.yaml.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "guidebook.yaml")
}

const videoConfig = "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n  videos:\n    manifest: docs/user/media/videos/manifest.json\n    tasks: [tasks/a]\n"

// TestCheckNoMedia: sem as mídias do e2e, o check -no-media passa, sem
// gerar os guias; as regras de escrita e de caminho continuam
// (docs/spec.md, seção 2.2).
func TestCheckNoMedia(t *testing.T) {
	p := project(t, map[string]string{
		"guidebook.yaml":       videoConfig,
		"docs/user/tasks/a.md": "# A\n\n![Tela](../media/tela.png)\n\n1. Passo.\n",
	})
	code, stdout, stderr := runCLI("check", "-config", p, "-no-media")
	if code != 0 || stderr != "" {
		t.Fatalf("código %d; esperava 0\n%s", code, stderr)
	}
	if len(sizes(t, stdout)) != 0 || !strings.Contains(stdout, "nenhum problema") {
		t.Errorf("sem geração, sem linha de tamanho: %q", stdout)
	}
	// Sem a opção, a mídia ausente é recusada.
	if code, _, _ := runCLI("check", "-config", p); code == 0 {
		t.Error("o check comum deveria recusar a mídia ausente")
	}
	if code, _, stderr := runCLI("build", "-config", p, "-no-media"); code != 2 || !strings.Contains(stderr, "-no-media") {
		t.Errorf("build -no-media: código %d\n%s", code, stderr)
	}

	long := strings.TrimSpace(strings.Repeat("palavra ", 30)) + "."
	bad := project(t, map[string]string{
		"guidebook.yaml":       videoConfig,
		"docs/user/tasks/a.md": "# A\n\n![](../media/a.png)\n\n![Fora](../../../../fora.png)\n\n![Rede](https://exemplo.com/a.png)\n\n" + long + "\n",
	})
	code, _, stderr = runCLI("check", "-config", bad, "-no-media")
	if code != 1 {
		t.Fatalf("código %d; esperava 1\n%s", code, stderr)
	}
	for _, want := range []string{"imagem sem texto alternativo", "sai do diretório do guidebook.yaml", "imagem externa não é permitida", "frase com 30 palavras"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr sem %q:\n%s", want, stderr)
		}
	}
}

// TestMediaWarnings: mídia sem uso e marcadores sem imagem usada saem
// como aviso, sem mudar o código de saída (docs/spec.md, seção 2.4).
func TestMediaWarnings(t *testing.T) {
	var png bytes.Buffer
	png.WriteString("\x89PNG\r\n\x1a\n") // não decodifica: sai como veio
	p := project(t, map[string]string{
		"guidebook.yaml":                       videoConfig,
		"docs/user/tasks/a.md":                 "# A\n\n![Tela](../media/used.png)\n",
		"docs/user/tasks/b.md":                 "# B\n",
		"docs/user/media/used.png":             png.String(),
		"docs/user/media/old.png":              png.String(),
		"docs/user/media/old.markers.json":     "{}",
		"docs/user/media/notes.txt":            "x",
		"docs/user/media/.DS_Store":            "x",
		"docs/user/media/videos/a.webm":        "v",
		"docs/user/media/videos/b.webm":        "v",
		"docs/user/media/videos/b.png":         png.String(),
		"docs/user/media/videos/manifest.json": `{"version": 1, "videos": [{"task": "tasks/a", "file": "a.webm", "duration_seconds": 5}, {"task": "tasks/b", "file": "b.webm", "duration_seconds": 5, "poster": "b.png"}]}`,
	})
	media := filepath.Join(filepath.Dir(p), "docs", "user", "media")
	want := "aviso: " + filepath.Join(media, "notes.txt") + ": mídia sem uso\n" +
		"aviso: " + filepath.Join(media, "old.png") + ": mídia sem uso\n" +
		"aviso: " + filepath.Join(media, "videos", "b.png") + ": mídia sem uso\n" +
		"aviso: " + filepath.Join(media, "videos", "b.webm") + ": mídia sem uso\n" +
		"aviso: " + filepath.Join(media, "old.markers.json") + ": marcadores sem imagem usada\n"
	for _, args := range [][]string{{"check"}, {"build", "-out", filepath.Join(t.TempDir(), "out")}} {
		code, _, stderr := runCLI(append(args, "-config", p)...)
		if code != 0 {
			t.Errorf("%s: código %d; aviso não muda o código\n%s", args[0], code, stderr)
		}
		if stderr != want {
			t.Errorf("%s: stderr =\n%s\nesperava\n%s", args[0], stderr, want)
		}
	}
	if _, _, stderr := runCLI("check", "-config", p, "-no-media"); stderr != "" {
		t.Errorf("-no-media não lê mídia e não avisa:\n%s", stderr)
	}
}
