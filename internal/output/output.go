// Package output prepara o diretório de saída do build (docs/spec.md,
// seção 2): só apaga diretório vazio ou já marcado como saída do
// guidebook.
package output

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Marker é o arquivo que identifica um diretório gravado pelo build.
const Marker = ".guidebook-output"

const markerText = "Diretório gravado pelo guidebook. O build apaga e regrava tudo aqui.\n"

// Prepare deixa dir vazio, só com o marcador, pronto para o build.
// Recusa diretório com conteúdo que não tenha o marcador.
func Prepare(dir string) error {
	entries, err := os.ReadDir(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("não foi possível criar %s: %v", dir, err)
		}
	case err != nil:
		if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
			return fmt.Errorf("%s não é um diretório", dir)
		}
		return fmt.Errorf("não foi possível ler %s: %v", dir, err)
	case len(entries) > 0:
		if _, err := os.Stat(filepath.Join(dir, Marker)); err != nil {
			return fmt.Errorf("%s tem conteúdo e não tem o arquivo %s; o build só apaga diretório vazio ou gravado por ele (escolha outro -out)", dir, Marker)
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("não foi possível limpar %s: %v", dir, err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dir, Marker), []byte(markerText), 0o644); err != nil {
		return fmt.Errorf("não foi possível gravar %s: %v", Marker, err)
	}
	return nil
}
