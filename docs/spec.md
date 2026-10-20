# Especificação do guidebook

Versão da especificação: 1 (fase 1). Mudanças passam por pull request e
entram no [CHANGELOG.md](../CHANGELOG.md). Decisões com alternativa real
ficam em [docs/adr/](adr/).

## 1. Visão geral

O guidebook lê a documentação em Markdown de um projeto e gera dois
guias em HTML:

- **guia do usuário**: para quem usa o sistema. Tem modo leitura e modo
  apresentação, capturas com marcadores, vídeo por tarefa e busca. Tem
  também glossário, "Comece por aqui", "O sistema não me deixa..." e "O
  que há de novo";
- **guia do desenvolvedor**: para quem mantém o sistema; mesmas bases,
  sem modo apresentação, com seções montadas a partir de fontes
  configuráveis e diagramas em Mermaid.

Os guias abrem com dois cliques no navegador (`file://`): sem servidor,
sem internet.

## 2. Linha de comando

```
guidebook build [-config guidebook.yaml] [-out out] [-guide all|user|developer]
guidebook check [-config guidebook.yaml] [-guide all|user|developer]
guidebook serve [-config guidebook.yaml] [-addr 127.0.0.1:8080] [-interval 2s]
guidebook version
```

- `build` valida configuração e conteúdo, aplica as regras de escrita e
  grava os guias em `-out`.
- `check` faz as mesmas validações sem gravar nada (para o CI do
  projeto).
- `serve` gera e serve só o guia do usuário por HTTP, regerando
  quando a entrada muda (seção 14). O guia do desenvolvedor nunca é
  servido; `-guide` não existe no `serve`.
- `-config` padrão: `guidebook.yaml` no diretório atual. Todo caminho
  dentro do arquivo é relativo ao diretório dele.
- `-out` padrão: `out/`, relativo ao diretório atual.
- Só o pacote `flag` da biblioteca padrão.

Códigos de saída:

| Código | Quando |
|---|---|
| 0 | sucesso |
| 1 | violação de regra de escrita ou erro de conteúdo (link quebrado, marcador sem item, página inexistente citada na configuração) |
| 2 | uso errado da linha de comando, configuração inválida ou erro de E/S |

Os erros vão para a saída de erro, um por linha, ordenados por arquivo
e linha, todos de uma vez:

```
docs/user/tasks/register-order.md:12: frase com 31 palavras (limite: 25): "Quando o atendente abre a tela de pedidos…"
docs/user/tasks/register-order.md:20: HTML cru não é permitido no Markdown
guidebook.yaml: user.profiles[1].start_here[0]: página "tasks/x" não existe
```

Segurança da saída: o `build` só apaga e regrava `-out` se o diretório
não existir, estiver vazio ou contiver o arquivo `.guidebook-output`
(gravado pelo próprio build). Caso contrário, recusa com código 2. Isso
impede que `-out .` apague o projeto.

## 3. Entrada

### 3.1 Estrutura de diretórios

```
<projeto>/
├── guidebook.yaml
├── CHANGELOG.md                 # "O que há de novo" (Keep a Changelog)
└── docs/
    ├── user/                    # user.dir
    │   ├── index.md             # opcional: texto da página inicial
    │   ├── glossary.md          # opcional: glossário
    │   ├── tutorials/           # Diátaxis: tutorial
    │   ├── tasks/               # Diátaxis: tarefa (how-to)
    │   ├── reference/           # Diátaxis: referência
    │   ├── explanation/         # Diátaxis: explicação
    │   └── media/               # capturas, marcadores e vídeos (e2e)
    └── developer/               # fontes do guia do desenvolvedor
```

Regras:

- Entradas que começam com `.` (como `.DS_Store`) são ignoradas em
  todo o `user.dir`.
- Os quatro diretórios Diátaxis têm nome fixo; o rótulo exibido vem dos
  textos da interface (seção 4.6). Qualquer um pode faltar.
- Só arquivos `.md` diretamente dentro deles (sem subdiretório). Outro
  arquivo ou subdiretório ali é erro de conteúdo.
- Na raiz de `user.dir`, além dos quatro diretórios e de `media/`, só
  `index.md` e `glossary.md`.
- **Identificador da página**: caminho relativo a `user.dir`, sem
  `.md`, com prefixo numérico `NN-` removido do nome do arquivo.
  `tasks/02-register-order.md` vira `tasks/register-order`.
  Identificador repetido depois de remover o prefixo é erro.
- **Ordem**: nome do arquivo, em ordem de bytes. O prefixo `NN-` serve
  para ordenar.
- **Título**: o primeiro título de nível 1 (`# `). Página sem ele é
  erro. Um segundo `# ` na mesma página também é erro.

### 3.2 Markdown

CommonMark puro, renderizado com goldmark sem `WithUnsafe`, mais as
extensões de tabela e tachado (GFM). Convenções além do CommonMark, e
só estas:

| Convenção | Onde | Efeito |
|---|---|---|
| bloco de código ` ```mermaid ` | guia do desenvolvedor | vira diagrama |
| `## ` que começa com `user.blocked_prefix` | guia do usuário | entra no índice "O sistema não me deixa..." |
| imagem com `<nome>.markers.json` ao lado | guia do usuário | captura com marcadores (seção 6) |
| `## ` e `---` | guia do usuário | limite de slide no modo apresentação |

As âncoras dos títulos são geradas do texto: minúsculas, acentos
removidos, o que não for letra ou dígito vira `-`, hífens repetidos
colapsados. Âncora repetida na mesma página ganha `-2`, `-3`...

### 3.3 Links

Todo link relativo é conferido no build. Link quebrado (arquivo ou
âncora inexistente) é erro de conteúdo com `arquivo:linha`.

No guia do usuário, o tipo do destino decide o comportamento:

| Destino | Comportamento |
|---|---|
| `glossary.md#termo` | abre o termo em `<dialog>`, sem sair do slide ou da posição de leitura |
| página de `reference/` ou `explanation/` ("veja também") | abre a página em `<dialog>` |
| título "O sistema não me deixa..." de qualquer página ("bloqueio") | abre aquela seção em `<dialog>` |
| página ou âncora de `tutorials/` ou `tasks/` | navega até lá e guarda o ponto de origem; aparece "Voltar para onde eu estava" |
| `http:`, `https:`, `mailto:` | link comum, abre em nova aba com `rel="noopener noreferrer"` |
| qualquer outro arquivo do projeto | erro de conteúdo |
| outro esquema (`javascript:`, `data:`, `vbscript:`, `file:`...) | erro de conteúdo |
| sem esquema (`//host/...`) ou começando com `/` | erro de conteúdo |

