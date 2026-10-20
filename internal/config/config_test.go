package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bpbastos/guidebook/internal/diag"
)

// writeProject cria um projeto temporário com os arquivos dados
// (caminho relativo → conteúdo); caminho terminado em "/" vira diretório.
func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func loadYAML(t *testing.T, yaml string, extra map[string]string) (*Config, diag.List) {
	t.Helper()
	files := map[string]string{"guidebook.yaml": yaml, "docs/user/": ""}
	for k, v := range extra {
		files[k] = v
	}
	dir := writeProject(t, files)
	return Load(filepath.Join(dir, "guidebook.yaml"))
}

const minimal = `version: 1
product:
  name: Produto
user:
  dir: docs/user
`

func TestLoadExample(t *testing.T) {
	cfg, problems := Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatalf("o exemplo deveria ser válido:\n%s", problems)
	}
	if cfg.User == nil || cfg.Developer == nil {
		t.Fatal("user e developer deveriam ter sido lidos")
	}
	if cfg.Product.Name != "Pedido Fácil" {
		t.Errorf("product.name = %q", cfg.Product.Name)
	}
	if got := cfg.User.Profiles[0].StartHere; !reflect.DeepEqual(got, []string{"tutorials/first-order", "tasks/register-order"}) {
		t.Errorf("start_here = %v", got)
	}
	if got := cfg.Developer.Sections.Modules.Files; !reflect.DeepEqual(got, []string{"docs/developer/modules/orders.md"}) {
		t.Errorf("modules.Files = %v", got)
	}
	if cfg.Developer.Sections.Security != nil {
		t.Errorf("seção security deveria ficar ausente")
	}
}

func TestDefaults(t *testing.T) {
	cfg, problems := loadYAML(t, minimal, nil)
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados:\n%s", problems)
	}
	if cfg.User == nil {
		t.Fatal("user deveria ter sido lido")
	}
	if cfg.Product.Colors.Primary != "#1f6feb" || cfg.Product.Colors.Accent != "#1f6feb" {
		t.Errorf("cores padrão = %+v", cfg.Product.Colors)
	}
	if cfg.User.Changelog != "CHANGELOG.md" {
		t.Errorf("changelog padrão = %q", cfg.User.Changelog)
	}
	if cfg.User.BlockedPrefix != "O sistema não me deixa" {
		t.Errorf("blocked_prefix padrão = %q", cfg.User.BlockedPrefix)
	}
	if got := *cfg.UserRules().MaxSentenceWords; got != 25 {
		t.Errorf("limite padrão = %d", got)
	}
	if got := cfg.UIText("start_here"); got != "Comece por aqui" {
		t.Errorf("UIText padrão = %q", got)
	}
	if got := cfg.UIText("slide_list"); got != "Lista de slides" {
		t.Errorf("UIText(slide_list) = %q", got)
	}
}

func TestRulesPerGuide(t *testing.T) {
	cfg, problems := Load(filepath.Join("..", "..", "testdata", "example", "guidebook.yaml"))
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	user, dev := cfg.UserRules(), cfg.DeveloperRules()
	if *user.MaxSentenceWords != 25 || *dev.MaxSentenceWords != 35 {
		t.Errorf("limites = %d e %d; esperava 25 e 35", *user.MaxSentenceWords, *dev.MaxSentenceWords)
	}
	if len(user.ForbiddenPatterns) != 2 || len(dev.ForbiddenPatterns) != 1 {
		t.Errorf("padrões = %d e %d; esperava 2 (gerais + usuário) e 1", len(user.ForbiddenPatterns), len(dev.ForbiddenPatterns))
	}
}

