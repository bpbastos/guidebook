// Package diag reúne os problemas encontrados pelo guidebook e os
// formata como o usuário vê na saída de erro.
package diag

import (
	"fmt"
	"sort"
	"strings"
)

// Problem é um erro de configuração ou de conteúdo.
//
// Com Line > 0, sai como "arquivo:linha: mensagem"; com Field, como
// "arquivo: campo: mensagem"; sem os dois, como "arquivo: mensagem".
type Problem struct {
	File  string
	Line  int
	Field string
	Msg   string
}

func (p Problem) String() string {
	switch {
	case p.Line > 0:
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Msg)
	case p.Field != "":
		return fmt.Sprintf("%s: %s: %s", p.File, p.Field, p.Msg)
	default:
		return fmt.Sprintf("%s: %s", p.File, p.Msg)
	}
}

// List acumula problemas para que todos saiam de uma vez.
type List []Problem

// Addf acrescenta um problema com mensagem formatada.
func (l *List) Addf(file string, line int, field, format string, args ...any) {
	*l = append(*l, Problem{File: file, Line: line, Field: field, Msg: fmt.Sprintf(format, args...)})
}

// Sorted devolve uma cópia ordenada por arquivo e linha; no mesmo
// arquivo e linha, mantém a ordem em que os problemas foram achados.
func (l List) Sorted() List {
	out := append(List(nil), l...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return out
}

// String devolve um problema por linha, já ordenados.
func (l List) String() string {
	var b strings.Builder
	for _, p := range l.Sorted() {
		b.WriteString(p.String())
		b.WriteByte('\n')
	}
	return b.String()
}
