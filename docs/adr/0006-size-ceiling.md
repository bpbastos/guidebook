# 0006. Teto de 160 KiB para o guia do usuário

Status: aceita (fase 4).

## Contexto

A spec pede um teto de tamanho para os guias, sem contar a mídia do
projeto, proposto a partir do medido. Na fase 4, o guia do usuário do
projeto de exemplo, sem `media/`, mede 17,6 KB:

- HTML com as sete páginas: cerca de 9 KB;
- CSS e JavaScript do guidebook: cerca de 8 KB;
- dados e tema: menos de 1 KB.

A fase 5 traz o MiniSearch 7.2.0. O pacote no npm não tem arquivo
minificado: o UMD distribuído (`dist/umd/index.js`) tem 86 KB.
Minificar exigiria uma ferramenta nova. As fases 6 a 8 acrescentam
JavaScript e CSS próprios, estimados em 25 KB.

## Decisão

Teto de 160 KiB para o guia do usuário do projeto de exemplo, sem
`media/`, conferido por teste (`TestSizeCeiling`). A conta: 18 KB de
hoje, 86 KB do MiniSearch, 25 KB das fases 6 a 8 e uma folga de cerca
de 30 KB.

## Consequência

O teste falha se uma fase crescer além do previsto, e o aumento vira
decisão explícita. O guia do desenvolvedor ganha teto próprio na fase
9, porque o Mermaid sozinho tem alguns megabytes. O nome
`minisearch.min.js` da seção 9.1 da spec é revisto na fase 5.
