# 0002. Índice de busca montado na abertura

Status: aceita (fase 1).

## Contexto

A especificação pede busca com MiniSearch e "índice gerado no build".
O build é Go; o MiniSearch é JavaScript. Gerar em Go o índice já
serializado (`MiniSearch.loadJSON`) exigiria reproduzir o formato
interno de uma versão específica, sem teste de navegador para pegar
uma quebra.

## Decisão

O build gera os documentos de busca já extraídos e normalizados (título,
texto sem marcação, tipo, âncora), serializados com `json.Marshal` nos
dados do guia. O navegador monta o índice MiniSearch ao abrir o guia.

## Consequência

Nada depende do formato interno do MiniSearch, e os documentos são
conferidos por teste Go. A montagem custa alguns milissegundos para as
centenas de páginas esperadas. Se um guia ficar grande a ponto de a
abertura pesar, revemos esta decisão.
