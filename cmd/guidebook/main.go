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
	"strings"
	"syscall"
	"time"

	"github.com/bpbastos/guidebook/internal/build"
	"github.com/bpbastos/guidebook/internal/serve"
)

// version é trocada no build de release com -ldflags "-X main.version=...".
var version = "dev"

const usage = `uso:
  guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
  guidebook check [-config guidebook.yaml] [-guide all|user|developer]
  guidebook serve [-config guidebook.yaml] [-addr 127.0.0.1:8080] [-interval 2s]
  guidebook version

build   valida a documentação e grava os guias em -out
check   faz as mesmas validações sem gravar nada
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
	var out *string
	if cmd == "build" {
		out = fs.String("out", "out", "")
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

	o := build.Options{Config: *configPath, Guide: *guide}
	if out != nil {
		o.Out = *out
	}
	r := build.Run(o)
	switch {
	case len(r.Problems) > 0:
		fmt.Fprint(stderr, r.Problems)
		return int(r.Status)
	case r.Err != nil:
		fmt.Fprintf(stderr, "guidebook: %v\n", r.Err)
		return int(r.Status)
	case cmd == "check":
		fmt.Fprintln(stdout, "check: nenhum problema encontrado")
		return exitOK
	}
	for _, path := range r.Wrote {
		kind := "do usuário"
		if strings.HasSuffix(path, "developer-guide.html") {
			kind = "do desenvolvedor"
		}
		fmt.Fprintf(stdout, "build: guia %s gravado em %s\n", kind, path)
	}
	return exitOK
}

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

func formatSize(n int64) string { return "" }
