// Package serve gera e serve o guia do usuário por HTTP, regerando
// quando a entrada muda (docs/spec.md, seção 14). O guia do
// desenvolvedor nunca é servido: nem gerado, nem lido, nem entregue.
package serve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bpbastos/guidebook/internal/build"
	"github.com/bpbastos/guidebook/internal/config"
)

// csp é a política de conteúdo de toda resposta (spec, seção 14.3; ADR
// 0011): nada de script em linha nem eval; estilo em linha só porque o
// Mermaid escreve <style> dentro do SVG.
const csp = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
	"media-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// shutdownGrace é quanto o servidor espera pelas requisições em curso.
const shutdownGrace = 10 * time.Second

func init() {
	// A imagem mínima não tem /etc/mime.types; sem isto, o tipo do vídeo
	// sairia por adivinhação.
	mime.AddExtensionType(".webm", "video/webm")
	mime.AddExtensionType(".mp4", "video/mp4")
}

// site é um build pronto. Quem lê segura o RLock; a remoção espera o
// Lock, então nenhuma requisição vê o diretório sumir no meio.
type site struct {
	dir    string
	root   *os.Root
	mu     sync.RWMutex
	closed bool
}

type healthState struct {
	code     int
	Status   string   `json:"status"`
	Problems []string `json:"problems,omitempty"`
}

// Server mantém o build atual e o estado de saúde.
type Server struct {
	opts    build.Options
	work    string
	log     io.Writer
	current atomic.Pointer[site]
	health  atomic.Pointer[healthState]

	refresh     sync.Mutex // um build por vez
	fingerprint string
	builds      int
	retiring    sync.WaitGroup
}

// New prepara o servidor; o diretório interno fica dentro de workBase.
// Recusa o projeto cujo guidebook.yaml válido não tem a seção user.
func New(configPath, workBase string, log io.Writer) (*Server, error) {
	if _, problems := config.LoadUser(configPath); len(problems) == 1 && problems[0].Msg == config.NoUserGuide {
		return nil, errors.New(problems[0].String())
	}
	work, err := os.MkdirTemp(workBase, "guidebook-serve-")
	if err != nil {
		return nil, fmt.Errorf("não foi possível criar o diretório interno: %v", err)
	}
	s := &Server{opts: build.Options{Config: configPath, UserOnly: true}, work: work, log: log}
	s.health.Store(&healthState{code: http.StatusServiceUnavailable, Status: "unavailable"})
	return s, nil
}

// WorkDir devolve o diretório interno.
func (s *Server) WorkDir() string { return s.work }

func (s *Server) logf(format string, args ...any) {
	fmt.Fprintf(s.log, "serve: "+format+"\n", args...)
}

// Refresh confere as entradas e, se algo mudou desde o último build (ou
// se nunca houve build), gera de novo. Diz se fez build.
func (s *Server) Refresh() bool {
	s.refresh.Lock()
	defer s.refresh.Unlock()
	fp := fingerprint(inputs(s.opts.Config))
	if s.builds > 0 && fp == s.fingerprint {
		return false
	}
	s.fingerprint = fp
	s.builds++
	dir := filepath.Join(s.work, fmt.Sprintf("build-%d", s.builds))
	s.logf("build %d iniciado", s.builds)

	o := s.opts
	o.Out = dir
	r := build.Run(o)
	if len(r.Problems) > 0 || r.Err != nil {
		var problems []string
		for _, p := range r.Problems.Sorted() {
			problems = append(problems, p.String())
		}
		if r.Err != nil {
			problems = append(problems, r.Err.Error())
		}
		s.logf("build %d falhou com %d problemas; segue a última versão boa", s.builds, len(problems))
		for _, p := range problems {
			s.logf("  %s", p)
		}
		os.RemoveAll(dir)
		state := &healthState{code: http.StatusServiceUnavailable, Status: "unavailable", Problems: problems}
		if s.current.Load() != nil {
			state.code, state.Status = http.StatusOK, "stale"
		}
		s.health.Store(state)
		return true
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		s.logf("build %d: não foi possível abrir a saída: %v", s.builds, err)
		return true
	}
	next := &site{dir: dir, root: root}
	if old := s.current.Swap(next); old != nil {
		s.retiring.Add(1)
		go func() {
			defer s.retiring.Done()
			retire(old)
		}()
	}
	s.health.Store(&healthState{code: http.StatusOK, Status: "ok"})
	s.logf("build %d concluído; troca feita", s.builds)
	return true
}

