# Visão geral

O guidebook é uma linha de comando em Go. Ele lê o Markdown de um
projeto e grava dois guias em HTML, que abrem sem servidor e sem
internet. A especificação completa está em [docs/spec.md](../spec.md).

## Contexto

```mermaid
C4Context
  title Contexto do guidebook
  UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
  Person(maintainer, "Mantenedor do projeto", "Escreve a documentação em Markdown")
  System_Ext(e2e, "e2e do projeto", "Grava capturas, marcadores e vídeos")
  Person(reader, "Leitor do guia", "Usa ou mantém o sistema")
  System_Ext(repo, "Repositório do projeto", "Markdown, guidebook.yaml e CHANGELOG")
  System(guidebook, "guidebook", "Gera guias em HTML a partir do Markdown")
  Rel(maintainer, repo, "Escreve")
  Rel(e2e, repo, "Grava a mídia")
  Rel(guidebook, repo, "Lê")
  Rel(reader, guidebook, "Lê os guias gerados")
```

## Containers

```mermaid
C4Container
  title Containers do guidebook
  Person(maintainer, "Mantenedor do projeto")
  Person(reader, "Leitor do guia")
  System_Ext(repo, "Repositório do projeto", "Markdown e mídia")
  System_Boundary(gb, "guidebook") {
    Container(cli, "CLI guidebook", "Go", "build, check e serve (só o guia do usuário)")
    Container(out, "Guias gerados", "HTML, CSS e JavaScript", "user-guide.html e developer-guide.html")
  }
  Rel(maintainer, cli, "Roda")
  Rel(cli, repo, "Lê")
  Rel(cli, out, "Grava")
  Rel(reader, out, "Abre no navegador")
```

## Pacotes

O código fica em `cmd/` e `internal/`:

| Pacote | Responsabilidade |
|---|---|
| `cmd/guidebook` | linha de comando: `build`, `check` e `version` |
| `internal/config` | leitura e validação do `guidebook.yaml` |
| `internal/content` | páginas, links, imagens, marcadores, vídeos e fontes do guia do desenvolvedor |
| `internal/rules` | regras de escrita |
| `internal/changelog` | leitura do CHANGELOG |
| `internal/markdown` | configuração do goldmark e utilitários de posição |
| `internal/site` | gravação dos guias, recursos embutidos e bibliotecas vendorizadas |
| `internal/output` | proteção do diretório de saída |
| `internal/diag` | formato dos problemas |

## Fluxo do build

1. O `config` lê e valida o `guidebook.yaml`.
2. O `content` lê as páginas e confere títulos, links e mídia.
3. O `rules` aplica as regras de escrita a todo Markdown lido.
4. Sem problemas, o `output` prepara a pasta e o `site` grava os guias.

Qualquer problema para o build antes de gravar. Todos os problemas saem
de uma vez, com arquivo e linha.
