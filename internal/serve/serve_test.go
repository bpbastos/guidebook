package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const wantCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; media-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// project cria um projeto pequeno e devolve o caminho do guidebook.yaml.
func project(t *testing.T, page string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"guidebook.yaml":       "version: 1\nproduct:\n  name: P\nuser:\n  dir: docs/user\n",
		"docs/user/tasks/a.md": page,
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "guidebook.yaml")
}

// edit troca o conteúdo da página e garante uma data de modificação nova.
var bump = time.Now()

func edit(t *testing.T, config, page string) {
	t.Helper()
	p := filepath.Join(filepath.Dir(config), "docs", "user", "tasks", "a.md")
	if err := os.WriteFile(p, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	bump = bump.Add(time.Second)
	os.Chtimes(p, bump, bump)
}

func newServer(t *testing.T, config string) (*Server, *bytes.Buffer) {
	t.Helper()
	var log syncBuffer
	s, err := New(config, t.TempDir(), &log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, &log.b
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/", nil)
	r.URL.Path = path // sem limpar: o servidor é quem precisa recusar
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func checkHeaders(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	want := map[string]string{
		"Content-Security-Policy": wantCSP,
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-cache",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q; esperava %q", k, got, v)
		}
	}
}

func TestServeFile(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n\nVersão 1.\n"))
	if !s.Refresh() {
		t.Fatal("o primeiro Refresh deveria fazer build")
	}
	w := get(t, s, "GET", "/user-guide.html")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Versão 1.") {
		t.Fatalf("código %d, corpo:\n%.200s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	checkHeaders(t, w)
	// Servido = gerado: os mesmos bytes do arquivo do build.
	matches, _ := filepath.Glob(filepath.Join(s.WorkDir(), "*", "user-guide.html"))
	if len(matches) != 1 {
		t.Fatalf("builds no diretório interno: %v", matches)
	}
	disk, _ := os.ReadFile(matches[0])
	if !bytes.Equal(disk, w.Body.Bytes()) {
		t.Errorf("o servidor alterou os bytes do build")
	}
	if js := get(t, s, "GET", "/assets/guidebook.js"); js.Code != 200 || !strings.Contains(js.Header().Get("Content-Type"), "javascript") {
		t.Errorf("guidebook.js: código %d, tipo %q", js.Code, js.Header().Get("Content-Type"))
	}
}

func TestRootRedirect(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n"))
	s.Refresh()
	w := get(t, s, "GET", "/")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/user-guide.html" {
		t.Errorf("código %d, Location %q", w.Code, w.Header().Get("Location"))
	}
}

func TestMethodNotAllowed(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n"))
	s.Refresh()
	for _, m := range []string{"POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		w := get(t, s, m, "/user-guide.html")
		if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("%s: código %d, Allow %q", m, w.Code, w.Header().Get("Allow"))
		}
		checkHeaders(t, w)
	}
}

func TestHead(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n"))
	s.Refresh()
	w := get(t, s, "HEAD", "/user-guide.html")
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
		t.Errorf("código %d, corpo %d bytes, Content-Length %q", w.Code, w.Body.Len(), w.Header().Get("Content-Length"))
	}
}

func TestNotFound(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n"))
	s.Refresh()
	for _, p := range []string{
		"/nada.html", "/assets", "/assets/", "/media/", "/.guidebook-output",
		"/../../../../etc/passwd", "/assets/../../../../etc/passwd", "/%2e%2e/etc/passwd",
	} {
		w := get(t, s, "GET", p)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: código %d; esperava 404", p, w.Code)
		}
		if strings.Contains(w.Body.String(), "guidebook.js") || strings.Contains(w.Body.String(), "root:") {
			t.Errorf("%s: o corpo vazou conteúdo: %q", p, w.Body.String())
		}
		checkHeaders(t, w)
	}
}