// retire espera as leituras em curso e apaga o build antigo.
func retire(old *site) {
	old.mu.Lock()
	defer old.mu.Unlock()
	old.closed = true
	old.root.Close()
	os.RemoveAll(old.dir)
}

// acquire devolve o build atual, já com a leitura garantida; o chamador
// libera com RUnlock.
func (s *Server) acquire() *site {
	for {
		cur := s.current.Load()
		if cur == nil {
			return nil
		}
		cur.mu.RLock()
		if !cur.closed {
			return cur
		}
		cur.mu.RUnlock() // trocado e apagado entre o Load e o RLock
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", csp)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-cache")

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.Set("Allow", "GET, HEAD")
		http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/healthz" {
		state := s.health.Load()
		h.Set("Content-Type", "application/json")
		w.WriteHeader(state.code)
		json.NewEncoder(w).Encode(state)
		return
	}

	cur := s.acquire()
	if cur == nil {
		http.Error(w, "guia ainda não gerado; veja /healthz", http.StatusServiceUnavailable)
		return
	}
	defer cur.mu.RUnlock()
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/user-guide.html", http.StatusFound)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" || hidden(name) || developerFile(name) {
		http.NotFound(w, r)
		return
	}
	// os.Root recusa sair do diretório, por ".." ou por link simbólico.
	f, err := cur.root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// hidden recusa qualquer parte do caminho que comece com ".", como o
// marcador .guidebook-output.
func hidden(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// developerFile recusa os arquivos do guia do desenvolvedor, em
// qualquer pasta e com qualquer caixa, mesmo que estejam no diretório.
func developerFile(name string) bool {
	base := strings.ToLower(path.Base(name))
	return base == "developer-guide.html" || base == "developer-data.js"
}

// Run faz o build inicial, serve em ln e regera a cada interval, até
// ctx acabar. Então para de aceitar conexões, espera as requisições em
// curso e apaga o diretório interno.
func (s *Server) Run(ctx context.Context, ln net.Listener, interval time.Duration) error {
	s.Refresh()
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      10 * time.Minute, // vídeos grandes
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	s.logf("ouvindo em http://%s", ln.Addr())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Refresh()
		case err := <-errs:
			s.Close()
			return fmt.Errorf("o servidor parou: %v", err)
		case <-ctx.Done():
			s.logf("desligando")
			shutdown, cancel := context.WithTimeout(context.Background(), shutdownGrace)
			err := srv.Shutdown(shutdown)
			cancel()
			if cerr := s.Close(); err == nil {
				err = cerr
			}
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			return err
		}
	}
}

// Close fecha o build atual e apaga o diretório interno.
func (s *Server) Close() error {
	s.retiring.Wait()
	if cur := s.current.Swap(nil); cur != nil {
		retire(cur)
	}
	return os.RemoveAll(s.work)
}

// inputs devolve os arquivos e diretórios que o build do guia do usuário
// lê (spec, seção 14.1). Com o guidebook.yaml inválido, só ele:
// corrigido, o build volta.
func inputs(configPath string) []string {
	list := []string{configPath}
	cfg, problems := config.LoadUser(configPath)
	if len(problems) > 0 {
		return list
	}
	at := func(rel string) string { return filepath.Join(cfg.Dir, filepath.FromSlash(rel)) }
	if cfg.Product.Logo != "" {
		list = append(list, at(cfg.Product.Logo))
	}
	if u := cfg.User; u != nil {
		list = append(list, at(u.Dir), at(u.Changelog))
		if u.Videos.Manifest != "" {
			list = append(list, filepath.Dir(at(u.Videos.Manifest)))
		}
	}
	return list
}

// fingerprint resume caminho, tamanho, data de modificação e tipo de
// tudo o que está nas entradas, inclusive dentro dos diretórios.
func fingerprint(paths []string) string {
	var lines []string
	add := func(p string, info fs.FileInfo) {
		lines = append(lines, fmt.Sprintf("%s|%d|%d|%s", p, info.Size(), info.ModTime().UnixNano(), info.Mode()))
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			lines = append(lines, p+"|ausente")
			continue
		}
		if !info.IsDir() {
			add(p, info)
			continue
		}
		filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
			if err != nil {
				lines = append(lines, q+"|erro")
				return nil
			}
			if info, err := d.Info(); err == nil {
				add(q, info)
			}
			return nil
		})
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}
