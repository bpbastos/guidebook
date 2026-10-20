# 0013. Bibliotecas de navegador em `third_party/`

Status: aceita (fase 13).

## Contexto

As bibliotecas vendorizadas ficavam em `internal/site/assets/vendor/`.
O zip de módulo do Go (`golang.org/x/mod/zip`), que o proxy e o
`go install` usam, descarta os arquivos de subdiretórios de qualquer
diretório `vendor`. No zip da v0.1.0 só sobrou o `vendor.json`, e o
`build` do binário instalado falha ao abrir o `minisearch.js`. O
`check` não lê os recursos e não pegou o defeito; os testes rodam no
clone, onde os arquivos existem.

Alternativas:

- outro nome para o diretório;
- tirar o nível de diretório e pôr os arquivos direto em `assets/`;
- embutir os arquivos de outro pacote.

## Decisão

O diretório passa a ser `internal/site/assets/third_party/`, com a
mesma estrutura e o mesmo manifesto `vendor.json`. O nome diz o que é,
e a estrutura por biblioteca continua, com a licença ao lado.

Dois testes guardam a regra:

- `TestNoVendorSegment` recusa o segmento `vendor` em qualquer caminho
  dos recursos embutidos. É barato e roda sempre.
- `TestGoInstall` monta um repositório git bare temporário com a árvore
  atual. Instala o guidebook por `go install` a partir dele, com as
  dependências do cache de módulos local, e gera o projeto de exemplo
  com o binário instalado. Fica fora do `-short`; a CI roda
  sem `-short`.

## Consequência

A v0.1.0 não instala por `go install`; a correção sai na v0.1.1. O
`TestGoInstall` pega qualquer outro arquivo que o zip de módulo
descarte, não só os de `vendor`.
