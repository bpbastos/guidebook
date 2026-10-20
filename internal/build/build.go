// Package build valida a documentação e grava os guias: o caminho comum
// de "guidebook build", "guidebook check" e "guidebook serve".
package build

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/content"
	"github.com/bpbastos/guidebook/internal/diag"
	"github.com/bpbastos/guidebook/internal/media"
	"github.com/bpbastos/guidebook/internal/output"
	"github.com/bpbastos/guidebook/internal/rules"
	"github.com/bpbastos/guidebook/internal/site"
)

// Status é o resultado, nos códigos de saída da spec (seção 2).
type Status int

const (
	OK      Status = 0 // sucesso
	Content Status = 1 // regra de escrita ou erro de conteúdo
	Usage   Status = 2 // configuração inválida ou erro de leitura e gravação
)

// Options escolhe o que fazer. Sem Out, só confere: gera num diretório
// temporário, para medir os guias, e o apaga no fim.
type Options struct {
	Config   string // caminho do guidebook.yaml
	Guide    string // all, user ou developer
	Out      string
	UserOnly bool         // só o guia do usuário, sem ler a seção developer (serve)
	Media    *media.Cache // cache da otimização de imagens; sem ele, um novo por Run
	NoMedia  bool         // check -no-media: não lê mídia nem gera os guias

	// ProductVersion e Date vão para o rodapé dos guias (docs/spec.md,
	// seção 2.3); Date já conferida, em AAAA-MM-DD.
	ProductVersion, Date string
}

// Result traz os problemas, ou o erro que impediu o build, os guias
// gravados e o tamanho de cada um.
type Result struct {
	Status   Status
	Problems diag.List
	Err      error
	Wrote    []string
	Sizes    []Size
	Warnings []string // linhas "aviso: <arquivo>: ...", já em ordem
}

// Size é quantos bytes um guia usa na saída (docs/spec.md, seção 2.1).
type Size struct {
	Guide string // "user" ou "developer"
	Bytes int64
}

// mib é o tamanho de um MiB, unidade de user.max_size_mib e
// developer.max_size_mib.
const mib = 1 << 20

// Run confere a documentação e, com Out, grava os guias.
func Run(o Options) Result {
	load := config.Load
	switch {
	case o.UserOnly:
		load, o.Guide = config.LoadUser, "user"
	case o.NoMedia:
		load = config.LoadNoMedia
	}
	cfg, problems := load(o.Config)
	if len(problems) > 0 {
		return Result{Status: Usage, Problems: problems}
	}
	cfg.ProductVersion, cfg.Date = o.ProductVersion, o.Date
	wantUser := o.Guide != "developer" && cfg.User != nil
	wantDev := o.Guide != "user" && cfg.Developer != nil
	if o.Guide == "user" && cfg.User == nil {
		return Result{Status: Usage, Err: fmt.Errorf("%s: o guidebook.yaml não tem a seção user", o.Config)}
	}
	if o.Guide == "developer" && cfg.Developer == nil {
		return Result{Status: Usage, Err: fmt.Errorf("%s: o guidebook.yaml não tem a seção developer", o.Config)}
	}

	var userGuide, devGuide *content.Guide
	if wantUser {
		userGuide, problems = content.Load(cfg)
	}
	if wantDev {
		var devProblems diag.List
		devGuide, devProblems = content.LoadDeveloper(cfg)
		problems = append(problems, devProblems...)
	}
	writing, err := checkWriting(cfg, userGuide, wantUser, wantDev)
	if err != nil {
		return Result{Status: Usage, Err: err}
	}
	problems = append(problems, writing...)
	var warnings []string
	if wantUser && !o.NoMedia {
		warnings = mediaWarnings(cfg, userGuide, devGuide)
	}
	if len(problems) > 0 {
		return Result{Status: Content, Problems: problems, Warnings: warnings}
	}
	if o.NoMedia {
		return Result{Status: OK}
	}
	out := o.Out
	if out == "" {
		tmp, err := os.MkdirTemp("", "guidebook-check-")
		if err != nil {
			return Result{Status: Usage, Err: fmt.Errorf("não foi possível criar o diretório temporário: %v", err)}
		}
		defer os.RemoveAll(tmp)
		out = tmp
	} else if err := output.Prepare(out); err != nil {
		return Result{Status: Usage, Err: err}
	}
	if o.Media == nil {
		o.Media = media.NewCache()
	}
	r := Result{Warnings: warnings}
	if wantUser {
		n, err := site.WriteUser(cfg, userGuide, out, o.Media)
		if err != nil {
			return Result{Status: Usage, Err: err}
		}
		r.Sizes = append(r.Sizes, Size{"user", n})
		r.ceiling("user", "guia do usuário", n, cfg.User.MaxSizeMiB, cfg.File)
	}
	if wantDev {
		n, err := site.WriteDeveloper(cfg, devGuide, out, o.Media)
		if err != nil {
			return Result{Status: Usage, Err: err}
		}
		r.Sizes = append(r.Sizes, Size{"developer", n})
		r.ceiling("developer", "guia do desenvolvedor", n, cfg.Developer.MaxSizeMiB, cfg.File)
	}
	if o.Out != "" {
		for _, s := range r.Sizes {
			r.Wrote = append(r.Wrote, filepath.Join(o.Out, s.Guide+"-guide.html"))
		}
	}
	if len(r.Problems) > 0 {
		r.Status = Content
	}
	return r
}

