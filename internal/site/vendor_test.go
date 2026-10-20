package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

type vendorPackage struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	TarballIntegrity string `json:"tarball_integrity"`
	File             string `json:"file"`
	VendoredAs       string `json:"vendored_as"`
	SHA256           string `json:"sha256"`
	License          string `json:"license"`
	LicenseFile      string `json:"license_file"`
	// ReviewedMatches é quantas ocorrências de networkAPI foram revisadas
	// à mão no arquivo (ADR do pacote); o teste exige exatamente esse
	// número.
	ReviewedMatches *int `json:"reviewed_matches"`
}

func vendorManifest(t *testing.T) []vendorPackage {
	t.Helper()
	body, err := assets.ReadFile("assets/third_party/vendor.json")
	if err != nil {
		t.Fatalf("manifesto ausente: %v", err)
	}
	var m struct {
		Packages []vendorPackage `json:"packages"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	return m.Packages
}

// TestVendorHashes confere cada arquivo vendorizado contra o manifesto.
func TestVendorHashes(t *testing.T) {
	pkgs := vendorManifest(t)
	names := map[string]bool{}
	for _, p := range pkgs {
		names[p.Name] = true
		if p.Version == "" || p.License == "" || p.File == "" || p.TarballIntegrity == "" {
			t.Errorf("%s: manifesto incompleto: %+v", p.Name, p)
		}
		if want := "https://registry.npmjs.org/" + p.Name + "/-/" + p.Name + "-" + p.Version + ".tgz"; p.Source != want {
			t.Errorf("%s: origem %q; esperava %q", p.Name, p.Source, want)
		}
		body, err := assets.ReadFile("assets/third_party/" + p.VendoredAs)
		if err != nil {
			t.Errorf("%s: arquivo ausente: %v", p.Name, err)
			continue
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != p.SHA256 {
			t.Errorf("%s: SHA-256 %s; o manifesto diz %s", p.Name, got, p.SHA256)
		}
		license, err := assets.ReadFile("assets/third_party/" + p.LicenseFile)
		if err != nil || !strings.Contains(string(license), "Permission is hereby granted") {
			t.Errorf("%s: licença ausente ou diferente da MIT: %v", p.Name, err)
		}
	}
	for _, want := range []string{"minisearch", "mermaid"} {
		if !names[want] {
			t.Errorf("%s fora do manifesto", want)
		}
	}
}

// networkAPI casa chamadas de rede e de código dinâmico; mapa.fetch(),
// método da estrutura interna do MiniSearch, não conta.
var networkAPI = regexp.MustCompile(`(^|[^.\w])fetch\(|XMLHttpRequest|WebSocket|sendBeacon|EventSource|\bimport\(|\beval\(|new Function|new Worker|importScripts`)

// methodDef casa a definição de um método, como "fetch(key, initial) {".
var methodDef = regexp.MustCompile(`^\s*\w+\([\w, ]*\)\s*\{\s*$`)

// TestVendorOffline: o arquivo vendorizado pode citar URL e fetch() em
// comentário, mas não pode usar a rede nem gerar código. Em arquivo
// minificado, a conferência por linha não separa definição de chamada;
// por isso cada ocorrência que sobra foi revisada à mão, e o manifesto
// guarda quantas são.
func TestVendorOffline(t *testing.T) {
	for _, p := range vendorManifest(t) {
		if p.ReviewedMatches == nil {
			t.Errorf("%s: falta reviewed_matches no manifesto", p.Name)
			continue
		}
		body, _ := assets.ReadFile("assets/third_party/" + p.VendoredAs)
		var found []string
		for i, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || methodDef.MatchString(line) {
				continue
			}
			for _, m := range networkAPI.FindAllStringIndex(line, -1) {
				start := max(0, m[0]-30)
				found = append(found, fmt.Sprintf("%d: %q", i+1, line[start:min(len(line), m[1]+20)]))
			}
		}
		if len(found) != *p.ReviewedMatches {
			t.Errorf("%s: %d ocorrências; o manifesto revisou %d:\n%s", p.Name, len(found), *p.ReviewedMatches, strings.Join(found, "\n"))
		}
	}
}

// TestNoVendorSegment: o zip de módulo do Go, que o go install usa,
// descarta os arquivos de subdiretórios de qualquer diretório "vendor".
// Recurso embutido sob esse nome some na instalação e o build falha.
func TestNoVendorSegment(t *testing.T) {
	err := fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		for _, seg := range strings.Split(path, "/") {
			if seg == "vendor" {
				t.Errorf("%s: segmento \"vendor\" some no zip de módulo", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNetworkAPIPattern(t *testing.T) {
	for _, bad := range []string{`fetch("dados.json")`, `x = new XMLHttpRequest()`, `await import("./m.js")`, `eval(s)`, `new Function("a")`} {
		if !networkAPI.MatchString(bad) {
			t.Errorf("deveria casar: %s", bad)
		}
	}
	for _, ok := range []string{`this._index.fetch(term, createMap)`, `const indexData = map.fetch(k)`} {
		if networkAPI.MatchString(ok) {
			t.Errorf("não deveria casar: %s", ok)
		}
	}
	if !methodDef.MatchString("        fetch(key, initial) {") || methodDef.MatchString(`fetch("x").then(f => {`) {
		t.Errorf("methodDef errado")
	}
}
