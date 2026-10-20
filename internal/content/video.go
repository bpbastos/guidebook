package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
)

type videoManifestFile struct {
	Version *int         `json:"version"`
	Videos  []videoEntry `json:"videos"`
}

type videoEntry struct {
	Task            string   `json:"task"`
	File            string   `json:"file"`
	DurationSeconds *float64 `json:"duration_seconds"`
	Poster          string   `json:"poster"`
}

// videos lê o manifesto de vídeos (docs/spec.md, seção 7) e liga cada
// tarefa de user.videos.tasks ao vídeo dela. Entrada do manifesto fora
// da lista tem só o formato conferido: os arquivos dela podem faltar.
func (l *loader) videos(byID map[string]*Page) {
	u := l.c.User
	if u == nil || u.Videos.Manifest == "" {
		return
	}
	file := l.abs(u.Videos.Manifest)
	dir := path.Dir(u.Videos.Manifest)
	bad := func(field, format string, args ...any) {
		l.problems.Addf(file, 0, field, format, args...)
	}
	body, err := os.ReadFile(file)
	if err != nil {
		bad("", "não foi possível ler o arquivo: %v", err)
		return
	}
	var mf videoManifestFile
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&mf); err != nil {
		if name, found := strings.CutPrefix(err.Error(), "json: unknown field "); found {
			bad(strings.Trim(name, `"`), "chave desconhecida")
		} else {
			bad("", "JSON inválido (%v)", err)
		}
		return
	}
	switch {
	case mf.Version == nil:
		bad("version", "obrigatório")
	case *mf.Version != 1:
		bad("version", "só a versão 1 é aceita")
	}

	listed := map[string]bool{}
	for _, t := range u.Videos.Tasks {
		listed[t] = true
	}
	inManifest := map[string]bool{}
	for i, v := range mf.Videos {
		f := fmt.Sprintf("videos[%d]", i)
		before := len(l.problems)
		switch {
		case v.Task == "":
			bad(f+".task", "obrigatório")
		case inManifest[v.Task]:
			bad(f+".task", "%q repetida; cada tarefa tem um vídeo só", v.Task)
		}
		inManifest[v.Task] = true

		checkName := func(field, name string, exts []string, extMsg string) {
			switch {
			case name == "":
				bad(field, "obrigatório")
			case strings.ContainsAny(name, `/\`) || name == "." || name == "..":
				bad(field, "use só o nome do arquivo, no diretório do manifesto")
			case !hasExt(name, exts):
				bad(field, "%s", extMsg)
			case listed[v.Task]:
				if info, err := os.Stat(l.abs(path.Join(dir, name))); err != nil || info.IsDir() {
					bad(field, "%q não existe", name)
				}
			}
		}
		checkName(f+".file", v.File, []string{".webm", ".mp4"}, "use um arquivo .webm ou .mp4")
		if v.Poster != "" {
			checkName(f+".poster", v.Poster, []string{".png", ".jpg", ".jpeg"}, "use PNG ou JPEG")
		}
		switch {
		case v.DurationSeconds == nil:
			bad(f+".duration_seconds", "obrigatório")
		case *v.DurationSeconds <= 0:
			bad(f+".duration_seconds", "precisa ser maior que 0")
		}
		if !listed[v.Task] || len(l.problems) > before {
			continue
		}

		video := &Video{
			Src:     "media/" + path.Join(dir, v.File),
			Minutes: max(1, int(math.Ceil(*v.DurationSeconds/60))),
		}
		l.media[path.Join(dir, v.File)] = true
		if v.Poster != "" {
			video.Poster = "media/" + path.Join(dir, v.Poster)
			l.media[path.Join(dir, v.Poster)] = true
		}
		if p := byID[v.Task]; p != nil {
			p.Video = video
		}
	}

	for i, t := range u.Videos.Tasks {
		if !inManifest[t] {
			l.problems.Addf(l.c.File, 0, fmt.Sprintf("user.videos.tasks[%d]", i), "%q não está no manifesto de vídeos", t)
		}
	}
}

func hasExt(name string, exts []string) bool {
	ext := strings.ToLower(path.Ext(name))
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

// renderVideoLink escreve o link "Ver em vídeo". Sem JavaScript, ele
// abre o arquivo no navegador; com JavaScript, abre o vídeo em diálogo.
func (l *loader) renderVideoLink(b *bytes.Buffer, v *Video) {
	label := strings.ReplaceAll(l.c.UIText("watch_video"), "{min}", strconv.Itoa(v.Minutes))
	poster := ""
	if v.Poster != "" {
		poster = fmt.Sprintf(` data-poster="%s"`, html.EscapeString(v.Poster))
	}
	fmt.Fprintf(b, "<p class=\"gb-video\"><a class=\"gb-video-link\" href=\"%s\"%s>%s</a></p>\n",
		html.EscapeString(v.Src), poster, html.EscapeString(label))
}
