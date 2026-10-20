# Changelog

Todas as mudanças relevantes do guidebook ficam registradas aqui.

O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto adota o [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Não publicado]

### Adicionado

- O `build` otimiza os PNG de mídia: paleta exata até 256 cores e, com
  mais cores, redução para 256 por quantizador próprio. Fica o original
  quando o resultado não é menor; JPEG, vídeo e logo saem como vieram.
- Imagens do guia com `loading="lazy"`, `decoding="async"` e, em PNG e
  JPEG, `width` e `height`.
- O `serve` guarda as imagens otimizadas em memória e não otimiza de
  novo a que não mudou.
- O `check` e o `build` informam o tamanho de cada guia, com HTML,
  recursos e mídia. O `check` gera numa pasta temporária para medir.
- Teto opcional por guia em `user.max_size_mib` e
  `developer.max_size_mib`; acima dele, código 1.
- Tarefa "Integrar à CI" no guia do usuário. A CI do guidebook gera os
  próprios guias e os guarda como artefato.

## [0.1.1] - 2026-10-20

### Modificado

- O repositório e a imagem `ghcr.io/bpbastos/guidebook` passam a ser
  públicos. O `go install` funciona direto, sem `GOPRIVATE`.

### Corrigido

- O `go install` gera os guias: as bibliotecas de navegador saem do
  diretório `vendor`, que o zip de módulo do Go descartava, e o `build`
  não falha mais ao abrir o `minisearch.js`.
- Marco do CHANGELOG sem itens, como um "Não publicado" vazio, não
  aparece mais em "O que há de novo" nem em "Novidades técnicas".

## [0.1.0] - 2026-10-20

### Adicionado

- Regras da casa no `AGENTS.md`, hook de mensagem de commit e
  `make hooks`.
- Especificação em `docs/spec.md` e primeiras decisões em `docs/adr/`.
- Fluxo de entrega por fases, com revisão do conselheiro.
- Comandos `guidebook build`, `check`, `serve` e `version`. O `build`
  grava os guias do usuário e do desenvolvedor; o `check` faz as mesmas
  conferências sem gravar nada.
- Leitura e validação estrita do `guidebook.yaml`, com erro em pt-BR e
  caminho do campo.
- Varredura das páginas do guia do usuário (convenção Diátaxis) e
  conferência das páginas citadas na configuração.
- Proteção do diretório de saída pelo arquivo `.guidebook-output`.
- Projeto de exemplo em `testdata/example/`.
- `make build`, `make test` e CI no GitHub Actions.
- Regras de escrita no `check` e no `build`: frase longa, citação
  proibida e HTML cru, com `arquivo:linha` e todas de uma vez.
- Leitura de Markdown com goldmark (CommonMark, tabela e tachado).
- CI roda o `check` sobre o projeto de exemplo.
- Guia do usuário em modo leitura, numa página HTML que abre por
  `file://`. Tem menu Diátaxis, glossário e "veja também" em janela e
  "Voltar para onde eu estava". Tem também "Comece por aqui" por perfil,
  o índice "O sistema não me deixa..." e "O que há de novo".
- Conferência de títulos, âncoras, links e imagens; esquemas fora de
  `http`, `https` e `mailto` são recusados.
- Leitura do CHANGELOG no formato Keep a Changelog.
- O guidebook gera os próprios guias (`guidebook.yaml` na raiz) e a CI
  roda o `check` sobre o repositório.
- Busca no guia do usuário, com MiniSearch vendorizado e conferido por
  hash. Ignora acento e maiúscula, aceita prefixo e destaca os termos.
- Modo apresentação no guia do usuário: slides 16:9 cortados em cada
  `##` e `---`, teclado, tela cheia, lista de slides e "slide n de N".
- Capturas com marcadores lidos do `.markers.json` do e2e, com destaque
  entre marcador e passo, visita guiada com zoom e lista acessível.
- Vídeo por tarefa lido do manifesto do e2e, com botão "Ver em vídeo
  (N min)", janela com controles nativos e ícone "tem vídeo".
- Guia do desenvolvedor com seções configuráveis, regras de negócio
  com filtro, novidades técnicas e diagramas Mermaid vendorizados.
- Comando `guidebook serve`: gera e serve o guia do usuário por HTTP,
  gera de novo quando a documentação muda e mantém a última versão boa.
  O guia do desenvolvedor nunca é servido: o `serve` não lê as fontes
  dele e responde 404 aos arquivos dele.
- Seção `development`, "Como manter e evoluir", no guia do
  desenvolvedor, entre o ciclo de entrega e a operação.
- Imagem de container com distroless, usuário não root e sistema de
  arquivos somente leitura; a CI publica no GHCR só em tag de versão.
- Guia do usuário do guidebook: instalação, `guidebook.yaml`, páginas,
  comandos, integração com o e2e, compose e recusas.
- Guia do desenvolvedor do guidebook: visão geral com diagramas C4,
  spec, ADRs, fluxo de entrega e "Como manter e evoluir".
- `make preview` gera os guias do guidebook e os do exemplo e imprime
  os caminhos para abrir.
