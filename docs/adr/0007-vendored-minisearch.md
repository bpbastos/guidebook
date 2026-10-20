# 0007. MiniSearch 7.2.0 vendorizado sem minificar

Status: aceita (fase 5).

## Contexto

A busca do guia usa o MiniSearch, aprovado pelo Bruno. A spec pedia o
arquivo `.min.js`, mas o pacote 7.2.0 no npm não traz versão
minificada. O UMD distribuído, `dist/umd/index.js`, tem 86 KB.
Minificar exigiria uma ferramenta nova.

## Decisão

Vendorizar o `package/dist/umd/index.js` do tarball como
`internal/site/assets/third_party/minisearch/minisearch.js`, sem mudança. A
licença MIT vai ao lado e também para a saída. Dados conferidos em
2026-10-20:

- tarball: `https://registry.npmjs.org/minisearch/-/minisearch-7.2.0.tgz`;
- integridade do tarball, igual à do registro:
  `sha512-dqT2XBYUOZOiC5t2HRnwADjhNS2cecp9u+TJRiJ1Qp/f5qjkeT5APcGPjHw+bz89Ms8Jp+cG4AlE+QZ/QnDglg==`;
- SHA-256 do arquivo:
  `03c27931eb603af1d3358579e9885eafdb6a2f5fa9929557b3e77d7e99d06d03`.

O arquivo cita URLs só em comentários. O `fetch` que aparece nele é
método da estrutura interna, não chamada de rede.

## Consequência

O guia do usuário do exemplo passa de 17,6 KB para 115 KB, dentro do
teto de 160 KiB ([ADR 0006](0006-size-ceiling.md)). Trocar de versão é
trocar o arquivo, o manifesto e este ADR; o teste de hash pega
qualquer diferença.
