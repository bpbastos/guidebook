# 0008. Slides cortados no build

Status: aceita (fase 6).

## Contexto

O modo apresentação corta cada página em slides em cada `## ` e em
cada `---`. O corte pode ser feito no navegador, percorrendo o HTML, ou
no build, em Go. A spec pede saída determinística e testes Go que
confiram o HTML; não há teste de navegador nesta rodada.

## Decisão

O build corta: ao renderizar a página, cada slide vira um
`<div class="gb-slide">`. O JavaScript só copia o slide atual para um
palco de 1280×720 px lógicos e o escala para a tela.

## Consequência

O corte é conferido por teste Go e não depende do navegador. O HTML da
leitura ganha os `div` dos slides, sem efeito visual. O diálogo, que
copia uma seção, segue para o slide seguinte quando ele começa num
`---`, porque o `---` corta o slide, não a seção.
