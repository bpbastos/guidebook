// Package content encontra as páginas do guia do usuário (docs/spec.md,
// seção 3.1) e confere as referências do guidebook.yaml a elas.
package content

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bpbastos/guidebook/internal/config"
	"github.com/bpbastos/guidebook/internal/diag"
)

// Kind é o tipo da página.
type Kind string

const (
	KindIndex       Kind = "index"
	KindGlossary    Kind = "glossary"
	KindTutorial    Kind = "tutorials"
	KindTask        Kind = "tasks"
	KindReference   Kind = "reference"
	KindExplanation Kind = "explanation"
)

// Page é uma página do guia do usuário.
type Page struct {
	ID   string // ex.: tasks/register-order
	Kind Kind
	File string // relativo ao diretório do guidebook.yaml, com barras normais

	// Preenchidos por Load.
	Title    string
	Headings []Heading
	Blocked  []Heading // títulos "O sistema não me deixa..."
	Sections []Section // texto para a busca
	Video    *Video    // só em tarefas de user.videos.tasks
	HTML     string
}

// Video é o vídeo de uma tarefa (docs/spec.md, seção 7).
type Video struct {
	Src     string // já reescrito para media/
	Poster  string // vazio se não houver
	Minutes int    // duração arredondada para cima, mínimo 1
}

// Section é um trecho da página para a busca: a introdução, antes do
// primeiro "## " (Anchor vazio), ou uma seção "## ".
type Section struct {
	Anchor string
	Title  string
	Text   string
}

// typeDirs são os diretórios Diátaxis, na ordem de exibição.
var typeDirs = []Kind{KindTutorial, KindTask, KindReference, KindExplanation}

var orderPrefix = regexp.MustCompile(`^[0-9]+-`)

// Scan lista as páginas de user.dir na ordem de exibição: index,
// glossary e os diretórios Diátaxis, cada um em ordem de nome de
// arquivo. Arquivos e diretórios que começam com "." são ignorados.
func Scan(c *config.Config) ([]Page, diag.List) {
	var pages []Page
	var problems diag.List
	if c.User == nil {
		return nil, nil
	}
	root := c.User.Dir
	display := func(rel string) string { return filepath.Join(c.Dir, filepath.FromSlash(rel)) }

	entries, err := readDir(filepath.Join(c.Dir, filepath.FromSlash(root)))
	if err != nil {
		problems.Addf(display(root), 0, "", "não foi possível ler o diretório: %v", err)
		return nil, problems
	}
	allowed := map[string]bool{"media": true}
	for _, k := range typeDirs {
		allowed[string(k)] = true
	}
	present := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		present[name] = true
		switch {
		case e.IsDir() && allowed[name]:
		case !e.IsDir() && (name == "index.md" || name == "glossary.md"):
		default:
			problems.Addf(display(path.Join(root, name)), 0, "",
				"não é esperado aqui; a raiz do guia aceita index.md, glossary.md, media/ e os diretórios tutorials/, tasks/, reference/ e explanation/")
		}
	}
	if present["index.md"] {
		pages = append(pages, Page{ID: "index", Kind: KindIndex, File: path.Join(root, "index.md")})
	}
	if present["glossary.md"] {
		pages = append(pages, Page{ID: "glossary", Kind: KindGlossary, File: path.Join(root, "glossary.md")})
	}

	seen := map[string]bool{}
	for _, kind := range typeDirs {
		if !present[string(kind)] {
			continue
		}
		dir := path.Join(root, string(kind))
		files, err := readDir(filepath.Join(c.Dir, filepath.FromSlash(dir)))
		if err != nil {
			problems.Addf(display(dir), 0, "", "não foi possível ler o diretório: %v", err)
			continue
		}
		for _, f := range files {
			rel := path.Join(dir, f.Name())
			switch {
			case f.IsDir():
				problems.Addf(display(rel), 0, "", "subdiretório não é aceito em %s/; deixe os arquivos .md direto nele", kind)
				continue
			case path.Ext(f.Name()) != ".md":
				problems.Addf(display(rel), 0, "", "só arquivos .md são aceitos em %s/", kind)
				continue
			}
			name := orderPrefix.ReplaceAllString(strings.TrimSuffix(f.Name(), ".md"), "")
			if name == "" {
				problems.Addf(display(rel), 0, "", "nome vazio depois de tirar o prefixo de ordem")
				continue
			}
			id := string(kind) + "/" + name
			if seen[id] {
				problems.Addf(display(rel), 0, "", "identificador %q repetido; outro arquivo dá o mesmo nome sem o prefixo de ordem", id)
				continue
			}
			seen[id] = true
			pages = append(pages, Page{ID: id, Kind: kind, File: rel})
		}
	}
	return pages, problems
}

// readDir lê o diretório sem as entradas ocultas, em ordem de bytes.
func readDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

// CheckReferences confere as páginas citadas no guidebook.yaml.
func CheckReferences(c *config.Config, pages []Page) diag.List {
	var problems diag.List
	if c.User == nil {
		return nil
	}
	kinds := map[string]Kind{}
	for _, p := range pages {
		kinds[p.ID] = p.Kind
	}
	missing := func(field, id string) bool {
		if _, ok := kinds[id]; !ok {
			problems.Addf(c.File, 0, field, "página %q não existe", id)
			return true
		}
		return false
	}
	for i, p := range c.User.Profiles {
		for j, id := range p.StartHere {
			missing(fmt.Sprintf("user.profiles[%d].start_here[%d]", i, j), id)
		}
	}
	for i, id := range c.User.Videos.Tasks {
		field := fmt.Sprintf("user.videos.tasks[%d]", i)
		if !missing(field, id) && kinds[id] != KindTask {
			problems.Addf(c.File, 0, field, "%q não é uma tarefa (tasks/)", id)
		}
	}
	return problems
}
