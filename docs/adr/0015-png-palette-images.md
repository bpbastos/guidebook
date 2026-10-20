# 0015. Imagens de mídia em PNG com paleta, por quantizador próprio

Status: aceita (fase 14), por decisão do Bruno. Medição na fase 15.

## Contexto

Numa prévia num projeto real, as capturas do e2e dominaram o tamanho
do guia do usuário. A referência do Bruno, com 89 capturas:

- originais: 22,2 MB;
- paleta de 256 cores pelo pngquant: 9,3 MB, sem diferença visível.

Capturas de tela têm poucas cores e muito texto. Uma paleta de 256
cores costuma bastar, e muitas telas cabem nela sem perda.

Alternativas:

- WebP com perda ou sem perda: exige dependência, porque a biblioteca
  padrão não grava WebP;
- chamar o pngquant ou outra ferramenta externa: ferramenta nova no
  fluxo e no ambiente de quem usa o guidebook;
- biblioteca Go de quantização: dependência nova;
- copiar como vieram, como na versão 0.1.

## Decisão

PNG de mídia (capturas, imagens das páginas e posters) sai como PNG
com paleta, por código próprio, só com a biblioteca padrão:

- até 256 cores distintas, contando o alfa: paleta exata, sem perda;
- mais que isso: quantização para 256 cores, com alfa, por median cut
  ou equivalente, determinística;
- se o resultado não for menor, fica o original.

O formato e a extensão não mudam. Caminho e dimensões também não, e os
`.markers.json` continuam valendo. JPEG, vídeos e o logo saem como
vieram.

O pontilhado (Floyd-Steinberg de `image/draw`) fica de fora, a menos
que as amostras mostrem texto mais legível com ele.

## Implementação

Pacote `internal/media`, só com a biblioteca padrão:

- As chaves de todos os pixels, ordenadas, dão as cores distintas e
  quantas vezes cada uma aparece. Até 256 cores, elas são a paleta.
- Acima disso, o median cut divide sempre o grupo de maior erro
  quadrático. O corte fica no canal e no valor que deixam a menor soma
  de erros dos dois lados, com os quatro canais, alfa incluído. A cor
  de cada grupo é a média ponderada.
- Cada cor vai para a cor mais próxima da paleta, pela distância
  quadrática; no empate, o menor índice.
- A paleta sai em ordem de alfa, e as cores transparentes vêm primeiro.
  Assim o bloco de transparência do PNG fica curto.
- A gravação usa `png.BestCompression`.
- PNG de 16 bits por canal sai como veio, porque a paleta perderia
  precisão.
- O `build` otimiza os arquivos em paralelo e grava em ordem. O cache
  por SHA-256 serve ao `serve` e aos dois guias de um mesmo `build`.

Pior caso sintético, 1920×1080 com quase todo pixel de cor própria:
cerca de 1,9 s por imagem numa máquina de 8 núcleos. Capturas reais
têm muito menos cores.

## Medição

A medir na fase 15, nas amostras de `.local/samples/`:

- tamanho total antes e depois, sem e com pontilhado: a medir na
  fase 15;
- quantas imagens cabem na paleta exata e quantas são quantizadas: a
  medir na fase 15;
- conferência visual do texto das telas na prévia: a fazer na fase 15;
- tempo do `build` sobre as amostras: a medir na fase 15.

## Consequência

Sem dependência nova; o quantizador fica dentro do teto de Go de
~5.000 linhas ([ADR 0014](0014-go-code-ceiling.md)). O `build` fica
mais lento com muitas capturas; o paralelismo por arquivo e o cache
do `serve` por SHA-256 compensam. A otimização regrava o PNG e
descarta os blocos auxiliares dele, como perfil de cor e texto. O
`check` faz a mesma otimização para informar o tamanho exato.
