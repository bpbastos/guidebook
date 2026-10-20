package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/jpeg" // registra o formato para image.DecodeConfig
	_ "image/png"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"

	"github.com/bpbastos/guidebook/internal/markdown"
)

// shot é uma captura com marcadores (docs/spec.md, seção 6).
type shot struct {
	id      string // id da <figure>, único no guia
	src     string // já reescrito para media/
	alt     string
	width   int
	height  int
	markers []marker
}

type markerFile struct {
	Version *int     `json:"version"`
	Image   string   `json:"image"`
	Width   *int     `json:"width"`
	Height  *int     `json:"height"`
	Markers []marker `json:"markers"`
}

type marker struct {
	N    *int     `json:"n"`
	X    *float64 `json:"x"`
	Y    *float64 `json:"y"`
	Text string   `json:"text"`
}

const maxMarkerText = 80

// markersFile devolve o caminho do .markers.json ao lado da captura.
func markersFile(imageRel string) string {
	return strings.TrimSuffix(imageRel, path.Ext(imageRel)) + ".markers.json"
}

// shot confere a captura com marcadores ligada à imagem e devolve nil se
// ela não tem .markers.json. Problemas do JSON saem no arquivo dele; os
// de ligação com a lista, na linha da imagem.
func (l *loader) shot(p *parsed, img *ast.Image, imageRel, src string, report func(ast.Node, string, ...any)) (result *shot) {
	jsonRel := markersFile(imageRel)
	body, err := os.ReadFile(l.abs(jsonRel))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	jsonFile := l.abs(jsonRel)
	bad := func(field, format string, args ...any) {
		l.problems.Addf(jsonFile, 0, field, format, args...)
	}
	if err != nil {
		bad("", "não foi possível ler o arquivo: %v", err)
		return nil
	}

	// Ligação com o Markdown: imagem sozinha num parágrafo de primeiro
	// nível, seguida de uma lista numerada com um item por marcador. O
	// problema de ligação sai junto com os do JSON, em qualquer saída.
	var list *ast.List
	linkProblem := ""
	if para, ok := img.Parent().(*ast.Paragraph); !ok || para.ChildCount() != 1 || para.Parent() != p.doc {
		linkProblem = "a captura com marcadores precisa ficar sozinha no parágrafo, fora de lista e citação"
	} else if list, ok = para.NextSibling().(*ast.List); !ok || !list.IsOrdered() {
		linkProblem = "depois da captura com marcadores precisa vir uma lista numerada"
	}
	defer func() {
		if linkProblem != "" {
			report(img, "%s", linkProblem)
			result = nil
		}
	}()

	var mf markerFile
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&mf); err != nil {
		if name, found := strings.CutPrefix(err.Error(), "json: unknown field "); found {
			bad(strings.Trim(name, `"`), "chave desconhecida")
		} else {
			bad("", "JSON inválido (%v)", err)
		}
		return nil
	}

	problems := len(l.problems)
	switch {
	case mf.Version == nil:
		bad("version", "obrigatório")
	case *mf.Version != 1:
		bad("version", "só a versão 1 é aceita")
	}
	if name := path.Base(imageRel); mf.Image != name {
		bad("image", "esperava %q, o nome da captura", name)
	}

	f, err := os.Open(l.abs(imageRel))
	if err != nil {
		bad("", "não foi possível ler a captura: %v", err)
		return nil
	}
	cfg, format, err := image.DecodeConfig(f)
	f.Close()
	if err != nil || (format != "png" && format != "jpeg") {
		bad("", "a captura com marcadores precisa ser PNG ou JPEG")
		return nil
	}
	checkSize := func(field string, v *int, actual int) {
		switch {
		case v == nil:
			bad(field, "obrigatório")
		case *v != actual:
			bad(field, "%d não bate com a captura, que tem %d px", *v, actual)
		}
	}
	checkSize("width", mf.Width, cfg.Width)
	checkSize("height", mf.Height, cfg.Height)

	if len(mf.Markers) == 0 {
		bad("markers", "liste pelo menos um marcador")
	}
	for i, m := range mf.Markers {
		f := fmt.Sprintf("markers[%d]", i)
		switch {
		case m.N == nil:
			bad(f+".n", "obrigatório")
		case *m.N != i+1:
			bad(f+".n", "esperava %d; os marcadores vão de 1 a N, em ordem", i+1)
		}
		for _, c := range []struct {
			name string
			v    *float64
		}{{"x", m.X}, {"y", m.Y}} {
			switch {
			case c.v == nil:
				bad(f+"."+c.name, "obrigatório")
			case *c.v < 0 || *c.v > 1:
				bad(f+"."+c.name, "precisa ficar entre 0 e 1 (posição relativa)")
			}
		}
		switch text := strings.TrimSpace(m.Text); {
		case text == "":
			bad(f+".text", "obrigatório")
		case utf8.RuneCountInString(text) > maxMarkerText:
			bad(f+".text", "use até %d caracteres", maxMarkerText)
		}
	}
	if len(l.problems) > problems || linkProblem != "" {
		return nil
	}

	if items := list.ChildCount(); items != len(mf.Markers) {
		report(img, "a captura tem %d marcadores e a lista seguinte tem %d itens", len(mf.Markers), items)
		return nil
	}
	list.SetAttributeString("data-gb-steps", "")
	p.shotCount++
	return &shot{
		id:      fmt.Sprintf("gb-shot-%s-%d", strings.ReplaceAll(p.page.ID, "/", "-"), p.shotCount),
		src:     src,
		alt:     markdown.PlainText(img, p.source),
		width:   cfg.Width,
		height:  cfg.Height,
		markers: mf.Markers,
	}
}

