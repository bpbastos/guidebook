# 0010. Mermaid 12.1.0 vendorizado e teto de 6 MiB do guia de dev

Status: aceita (fase 9).

## Contexto

O guia do desenvolvedor desenha diagramas com o Mermaid, aprovado pelo
Bruno. Em 2026-10-20, a versão mais nova era a 12.2.0, de 2026-10-13,
com uma semana de publicada. A spec pede um teto de tamanho por guia,
proposto a partir do medido.

## Decisão

Vendorizar o Mermaid 12.1.0, de 2026-10-02: a mais nova com pelo menos
duas semanas de publicada, por prudência com a cadeia de suprimentos.
O arquivo é o `package/dist/mermaid.min.js` do tarball, sem mudança,
com a licença MIT ao lado.

- tarball: `https://registry.npmjs.org/mermaid/-/mermaid-12.1.0.tgz`;
- integridade do tarball, igual à do registro:
  `sha512-wlVCp+8eTupfCeeFvoZNNiTuHrvag0P2jz/ILgb/f/6jkVokUefOcujefi8qUe/j2asHiSePncVsz/xzzA80LQ==`;
- SHA-256 do arquivo:
  `6484afc32872a3aa16cac9a76ba1816a1ed4cc870a6593cc2e17757750f518b2`;
- tamanho: 5.493.176 bytes.

Revisão do arquivo, que é minificado:

- as URLs são namespaces (`www.w3.org`) e links de documentação em
  textos; nenhum CDN;
- os `fetch(` são métodos do analisador do KaTeX; um deles é a
  definição `fetch(){`;
- `new Worker` aparece no ELK e só roda com `workerUrl`, que o
  guidebook não configura;
- `Function("return this")` é a detecção de objeto global do lodash e
  não roda no navegador, onde `self` existe.

Sobram duas ocorrências para o teste (`reviewed_matches: 2`): a
definição `fetch(){` e o `new Worker`.

O teto do guia do desenvolvedor do projeto de exemplo, sem `media/`, é
de 6 MiB. Medido na fase 9: 5.625.097 bytes, dos quais 5.493.176 são do
Mermaid.

## Consequência

O guia do desenvolvedor fica pesado, mas abre sem rede. O guia do
usuário não carrega o Mermaid e continua dentro dos 160 KiB. Trocar de
versão é trocar o arquivo, o manifesto e este ADR; o teste de hash e o
de ocorrências pegam qualquer diferença.
