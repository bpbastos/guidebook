# Integrar à CI

A CI do projeto pode conferir a documentação e entregar os guias
prontos a cada execução. Quem revisa baixa os guias e abre direto do
disco.

## Gere as mídias no mesmo job

As capturas, os marcadores e os vídeos vêm do e2e e ficam fora do git.
Por isso o `check` e o `build` rodam no mesmo job do e2e, depois dele.
Em outro job, as mídias não existem, e o `check` recusa as imagens que
faltam.

Para conferir a escrita antes do e2e, num job sem navegador, use
`guidebook check -no-media`. Ele não lê as mídias; o `check` completo
continua depois do e2e.

Ponha a pasta das mídias no `.gitignore` do projeto, para não
versionar o que o e2e gera:

```gitignore
docs/user/media/
```

## Monte o job

Um exemplo no GitHub Actions, com cada action fixada por SHA:

```yaml
guides:
  runs-on: ubuntu-24.04
  steps:
    - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      with:
        persist-credentials: false

    - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
      with:
        go-version: "1.27.2"

    - name: instalar o guidebook
      run: go install github.com/bpbastos/guidebook/cmd/guidebook@v0.1.1

    # O e2e do projeto grava capturas, marcadores e vídeos em docs/user/media/.
    - name: e2e
      run: make e2e

    # Versão: a tag, ou o SHA curto fora de tag. Data: a do commit.
    - name: versão e data
      run: |
        echo "GUIDE_VERSION=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)" >> "$GITHUB_ENV"
        echo "GUIDE_DATE=$(git log -1 --format=%cs)" >> "$GITHUB_ENV"

    - name: check
      run: guidebook check -product-version "$GUIDE_VERSION" -date "$GUIDE_DATE"

    - name: build
      run: guidebook build -out out/guides -product-version "$GUIDE_VERSION" -date "$GUIDE_DATE"

    - uses: actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9 # v7.0.2
      with:
        name: guias
        path: out/guides
        if-no-files-found: error
```

Troque `make e2e` pelo comando do e2e do projeto. Num push de tag, o
`actions/checkout` traz a própria tag, e a versão sai como ela, por
exemplo `v1.4.0`. Fora disso, nenhuma tag vem, e a versão sai como o
SHA curto do commit. A versão e a data
aparecem no rodapé dos dois guias. A data vem do commit, não do relógio
da máquina: o mesmo commit gera os mesmos guias. O `check` recebe as
mesmas opções para informar o mesmo tamanho do `build`. Em outra forja, os
passos são os mesmos: instalar, rodar o e2e, `check`, `build` e guardar
`out/guides` como artefato.

A CI do próprio guidebook faz o mesmo com os guias dele.

## Acompanhe o tamanho

O `check` e o `build` informam o tamanho de cada guia na saída, com
HTML, recursos e mídia:

```
build: guia do usuário com 9.876.543 bytes (9,4 MiB)
```

Para a CI recusar um guia grande demais, defina um teto em MiB no
`guidebook.yaml`:

```yaml
user:
  dir: docs/user
  max_size_mib: 20
```

Acima do teto, o `check` e o `build` saem com código 1 e dizem o
tamanho e o limite. Veja os campos em
[guidebook.yaml](../reference/05-guidebook-yaml.md).
