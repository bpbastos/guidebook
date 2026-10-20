# 0015. Imagens de mídia em PNG com paleta, por quantizador próprio

Status: aceita (fase 14), por decisão do Bruno. Medida na fase 15.

## Contexto

Numa prévia num projeto real, as capturas do e2e dominaram o tamanho
do guia do usuário. Capturas de tela têm poucas cores e muito texto. Uma paleta de 256
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

Sem pontilhado: as amostras não mostraram texto mais legível com o
Floyd-Steinberg de `image/draw`, e ele aumenta o arquivo (seção
Medição).

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

## Medição

A referência anterior, de 89 capturas, foi descartada. Ela foi medida
sobre PNG refeito a partir de WebP com perda, e o ruído dessa
compressão quase dobra o PNG. Valem só as medições abaixo.

Amostras: 12 capturas originais do Playwright, PNG RGB de 8 bits, sem
perda, com dados fictícios. São 10 de tela inteira com 1440 px de
largura, nos temas claro e escuro, e 2 de celular com 390 px. Medidas
fora do repositório, sem versionar as amostras.

| Medida | Resultado |
|---|---|
| total antes | 1.204.364 bytes |
| total depois, sem pontilhado | 541.847 bytes (45,0%) |
| total depois, com pontilhado | 678.735 bytes (56,4%) |
| na paleta exata | 0 de 12; todas têm de 768 a 8.336 cores |
| original mantido por não ficar menor | 0 de 12 |
| pior caso por imagem, um núcleo | 312 ms (1440×1532) |
| `build` das 12 numa página, 8 núcleos | 0,86 a 1,03 s, pico de ~300 MB |

Por imagem, o resultado sem pontilhado ficou entre 38% e 54% do
original. A conferência visual comparou recortes ampliados 2×, lado a
lado, nas três versões:

- números e rótulos pequenos dos cartões, no claro;
- texto apagado sob um modal, no claro;
- cabeçalho de tabela e etiquetas coloridas, no escuro;
- tela de celular, no escuro.

O texto ficou igual ao original nas duas versões quantizadas. O
pontilhado só acrescentou granulado nas áreas lisas e custou cerca de
25% a mais.

Pior caso sintético, 1920×1080 com quase todo pixel de cor própria:
cerca de 1,9 s por imagem, num núcleo. Capturas reais têm muito menos
cores.

## Consequência

Sem dependência nova; o quantizador fica dentro do teto de Go de
~5.000 linhas ([ADR 0014](0014-go-code-ceiling.md)). O `build` fica
mais lento com muitas capturas; o paralelismo por arquivo e o cache
do `serve` por SHA-256 compensam. A memória cresce com o paralelismo,
cerca de 40 MB por captura grande em processamento. Por isso o
`copyMedia` otimiza no máximo 4 imagens ao mesmo tempo, mesmo com mais
núcleos: o `serve` costuma rodar em container com limite de memória, e
o `check` também otimiza para medir. Com o limite, o pico medido nas 12
amostras caiu de ~300 MB para 168 a 180 MB, no mesmo tempo (0,8 s). A otimização regrava o PNG e
descarta os blocos auxiliares dele, como perfil de cor e texto. O
`check` faz a mesma otimização para informar o tamanho exato.
