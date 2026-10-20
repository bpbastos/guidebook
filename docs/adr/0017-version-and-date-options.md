# 0017. Versão do produto e data por opção da linha de comando

Status: aceita (fase 14), implementada na fase 18.

## Contexto

Quem lê um guia gerado pela CI quer saber de que versão do produto ele
fala e quando foi gerado. A spec exige determinismo: mesma entrada,
mesmos bytes.

Alternativas:

- ler o relógio: quebra o determinismo;
- ler `SOURCE_DATE_EPOCH` do ambiente: entrada escondida, fora da
  linha de comando, e só resolve a data;
- tirar a versão do último marco do CHANGELOG: o marco publicado nem
  sempre é a versão gerada, e a data dele não é a da geração;
- perguntar ao git (`git describe`, data do commit): exige git e um
  clone completo no ambiente de quem gera;
- chaves no `guidebook.yaml`: obrigam a editar o arquivo a cada
  versão;
- opções do `build` e do `check`.

## Decisão

`-product-version <texto>` e `-date AAAA-MM-DD` no `build` e no
`check`, opcionais e independentes. São entrada, como o resto: a mesma linha de comando
dá os mesmos bytes. Os dois guias mostram os valores no rodapé, com os
textos `product_version` e `generated_on`, e a data em DD/MM/AAAA.
Data inválida é erro de uso.

O `check` aceita as mesmas opções, porque o tamanho que ele informa
precisa ser o mesmo do `build`. O `serve` não as tem: ele regera a cada
mudança, e uma data fixa na linha de comando envelheceria sem aviso.

## Consequência

A CI do projeto passa a versão e a data que já conhece, por exemplo a
tag e a data do commit. Sem as opções, o guia sai como na versão 0.1.