func TestSymlinkEscape(t *testing.T) {
	s, _ := newServer(t, project(t, "# A\n"))
	s.Refresh()
	secret := filepath.Join(t.TempDir(), "segredo.txt")
	os.WriteFile(secret, []byte("segredo"), 0o644)
	matches, _ := filepath.Glob(filepath.Join(s.WorkDir(), "*", "assets"))
	if len(matches) != 1 {
		t.Fatalf("build não encontrado no diretório interno: %v", matches)
	}
	if err := os.Symlink(secret, filepath.Join(matches[0], "link.txt")); err != nil {
		t.Skip("sem link simbólico:", err)
	}
	if w := get(t, s, "GET", "/assets/link.txt"); w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "segredo") {
		t.Errorf("link para fora: código %d, corpo %q", w.Code, w.Body.String())
	}
}

type health struct {
	Status   string   `json:"status"`
	Problems []string `json:"problems"`
}

func healthz(t *testing.T, s *Server) (int, health) {
	t.Helper()
	w := get(t, s, "GET", "/healthz")
	checkHeaders(t, w)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type do /healthz = %q", ct)
	}
	var h health
	if err := json.Unmarshal(w.Body.Bytes(), &h); err != nil {
		t.Fatalf("/healthz não é JSON: %q", w.Body.String())
	}
	return w.Code, h
}

func TestHealthAndLastGoodVersion(t *testing.T) {
	config := project(t, "# A\n\n<b>HTML cru</b>\n")
	s, log := newServer(t, config)

	if code, h := healthz(t, s); code != 503 || h.Status != "unavailable" {
		t.Errorf("antes do build: %d %+v", code, h)
	}
	s.Refresh()
	code, h := healthz(t, s)
	if code != 503 || h.Status != "unavailable" || len(h.Problems) != 1 || !strings.Contains(h.Problems[0], "a.md:3: HTML cru") {
		t.Errorf("nunca houve build bom: %d %+v", code, h)
	}
	if w := get(t, s, "GET", "/user-guide.html"); w.Code != 503 {
		t.Errorf("sem build bom, arquivos dão 503; veio %d", w.Code)
	}
	if !strings.Contains(log.String(), "a.md:3: HTML cru") {
		t.Errorf("o log deveria trazer os problemas:\n%s", log.String())
	}

	edit(t, config, "# A\n\nVersão boa.\n")
	s.Refresh()
	if code, h := healthz(t, s); code != 200 || h.Status != "ok" || len(h.Problems) != 0 {
		t.Errorf("depois do build bom: %d %+v", code, h)
	}

	edit(t, config, "# A\n\n<i>quebrou</i>\n")
	s.Refresh()
	if code, h := healthz(t, s); code != 200 || h.Status != "stale" || len(h.Problems) != 1 {
		t.Errorf("build falho depois de um bom: %d %+v", code, h)
	}
	if w := get(t, s, "GET", "/user-guide.html"); w.Code != 200 || !strings.Contains(w.Body.String(), "Versão boa.") {
		t.Errorf("deveria continuar servindo a última versão boa: %d", w.Code)
	}
}

func TestRefreshOnlyOnChange(t *testing.T) {
	config := project(t, "# A\n\nUm.\n")
	s, _ := newServer(t, config)
	s.Refresh()
	if s.Refresh() {
		t.Errorf("sem mudança, não deveria fazer build")
	}
	edit(t, config, "# A\n\nDois.\n")
	if !s.Refresh() {
		t.Fatalf("a mudança na página deveria gerar build")
	}
	if w := get(t, s, "GET", "/user-guide.html"); !strings.Contains(w.Body.String(), "Dois.") {
		t.Errorf("o build novo não foi servido")
	}
	// Arquivo novo numa pasta configurada também conta.
	os.WriteFile(filepath.Join(filepath.Dir(config), "docs", "user", "tasks", "b.md"), []byte("# B\n"), 0o644)
	if !s.Refresh() {
		t.Errorf("arquivo novo deveria gerar build")
	}
	// O diretório do build anterior é apagado.
	time.Sleep(50 * time.Millisecond)
	if matches, _ := filepath.Glob(filepath.Join(s.WorkDir(), "*", "user-guide.html")); len(matches) != 1 {
		t.Errorf("builds que sobraram: %v", matches)
	}
}

