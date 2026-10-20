# 0001. Uma página HTML por guia

Status: aceita (fase 1).

## Contexto

Os guias abrem por `file://`. "Voltar para onde eu estava" e o vínculo
de apoio em `<dialog>` precisam de estado entre páginas. Com vários
arquivos HTML, esse estado teria de passar por `localStorage` ou
`sessionStorage`, que por `file://` varia entre navegadores (o Firefox
trata cada arquivo como origem própria).

## Decisão

Cada guia é um único `user-guide.html` ou `developer-guide.html`, com
todas as páginas pré-renderizadas e navegação por `location.hash`. O
estado de navegação fica na memória da página e no histórico.

## Consequência

Navegação, `<dialog>` e "Voltar" funcionam igual em qualquer navegador,
sem armazenamento. O HTML cresce com o número de páginas; o teto de
tamanho (seção 9.2 da spec) vigia isso. Sem JavaScript, o guia continua
legível como um documento longo.