func TestUIOverride(t *testing.T) {
	cfg, problems := loadYAML(t, minimal+"ui:\n  start_here: Por onde começar\n", nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if got := cfg.UIText("start_here"); got != "Por onde começar" {
		t.Errorf("UIText = %q", got)
	}
}

func TestProblems(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		extra map[string]string
		field string
		msg   string
	}{
		{"chave desconhecida na raiz", minimal + "colour: azul\n", nil, "colour", "chave desconhecida"},
		{"chave desconhecida aninhada", "version: 1\nproduct:\n  name: X\n  nome: Y\nuser:\n  dir: docs/user\n", nil, "product.nome", "chave desconhecida"},
		{"chave repetida", minimal + "version: 1\n", nil, "version", "chave repetida"},
		{"versão errada", strings.Replace(minimal, "version: 1", "version: 2", 1), nil, "version", "só a versão 1"},
		{"versão ausente", strings.Replace(minimal, "version: 1\n", "", 1), nil, "version", "obrigatório"},
		{"nome ausente", "version: 1\nproduct: {}\nuser:\n  dir: docs/user\n", nil, "product.name", "obrigatório"},
		{"cor inválida", "version: 1\nproduct:\n  name: X\n  colors:\n    primary: azul\nuser:\n  dir: docs/user\n", nil, "product.colors.primary", "#rrggbb"},
		{"tipo errado", minimal + "rules:\n  max_sentence_words: muitas\n", nil, "rules.max_sentence_words", "número inteiro"},
		{"lista esperada", minimal + "rules:\n  forbidden_patterns: x\n", nil, "rules.forbidden_patterns", "lista"},
		{"objeto esperado", "version: 1\nproduct: X\nuser:\n  dir: docs/user\n", nil, "product", "objeto"},
		{"limite fora da faixa", minimal + "rules:\n  max_sentence_words: 3\n", nil, "rules.max_sentence_words", "entre 5 e 200"},
		{"regex inválida", minimal + "rules:\n  forbidden_patterns:\n    - pattern: '('\n      message: m\n", nil, "rules.forbidden_patterns[0].pattern", "expressão regular inválida"},
		{"mensagem ausente", minimal + "rules:\n  forbidden_patterns:\n    - pattern: x\n", nil, "rules.forbidden_patterns[0].message", "obrigatório"},
		{"logo com formato errado", "version: 1\nproduct:\n  name: X\n  logo: logo.gif\nuser:\n  dir: docs/user\n", map[string]string{"logo.gif": "GIF89a"}, "product.logo", ".svg ou .png"},
		{"logo inexistente", "version: 1\nproduct:\n  name: X\n  logo: logo.svg\nuser:\n  dir: docs/user\n", nil, "product.logo", "não existe"},
		{"dir sai do projeto", "version: 1\nproduct:\n  name: X\nuser:\n  dir: ../fora\n", nil, "user.dir", "sai do diretório do guidebook.yaml"},
		{"dir absoluto", "version: 1\nproduct:\n  name: X\nuser:\n  dir: /tmp\n", nil, "user.dir", "caminho absoluto"},
		{"dir inexistente", "version: 1\nproduct:\n  name: X\nuser:\n  dir: docs/nada\n", nil, "user.dir", "não existe"},
		{"dir é arquivo", "version: 1\nproduct:\n  name: X\nuser:\n  dir: docs/f.md\n", map[string]string{"docs/f.md": "# F\n"}, "user.dir", "não é um diretório"},
		{"perfil sem kebab-case", minimal + "  profiles:\n    - id: Atendente\n      name: A\n      start_here: [index]\n", nil, "user.profiles[0].id", "kebab-case"},
		{"perfil repetido", minimal + "  profiles:\n    - id: a\n      name: A\n      start_here: [index]\n    - id: a\n      name: B\n      start_here: [index]\n", nil, "user.profiles[1].id", "repetido"},
		{"perfil sem trilha", minimal + "  profiles:\n    - id: a\n      name: A\n", nil, "user.profiles[0].start_here", "pelo menos uma página"},
		{"trilha com repetição", minimal + "  profiles:\n    - id: a\n      name: A\n      start_here: [index, index]\n", nil, "user.profiles[0].start_here[1]", "repetida"},
		{"vídeo sem manifesto", minimal + "  videos:\n    tasks: [tasks/x]\n", nil, "user.videos.manifest", "obrigatório"},
		{"manifesto inexistente", minimal + "  videos:\n    manifest: m.json\n    tasks: [tasks/x]\n", nil, "user.videos.manifest", "não existe"},
		{"url sem https", minimal + "developer:\n  repository_url: http://x.example\n", nil, "developer.repository_url", "https://"},
		{"seção desconhecida", minimal + "developer:\n  sections:\n    extras:\n      sources: [a.md]\n", nil, "developer.sections.extras", "chave desconhecida"},
		{"seção sem fontes", minimal + "developer:\n  sections:\n    overview:\n      sources: []\n", nil, "developer.sections.overview.sources", "pelo menos uma fonte"},
		{"glob sai do projeto", minimal + "developer:\n  sections:\n    modules:\n      sources: ['../*.md']\n", nil, "developer.sections.modules.sources[0]", "sai do diretório do guidebook.yaml"},
		{"glob escondido sai do projeto", minimal + "developer:\n  sections:\n    modules:\n      sources: ['docs/../../*.md']\n", nil, "developer.sections.modules.sources[0]", "sai do diretório do guidebook.yaml"},
		{"glob absoluto", minimal + "developer:\n  sections:\n    modules:\n      sources: ['/etc/*.conf']\n", nil, "developer.sections.modules.sources[0]", "caminho absoluto"},
		{"glob sem arquivo", minimal + "developer:\n  sections:\n    modules:\n      sources: ['docs/nada/*.md']\n", nil, "developer.sections.modules.sources[0]", "nenhum arquivo"},
		{"glob inválido", minimal + "developer:\n  sections:\n    modules:\n      sources: ['docs/[.md']\n", nil, "developer.sections.modules.sources[0]", "padrão inválido"},
		{"fonte que não é markdown", minimal + "developer:\n  sections:\n    overview:\n      sources: ['*.yaml']\n", nil, "developer.sections.overview.sources[0]", "não é Markdown"},
		{"arquivo em duas seções", minimal + "developer:\n  sections:\n    overview:\n      sources: [a.md]\n    security:\n      sources: ['*.md']\n", map[string]string{"a.md": "# A\n"}, "developer.sections.security.sources[0]", "já está na seção overview"},
		{"ui desconhecida", minimal + "ui:\n  botao: x\n", nil, "ui.botao", "chave desconhecida"},
		{"ui vazia", minimal + "ui:\n  close: ''\n", nil, "ui.close", "não pode ficar vazio"},
		{"ui sem marcador", minimal + "ui:\n  slide_of: Slide {n}\n", nil, "ui.slide_of", "{total}"},
		{"nenhum guia", "version: 1\nproduct:\n  name: X\n", nil, "user", "defina user, developer ou os dois"},
		{"whats_new com tipo errado", minimal + "  whats_new: talvez\n", nil, "user.whats_new", "booleano"},
		{"ui de versão sem marcador", minimal + "ui:\n  product_version: Versão\n", nil, "ui.product_version", "{version}"},
		{"ui de data sem marcador", minimal + "ui:\n  generated_on: Gerado\n", nil, "ui.generated_on", "{date}"},
		{"teto zero", minimal + "  max_size_mib: 0\n", nil, "user.max_size_mib", "maior que 0"},
		{"teto negativo", minimal + "developer:\n  max_size_mib: -3\n  sections:\n    overview:\n      sources: [a.md]\n", map[string]string{"a.md": "# A\n"}, "developer.max_size_mib", "maior que 0"},
		{"teto fracionário", minimal + "  max_size_mib: 1.5\n", nil, "user.max_size_mib", "número inteiro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := loadYAML(t, tt.yaml, tt.extra)
			for _, p := range problems {
				if p.Field == tt.field && strings.Contains(p.Msg, tt.msg) {
					if !strings.HasSuffix(p.File, "guidebook.yaml") {
						t.Errorf("arquivo do problema = %q", p.File)
					}
					return
				}
			}
			t.Errorf("esperava %q com %q; veio:\n%s", tt.field, tt.msg, problems)
		})
	}
}

