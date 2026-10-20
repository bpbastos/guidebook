package content

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"testing"
)

// picture devolve uma imagem codificada no formato pedido.
func picture(t *testing.T, format string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var b bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&b, img)
	case "jpeg":
		err = jpeg.Encode(&b, img, nil)
	case "gif":
		err = gif.Encode(&b, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

const threeMarkers = `{
  "version": 1,
  "image": "tela.png",
  "width": 640,
  "height": 400,
  "markers": [
    {"n": 1, "x": 0.125, "y": 0.2, "text": "Campo Cliente"},
    {"n": 2, "x": 0.61, "y": 0.2, "text": "Botão Adicionar item"},
    {"n": 3, "x": 0.91, "y": 0.92, "text": "Botão Salvar"}
  ]
}`

const threeSteps = "1. Escolha o cliente.\n2. Adicione os itens.\n3. Clique em Salvar."

// shotProject cria tasks/a.md com body e a captura media/tela.png (ou
// o nome dado) com os marcadores dados.
func shotProject(t *testing.T, body string, files map[string]string) (*Guide, []string) {
	t.Helper()
	all := map[string]string{
		"docs/user/tasks/a.md": "# A\n\n" + body + "\n",
	}
	if _, ok := files["docs/user/media/tela.png"]; !ok {
		all["docs/user/media/tela.png"] = picture(t, "png", 640, 400)
	}
	for k, v := range files {
		all[k] = v
	}
	cfg := projectFiles(t, linkConfig, all)
	g, problems := Load(cfg)
	var out []string
	for _, p := range problems.Sorted() {
		out = append(out, strings.TrimPrefix(filepath.ToSlash(p.String()), filepath.ToSlash(cfg.Dir)+"/"))
	}
	return g, out
}

func TestMarkersRender(t *testing.T) {
	g, problems := shotProject(t, "![Tela de novo pedido](../media/tela.png)\n\n"+threeSteps,
		map[string]string{"docs/user/media/tela.markers.json": threeMarkers})
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados: %v", problems)
	}
	html := page(t, g, "tasks/a").HTML
	for _, want := range []string{
		`<figure class="gb-shot" id="gb-shot-tasks-a-1">`,
		`<img src="media/docs/user/media/tela.png" alt="Tela de novo pedido" width="640" height="400" aria-describedby="gb-shot-tasks-a-1-points">`,
		`<button type="button" class="gb-marker" data-n="1" data-x="0.125" data-y="0.2" aria-label="1. Campo Cliente" hidden>1</button>`,
		`<button type="button" class="gb-marker" data-n="3" data-x="0.91" data-y="0.92" aria-label="3. Botão Salvar" hidden>3</button>`,
		`<button type="button" class="gb-tour" hidden>Visita guiada</button>`,
		`<p class="gb-shot-points-title">Pontos desta tela</p>`,
		`<ol class="gb-shot-points" id="gb-shot-tasks-a-1-points">`,
		`<li>Botão Adicionar item</li>`,
		`<ol data-gb-steps="">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML sem %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "<p><img") {
		t.Errorf("a captura com marcadores não deveria sair como parágrafo comum")
	}
	if strings.Contains(html, "style=") {
		t.Errorf("posição dos marcadores não pode ir em estilo em linha")
	}
}

func TestMarkersJPEG(t *testing.T) {
	_, problems := shotProject(t, "![Tela](../media/tela.jpg)\n\n"+threeSteps, map[string]string{
		"docs/user/media/tela.jpg":          picture(t, "jpeg", 640, 400),
		"docs/user/media/tela.markers.json": strings.Replace(threeMarkers, `"tela.png"`, `"tela.jpg"`, 1),
	})
	if len(problems) > 0 {
		t.Errorf("JPEG deveria ser aceito: %v", problems)
	}
}

func TestImageWithoutMarkersIsPlain(t *testing.T) {
	g, problems := shotProject(t, "![Tela](../media/tela.png)", nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if html := page(t, g, "tasks/a").HTML; !strings.Contains(html, `<p><img src="media/docs/user/media/tela.png" alt="Tela"></p>`) {
		t.Errorf("imagem sem marcadores deveria sair comum:\n%s", html)
	}
}

func TestMarkerProblems(t *testing.T) {
	const img = "![Tela](../media/tela.png)\n\n"
	const mk = "docs/user/media/tela.markers.json"
	tests := []struct {
		name, body, markers, want string
	}{
		{"lista com menos itens", img + "1. Um.\n2. Dois.", threeMarkers, "docs/user/tasks/a.md:3: a captura tem 3 marcadores e a lista seguinte tem 2 itens"},
		{"sem lista", img + "Texto.", threeMarkers, "docs/user/tasks/a.md:3: depois da captura com marcadores precisa vir uma lista numerada"},
		{"lista sem número", img + "- Um.\n- Dois.\n- Três.", threeMarkers, "docs/user/tasks/a.md:3: depois da captura com marcadores precisa vir uma lista numerada"},
		{"imagem acompanhada", "Veja ![Tela](../media/tela.png) aqui.\n\n" + threeSteps, threeMarkers, "docs/user/tasks/a.md:3: a captura com marcadores precisa ficar sozinha no parágrafo"},
		{"dentro de citação", "> ![Tela](../media/tela.png)\n\n" + threeSteps, threeMarkers, "docs/user/tasks/a.md:3: a captura com marcadores precisa ficar sozinha no parágrafo"},
		{"largura errada", img + threeSteps, strings.Replace(threeMarkers, `"width": 640`, `"width": 1280`, 1), mk + ": width: 1280 não bate com a captura, que tem 640 px"},
		{"altura errada", img + threeSteps, strings.Replace(threeMarkers, `"height": 400`, `"height": 800`, 1), mk + ": height: 800 não bate com a captura, que tem 400 px"},
		{"n fora de ordem", img + threeSteps, strings.Replace(threeMarkers, `"n": 2`, `"n": 3`, 1), mk + ": markers[1].n: esperava 2"},
		{"x fora da faixa", img + threeSteps, strings.Replace(threeMarkers, `"x": 0.125`, `"x": 1.5`, 1), mk + ": markers[0].x: precisa ficar entre 0 e 1"},
		{"y ausente", img + threeSteps, strings.Replace(threeMarkers, `"y": 0.2, "text": "Campo Cliente"`, `"text": "Campo Cliente"`, 1), mk + ": markers[0].y: obrigatório"},
		{"texto vazio", img + threeSteps, strings.Replace(threeMarkers, `"Campo Cliente"`, `""`, 1), mk + ": markers[0].text: obrigatório"},
		{"texto longo", img + threeSteps, strings.Replace(threeMarkers, `"Campo Cliente"`, `"`+strings.Repeat("á", 81)+`"`, 1), mk + ": markers[0].text: use até 80 caracteres"},
		{"chave desconhecida", img + threeSteps, strings.Replace(threeMarkers, `"version": 1,`, `"version": 1, "zoom": 2,`, 1), mk + ": zoom: chave desconhecida"},
		{"versão", img + threeSteps, strings.Replace(threeMarkers, `"version": 1`, `"version": 2`, 1), mk + ": version: só a versão 1 é aceita"},
		{"nome da imagem", img + threeSteps, strings.Replace(threeMarkers, `"tela.png"`, `"outra.png"`, 1), mk + `: image: esperava "tela.png"`},
		{"sem marcadores", img + threeSteps, `{"version": 1, "image": "tela.png", "width": 640, "height": 400, "markers": []}`, mk + ": markers: liste pelo menos um marcador"},
		{"JSON inválido", img + threeSteps, `{"version": 1,`, mk + ": JSON inválido"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := shotProject(t, tt.body, map[string]string{mk: tt.markers})
			for _, p := range problems {
				if strings.HasPrefix(p, tt.want) {
					return
				}
			}
			t.Errorf("esperava %q; veio:\n%s", tt.want, strings.Join(problems, "\n"))
		})
	}
}

func TestMarkersNeedPNGOrJPEG(t *testing.T) {
	_, problems := shotProject(t, "![Tela](../media/tela.gif)\n\n"+threeSteps, map[string]string{
		"docs/user/media/tela.gif":          picture(t, "gif", 640, 400),
		"docs/user/media/tela.markers.json": strings.Replace(threeMarkers, `"tela.png"`, `"tela.gif"`, 1),
	})
	want := "docs/user/media/tela.markers.json: a captura com marcadores precisa ser PNG ou JPEG"
	for _, p := range problems {
		if strings.HasPrefix(p, want) {
			return
		}
	}
	t.Errorf("esperava %q; veio: %v", want, problems)
}

// A ligação com a lista e o JSON são conferidos juntos: todos os
// problemas saem de uma vez.
func TestMarkerProblemsAllAtOnce(t *testing.T) {
	bad := strings.Replace(strings.Replace(threeMarkers, `"width": 640`, `"width": 1280`, 1), `"x": 0.125`, `"x": 2`, 1)
	_, problems := shotProject(t, "![Tela](../media/tela.png)\n\nTexto sem lista.", map[string]string{"docs/user/media/tela.markers.json": bad})
	got := strings.Join(problems, "\n")
	for _, want := range []string{
		"docs/user/tasks/a.md:3: depois da captura com marcadores precisa vir uma lista numerada",
		"docs/user/media/tela.markers.json: width: 1280 não bate com a captura, que tem 640 px",
		"docs/user/media/tela.markers.json: markers[0].x: precisa ficar entre 0 e 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("esperava %q; veio:\n%s", want, got)
		}
	}
}

func TestMarkerLinkAndInvalidJSON(t *testing.T) {
	_, problems := shotProject(t, "![Tela](../media/tela.png)\n\nTexto sem lista.", map[string]string{"docs/user/media/tela.markers.json": `{"version": 1,`})
	got := strings.Join(problems, "\n")
	for _, want := range []string{
		"docs/user/tasks/a.md:3: depois da captura com marcadores precisa vir uma lista numerada",
		"docs/user/media/tela.markers.json: JSON inválido",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("esperava %q; veio:\n%s", want, got)
		}
	}
}
