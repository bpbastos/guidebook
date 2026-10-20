// Comando guidebook: gera guias em HTML a partir de Markdown
// (docs/spec.md, seção 2).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/bpbastos/guidebook/internal/build"
	"github.com/bpbastos/guidebook/internal/serve"
)

// version é trocada no build de release com -ldflags "-X main.version=...".
var version = "dev"

const usage = `uso:
  guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
  guidebook check [-config guidebook.yaml] [-guide all|user|developer] [-no-media]
  guidebook serve [-config guidebook.yaml] [-addr 127.0.0.1:8080] [-interval 2s]
  guidebook version

build   valida a documentação e grava os guias em -out
check   faz as mesmas validações sem gravar nada no projeto; com
        -no-media, não exige a mídia gerada pelo e2e
serve   gera e serve só o guia do usuário por HTTP, regerando quando a
        entrada muda; o guia do desenvolvedor nunca é servido
version mostra a versão

Códigos de saída: 0 sucesso; 1 violação de regra ou erro de conteúdo;
2 uso errado, configuração inválida ou erro de leitura e gravação.
`

const (
	exitOK      = 0
	exitContent = 1
	exitUsage   = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	switch cmd, rest := args[0], args[1:]; cmd {
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return exitOK
	case "version":
		if len(rest) > 0 {
			return usageError(stderr, "argumento inesperado: %q", rest[0])
		}
		info, _ := debug.ReadBuildInfo()
		fmt.Fprintf(stdout, "guidebook %s\n", versionString(version, info))
		return exitOK
	case "build", "check":
		return validateAndBuild(cmd, rest, stdout, stderr)
	case "serve":
		return serveCmd(rest, stdout, stderr)
	default:
		return usageError(stderr, "subcomando desconhecido: %q", cmd)
	}
}

func validateAndBuild(cmd string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "guidebook.yaml", "")
	guide := fs.String("guide", "all", "")
	productVersion := fs.String("product-version", "", "")
	date := fs.String("date", "", "")
	var out *string
	noMedia := new(bool)
	if cmd == "build" {
		out = fs.String("out", "out", "")
	} else {
		noMedia = fs.Bool("no-media", false, "")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return exitOK
		}
		return usageError(stderr, "%s", translateFlagError(err))
	}
	if fs.NArg() > 0 {
		return usageError(stderr, "argumento inesperado: %q", fs.Arg(0))
	}
	if *guide != "all" && *guide != "user" && *guide != "developer" {
		return usageError(stderr, "-guide aceita all, user ou developer (veio %q)", *guide)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if v := *productVersion; set["product-version"] && (strings.TrimSpace(v) == "" || strings.ContainsAny(v, "\r\n") || utf8.RuneCountInString(v) > maxProductVersion) {
		return usageError(stderr, "-product-version aceita um texto de 1 a %d caracteres, sem quebra de linha", maxProductVersion)
	}
	if d := *date; set["date"] {
		if t, err := time.Parse(time.DateOnly, d); err != nil || t.Format(time.DateOnly) != d {
			return usageError(stderr, "-date aceita uma data válida no formato AAAA-MM-DD (veio %q)", d)
		}
	}

	o := build.Options{Config: *configPath, Guide: *guide, NoMedia: *noMedia, ProductVersion: *productVersion, Date: *date}
	if out != nil {
		o.Out = *out
	}
	r := build.Run(o)
	for _, s := range r.Sizes {
		fmt.Fprintf(stdout, "%s: guia %s com %s\n", cmd, guideName(s.Guide), formatSize(s.Bytes))
	}
	if len(r.Problems) > 0 {
		fmt.Fprint(stderr, r.Problems)
	} else if r.Err != nil {
		fmt.Fprintf(stderr, "guidebook: %v\n", r.Err)
	}
	// Avisos depois dos erros, sem mudar o código (docs/spec.md, seção 2.4).
	for _, w := range r.Warnings {
		fmt.Fprintln(stderr, w)
	}
	switch {
	case len(r.Problems) > 0 || r.Err != nil:
		return int(r.Status)
	case cmd == "check":
		fmt.Fprintln(stdout, "check: nenhum problema encontrado")
		return exitOK
	}
	for i, path := range r.Wrote {
		fmt.Fprintf(stdout, "build: guia %s gravado em %s\n", guideName(r.Sizes[i].Guide), path)
	}
	return exitOK
}

// guideName dá o nome do guia na saída: "do usuário" ou "do
// desenvolvedor".
func guideName(guide string) string {
	if guide == "developer" {
		return "do desenvolvedor"
	}
	return "do usuário"
}

// maxProductVersion é o tamanho máximo de -product-version, em
// caracteres (docs/spec.md, seção 2.3).
const maxProductVersion = 40

// minInterval é o menor intervalo de conferência do serve.
const minInterval = 500 * time.Millisecond

func serveCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "guidebook.yaml", "")
	addr := fs.String("addr", "127.0.0.1:8080", "")
	interval := fs.Duration("interval", 2*time.Second, "")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return exitOK
		}
		if strings.HasSuffix(err.Error(), "defined: -guide") {
			return usageError(stderr, "o serve não tem -guide: ele serve só o guia do usuário")
		}
		return usageError(stderr, "%s", translateFlagError(err))
	}
	switch {
	case fs.NArg() > 0:
		return usageError(stderr, "argumento inesperado: %q", fs.Arg(0))
	case *interval < minInterval:
		return usageError(stderr, "-interval precisa ser de pelo menos %s (veio %s)", minInterval, *interval)
	}

	s, err := serve.New(*configPath, os.TempDir(), stderr)
	if err != nil {
		fmt.Fprintf(stderr, "guidebook: %v\n", err)
		return exitUsage
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		s.Close()
		fmt.Fprintf(stderr, "guidebook: não foi possível escutar em %s: %v\n", *addr, err)
		return exitUsage
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	if err := s.Run(ctx, ln, *interval); err != nil {
		fmt.Fprintf(stderr, "guidebook: %v\n", err)
		return exitUsage
	}
	return exitOK
}

// versionString escolhe a versão mostrada: a do -ldflags (imagem e
// release) ou, sem ela, a do módulo, que o "go install ...@vX.Y.Z" grava
// no binário.
func versionString(ldflags string, info *debug.BuildInfo) string {
	if ldflags != "dev" || info == nil {
		return ldflags
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return ldflags
}

func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "guidebook: "+format+"\n\n", args...)
	fmt.Fprint(stderr, usage)
	return exitUsage
}

// translateFlagError traduz as mensagens do pacote flag, que vêm em
// inglês.
func translateFlagError(err error) string {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "flag provided but not defined: "):
		return "opção desconhecida: " + strings.TrimPrefix(msg, "flag provided but not defined: ")
	case strings.HasPrefix(msg, "flag needs an argument: "):
		return "a opção " + strings.TrimPrefix(msg, "flag needs an argument: ") + " precisa de um valor"
	case strings.HasPrefix(msg, "bad flag syntax: "):
		return "opção mal escrita: " + strings.TrimPrefix(msg, "bad flag syntax: ")
	default:
		return "opção inválida: " + msg
	}
}

// formatSize escreve o tamanho de um guia como na spec (seção 2.1):
// bytes com separador de milhar e MiB com uma casa decimal.
func formatSize(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	return fmt.Sprintf("%s bytes (%s MiB)", b.String(), build.MiB(n))
}