Imagem segue as mesmas regras de caminho e precisa existir. Imagem
externa (`http:`, `https:`) é erro, porque o guia abre sem internet. A
imagem é copiada para `media/`, com o caminho relativo ao
`guidebook.yaml`: `docs/user/media/a.png` vira `media/docs/user/media/a.png`.

"Voltar para onde eu estava" guarda página, posição de rolagem e, no
modo apresentação, o slide. É uma pilha: voltar desfaz uma navegação
por vez. Vive na memória da página e no histórico (`location.hash`),
não em armazenamento do navegador.

No guia do desenvolvedor, links entre fontes do guia viram âncoras
internas. Link relativo para arquivo do repositório que não é fonte do
guia vira link para `developer.repository_url` (se configurado,
`<repository_url>/<caminho>`); sem essa chave, é erro de conteúdo.

### 3.4 Glossário

`glossary.md` tem um `# ` e um `## ` por termo. O texto até o próximo
`## ` é a definição. A âncora do termo segue a regra da seção 3.2. O
glossário também aparece como página própria no índice.

### 3.5 "O sistema não me deixa..."

Toda seção `## ` cujo texto começa com `user.blocked_prefix` (padrão:
`O sistema não me deixa`) entra no índice "O sistema não me deixa...",
ordenado pelo texto do título. Cada entrada leva à seção na página de
origem. O índice só aparece se houver pelo menos uma entrada.

Exemplo, em `tasks/register-order.md`:

```markdown
## O sistema não me deixa salvar o pedido

O botão Salvar fica desligado enquanto falta o cliente. Escolha o
cliente e o botão liga.
```

### 3.6 CHANGELOG

