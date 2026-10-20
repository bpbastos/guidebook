package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type validator struct {
	c      *Config
	report func(field, format string, args ...any)
}

func (v *validator) validate() {
	c := v.c

	switch c.Version {
	case 0:
		v.report("version", "obrigatório")
	case 1:
	default:
		v.report("version", "só a versão 1 é aceita")
	}

	if strings.TrimSpace(c.Product.Name) == "" {
		v.report("product.name", "obrigatório")
	}
	if c.Product.Logo != "" {
		if p, ok := v.file("product.logo", c.Product.Logo); ok {
			if ext := strings.ToLower(path.Ext(p)); ext != ".svg" && ext != ".png" {
				v.report("product.logo", "use um arquivo .svg ou .png")
			}
		}
	}
	v.color("product.colors.primary", &c.Product.Colors.Primary, DefaultColor)
	v.color("product.colors.accent", &c.Product.Colors.Accent, c.Product.Colors.Primary)

	v.rules("rules", c.Rules)

	if c.User == nil && c.Developer == nil {
		v.report("user", "defina user, developer ou os dois; sem eles não há guia a gerar")
	}
	if c.User != nil {
		v.user(c.User)
		v.maxSize("user.max_size_mib", c.User.MaxSizeMiB)
	}
	if c.Developer != nil {
		v.developer(c.Developer)
		v.maxSize("developer.max_size_mib", c.Developer.MaxSizeMiB)
	}

	keys := make([]string, 0, len(c.UI))
	for k := range c.UI {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		field := "ui." + k
		def, known := uiDefaults[k]
		switch {
		case !known:
			v.report(field, "chave desconhecida")
		case strings.TrimSpace(c.UI[k]) == "":
			v.report(field, "não pode ficar vazio (padrão: %q)", def)
		}
		for _, ph := range uiPlaceholders[k] {
			if known && !strings.Contains(c.UI[k], ph) {
				v.report(field, "falta o marcador %s", ph)
			}
		}
	}
}

// maxSize confere o teto opcional de um guia (docs/spec.md, seção 2.1).
func (v *validator) maxSize(field string, n *int) {
	if n != nil && *n <= 0 {
		v.report(field, "deve ser um inteiro maior que 0")
	}
}

func (v *validator) color(field string, value *string, def string) {
	if *value == "" {
		*value = def
		return
	}
	if !colorRe.MatchString(*value) {
		v.report(field, "use uma cor no formato #rrggbb (entre aspas no YAML)")
	}
}

func (v *validator) rules(field string, r Rules) {
	if n := r.MaxSentenceWords; n != nil && (*n < 5 || *n > 200) {
		v.report(field+".max_sentence_words", "deve ficar entre 5 e 200")
	}
	for i, fp := range r.ForbiddenPatterns {
		f := fmt.Sprintf("%s.forbidden_patterns[%d]", field, i)
		if fp.Pattern == "" {
			v.report(f+".pattern", "obrigatório")
		} else if _, err := regexp.Compile(fp.Pattern); err != nil {
			v.report(f+".pattern", "expressão regular inválida: %s", strings.TrimPrefix(err.Error(), "error parsing regexp: "))
		}
		if strings.TrimSpace(fp.Message) == "" {
			v.report(f+".message", "obrigatório")
		}
	}
}

func (v *validator) user(u *User) {
	if u.Dir == "" {
		v.report("user.dir", "obrigatório")
	} else if p, ok := v.path("user.dir", u.Dir); ok {
		u.Dir = p
		v.exists("user.dir", p, true)
	}

	if u.Changelog == "" {
		u.Changelog = DefaultChangelog
	}
	if p, ok := v.path("user.changelog", u.Changelog); ok {
		u.Changelog = p // pode faltar: sem ele, não há "O que há de novo"
	}
	if strings.TrimSpace(u.BlockedPrefix) == "" {
		u.BlockedPrefix = DefaultBlockedPrefix
	}
	v.rules("user.rules", u.Rules)

	ids := map[string]bool{}
	for i, p := range u.Profiles {
		f := fmt.Sprintf("user.profiles[%d]", i)
		switch {
		case p.ID == "":
			v.report(f+".id", "obrigatório")
		case !idRe.MatchString(p.ID):
			v.report(f+".id", "use kebab-case: minúsculas, dígitos e hífen (ex.: atendente-caixa)")
		case ids[p.ID]:
			v.report(f+".id", "perfil %q repetido", p.ID)
		}
		ids[p.ID] = true
		if strings.TrimSpace(p.Name) == "" {
			v.report(f+".name", "obrigatório")
		}
		if len(p.StartHere) == 0 {
			v.report(f+".start_here", "liste pelo menos uma página")
		}
		v.uniqueStrings(f+".start_here", p.StartHere, "página %q repetida na trilha")
	}

	if len(u.Videos.Tasks) > 0 && u.Videos.Manifest == "" {
		v.report("user.videos.manifest", "obrigatório quando há user.videos.tasks")
	}
	if u.Videos.Manifest != "" {
		if p, ok := v.file("user.videos.manifest", u.Videos.Manifest); ok {
			u.Videos.Manifest = p
		}
	}
	v.uniqueStrings("user.videos.tasks", u.Videos.Tasks, "tarefa %q repetida")
}