// imageAttributes põe no <img> as dimensões, se for PNG ou JPEG, e a
// carga preguiçosa (docs/spec.md, seção 9.4).
func (l *loader) imageAttributes(img *ast.Image, rel string) {
	if f, err := os.Open(l.abs(rel)); err == nil {
		cfg, format, err := image.DecodeConfig(f)
		f.Close()
		if err == nil && (format == "png" || format == "jpeg") {
			img.SetAttributeString("width", []byte(strconv.Itoa(cfg.Width)))
			img.SetAttributeString("height", []byte(strconv.Itoa(cfg.Height)))
		}
	}
	img.SetAttributeString("loading", []byte("lazy"))
	img.SetAttributeString("decoding", []byte("async"))
}

// renderShot escreve a captura como <figure>: imagem, marcadores (que o
// JavaScript posiciona a partir de data-x e data-y), visita guiada e a
// lista acessível dos pontos.
func (l *loader) renderShot(b *bytes.Buffer, s *shot) {
	e := html.EscapeString
	fmt.Fprintf(b, "<figure class=\"gb-shot\" id=\"%s\">\n<div class=\"gb-shot-frame\"><div class=\"gb-shot-zoom\">\n", s.id)
	fmt.Fprintf(b, "<img src=\"%s\" alt=\"%s\" width=\"%d\" height=\"%d\" loading=\"lazy\" decoding=\"async\" aria-describedby=\"%s-points\">\n", e(s.src), e(s.alt), s.width, s.height, s.id)
	for _, m := range s.markers {
		fmt.Fprintf(b, "<button type=\"button\" class=\"gb-marker\" data-n=\"%d\" data-x=\"%s\" data-y=\"%s\" aria-label=\"%d. %s\" hidden>%d</button>\n",
			*m.N, strconv.FormatFloat(*m.X, 'f', -1, 64), strconv.FormatFloat(*m.Y, 'f', -1, 64), *m.N, e(strings.TrimSpace(m.Text)), *m.N)
	}
	b.WriteString("</div></div>\n")
	fmt.Fprintf(b, "<button type=\"button\" class=\"gb-tour\" hidden>%s</button>\n", e(l.c.UIText("guided_tour")))
	fmt.Fprintf(b, "<div class=\"gb-tour-panel\" hidden><p class=\"gb-tour-text\" aria-live=\"polite\"></p>"+
		"<button type=\"button\" class=\"gb-tour-prev\">%s</button> <button type=\"button\" class=\"gb-tour-next\">%s</button> "+
		"<button type=\"button\" class=\"gb-tour-close\">%s</button></div>\n",
		e(l.c.UIText("previous")), e(l.c.UIText("next")), e(l.c.UIText("close")))
	fmt.Fprintf(b, "<figcaption><p class=\"gb-shot-points-title\">%s</p>\n<ol class=\"gb-shot-points\" id=\"%s-points\">\n", e(l.c.UIText("screen_points")), s.id)
	for _, m := range s.markers {
		fmt.Fprintf(b, "<li>%s</li>\n", e(strings.TrimSpace(m.Text)))
	}
	b.WriteString("</ol></figcaption>\n</figure>\n")
}