// TestAtomicSwap: durante as trocas, cada resposta é um build inteiro.
func TestAtomicSwap(t *testing.T) {
	config := project(t, "# A\n\nVersão 0.\n")
	s, _ := newServer(t, config)
	s.Refresh()
	srv := httptest.NewServer(s)
	defer srv.Close()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var bad []string
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				resp, err := http.Get(srv.URL + "/user-guide.html")
				if err != nil {
					mu.Lock()
					bad = append(bad, err.Error())
					mu.Unlock()
					continue
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode != 200 || !strings.HasSuffix(string(body), "</html>\n") || strings.Count(string(body), "Versão ") != 1 {
					mu.Lock()
					bad = append(bad, resp.Status)
					mu.Unlock()
				}
			}
		}()
	}
	for v := 1; v <= 15; v++ {
		edit(t, config, "# A\n\nVersão "+string(rune('a'+v))+".\n")
		if !s.Refresh() {
			t.Fatalf("versão %d não gerou build", v)
		}
	}
	close(stop)
	wg.Wait()
	if len(bad) > 0 {
		t.Errorf("%d respostas quebradas durante as trocas, a primeira: %s", len(bad), bad[0])
	}
}

func TestRunAndShutdown(t *testing.T) {
	s, err := New(project(t, "# A\n"), t.TempDir(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx, ln, time.Second) }()

	client := &http.Client{Timeout: 2 * time.Second}
	var resp *http.Response
	for i := 0; i < 50; i++ {
		if resp, err = client.Get("http://" + ln.Addr().String() + "/healthz"); err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil || resp.StatusCode != 200 {
		cancel()
		t.Fatalf("servidor não respondeu: %v", err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v", err)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("Run não terminou depois do cancelamento")
	}
	if _, err := os.Stat(s.WorkDir()); !os.IsNotExist(err) {
		t.Errorf("o diretório interno deveria ter sido apagado: %v", err)
	}
}

// withDeveloper acrescenta ao projeto um guia do desenvolvedor com uma
// fonte que existe (docs/dev/a.md) e devolve o caminho dela.
func withDeveloper(t *testing.T, config string) string {
	t.Helper()
	dir := filepath.Dir(config)
	src := filepath.Join(dir, "docs", "dev", "a.md")
	os.MkdirAll(filepath.Dir(src), 0o755)
	if err := os.WriteFile(src, []byte("# Dev\n\nSegredo interno.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(config, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString("developer:\n  sections:\n    overview:\n      sources: [docs/dev/*.md]\n")
	return src
}

// TestDeveloperGuideNeverServed: o serve gera só o guia do usuário e,
// mesmo com os arquivos do desenvolvedor no diretório, responde 404.
func TestDeveloperGuideNeverServed(t *testing.T) {
	config := project(t, "# A\n")
	withDeveloper(t, config)
	s, _ := newServer(t, config)
	s.Refresh()
	if code, h := healthz(t, s); code != 200 || h.Status != "ok" {
		t.Fatalf("/healthz: %d %+v", code, h)
	}
	dirs, _ := filepath.Glob(filepath.Join(s.WorkDir(), "build-*"))
	if len(dirs) != 1 {
		t.Fatalf("builds no diretório interno: %v", dirs)
	}
	for _, name := range []string{"developer-guide.html", "assets/developer-data.js"} {
		if _, err := os.Stat(filepath.Join(dirs[0], filepath.FromSlash(name))); err == nil {
			t.Errorf("o serve não deveria gerar %s", name)
		}
	}
	// Mesmo plantados no build, os arquivos do desenvolvedor dão 404.
	planted := map[string]string{
		"developer-guide.html":     "segredo",
		"Developer-Guide.HTML":     "segredo",
		"assets/developer-data.js": "segredo",
		"assets/DEVELOPER-DATA.JS": "segredo",
	}
	for name, body := range planted {
		os.WriteFile(filepath.Join(dirs[0], filepath.FromSlash(name)), []byte(body), 0o644)
	}
	for _, p := range []string{
		"/developer-guide.html", "/Developer-Guide.HTML", "/DEVELOPER-GUIDE.html",
		"//developer-guide.html", "/./developer-guide.html", "/assets/../developer-guide.html",
		"/developer-guide.html/", "/assets/developer-data.js", "/assets/DEVELOPER-DATA.JS",
		"/assets//developer-data.js", "/media/../assets/developer-data.js",
	} {
		for _, m := range []string{"GET", "HEAD"} {
			w := get(t, s, m, p)
			if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "segredo") {
				t.Errorf("%s %s: código %d, corpo %q; esperava 404", m, p, w.Code, w.Body.String())
			}
			checkHeaders(t, w)
		}
	}
	// Pelo servidor de verdade, com o caminho codificado.
	srv := httptest.NewServer(s)
	defer srv.Close()
	for _, p := range []string{"/%64eveloper-guide.html", "/assets/developer%2Ddata.js", "/assets%2Fdeveloper-data.js"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound || strings.Contains(string(body), "segredo") {
			t.Errorf("%s: código %d, corpo %q; esperava 404", p, resp.StatusCode, body)
		}
	}
}

// TestDeveloperSourcesIgnored: o serve não lê, não valida e não observa
// as fontes do desenvolvedor.
func TestDeveloperSourcesIgnored(t *testing.T) {
	config := project(t, "# A\n")
	src := withDeveloper(t, config)
	// Fonte quebrada: HTML cru, link para arquivo que não existe e frase
	// longa demais.
	long := strings.Repeat("palavra ", 40)
	os.WriteFile(src, []byte("# Dev\n\n<b>cru</b>\n\n[x](falta.md)\n\n"+long+".\n"), 0o644)
	s, log := newServer(t, config)
	s.Refresh()
	if code, h := healthz(t, s); code != 200 || h.Status != "ok" || len(h.Problems) != 0 {
		t.Errorf("as fontes do desenvolvedor não deveriam pesar no /healthz: %d %+v", code, h)
	}
	if strings.Contains(log.String(), "docs/dev") || strings.Contains(log.String(), "developer") {
		t.Errorf("o log não deveria citar fontes do desenvolvedor:\n%s", log.String())
	}
	// Mudar ou criar fonte do desenvolvedor não gera build.
	bump = bump.Add(time.Second)
	os.WriteFile(src, []byte("# Dev\n\nOutra.\n"), 0o644)
	os.Chtimes(src, bump, bump)
	os.WriteFile(filepath.Join(filepath.Dir(src), "b.md"), []byte("# B\n"), 0o644)
	if s.Refresh() {
		t.Errorf("mudança nas fontes do desenvolvedor não deveria gerar build")
	}
	// Padrão que não casa nada também não pesa.
	os.Remove(src)
	os.Remove(filepath.Join(filepath.Dir(src), "b.md"))
	edit(t, config, "# A\n\nNova.\n")
	s.Refresh()
	if code, h := healthz(t, s); code != 200 || h.Status != "ok" {
		t.Errorf("sem fontes do desenvolvedor: %d %+v", code, h)
	}
}

func TestDeveloperOnlyRefused(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "guidebook.yaml")
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "dev.md"), []byte("# Dev\n"), 0o644)
	os.WriteFile(config, []byte("version: 1\nproduct:\n  name: P\ndeveloper:\n  sections:\n    overview:\n      sources: [docs/dev.md]\n"), 0o644)
	_, err := New(config, t.TempDir(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "guidebook.yaml: user: o serve só serve o guia do usuário; defina a seção user") {
		t.Errorf("New = %v; esperava recusa por falta do guia do usuário", err)
	}
}
