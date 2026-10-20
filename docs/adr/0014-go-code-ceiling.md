# 0014. Teto de Go próprio sobe para ~5.000 linhas

Status: aceita (fase 13), por decisão do Bruno.

## Contexto

A spec fixava o Go próprio sem testes em ~4.000 linhas. Na fase 13, o
código mede 3.491. A versão 0.2.0 traz a otimização das imagens de
mídia em PNG com paleta, por quantizador próprio. Nenhuma ferramenta
pronta dentro das dependências aprovadas faz isso: a biblioteca padrão
grava PNG com paleta, mas não escolhe a paleta. As fases 16 a 19 também
acrescentam código.

Alternativas:

- manter o teto e parar para reavaliar no meio da versão;
- trazer uma biblioteca de quantização, dependência nova;
- gravar WebP, que também exigiria dependência.

## Decisão

O teto de Go próprio sem testes passa a ~5.000 linhas. O de
JavaScript continua em ~1.500.

## Consequência

A versão 0.2.0 cabe no teto sem dependência nova. Passar de ~5.000
continua sendo motivo para parar e reavaliar com o Bruno.
