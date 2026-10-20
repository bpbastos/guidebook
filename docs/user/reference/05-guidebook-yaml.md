# guidebook.yaml

O `guidebook.yaml` fica na raiz do projeto. Todo caminho dentro dele é
relativo a essa raiz e não pode sair dela.

## Exemplo

```yaml
version: 1

product:
  name: Pedido Fácil
  logo: docs/assets/logo.svg
  colors:
    primary: "#1f6feb"
    accent: "#bf3989"

rules:
  max_sentence_words: 25
  forbidden_patterns:
    - pattern: '\b[A-Z]{2,}-\d+\b'
      message: não cite chamado interno no guia

user:
  dir: docs/user
  changelog: CHANGELOG.md
  blocked_prefix: O sistema não me deixa
  profiles:
    - id: attendant
      name: Atendente
      start_here:
        - tutorials/first-order
        - tasks/register-order
  videos:
    manifest: docs/user/media/videos/manifest.json
    tasks:
      - tasks/register-order

developer:
  repository_url: https://github.com/example/pedido-facil/blob/main
  sections:
    overview:
      sources: [docs/developer/overview.md]
    decisions:
      sources: [docs/adr/*.md]
    development:
      sources: [docs/developer/development.md]
    changes:
      sources: [CHANGELOG.md]

ui:
  start_here: Comece por aqui
```

## Campos principais

| Campo | Para quê |
|---|---|
| `version` | sempre `1` |
| `product.name` | nome no alto dos guias |
| `product.logo` | logo em SVG ou PNG |
| `product.colors` | cores `#rrggbb`; o texto sobre elas escolhe o contraste sozinho |
| `rules.max_sentence_words` | limite de palavras por frase; o padrão é 25 |
| `rules.forbidden_patterns` | textos que o guia não pode citar, com a mensagem da recusa |
| `user.dir` | pasta das páginas do guia do usuário |
| `user.changelog` | CHANGELOG do "O que há de novo" |
| `user.blocked_prefix` | começo dos títulos do índice de bloqueios |
| `user.profiles` | perfis de "Comece por aqui", cada um com a trilha dele |
| `user.videos` | manifesto e lista das tarefas com vídeo |
| `user.max_size_mib` | teto opcional do guia do usuário, em MiB, inteiro maior que 0 |
| `developer.repository_url` | endereço para links de arquivos que não são fonte do guia |
| `developer.sections` | fontes de cada seção do guia do desenvolvedor |
| `developer.max_size_mib` | teto opcional do guia do desenvolvedor, em MiB, inteiro maior que 0 |
| `ui` | troca de qualquer texto da interface |

Os dois guias são opcionais, mas pelo menos um precisa existir. Um
guia pode ter regras próprias em `user.rules` ou `developer.rules`. O
limite do guia substitui o geral, e os padrões se somam.

## O guidebook não me deixa usar a configuração

O `check` diz o campo, como `user.profiles[0].id: obrigatório`. Chave
escrita errado também é recusada, com o nome dela.

Veja as seções do guia do desenvolvedor em
[Guia do desenvolvedor](04-developer-guide.md).
