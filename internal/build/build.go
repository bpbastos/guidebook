// Package build valida a documentação e grava os guias: o caminho comum
// de "guidebook build", "guidebook check" e "guidebook serve".
package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/content"
	"github.com/bpbastos/guidebook/internal/diag"
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

// Options escolhe o que fazer. Sem Out, só confere.
type Options struct {
	Config   string // caminho do guidebook.yaml
	Guide    string // all, user ou developer
	Out      string
	UserOnly bool // só o guia do usuário, sem ler a seção developer (serve)
}

// Result traz os problemas, ou o erro que impediu o build, e os guias
// gravados.
type Result struct {
	Status   Status
	Problems diag.List
	Err      error
	Wrote    []string
}

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
	if o.Out == "" {
		return Result{Status: OK}
	}

	if err := output.Prepare(o.Out); err != nil {
		return Result{Status: Usage, Err: err}
	}
	var r Result
	if wantUser {
		if err := site.WriteUser(cfg, userGuide, o.Out); err != nil {
			return Result{Status: Usage, Err: err}
		}
		r.Wrote = append(r.Wrote, filepath.Join(o.Out, "user-guide.html"))
	}
	if wantDev {
		if err := site.WriteDeveloper(cfg, devGuide, o.Out); err != nil {
			return Result{Status: Usage, Err: err}
		}
		r.Wrote = append(r.Wrote, filepath.Join(o.Out, "developer-guide.html"))
	}
	return r
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