// mediaWarnings lista o que fica em <user.dir>/media sem uso por nenhum
// dos guias lidos (docs/spec.md, seção 2.4). "Usado" é a mesma lista de
// mídia que o build copia, mais o manifesto de vídeos e o .markers.json
// das imagens usadas.
func mediaWarnings(cfg *config.Config, guides ...*content.Guide) []string {
	used := map[string]bool{}
	for _, g := range guides {
		if g == nil {
			continue
		}
		for _, m := range g.Media {
			used[m] = true
			used[content.MarkersFile(m)] = true
		}
	}
	if cfg.User.Videos.Manifest != "" {
		used[cfg.User.Videos.Manifest] = true
	}
	var unused, markers []string
	root := path.Join(cfg.User.Dir, "media")
	fs.WalkDir(os.DirFS(cfg.Dir), root, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil // sem a pasta de mídia, não há o que avisar
		case p != root && strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return fs.SkipDir
			}
		case d.IsDir() || used[p]:
		case strings.HasSuffix(p, ".markers.json"):
			markers = append(markers, p)
		default:
			unused = append(unused, p)
		}
		return nil
	})
	// Ordem de bytes do caminho inteiro: a do WalkDir põe "a/x" antes de
	// "a-b".
	slices.Sort(unused)
	slices.Sort(markers)
	var lines []string
	for _, p := range unused {
		lines = append(lines, "aviso: "+filepath.Join(cfg.Dir, filepath.FromSlash(p))+": mídia sem uso")
	}
	for _, p := range markers {
		lines = append(lines, "aviso: "+filepath.Join(cfg.Dir, filepath.FromSlash(p))+": marcadores sem imagem usada")
	}
	return lines
}

// ceiling confere o teto opcional do guia (docs/spec.md, seção 2.1).
func (r *Result) ceiling(guide, name string, n int64, limit *int, file string) {
	if limit != nil && n > int64(*limit)*mib {
		r.Problems.Addf(file, 0, guide+".max_size_mib", "%s com %s MiB (limite: %d MiB)", name, MiB(n), *limit)
	}
}

// MiB escreve n bytes em MiB, com uma casa decimal e vírgula.
func MiB(n int64) string {
	return strings.Replace(fmt.Sprintf("%.1f", float64(n)/mib), ".", ",", 1)
}

// checkWriting aplica as regras de escrita de cada guia aos arquivos
// dele. Arquivo lido pelos dois guias com a mesma violação aparece uma
// vez só.
func checkWriting(cfg *config.Config, guide *content.Guide, user, dev bool) (diag.List, error) {
	var problems diag.List
	seen := map[string]bool{}
	apply := func(r config.Rules, files []string) error {
		checker := rules.New(r)
		for _, rel := range files {
			path := filepath.Join(cfg.Dir, filepath.FromSlash(rel))
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("não foi possível ler %s: %v", path, err)
			}
			for _, p := range checker.Check(path, source) {
				if key := p.String(); !seen[key] {
					seen[key] = true
					problems = append(problems, p)
				}
			}
		}
		return nil
	}
	if user {
		var files []string
		for _, p := range guide.Pages {
			files = append(files, p.File)
		}
		if _, err := os.Stat(filepath.Join(cfg.Dir, filepath.FromSlash(cfg.User.Changelog))); err == nil && cfg.User.ShowWhatsNew() {
			files = append(files, cfg.User.Changelog)
		}
		if err := apply(cfg.UserRules(), files); err != nil {
			return nil, err
		}
	}
	if dev {
		var files []string
		for _, s := range cfg.Developer.Sections.SectionList() {
			files = append(files, s.Files...)
		}
		if err := apply(cfg.DeveloperRules(), files); err != nil {
			return nil, err
		}
	}
	return problems, nil
}
