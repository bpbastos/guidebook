# 0018. `check -no-media` para a CI antes do e2e

Status: aceita (fase 14), implementada na fase 17.

## Contexto

As capturas, os `.markers.json` e os vídeos são gerados pelo e2e e
ficam fora do git. Uma CI que roda o `check` antes do e2e, ou num job
sem navegador, recebe erro por mídia ausente. A conferência de escrita
e de vínculos é útil mesmo assim, e é a mais barata.

Alternativas:

- segundo arquivo de configuração para a CI: o Bruno disse que não é
  pedido, e duplicaria a configuração;
- rebaixar a mídia ausente a aviso sempre: afrouxa o `check` de quem
  tem as mídias;
- opção que só ignora arquivos ausentes e confere os presentes: o
  resultado dependeria do que sobrou no diretório;
- opção que não lê mídia nenhuma.

## Decisão

`check -no-media` não lê nenhum arquivo de mídia. Deixa de exigir as
imagens, os `.markers.json`, o manifesto e os vídeos, a ligação entre
marcador e lista e o tamanho dos guias. Continua conferindo as regras
de escrita, o texto alternativo, os vínculos entre páginas e as regras
de caminho. `user.videos.manifest` ausente deixa de recusar a
configuração nesse modo. Os avisos de mídia não saem, porque nada foi
lido.

A opção existe só no `check`: o `build` precisa das mídias para gravar
o guia.

## Consequência

A CI pode rodar o `check -no-media` cedo e o `check` completo, ou o
`build`, depois do e2e. Sem a opção, nada fica mais frouxo.
