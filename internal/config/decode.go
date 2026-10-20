package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// decoder converte a árvore de nós YAML na struct de destino, campo a
// campo, e anota cada problema com o caminho do campo. O yaml.v3 também
// decodifica de forma estrita (KnownFields), mas para no primeiro erro
// e responde em inglês, sem caminho.
type decoder struct {
	report func(field, format string, args ...any)
}

func (d *decoder) decode(n *yaml.Node, v reflect.Value, path string) {
	for n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return // campo vazio vale como ausente
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		d.decode(n, v.Elem(), path)

	case reflect.Struct:
		if n.Kind != yaml.MappingNode {
			d.report(path, "esperava um objeto (chave: valor)")
			return
		}
		fields := map[string]reflect.Value{}
		for i := 0; i < v.NumField(); i++ {
			if tag := v.Type().Field(i).Tag.Get("yaml"); tag != "" && tag != "-" {
				fields[tag] = v.Field(i)
			}
		}
		d.eachKey(n, path, func(key string, value *yaml.Node, p string) {
			f, ok := fields[key]
			if !ok {
				d.report(p, "chave desconhecida")
				return
			}
			d.decode(value, f, p)
		})

	case reflect.Map:
		if n.Kind != yaml.MappingNode {
			d.report(path, "esperava um objeto (chave: valor)")
			return
		}
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		d.eachKey(n, path, func(key string, value *yaml.Node, p string) {
			elem := reflect.New(v.Type().Elem()).Elem()
			d.decode(value, elem, p)
			v.SetMapIndex(reflect.ValueOf(key), elem)
		})

	case reflect.Slice:
		if n.Kind != yaml.SequenceNode {
			d.report(path, "esperava uma lista")
			return
		}
		s := reflect.MakeSlice(v.Type(), len(n.Content), len(n.Content))
		for i, item := range n.Content {
			d.decode(item, s.Index(i), fmt.Sprintf("%s[%d]", path, i))
		}
		v.Set(s)

	case reflect.String:
		if n.Kind != yaml.ScalarNode {
			d.report(path, "esperava um texto")
			return
		}
		v.SetString(n.Value)

	case reflect.Int:
		i, err := strconv.Atoi(n.Value)
		if n.Kind != yaml.ScalarNode || n.Tag != "!!int" || err != nil {
			d.report(path, "esperava um número inteiro")
			return
		}
		v.SetInt(int64(i))

	case reflect.Bool:
		if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" {
			d.report(path, "esperava um booleano (true ou false)")
			return
		}
		v.SetBool(strings.EqualFold(n.Value, "true"))

	default:
		panic("config: tipo sem decodificação: " + v.Type().String())
	}
}

// eachKey percorre as chaves de um objeto YAML, recusando chave que não
// seja texto e chave repetida.
func (d *decoder) eachKey(n *yaml.Node, path string, fn func(key string, value *yaml.Node, p string)) {
	seen := map[string]bool{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, value := n.Content[i], n.Content[i+1]
		p := k.Value
		if path != "" {
			p = path + "." + k.Value
		}
		switch {
		case k.Kind != yaml.ScalarNode || strings.TrimSpace(k.Value) == "":
			d.report(path, "chave inválida na linha %d", k.Line)
		case seen[k.Value]:
			d.report(p, "chave repetida (linha %d)", k.Line)
		default:
			seen[k.Value] = true
			fn(k.Value, value, p)
		}
	}
}