Formato [Keep a Changelog 1.1.0](https://keepachangelog.com/pt-BR/1.1.0/):
cada `## [versão] - AAAA-MM-DD` é um marco; os `### ` de categoria são
aceitos em inglês (`Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`,
`Security`) ou em pt-BR (`Adicionado`, `Modificado`, `Obsoleto`,
`Removido`, `Corrigido`, `Segurança`).

- Guia do usuário ("O que há de novo"): lê `user.changelog`, mostra os
  marcos publicados, do mais novo para o mais antigo; ignora
  `[Unreleased]`/`[Não publicado]`.
- Guia do desenvolvedor ("Novidades técnicas"): lê as fontes da seção
  `changes`; mostra também o marco não publicado.
- Marco sem itens não aparece em nenhum dos dois guias; categoria sem
  itens também não. Sem nenhum marco com itens, a seção inteira some.
- Arquivo fora do formato (marco sem data, categoria desconhecida) é
  erro de conteúdo com `arquivo:linha`.
- O CHANGELOG não é página do guia: link relativo nele é erro. Links
  externos seguem a tabela da seção 3.3.
- As categorias aparecem sempre em pt-BR; `Added` vira `Adicionado`. As
  datas aparecem como DD/MM/AAAA.

## 4. guidebook.yaml

### 4.1 Regras gerais

- YAML decodificado de forma estrita: chave desconhecida, chave
  repetida e tipo errado são erro
  ([ADR 0005](adr/0005-strict-yaml-decoder.md)).
- Erro sempre em pt-BR, com o caminho do campo:
  `guidebook.yaml: user.profiles[0].id: obrigatório`.
- Todos os erros de validação são listados de uma vez (código 2).
- Pelo menos um dos guias (`user` ou `developer`) precisa estar
  configurado.
- Caminhos relativos ao diretório do `guidebook.yaml`. É erro o caminho
  absoluto, o que sai do diretório (`../`, inclusive no meio, como
  `docs/../../x`) e o que sai por link simbólico. A regra vale também
  para os padrões glob de `developer.sections.*.sources` e para cada
  arquivo que eles casam.
- Fonte do guia do desenvolvedor precisa ser `.md`; padrão que não casa
  nenhum arquivo é erro.
- Página citada no `guidebook.yaml` que não existe é erro de conteúdo
  (código 1), porque depende das páginas, não só do arquivo.

### 4.2 Exemplo completo

```yaml
version: 1

product:
  name: Pedido Fácil
  logo: docs/assets/logo.svg      # opcional; .svg ou .png
  colors:                         # opcionais; hexadecimal #rrggbb
    primary: "#1f6feb"
    accent: "#bf3989"

rules:                            # valem para os dois guias
  max_sentence_words: 25
  forbidden_patterns:
    - pattern: '\b[A-Z]{2,}-\d+\b'
      message: não cite chamado interno no guia

user:
  dir: docs/user
  changelog: CHANGELOG.md
  blocked_prefix: O sistema não me deixa
  rules:                          # somam-se às de cima; o limite substitui
    forbidden_patterns:
      - pattern: '\b(src|internal|cmd)/\S+'
        message: o guia do usuário não cita caminho de código
  profiles:
    - id: attendant
      name: Atendente
      start_here:
        - tutorials/first-order
        - tasks/register-order
    - id: manager
      name: Gerente
      start_here:
        - tasks/close-day
  videos:
    manifest: docs/user/media/videos/manifest.json
    tasks:
      - tasks/register-order

developer:
  repository_url: https://github.com/example/pedido-facil/blob/main
  sections:
    overview:
      sources: [docs/developer/overview.md]
    modules:
      sources: [docs/developer/modules/*.md]
    business_rules:
      sources: [docs/developer/rules/*.md]
    data_model:
      sources: [docs/developer/data-model.md]
    decisions:
      title: Decisões de arquitetura   # opcional; substitui o padrão
      sources: [docs/adr/*.md]
    security:
      sources: [docs/developer/security.md]
    delivery:
      sources: [docs/developer/delivery.md]
    development:
      sources: [docs/developer/development.md]
    operations:
      sources: [docs/developer/operations.md]
    out_of_scope:
      sources: [docs/developer/out-of-scope.md]
    changes:
      sources: [CHANGELOG.md]

ui:                               # opcional; substitui textos padrão
  start_here: Comece por aqui
```

### 4.3 Campos

| Campo | Tipo | Obrigatório | Padrão / validação |
|---|---|---|---|
| `version` | inteiro | sim | só `1` |
| `product.name` | texto | sim | não vazio |
| `product.logo` | caminho | não | arquivo existente, `.svg` ou `.png` |
| `product.colors.primary` | cor | não | `#rrggbb`; padrão `#1f6feb` |
| `product.colors.accent` | cor | não | `#rrggbb`; padrão igual a `primary` |
| `rules.max_sentence_words` | inteiro | não | 5 a 200; padrão 25 |
| `rules.forbidden_patterns[]` | lista | não | cada item com `pattern` (RE2, compila) e `message` (não vazia) |
| `user` | objeto | não | sem ele, o guia do usuário não é gerado |
| `user.dir` | caminho | sim, se `user` | diretório existente |
| `user.changelog` | caminho | não | padrão `CHANGELOG.md`; se não existir, sem "O que há de novo" |
| `user.blocked_prefix` | texto | não | padrão `O sistema não me deixa` |
| `user.rules` | objeto | não | mesmo formato de `rules` |
| `user.profiles[]` | lista | não | `id` (kebab-case, único), `name` (não vazio), `start_here` (identificadores de página existentes, sem repetição, pelo menos um) |
| `user.videos.manifest` | caminho | se houver `tasks` | arquivo existente (seção 7) |
| `user.videos.tasks[]` | lista | não | identificadores de `tasks/` existentes; cada um precisa estar no manifesto |
| `developer` | objeto | não | sem ele, o guia do desenvolvedor não é gerado |
| `developer.repository_url` | URL | não | `https://`; ver seção 3.3 |
| `developer.rules` | objeto | não | mesmo formato de `rules` |
| `developer.sections.<chave>` | objeto | não | chaves da seção 8, de `overview` a `changes`; `sources` (lista de caminhos ou padrões glob, pelo menos um arquivo casado) e `title` opcional |
| `ui.<chave>` | texto | não | chaves da seção 4.6; não vazio |

Regras por guia: `forbidden_patterns` do guia somam-se às gerais;
`max_sentence_words` do guia substitui o geral.

### 4.4 Perfis e "Comece por aqui"

Cada perfil tem uma trilha ordenada de páginas. Na página inicial do
guia do usuário, um seletor de perfil mostra a trilha. Sem perfis, a
seção não aparece. O perfil escolhido é lembrado em `localStorage`
quando o navegador permite; sem isso, volta ao primeiro perfil.

### 4.5 Cores e logo

`primary` pinta cabeçalho, links e botões; `accent`, marcadores e
destaques. O guia respeita `prefers-color-scheme` com tema claro e
escuro próprios; as cores do produto valem nos dois. O contraste do
texto sobre `primary` é escolhido no build (branco ou quase preto, o
que der contraste maior). Fonte: pilha de fontes do sistema.

### 4.6 Textos da interface

Todos em pt-BR por padrão; qualquer um pode ser trocado em `ui`.

| Chave | Padrão |
|---|---|
| `start_here` | Comece por aqui |
| `profile` | Perfil |
| `tutorials` | Tutoriais |
| `tasks` | Tarefas |
| `reference` | Referência |
| `explanation` | Explicações |
| `glossary` | Glossário |
| `blocked_index` | O sistema não me deixa... |
| `whats_new` | O que há de novo |
| `search` | Buscar |
| `search_no_results` | Nada encontrado |
| `back_to_where_i_was` | Voltar para onde eu estava |
| `close` | Fechar |
| `present` | Apresentar |
| `exit_presentation` | Sair da apresentação |
| `fullscreen` | Tela cheia |
| `slide_of` | Slide {n} de {total} |
| `slide_list` | Lista de slides |
| `rules_filter` | Filtrar regras |
| `rules_module` | Módulo |
| `rules_all` | Todos os módulos |
| `rules_none` | Nenhuma regra encontrada |
| `watch_video` | Ver em vídeo ({min} min) |
| `has_video` | Tem vídeo |
| `guided_tour` | Visita guiada |
| `next` | Próximo |
| `previous` | Anterior |
| `screen_points` | Pontos desta tela |
| `user_guide` | Guia do usuário |
| `developer_guide` | Guia do desenvolvedor |

Os marcadores `{n}`, `{total}` e `{min}` são obrigatórios nos textos
que os usam; faltar um é erro de configuração.

## 5. Regras de escrita

Recusa mecânica. O build e o `check` listam todas as violações com
`arquivo:linha` e saem com código 1. Valem para todo Markdown lido
pelos dois guias, incluindo CHANGELOG e glossário.

1. **Frase longa**: frase com mais palavras que `max_sentence_words`
   (padrão 25; ver [ADR 0003](adr/0003-sentence-word-limit.md)).
   - Texto considerado: parágrafos, itens de lista, títulos, células de
     tabela, citações em bloco e texto alternativo de imagem (este como
     bloco próprio, separado do parágrafo em volta). Código (em linha
     ou em bloco), destino de link e URLs (`http:`, `https:`, `mailto:`)
     ficam fora; o texto do link conta.
   - Frase termina em `.`, `!`, `?`, `:` ou `…`, talvez seguidos de
     fechamento (`)`, `]`, aspas), e depois espaço ou quebra de linha; o
     fim de cada bloco também termina a frase. Abreviação como `ex.`
     corta a frase: a regra erra para o lado de aceitar.
   - Palavra: sequência de letras ou dígitos, com hífen ou apóstrofo
     interno (`guarda-chuva` é uma palavra).
   - A linha informada é a do início da frase; a mensagem traz as oito
     primeiras palavras, para achar a frase.
2. **Citação proibida**: texto que casa com algum `forbidden_patterns`.
   Vale para o mesmo texto da regra 1 mais código em linha e destino de
   links; blocos de código ficam fora. A mensagem da violação é o
   `message` configurado, seguido do trecho que casou; cada ocorrência
   é uma violação.
3. **HTML cru**: bloco ou trecho de HTML no Markdown (nó HTML do
   goldmark), inclusive comentário `<!-- -->`. Autolink
   (`<https://...>`) e HTML dentro de código não contam. Uma violação
   por linha.

Um arquivo lido pelos dois guias (o CHANGELOG, por exemplo) passa pelas
regras de cada guia; a mesma violação sai uma vez só.

## 6. Capturas com marcadores

### 6.1 Formato do arquivo de marcadores

O e2e do projeto grava, ao lado de cada captura, um JSON com o mesmo
nome-base e o sufixo `.markers.json`:

```
docs/user/media/register-order-1.png
docs/user/media/register-order-1.markers.json
```

```json
{
  "version": 1,
  "image": "register-order-1.png",
  "width": 1280,
  "height": 800,
  "markers": [
    { "n": 1, "x": 0.125, "y": 0.20, "text": "Campo Cliente" },
    { "n": 2, "x": 0.610, "y": 0.20, "text": "Botão Adicionar item" },
    { "n": 3, "x": 0.910, "y": 0.92, "text": "Botão Salvar" }
  ]
}
```

| Campo | Regra |
|---|---|
| `version` | só `1` |
| `image` | nome do arquivo da captura, no mesmo diretório |
| `width`, `height` | tamanho da captura em pixels; precisa bater com o arquivo |
| `markers[].n` | inteiro, de 1 a N, sem buraco nem repetição, em ordem |
| `markers[].x`, `markers[].y` | centro do marcador, relativo à largura e à altura (0 a 1), origem no canto superior esquerdo |
| `markers[].text` | rótulo curto, não vazio, até 80 caracteres |

Capturas aceitas: PNG e JPEG (o build lê o tamanho com a biblioteca
padrão). Chave desconhecida, JSON inválido ou valor fora da faixa é
erro de conteúdo com o caminho do JSON e do campo.

### 6.2 Ligação com o passo

No Markdown, a imagem fica sozinha num parágrafo e logo depois vem uma
lista numerada. O marcador `n` corresponde ao item `n` da lista.

```markdown
## Registre o pedido

![Tela de novo pedido](../media/register-order-1.png)

1. Escolha o cliente.
2. Adicione os itens.
3. Clique em Salvar.
```

- Quantidade de marcadores diferente da quantidade de itens é erro de
  conteúdo, na linha da imagem.
- Os problemas do `.markers.json` saem mesmo quando a ligação com a
  lista falha; só a contagem depende da lista.
- Imagem sem `.markers.json` ao lado é uma imagem comum.

### 6.3 Comportamento

- Marcadores numerados desenhados sobre a captura (posição em
  porcentagem, acompanham o redimensionamento).
- Cada marcador é um botão com o número; ativá-lo destaca o item da
  lista e vice-versa.
- **Visita guiada**: botão `guided_tour` amplia a captura (2×) centrada
  no marcador atual, mostra o `text` e os botões Anterior/Próximo; Esc
  encerra. Com `prefers-reduced-motion: reduce`, a troca é instantânea,
  sem animação de zoom.
- **Lista acessível**: abaixo da captura, uma lista "Pontos desta tela"
  com `n. text`, sempre presente no HTML. A imagem aponta para ela com
  `aria-describedby`.

Detalhes:

- A captura com marcadores sai como `<figure class="gb-shot">`, no lugar
  do parágrafo da imagem; a lista numerada seguinte ganha o atributo
  `data-gb-steps`.
- A posição vai em `data-x` e `data-y`, nunca em estilo no HTML. O
  JavaScript aplica a posição, mostra os marcadores e o botão da visita.
  Sem JavaScript, só a imagem e a lista acessível aparecem.
- Clicar num item da lista de passos ou da lista acessível também
  destaca o marcador.
- A visita centraliza o marcador sem mostrar área fora da imagem: perto
  da borda, o zoom encosta na borda.
- Teclado durante a visita: → e ↓ avançam, ← e ↑ voltam, Esc encerra e
  devolve o foco ao botão `guided_tour`.
- A visita funciona também num slide. Durante ela, as setas e o Esc são
  da visita, não da apresentação. Trocar de slide ou de página encerra a
  visita.
- A cópia de um slide ou de um diálogo renomeia os ids (sufixo
  `--slide` ou `--dialog`) e mantém o `aria-describedby` apontando para
  a lista certa.

## 7. Vídeo por tarefa

### 7.1 Formato

O e2e grava um vídeo por tarefa e um manifesto no mesmo diretório:

```
docs/user/media/videos/register-order.webm
docs/user/media/videos/manifest.json
```

```json
{
  "version": 1,
  "videos": [
    {
      "task": "tasks/register-order",
      "file": "register-order.webm",
      "duration_seconds": 94.2,
      "poster": "register-order.png"
    }
  ]
}
```

| Campo | Regra |
|---|---|
| `version` | só `1` |
| `videos[].task` | identificador de página de `tasks/`, único no manifesto |
| `videos[].file` | `.webm` ou `.mp4`, no diretório do manifesto, existente |
| `videos[].duration_seconds` | número maior que 0 |
| `videos[].poster` | opcional; PNG ou JPEG no diretório do manifesto |

Só as tarefas listadas em `user.videos.tasks` ganham vídeo; cada uma
precisa estar no manifesto (senão, erro de conteúdo). Entrada do
manifesto fora da lista é ignorada.

### 7.2 Comportamento

- Botão `watch_video` com `{min}` = duração arredondada para cima, em
  minutos, mínimo 1.
- Abre um `<dialog>` grande (até 90% da janela) com `<video controls
  preload="none">`. Sem sincronia com o texto ou os slides. Fechar o
  diálogo pausa o vídeo.
- Ícone `has_video` ao lado da tarefa no índice e no primeiro slide da
  tarefa na lista de slides.

Detalhes:

- O botão é um link para o arquivo, logo depois do título da tarefa:
  `<a class="gb-video-link" href="media/..." data-poster="media/...">`.
  Sem JavaScript, ele abre o vídeo no navegador; com JavaScript, abre o
  diálogo.
- Vídeo e poster vão para `media/`, com o caminho relativo ao
  `guidebook.yaml`, como as imagens.
- A caixa do vídeo ocupa a altura disponível; o vídeo cabe nela com
  tarja preta nas sobras, porque com `preload="none"` a proporção só
  aparece ao tocar.
- Fechar o diálogo pausa o vídeo, solta o arquivo e devolve o foco ao
  link. Com o vídeo aberto, as teclas não trocam de slide.
- O manifesto é lido de forma estrita: chave desconhecida, versão,
  tarefa vazia ou repetida, extensão, nome com caminho e duração são
  erro, com o campo. Os arquivos só precisam existir nas entradas
  usadas. Problemas do manifesto são erro de conteúdo (código 1).
- O ícone "tem vídeo" é decorativo; o texto `has_video` fica para
  leitores de tela.

## 8. Guia do desenvolvedor

Mesmas bases do guia do usuário (leitura, busca, `<dialog>`, regras de
escrita, links conferidos), sem modo apresentação, perfis nem
marcadores.

Seções, nesta ordem; seção sem `sources` não aparece:

| Chave | Título padrão | Montagem | arc42 |
|---|---|---|---|
| `overview` | Visão geral | fontes em sequência; espera diagramas C4 de contexto e de containers em Mermaid | 1, 3 |
| `modules` | Módulos | uma subseção por arquivo | 5 |
| `business_rules` | Regras de negócio | cada `## ` é uma regra; filtro por módulo (título do arquivo) e por texto | — |
| `data_model` | Modelo de dados | fontes em sequência; diagramas `erDiagram` em Mermaid | 5, 8 |
| `decisions` | Decisões | um item por arquivo (ADR), em ordem de nome | 9 |
| `security` | Segurança | fontes em sequência | 8 |
| `delivery` | Ciclo de entrega | fontes em sequência | 7 |
| `development` | Como manter e evoluir | fontes em sequência; espera: subir o ambiente, fluxo de trabalho (branch, commits, revisão, pull request, CI), acrescentar uma funcionalidade, rodar os testes e publicar uma versão | — |
| `operations` | Operação | fontes em sequência | 7, 8 |
| `out_of_scope` | Fora da versão | fontes em sequência | 11 |
| `changes` | Novidades técnicas | CHANGELOG (seção 3.6) | — |

Padrões glob casam em ordem de bytes do caminho. Um arquivo só pode
aparecer em uma seção.

Mermaid: blocos ` ```mermaid ` vão para o HTML como texto escapado
dentro de `<pre class="mermaid">` e são desenhados no navegador pelo
Mermaid vendorizado (seção 10). Diagrama que falha mostra o texto
original e a mensagem de erro do Mermaid.

Detalhes:

- O guidebook inicializa o Mermaid com `securityLevel: "strict"` e
  `startOnLoad: false`; teste Go confere essa configuração no JavaScript
  gerado. O tema segue `prefers-color-scheme`.
- Cada diagrama é desenhado quando a página dele aparece, uma vez só:
  desenhar escondido erra o tamanho.
- No guia do usuário, ` ```mermaid ` continua bloco de código comum.
- Cada arquivo de seção vira uma página `<seção>/<nome do arquivo>`,
  com o nome pela regra das âncoras (seção 3.2): `docs/adr/0001-x.md`
  em `decisions` vira `decisions/0001-x`. Toda fonte precisa de um
  `# ` e só um.
- Menu: módulos e decisões viram grupo, com um item por arquivo. As
  outras seções com um arquivo só viram um item direto, com o título
  da seção; com mais arquivos, viram grupo.
- Regras de negócio: um artigo só, `business_rules`, com uma parte por
  arquivo (`data-module` = título do arquivo). Cada `## ` é uma regra;
  o filtro por texto ignora acento e maiúscula, e o filtro por módulo
  usa os títulos. Sem resultado, aparece `rules_none`.
- Novidades técnicas: um artigo `changes`, com todos os marcos das
  fontes, inclusive o não publicado, na ordem dos arquivos.
- Links no guia do desenvolvedor sempre navegam; não há diálogo, modo
  apresentação, vídeo nem "Comece por aqui". Link para uma fonte de
  `changes` leva ao artigo `changes`.

## 9. Saída

### 9.1 Estrutura

```
out/
├── .guidebook-output
├── user-guide.html
├── developer-guide.html
├── assets/                 # recursos do guidebook (go:embed) e dados
│   ├── guidebook.css
│   ├── guidebook.js
│   ├── minisearch.js       # vendorizado, sem minificar (ADR 0007)
│   ├── minisearch.LICENSE.txt
│   ├── mermaid.min.js      # só se houver guia do desenvolvedor (ADR 0010)
│   ├── mermaid.LICENSE.txt
│   ├── user-data.js
│   ├── theme.css           # cores do produto, comuns aos dois guias
│   └── developer-data.js
└── media/                  # cópia da mídia do projeto usada pelos guias
```

Uma página HTML por guia, com navegação por `location.hash`
([ADR 0001](adr/0001-single-page-per-guide.md)). O conteúdo das páginas
vai pré-renderizado no HTML; sem JavaScript, o guia continua legível
como um documento longo.

### 9.2 Restrições

- Abre por `file://`: nenhum `fetch`, `XMLHttpRequest`, `import()`
  dinâmico, CDN ou recurso externo. Teste confere que o HTML e os `.js`
  e `.css` próprios não referenciam `http:`/`https:` fora de links de
  conteúdo. Biblioteca vendorizada pode citar URL em comentário; teste
  próprio confere que ela não usa rede nem gera código (seção 10).
- Dados para o JavaScript em `assets/*-data.js`, no formato
  `window.GUIDEBOOK_DATA = <json>;`, com o JSON produzido por
  `json.Marshal` (que escapa `<`, `>`, `&`, U+2028 e U+2029). Nenhum
  texto do projeto entra cru em `<script>`.
- HTML do conteúdo: saída do goldmark sem `WithUnsafe`; textos da
  configuração passam por `html/template`.
- Recursos do guidebook (CSS, JS, bibliotecas vendorizadas) embutidos no
  binário com `go:embed`.
- **Determinismo**: mesma entrada, mesmos bytes. Sem data, hora, caminho
  absoluto ou valor aleatório na saída; mapas sempre percorridos em
  ordem; arquivos gravados em ordem. Teste gera duas vezes e compara.
- **Teto de tamanho**: o conjunto de `out/` sem `media/` tem teto
  conferido por teste, por guia, sobre o projeto de exemplo. O guia do
  usuário tem teto de 160 KiB ([ADR 0006](adr/0006-size-ceiling.md)). O
  guia do desenvolvedor tem teto de 6 MiB
  ([ADR 0010](adr/0010-vendored-mermaid.md)), quase todo do Mermaid.

### 9.3 Interface do guia do usuário

- **Leitura**: cabeçalho com logo, nome do produto, busca e botão
  `present`. À esquerda, o menu; à direita, o conteúdo. O menu traz, em
  ordem:
  - a página inicial (`index.md`);
  - "Comece por aqui", se houver perfis;
  - os quatro tipos Diátaxis, cada um com as páginas dele;
  - o glossário;
  - "O sistema não me deixa...", se houver entradas;
  - "O que há de novo", se houver marco publicado.

  Sem `index.md`, a página inicial é a primeira do menu. Os artigos
  gerados pelo guidebook têm os identificadores `start-here`, `blocked`
  e `whats-new`, que não colidem com os das páginas. O endereço de uma
  página é `#/<identificador>`; o de uma seção, `#/<identificador>/<âncora>`.
  O `id` dos elementos é o próprio endereço sem o `#`. Assim, sem
  JavaScript, os links continuam levando ao lugar certo.
- **Apresentação**: cada página vira uma sequência de slides 16:9,
  cortada em cada `## ` e em cada `---`. O primeiro slide leva o `# ` e
  o texto antes do primeiro corte. Teclas:
  - →, PageDown e espaço avançam;
  - ←, PageUp e Shift+espaço voltam;
  - Home e End vão ao primeiro e ao último;
  - F alterna tela cheia (Fullscreen API);
  - Esc sai da tela cheia e, fora dela, da apresentação.

  Detalhes ([ADR 0008](adr/0008-slides-cut-at-build.md)):
  - O build corta os slides: cada um vira um `<div class="gb-slide">`
    no HTML da página. Na leitura, os cortes não aparecem; o `---`
    continua como linha e some no slide.
  - A apresentação usa a mesma página (ADR 0001). O botão `present`
    abre a página atual no slide da seção apontada pelo endereço.
    Artigo sem cortes, como "Comece por aqui", é um slide só.
  - Barra com lista de slides (recolhível, fechada ao abrir), contador
    `slide_of` anunciado a leitores de tela, Anterior, Próximo, Voltar,
    tela cheia e saída.
  - O slide tem 1280×720 px lógicos e é escalado para a largura do
    palco. Conteúdo mais alto que o slide reduz a escala até caber, até
    60% dela; abaixo disso, o palco rola.
  - Link de diálogo abre por cima, sem perder o slide. Link de tarefa
    troca de página e continua apresentando; "Voltar" devolve página e
    slide.
  - Troca de slide com transição curta de opacidade; com
    `prefers-reduced-motion: reduce`, sem transição.
  - Ao entrar, o foco vai para o palco; ao sair, volta ao botão
    `present`, e a leitura fica no ponto do último slide.
  - Espaço e Enter num botão ou link ficam com o próprio elemento. Sem
    JavaScript, o botão `present` não aparece.
- **Busca**: MiniSearch vendorizado. O build gera os documentos já
  extraídos; o índice é montado na abertura do guia
  ([ADR 0002](adr/0002-search-index-at-load.md)).
  - Cada página dá um documento para a introdução, antes do primeiro
    `## `, e um para cada seção `## `. O texto inclui subtítulos,
    tabelas e código; HTML fica fora.
  - Campos: título da página, título da seção e texto. Título pesa 3 e
    seção pesa 2.
  - Acento e maiúscula não contam: "cobranca" acha "cobrança". Vale
    prefixo ("fech" acha "Fechar") e erro pequeno de digitação.
  - Um resultado só aparece quando todos os termos da busca estão no
    documento.
  - Até 20 resultados, cada um com seção, página e trecho com os termos
    destacados. Sem resultado, aparece `search_no_results`.
  - Teclado: `/` leva à busca; ↓ e ↑ escolhem o resultado; Enter abre;
    Esc limpa. Abrir um resultado guarda o ponto de origem, como um link
    de tarefa.
  - Sem JavaScript, a caixa de busca não aparece.
- **Acessibilidade**: navegação por teclado em tudo e foco visível. O
  `<dialog>` prende o foco e o devolve ao link ao fechar. Ao trocar de
  página, o foco vai para o título. Texto alternativo é obrigatório em
  imagem; imagem sem ele é erro de conteúdo.

## 10. Bibliotecas de navegador vendorizadas

MiniSearch e Mermaid, versões fixas, baixadas uma vez do registro npm
para `internal/site/assets/vendor/<nome>/`. Cada uma tem:

- o arquivo distribuído, tal como vem no pacote;
- o arquivo de licença original ao lado, copiado também para a saída;
- uma entrada no manifesto `internal/site/assets/vendor/vendor.json`.

Campos da entrada no manifesto:

| Campo | Conteúdo |
|---|---|
| `name`, `version` | pacote npm e versão exata |
| `source` | URL do tarball no registro npm |
| `tarball_integrity` | `dist.integrity` do registro, conferido no download |
| `file` | caminho do arquivo dentro do tarball |
| `vendored_as` | caminho no repositório, relativo a `vendor/` |
| `sha256` | SHA-256 do arquivo vendorizado |
| `license`, `license_file` | licença e arquivo dela |
| `reviewed_matches` | quantas ocorrências de API de rede foram revisadas à mão |

Testes Go conferem:

- o SHA-256 de cada arquivo contra o manifesto;
- que a licença existe;
- que a origem é o tarball do registro npm;
- que o arquivo não usa rede nem gera código (`fetch`, `XMLHttpRequest`,
  `import()`, `eval`, `new Worker`...), fora de comentários. Em arquivo
  minificado, a conferência por linha não separa definição de chamada.
  Por isso, cada ocorrência que sobra é revisada à mão e contada em
  `reviewed_matches`, e o teste exige esse número exato.

Nunca CDN. A versão do MiniSearch está no
[ADR 0007](adr/0007-vendored-minisearch.md).

## 11. Limites do projeto

- Dependências Go: só goldmark e `go.yaml.in/yaml/v3`
  ([ADR 0004](adr/0004-yaml-library.md)), com versões fixas.
- Teto de código: Go próprio sem testes até ~4.000 linhas; JavaScript
  próprio até ~1.500. Passou disso, para e reavalia com o Bruno o uso de
  ferramenta pronta.

## 12. Fases

| Fase | Branch | Entrega |
|---|---|---|
| 1 | `docs/spec-and-flow` | esta especificação, fluxo no AGENTS.md, CHANGELOG |
| 2 | `feat/cli-config` | `go.mod`, `cmd/guidebook` com `build`, leitura e validação do `guidebook.yaml`, projeto de exemplo em `testdata/`, `make build`/`make test`, CI |
| 3 | `feat/markdown-rules` | goldmark e regras de escrita |
| 4 | `feat/user-guide-reading` | guia do usuário em modo leitura e offline, com o que a seção 9.3 descreve; teto de tamanho; `check` do próprio repositório na CI |
| 5 | `feat/search` | MiniSearch vendorizado com manifesto e hash |
| 6 | `feat/presentation-mode` | modo apresentação |
| 7 | `feat/screenshot-markers` | marcadores, visita guiada, lista acessível |
| 8 | `feat/task-video` | vídeo por tarefa |
| 9 | `feat/developer-guide` | seções configuráveis e Mermaid vendorizado |
| 10 | `feat/serve-container` | `guidebook serve`, Dockerfile, CI que constrói a imagem e só publica em tag (seção 14) |
| 11 | `docs/usage` | guias do próprio guidebook no formato dele (seção 13), README curto apontando para eles, como pôr o guidebook no compose de um projeto e `make preview` |

A partir da fase 4, toda fase que muda comportamento atualiza também a
página correspondente dos guias do guidebook (seção 13), no mesmo
pull request.

## 13. O guidebook como projeto de entrada

O repositório do guidebook é também um projeto que o guidebook lê: tem
`guidebook.yaml` na raiz e páginas na convenção da seção 3. Gerar os
próprios guias é um teste da ferramenta.

- **Guia do usuário do guidebook** (`docs/user/`). Assuntos:
  - instalar;
  - configurar o `guidebook.yaml`;
  - escrever as páginas;
  - rodar `build` e `check`;
  - integrar ao e2e de um projeto, com os formatos das seções 6 e 7;
  - interpretar as recusas das regras de escrita.

  É uma CLI: sem capturas nem vídeos aqui. Esses recursos continuam testados
  pelo projeto de exemplo em `testdata/`.
- **Guia do desenvolvedor do guidebook**: montado das fontes reais do
  repositório — `AGENTS.md`, `docs/spec.md`, `docs/adr/` e
  `CHANGELOG.md` — mais uma visão geral com diagramas C4 de contexto e
  de containers em Mermaid.
- **CI**: assim que o `build` existir (fase 4), a CI roda `guidebook
  check` sobre o próprio repositório. Violação das regras de escrita na
  documentação do guidebook quebra a CI. Os documentos já existentes
  (este inclusive) são ajustados às regras nessa fase.
- **Prévia**: `make preview` gera os guias do próprio guidebook em
  `out/guidebook/` e os do projeto de exemplo de `testdata/` em
  `out/example/`. No fim, imprime os caminhos `file://` dos quatro HTML.

### 13.1 Lista de conferência da prévia

O que o Bruno confere antes da v0.1.0, porque o teste automático não
cobre:

1. **Tela cheia de verdade**, no guia do usuário do exemplo: F entra e
   sai; Esc sai da tela cheia e, fora dela, da apresentação; o botão
   Tela cheia também sai.
2. **Temas claro e escuro**: troque o tema do sistema e recarregue os
   quatro guias; confira contraste do cabeçalho, dos marcadores e dos
   diagramas.
3. **Tela estreita**: janela de celular (até 48rem); menu acima do
   conteúdo, busca em linha própria, apresentação legível.
4. **Visita guiada** em "Registrar um pedido": com o mouse e só com o
   teclado (Tab até "Visita guiada", setas, Esc), na leitura e no modo
   apresentação.
5. **Vídeo** em "Registrar um pedido": abrir e fechar com o mouse e com
   o teclado; o foco volta ao link.
6. **Leitor de tela**, se houver à mão: a lista "Pontos desta tela", o
   contador de slides e a busca.
7. **Serve pelo container**: `docker build -t guidebook:previa .` e o
   compose da tarefa "Servir os guias", trocando a imagem por
   `guidebook:previa` e publicando a porta só para a conferência.
   Confira `/healthz`, o guia do usuário, o 404 de
   `/developer-guide.html`, uma mudança num `.md` aparecendo e o
   `docker compose stop`.
- O README continua curto e aponta para os guias.

## 14. Servidor e container

Comando para servir o guia do usuário de um projeto, por exemplo ao
lado da aplicação no compose dela. Só biblioteca padrão; sem dependência
nova.

**O guia do desenvolvedor nunca é servido.** É regra fixa, sem opção.
Ele existe só como arquivo local, gerado com `guidebook build` por quem
tem acesso ao repositório.

### 14.1 Comando

```
guidebook serve [-config guidebook.yaml] [-addr 127.0.0.1:8080] [-interval 2s]
```

- `-addr` padrão `127.0.0.1:8080`; no container, `:8080`.
- `-interval` padrão `2s`, mínimo `500ms`.
- `-guide` é erro de uso (código 2): o `serve` gera só o guia do
  usuário.
- O `serve` não lê nem valida a seção `developer` do `guidebook.yaml`,
  nem as fontes dela. Erro nelas não aparece no log nem no `/healthz`.
- Com o `guidebook.yaml` válido mas sem `user`, o `serve` recusa ao
  subir, com código 2: `guidebook.yaml: user: o serve só serve o guia
  do usuário; defina a seção user`.
- Ao subir, faz o build inicial num diretório interno, criado com
  `os.MkdirTemp` no diretório temporário do sistema (`$TMPDIR`), e
  serve os arquivos dele.
- **Atualização**: a cada intervalo, varre as entradas com a biblioteca
  padrão e compara caminho, tamanho e data de modificação. Entradas:
  - o `guidebook.yaml`;
  - todo o `user.dir`, incluindo `media/`;
  - o `user.changelog` e o logo;
  - o diretório do manifesto de vídeos.

  As fontes do guia do desenvolvedor ficam fora da varredura.

  Havendo mudança, faz o build num diretório novo e troca o diretório servido
  de forma atômica: cada requisição vê o build antigo inteiro ou o novo
  inteiro, nunca a metade. O diretório anterior é apagado depois da
  troca.
- **Build que falha** (regra de escrita, conteúdo ou configuração):
  continua servindo a última versão boa e registra os problemas no log,
  no mesmo formato do `build`.
- **Servido = gerado**: os arquivos servidos são os bytes do build, sem
  nada injetado. O mesmo diretório abre por `file://`.
- **Log**: saída de erro, uma linha por evento, em pt-BR (build
  iniciado, concluído, falhou com N problemas, troca feita).
- **Desligamento**: em SIGTERM ou SIGINT, para de aceitar conexões,
  espera até 10 s pelas requisições em curso
  (`http.Server.Shutdown`), apaga o diretório interno e sai com 0.
- Erro de uso (opção inválida, `-interval` abaixo do mínimo) e porta que
  não abre saem com código 2. Build que falha ao subir não derruba o
  servidor: ele fica no ar com `/healthz` em 503 até um build dar certo.
- Com o `guidebook.yaml` inválido, a varredura olha só ele; corrigido o
  arquivo, o build volta e as outras entradas voltam a ser olhadas.
- Cada build vai para `build-N` dentro do diretório interno. A troca é
  um ponteiro atômico; o build antigo só é apagado depois que as
  leituras em curso terminam.

### 14.2 Rotas

| Rota | Resposta |
|---|---|
| `GET /` | redireciona (302) para `/user-guide.html` |
| `GET /healthz` | estado do build, em JSON (abaixo) |
| `GET /<arquivo>` | arquivo do build atual; diretório ou arquivo inexistente dá 404, sem listagem |
| `GET /.../developer-guide.html`, `GET /.../developer-data.js` | 404 sempre, em qualquer pasta e com qualquer caixa, mesmo que o arquivo exista no diretório |
| outro método | 405 com `Allow: GET, HEAD` |

`HEAD` vale em todas as rotas de `GET`. `/healthz`:

| Situação | Código | Corpo |
|---|---|---|
| último build bom e nenhum erro pendente | 200 | `{"status":"ok"}` |
| há versão boa, mas o build mais recente falhou | 200 | `{"status":"stale","problems":["docs/user/tasks/x.md:3: ..."]}` |
| nunca houve build bom | 503 | `{"status":"unavailable","problems":[...]}` |

O 200 em `stale` mantém o serviço no ar para quem lê o guia; o
monitoramento distingue pelo campo `status`.

### 14.3 Segurança

- Escuta em `127.0.0.1` por padrão.
- Só `GET` e `HEAD`.
- Sem listagem de diretório; arquivos lidos por `os.Root`, que impede
  sair do diretório servido, inclusive por `..` ou link simbólico.
- Cabeçalhos em toda resposta:
  - `Content-Security-Policy: default-src 'self'; script-src 'self';
    style-src 'self' 'unsafe-inline'; img-src 'self' data:; media-src
    'self'; object-src 'none'; base-uri 'none'; form-action 'none';
    frame-ancestors 'none'`. O `'unsafe-inline'` em `style-src` é o
    mínimo que o Mermaid pede, porque ele escreve `<style>` dentro do
    SVG. Confirmado no navegador na fase 10
    ([ADR 0011](adr/0011-serve-and-image.md)). Script só de `'self'`,
    sem `'unsafe-inline'` nem `'unsafe-eval'`.
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: no-referrer`
  - `Cache-Control: no-cache`, para o navegador revalidar depois de
    cada troca.
- Os guias já cumprem essa política: nenhum script em linha, dados em
  `assets/*-data.js` (seção 9.2).
- `http.Server` com `ReadHeaderTimeout` 5 s, `ReadTimeout` 15 s,
  `WriteTimeout` 10 min (vídeos grandes), `IdleTimeout` 2 min e
  `MaxHeaderBytes` 64 KiB.
- **Sem autenticação.** O guidebook não controla quem acessa. Proteger
  o acesso (autenticação, TLS, limite de taxa) é responsabilidade de
  quem implanta, no proxy reverso do projeto.

### 14.4 Container

- `Dockerfile` no repositório, em múltiplos estágios: compila com
  `CGO_ENABLED=0` (binário estático) numa imagem Go e copia só o
  binário para uma imagem final mínima, sem shell. Imagens base fixadas
  por digest: `golang:1.27.2-trixie` para compilar e
  `gcr.io/distroless/static-debian13:nonroot` na final
  ([ADR 0011](adr/0011-serve-and-image.md)).
- Usuário não root (`nonroot`, UID 65532).
- A imagem não tem healthcheck próprio, porque não há shell nem cliente
  HTTP dentro dela. O proxy ou o orquestrador consulta `/healthz`.
- Funciona com sistema de arquivos somente leitura: o único lugar
  gravável é o diretório temporário (`/tmp`), montado como `tmpfs`.
- Entrada padrão: `guidebook serve -config /project/guidebook.yaml
  -addr :8080`, com diretório de trabalho `/project`.
- O projeto monta, somente leitura, só o que o guia do usuário lê: o
  `guidebook.yaml`, o `docs/user/` e o `CHANGELOG.md`, em `/project`.
  As fontes do guia do desenvolvedor não entram no container. Os
  caminhos do `guidebook.yaml` valem a partir de `/project`, como no
  repositório. Um logo fora de `docs/user/` pede a pasta dele também.
  Prefira montar diretórios: um arquivo montado sozinho deixa de
  refletir mudanças quando o editor o substitui por outro.

Exemplo de serviço no compose do projeto:

```yaml
services:
  guidebook:
    image: ghcr.io/bpbastos/guidebook:v0.1.0
    read_only: true
    tmpfs:
      - /tmp
    cap_drop: [ALL]
    security_opt:
      - no-new-privileges:true
    volumes:
      - ./guidebook.yaml:/project/guidebook.yaml:ro
      - ./docs/user:/project/docs/user:ro
      - ./CHANGELOG.md:/project/CHANGELOG.md:ro
    expose:
      - "8080"   # publicado só pelo proxy reverso do projeto
```

### 14.5 Publicação

- A CI constrói a imagem em todo pull request, sem publicar. Confere
  que ela serve o guia do usuário do projeto de exemplo com
  `--read-only`, `tmpfs` em `/tmp` e `cap-drop ALL`. Monta só o que o
  guia do usuário lê e confere que `/developer-guide.html` dá 404.
- O workflow `release.yml` roda só em tag `vX.Y.Z`. Ele confere o
  formato da tag, roda `go test` e publica. Só esse job tem
  `packages: write`; o resto do workflow não tem permissão nenhuma.
- Quando o Bruno cria uma tag de versão (`v0.1.0`...), a CI publica a
  imagem no GitHub Container Registry, pública, como
  `ghcr.io/bpbastos/guidebook:<tag>`. A publicação usa o `GITHUB_TOKEN`
  do próprio workflow, sem segredo no repositório.
- Os projetos usam a versão fixa (nunca `latest`).

### 14.6 Testes

Servidor testado com `httptest`. Os testes cobrem:

- cabeçalhos de segurança;
- 404 sem listagem, 405 e recusa de saída do diretório;
- 404 para `developer-guide.html` e `developer-data.js`, mesmo
  plantados no diretório, com variações de caminho e de caixa;
- fontes do desenvolvedor fora do build, do log, do `/healthz` e da
  varredura;
- recusa do projeto só com `developer`;
- `/healthz` nos três estados;
- troca atômica;
- última versão boa servida depois de um build que falhou;
- build só quando a entrada muda, inclusive arquivo novo;
- desligamento por cancelamento e por SIGTERM de verdade.