func TestListsAllProblemsAtOnce(t *testing.T) {
	yaml := "version: 3\nproduct:\n  name: ''\n  colors:\n    accent: x\nrules:\n  max_sentence_words: 1\nuser:\n  dir: docs/user\n"
	_, problems := loadYAML(t, yaml, nil)
	want := []string{"version", "product.name", "product.colors.accent", "rules.max_sentence_words"}
	var got []string
	for _, p := range problems {
		got = append(got, p.Field)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("campos = %v; esperava %v", got, want)
	}
}

func TestFileProblems(t *testing.T) {
	tests := []struct {
		name, yaml, msg string
	}{
		{"vazio", "", "arquivo vazio"},
		{"só comentário", "# nada\n", "arquivo vazio"},
		{"sintaxe", "version: [1\n", "YAML inválido"},
		{"raiz não é objeto", "- a\n", "objeto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := loadYAML(t, tt.yaml, nil)
			if len(problems) != 1 || !strings.Contains(problems[0].Msg, tt.msg) {
				t.Errorf("esperava um problema com %q; veio:\n%s", tt.msg, problems)
			}
		})
	}
	t.Run("ausente", func(t *testing.T) {
		_, problems := Load(filepath.Join(t.TempDir(), "guidebook.yaml"))
		if len(problems) != 1 || !strings.Contains(problems[0].Msg, "não foi possível ler") {
			t.Errorf("veio:\n%s", problems)
		}
	})
}

