package content

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const videoConfig = `version: 1
product:
  name: P
user:
  dir: docs/user
  videos:
    manifest: docs/user/media/videos/manifest.json
    tasks: [tasks/a]
`

const videoManifest = `{
  "version": 1,
  "videos": [
    {"task": "tasks/a", "file": "a.webm", "duration_seconds": 94.2, "poster": "a.png"},
    {"task": "tasks/fora-da-lista", "file": "nao-existe.webm", "duration_seconds": 10}
  ]
}`

func videoProject(t *testing.T, manifest string, extra map[string]string) (*Guide, []string) {
	t.Helper()
	files := map[string]string{
		"docs/user/tasks/a.md":                 "# A\n\nTexto.\n",
		"docs/user/tasks/b.md":                 "# B\n",
		"docs/user/media/videos/a.webm":        "webm",
		"docs/user/media/videos/a.png":         picture(t, "png", 32, 18),
		"docs/user/media/videos/manifest.json": manifest,
	}
	for k, v := range extra {
		files[k] = v
	}
	cfg := projectFiles(t, videoConfig, files)
	g, problems := Load(cfg)
	var out []string
	for _, p := range problems.Sorted() {
		out = append(out, strings.TrimPrefix(filepath.ToSlash(p.String()), filepath.ToSlash(cfg.Dir)+"/"))
	}
	return g, out
}

func TestVideo(t *testing.T) {
	g, problems := videoProject(t, videoManifest, nil)
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados: %v", problems)
	}
	a := page(t, g, "tasks/a")
	want := &Video{Src: "media/docs/user/media/videos/a.webm", Poster: "media/docs/user/media/videos/a.png", Minutes: 2}
	if !reflect.DeepEqual(a.Video, want) {
		t.Errorf("vídeo = %+v; esperava %+v", a.Video, want)
	}
	link := `<p class="gb-video"><a class="gb-video-link" href="media/docs/user/media/videos/a.webm" data-poster="media/docs/user/media/videos/a.png">Ver em vídeo (2 min)</a></p>`
	if !strings.Contains(a.HTML, `<h1 id="/tasks/a/a">A</h1>`+"\n"+link) {
		t.Errorf("o link do vídeo deveria vir logo depois do título:\n%s", a.HTML)
	}
	if page(t, g, "tasks/b").Video != nil {
		t.Errorf("tarefa fora da lista não tem vídeo")
	}
	for _, m := range []string{"docs/user/media/videos/a.png", "docs/user/media/videos/a.webm"} {
		found := false
		for _, got := range g.Media {
			found = found || got == m
		}
		if !found {
			t.Errorf("mídia sem %s: %v", m, g.Media)
		}
	}
}

func TestVideoMinutes(t *testing.T) {
	for seconds, want := range map[string]int{"20": 1, "60": 1, "120": 2, "121": 3, "0.5": 1} {
		m := strings.Replace(videoManifest, `"duration_seconds": 94.2`, `"duration_seconds": `+seconds, 1)
		g, problems := videoProject(t, m, nil)
		if len(problems) > 0 {
			t.Fatal(problems)
		}
		v := page(t, g, "tasks/a").Video
		if v == nil {
			t.Fatalf("%s s: tarefa sem vídeo", seconds)
		}
		if got := v.Minutes; got != want {
			t.Errorf("%s s = %d min; esperava %d", seconds, got, want)
		}
	}
}

func TestVideoWithoutPoster(t *testing.T) {
	g, problems := videoProject(t, strings.Replace(videoManifest, `, "poster": "a.png"`, "", 1), nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if html := page(t, g, "tasks/a").HTML; !strings.Contains(html, `<a class="gb-video-link" href="media/docs/user/media/videos/a.webm">`) {
		t.Errorf("sem poster, o link não leva data-poster:\n%s", html)
	}
}

func TestVideoProblems(t *testing.T) {
	const mf = "docs/user/media/videos/manifest.json"
	tests := []struct {
		name, manifest, want string
	}{
		{"JSON inválido", `{"version": 1,`, mf + ": JSON inválido"},
		{"chave desconhecida", strings.Replace(videoManifest, `"version": 1,`, `"version": 1, "fps": 30,`, 1), mf + ": fps: chave desconhecida"},
		{"versão", strings.Replace(videoManifest, `"version": 1`, `"version": 2`, 1), mf + ": version: só a versão 1 é aceita"},
		{"tarefa vazia", strings.Replace(videoManifest, `"task": "tasks/a"`, `"task": ""`, 1), mf + ": videos[0].task: obrigatório"},
		{"tarefa repetida", strings.Replace(videoManifest, `"tasks/fora-da-lista"`, `"tasks/a"`, 1), mf + `: videos[1].task: "tasks/a" repetida`},
		{"extensão", strings.Replace(videoManifest, `"a.webm"`, `"a.avi"`, 1), mf + ": videos[0].file: use um arquivo .webm ou .mp4"},
		{"caminho no nome", strings.Replace(videoManifest, `"a.webm"`, `"../a.webm"`, 1), mf + ": videos[0].file: use só o nome do arquivo, no diretório do manifesto"},
		{"arquivo inexistente", strings.Replace(videoManifest, `"a.webm"`, `"b.webm"`, 1), mf + `: videos[0].file: "b.webm" não existe`},
		{"duração zero", strings.Replace(videoManifest, `94.2`, `0`, 1), mf + ": videos[0].duration_seconds: precisa ser maior que 0"},
		{"duração ausente", strings.Replace(videoManifest, `, "duration_seconds": 94.2`, ``, 1), mf + ": videos[0].duration_seconds: obrigatório"},
		{"poster com formato errado", strings.Replace(videoManifest, `"a.png"`, `"a.gif"`, 1), mf + ": videos[0].poster: use PNG ou JPEG"},
		{"poster inexistente", strings.Replace(videoManifest, `"a.png"`, `"b.png"`, 1), mf + `: videos[0].poster: "b.png" não existe`},
		{"tarefa fora do manifesto", strings.Replace(videoManifest, `"task": "tasks/a"`, `"task": "tasks/b"`, 1), `guidebook.yaml: user.videos.tasks[0]: "tasks/a" não está no manifesto de vídeos`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := videoProject(t, tt.manifest, nil)
			for _, p := range problems {
				if strings.HasPrefix(p, tt.want) {
					return
				}
			}
			t.Errorf("esperava %q; veio:\n%s", tt.want, strings.Join(problems, "\n"))
		})
	}
}
