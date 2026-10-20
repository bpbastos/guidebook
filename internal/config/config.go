// Package config lê e valida o guidebook.yaml (docs/spec.md, seção 4).
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/bpbastos/guidebook/internal/diag"
)

// Config é o conteúdo do guidebook.yaml, já com os padrões aplicados.
type Config struct {
	Version   int               `yaml:"version"`
	Product   Product           `yaml:"product"`
	Rules     Rules             `yaml:"rules"`
	User      *User             `yaml:"user"`
	Developer *Developer        `yaml:"developer"`
	UI        map[string]string `yaml:"ui"`

	// File é o caminho do guidebook.yaml como o usuário informou; Dir é
	// o diretório dele, base de todos os caminhos relativos.
	File string `yaml:"-"`
	Dir  string `yaml:"-"`

	// NoMedia é o modo do "check -no-media": nenhum arquivo de mídia é
	// lido nem exigido (docs/spec.md, seção 2.2).
	NoMedia bool `yaml:"-"`

	// ProductVersion e Date vêm de -product-version e -date (docs/spec.md,
	// seção 2.3); Date em AAAA-MM-DD. Vazios, o rodapé não aparece.
	ProductVersion string `yaml:"-"`
	Date           string `yaml:"-"`
}

type Product struct {
	Name   string `yaml:"name"`
	Logo   string `yaml:"logo"`
	Colors Colors `yaml:"colors"`
}

type Colors struct {
	Primary string `yaml:"primary"`
	Accent  string `yaml:"accent"`
}

type Rules struct {
	MaxSentenceWords  *int               `yaml:"max_sentence_words"`
	ForbiddenPatterns []ForbiddenPattern `yaml:"forbidden_patterns"`
}

type ForbiddenPattern struct {
	Pattern string `yaml:"pattern"`
	Message string `yaml:"message"`
}

type User struct {
	Dir           string    `yaml:"dir"`
	Changelog     string    `yaml:"changelog"`
	BlockedPrefix string    `yaml:"blocked_prefix"`
	Rules         Rules     `yaml:"rules"`
	Profiles      []Profile `yaml:"profiles"`
	Videos        Videos    `yaml:"videos"`
	MaxSizeMiB    *int      `yaml:"max_size_mib"`
	WhatsNew      *bool     `yaml:"whats_new"`
}

// ShowWhatsNew diz se o guia tem "O que há de novo"; o padrão é ter.
func (u *User) ShowWhatsNew() bool {
	return u.WhatsNew == nil || *u.WhatsNew
}

type Profile struct {
	ID        string   `yaml:"id"`
	Name      string   `yaml:"name"`
	StartHere []string `yaml:"start_here"`
}

type Videos struct {
	Manifest string   `yaml:"manifest"`
	Tasks    []string `yaml:"tasks"`
}

type Developer struct {
	RepositoryURL string   `yaml:"repository_url"`
	Rules         Rules    `yaml:"rules"`
	Sections      Sections `yaml:"sections"`
	MaxSizeMiB    *int     `yaml:"max_size_mib"`
}

// Sections segue a ordem de exibição da seção 8 da spec.
type Sections struct {
	Overview      *Section `yaml:"overview"`
	Modules       *Section `yaml:"modules"`
	BusinessRules *Section `yaml:"business_rules"`
	DataModel     *Section `yaml:"data_model"`
	Decisions     *Section `yaml:"decisions"`
	Security      *Section `yaml:"security"`
	Delivery      *Section `yaml:"delivery"`
	Development   *Section `yaml:"development"`
	Operations    *Section `yaml:"operations"`
	OutOfScope    *Section `yaml:"out_of_scope"`
	Changes       *Section `yaml:"changes"`
}

type Section struct {
	Title   string   `yaml:"title"`
	Sources []string `yaml:"sources"`

	// Files são os arquivos casados por Sources, relativos a Config.Dir,
	// em ordem de bytes.
	Files []string `yaml:"-"`
}

// Load lê o guidebook.yaml em path e devolve todos os problemas
// encontrados de uma vez. Com problemas, o Config devolvido é parcial e
// não deve ser usado.
func Load(path string) (*Config, diag.List) {
	return load(path, false, false)
}

// NoUserGuide é a recusa de LoadUser quando o guidebook.yaml não tem a
// seção user.
const NoUserGuide = "o serve só serve o guia do usuário; defina a seção user"

// LoadUser é o Load do serve (spec, seção 14): descarta a seção
// developer antes de validar, sem ler nem conferir as fontes dela.
func LoadUser(path string) (*Config, diag.List) {
	return load(path, true, false)
}

func load(path string, userOnly, noMedia bool) (*Config, diag.List) {
	c := &Config{File: path, Dir: filepath.Dir(path), NoMedia: noMedia}
	var problems diag.List
	report := func(field, format string, args ...any) {
		problems.Addf(path, 0, field, format, args...)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		report("", "não foi possível ler o arquivo: ele não existe")
		return c, problems
	} else if err != nil {
		report("", "não foi possível ler o arquivo: %v", err)
		return c, problems
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		report("", "YAML inválido (%s)", strings.TrimPrefix(err.Error(), "yaml: "))
		return c, problems
	}
	if len(root.Content) == 0 {
		report("", "arquivo vazio")
		return c, problems
	}

	d := &decoder{report: report}
	d.decode(root.Content[0], reflect.ValueOf(c).Elem(), "")
	if len(problems) > 0 {
		return c, problems // sem estrutura confiável, não dá para validar valores
	}

	if userOnly {
		c.Developer = nil
		if c.User == nil {
			report("user", NoUserGuide)
			return c, problems
		}
	}
	v := &validator{c: c, report: report}
	v.validate()
	return c, problems
}