func TestDevelopmentSection(t *testing.T) {
	yaml := `version: 1
product:
  name: P
developer:
  sections:
    operations:
      sources: [docs/ops.md]
    development:
      sources: [docs/dev.md]
    delivery:
      sources: [docs/delivery.md]
`
	cfg, problems := loadYAML(t, yaml, map[string]string{
		"docs/ops.md": "# O\n", "docs/dev.md": "# D\n", "docs/delivery.md": "# E\n",
	})
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	var keys, titles []string
	for _, s := range cfg.Developer.Sections.SectionList() {
		keys = append(keys, s.Key)
		titles = append(titles, s.DisplayTitle())
	}
	if want := []string{"delivery", "development", "operations"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("ordem = %v; esperava %v", keys, want)
	}
	if want := []string{"Ciclo de entrega", "Como manter e evoluir", "Operação"}; !reflect.DeepEqual(titles, want) {
		t.Errorf("títulos = %v; esperava %v", titles, want)
	}
}

// TestNoMedia: no check -no-media, o manifesto pode faltar, mas as
// regras de caminho continuam (docs/spec.md, seção 2.2).
func TestNoMedia(t *testing.T) {
	load := func(yaml string) diag.List {
		dir := writeProject(t, map[string]string{"guidebook.yaml": yaml, "docs/user/": ""})
		_, problems := LoadNoMedia(filepath.Join(dir, "guidebook.yaml"))
		return problems
	}
	for name, yaml := range map[string]string{
		"sem manifesto":         minimal + "  videos:\n    tasks: [tasks/x]\n",
		"manifesto inexistente": minimal + "  videos:\n    manifest: docs/user/media/videos/manifest.json\n    tasks: [tasks/x]\n",
	} {
		if problems := load(yaml); len(problems) > 0 {
			t.Errorf("%s: deveria aceitar no modo sem mídia:\n%s", name, problems)
		}
	}
	problems := load(minimal + "  videos:\n    manifest: ../fora/manifest.json\n    tasks: [tasks/x]\n")
	if len(problems) != 1 || problems[0].Field != "user.videos.manifest" || !strings.Contains(problems[0].Msg, "sai do diretório") {
		t.Errorf("caminho que sai do projeto deveria continuar recusado: %v", problems)
	}
	// Sem o modo, nada fica mais frouxo.
	if _, problems := loadYAML(t, minimal+"  videos:\n    tasks: [tasks/x]\n", nil); len(problems) == 0 {
		t.Error("o Load comum deveria exigir o manifesto")
	}
}