func (v *validator) developer(d *Developer) {
	if d.RepositoryURL != "" {
		u, err := url.Parse(d.RepositoryURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			v.report("developer.repository_url", "use um endereço https:// completo")
		}
		d.RepositoryURL = strings.TrimRight(d.RepositoryURL, "/")
	}
	v.rules("developer.rules", d.Rules)

	owner := map[string]string{} // arquivo → chave da seção que já o usa
	for _, s := range d.Sections.SectionList() {
		f := "developer.sections." + s.Key
		if len(s.Sources) == 0 {
			v.report(f+".sources", "liste pelo menos uma fonte")
		}
		for i, pattern := range s.Sources {
			sf := fmt.Sprintf("%s.sources[%d]", f, i)
			for _, file := range v.glob(sf, pattern) {
				if prev, ok := owner[file]; ok {
					v.report(sf, "%s já está na seção %s", file, prev)
					continue
				}
				owner[file] = s.Key
				s.Files = append(s.Files, file)
			}
		}
	}
}

func (v *validator) uniqueStrings(field string, list []string, format string) {
	seen := map[string]bool{}
	for i, s := range list {
		if seen[s] {
			v.report(fmt.Sprintf("%s[%d]", field, i), format, s)
		}
		seen[s] = true
	}
}

// path confere que p é relativo e não sai do diretório do
// guidebook.yaml, e o devolve limpo, com barras normais.
func (v *validator) path(field, p string) (string, bool) {
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		v.report(field, "caminho absoluto não é aceito; use um caminho relativo ao guidebook.yaml")
		return "", false
	}
	clean := path.Clean(filepath.ToSlash(p))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		v.report(field, "%q sai do diretório do guidebook.yaml", p)
		return "", false
	}
	return clean, true
}

// file confere path e que p é um arquivo existente.
func (v *validator) file(field, p string) (string, bool) {
	clean, ok := v.path(field, p)
	if !ok {
		return "", false
	}
	return clean, v.exists(field, clean, false)
}

// exists confere que rel existe, é do tipo certo e, resolvidos os links
// simbólicos, continua dentro do diretório do guidebook.yaml.
func (v *validator) exists(field, rel string, wantDir bool) bool {
	abs := filepath.Join(v.c.Dir, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		v.report(field, "%q não existe", rel)
		return false
	case err != nil:
		v.report(field, "não foi possível ler %q: %v", rel, err)
		return false
	case wantDir && !info.IsDir():
		v.report(field, "%q não é um diretório", rel)
		return false
	case !wantDir && info.IsDir():
		v.report(field, "%q é um diretório; esperava um arquivo", rel)
		return false
	}
	if !v.inside(abs) {
		v.report(field, "%q aponta, por link simbólico, para fora do diretório do guidebook.yaml", rel)
		return false
	}
	return true
}

func (v *validator) inside(abs string) bool {
	base, err1 := filepath.EvalSymlinks(v.c.Dir)
	target, err2 := filepath.EvalSymlinks(abs)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(base, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// glob expande pattern dentro do diretório do guidebook.yaml e devolve
// os arquivos casados, relativos a ele, em ordem de bytes.
func (v *validator) glob(field, pattern string) []string {
	clean, ok := v.path(field, pattern)
	if !ok {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(v.c.Dir, filepath.FromSlash(clean)))
	if err != nil {
		v.report(field, "padrão inválido: %q", pattern)
		return nil
	}
	var files []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(v.c.Dir, m)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if path.Ext(rel) != ".md" {
			v.report(field, "%q não é Markdown; as fontes precisam ser .md", rel)
			continue
		}
		if !v.inside(m) {
			v.report(field, "%q aponta, por link simbólico, para fora do diretório do guidebook.yaml", rel)
			continue
		}
		files = append(files, rel)
	}
	if len(matches) == 0 {
		v.report(field, "nenhum arquivo casa com %q", pattern)
	}
	sort.Strings(files)
	return files
}
