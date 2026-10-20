// Package build valida a documentação e grava os guias: o caminho comum
// de "guidebook build", "guidebook check" e "guidebook serve".
package build

import (
	"fmt"
	"os"
	"path/filepath"
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
}

// Result traz os problemas, ou o erro que impediu o build, os guias
// gravados e o tamanho de cada um.
type Result struct {
	Status   Status
	Problems diag.List
	Err      error
	Wrote    []string
	Sizes    []Size
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
	if o.UserOnly {
		load, o.Guide = config.LoadUser, "user"
	}
	cfg, problems := load(o.Config)
	if len(problems) > 0 {
		return Result{Status: Usage, Problems: problems}
	}
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
	if len(problems) > 0 {
		return Result{Status: Content, Problems: problems}
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
	var r Result
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
		if _, err := os.Stat(filepath.Join(cfg.Dir, filepath.FromSlash(cfg.User.Changelog))); err == nil {
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