// UserRules devolve as regras de escrita efetivas do guia do usuário.
func (c *Config) UserRules() Rules {
	var r Rules
	if c.User != nil {
		r = c.User.Rules
	}
	return c.Rules.merge(r)
}

// DeveloperRules devolve as regras de escrita efetivas do guia do
// desenvolvedor.
func (c *Config) DeveloperRules() Rules {
	var r Rules
	if c.Developer != nil {
		r = c.Developer.Rules
	}
	return c.Rules.merge(r)
}

// merge aplica as regras de um guia sobre as gerais: o limite do guia
// substitui o geral e os padrões se somam.
func (general Rules) merge(guide Rules) Rules {
	limit := DefaultMaxSentenceWords
	if general.MaxSentenceWords != nil {
		limit = *general.MaxSentenceWords
	}
	if guide.MaxSentenceWords != nil {
		limit = *guide.MaxSentenceWords
	}
	patterns := append([]ForbiddenPattern(nil), general.ForbiddenPatterns...)
	return Rules{
		MaxSentenceWords:  &limit,
		ForbiddenPatterns: append(patterns, guide.ForbiddenPatterns...),
	}
}

// UIText devolve o texto da interface para key, com a troca de ui
// aplicada.
func (c *Config) UIText(key string) string {
	if t, ok := c.UI[key]; ok {
		return t
	}
	return uiDefaults[key]
}

// SectionList devolve as seções configuradas do guia do desenvolvedor,
// na ordem de exibição.
func (s *Sections) SectionList() []NamedSection {
	var out []NamedSection
	v := reflect.ValueOf(s).Elem()
	for i := 0; i < v.NumField(); i++ {
		if sec := v.Field(i).Interface().(*Section); sec != nil {
			out = append(out, NamedSection{Key: v.Type().Field(i).Tag.Get("yaml"), Section: sec})
		}
	}
	return out
}

// NamedSection é uma seção com a chave dela no guidebook.yaml.
type NamedSection struct {
	Key string
	*Section
}

// DisplayTitle devolve o título da seção: o do guidebook.yaml ou o
// padrão da spec (seção 8).
func (s NamedSection) DisplayTitle() string {
	if strings.TrimSpace(s.Title) != "" {
		return s.Title
	}
	return sectionTitles[s.Key]
}

var sectionTitles = map[string]string{
	"overview":       "Visão geral",
	"modules":        "Módulos",
	"business_rules": "Regras de negócio",
	"data_model":     "Modelo de dados",
	"decisions":      "Decisões",
	"security":       "Segurança",
	"delivery":       "Ciclo de entrega",
	"development":    "Como manter e evoluir",
	"operations":     "Operação",
	"out_of_scope":   "Fora da versão",
	"changes":        "Novidades técnicas",
}

const (
	DefaultMaxSentenceWords = 25
	DefaultColor            = "#1f6feb"
	DefaultChangelog        = "CHANGELOG.md"
	DefaultBlockedPrefix    = "O sistema não me deixa"
)

// uiDefaults são os textos da interface (spec, seção 4.6).
var uiDefaults = map[string]string{
	"start_here":          "Comece por aqui",
	"profile":             "Perfil",
	"tutorials":           "Tutoriais",
	"tasks":               "Tarefas",
	"reference":           "Referência",
	"explanation":         "Explicações",
	"glossary":            "Glossário",
	"blocked_index":       "O sistema não me deixa...",
	"whats_new":           "O que há de novo",
	"search":              "Buscar",
	"search_no_results":   "Nada encontrado",
	"back_to_where_i_was": "Voltar para onde eu estava",
	"close":               "Fechar",
	"present":             "Apresentar",
	"exit_presentation":   "Sair da apresentação",
	"fullscreen":          "Tela cheia",
	"slide_of":            "Slide {n} de {total}",
	"slide_list":          "Lista de slides",
	"rules_filter":        "Filtrar regras",
	"rules_module":        "Módulo",
	"rules_all":           "Todos os módulos",
	"rules_none":          "Nenhuma regra encontrada",
	"watch_video":         "Ver em vídeo ({min} min)",
	"has_video":           "Tem vídeo",
	"guided_tour":         "Visita guiada",
	"next":                "Próximo",
	"previous":            "Anterior",
	"screen_points":       "Pontos desta tela",
	"user_guide":          "Guia do usuário",
	"developer_guide":     "Guia do desenvolvedor",
	"product_version":     "Versão {version}",
	"generated_on":        "Gerado em {date}",
}

// uiPlaceholders são os marcadores obrigatórios de cada texto.
var uiPlaceholders = map[string][]string{
	"slide_of":        {"{n}", "{total}"},
	"watch_video":     {"{min}"},
	"product_version": {"{version}"},
	"generated_on":    {"{date}"},
}

var (
	colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	idRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// LoadNoMedia é o Load do "check -no-media" (docs/spec.md, seção 2.2):
// o manifesto de vídeos pode faltar; as regras de caminho continuam.
func LoadNoMedia(path string) (*Config, diag.List) {
	return load(path, false, true)
}
