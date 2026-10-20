# 0011. Servidor, política de conteúdo e imagem do container

Status: aceita (fase 10).

## Contexto

O `guidebook serve` serve os guias por HTTP com uma política de
conteúdo (CSP) restrita. A spec deixou para a fase 10 confirmar o
mínimo que o Mermaid exige e escolher a imagem final do container, entre
distroless `static` e `scratch`.

## Decisão

**CSP.** Fica `style-src 'self' 'unsafe-inline'`; o resto é só `'self'`.
Conferido no Chrome, servindo o projeto de exemplo:

- com a CSP do `serve`, nenhuma violação: diagramas, busca, visita
  guiada, apresentação e vídeo funcionam;
- com `style-src 'self'`, o navegador recusa o estilo em linha que o
  Mermaid escreve no SVG, e o diagrama fica sem estilo;
- `script-src 'self'` basta: o Mermaid com `securityLevel: "strict"`
  não pede `eval` nem script em linha.

**Imagem.** Compilação em `golang:1.27.2-trixie` e imagem final
`gcr.io/distroless/static-debian13:nonroot`, ambas fixadas por digest
no `Dockerfile`. A distroless já traz o usuário `nonroot` (UID 65532),
o `/tmp` e os certificados, sem shell. Com `scratch`, o usuário e o
`/tmp` teriam de ser montados à mão. Medido na fase 10: 29,3 MB.

**Troca do build.** Cada build vai para um diretório novo, e o servidor
troca um ponteiro atômico. O build antigo só é apagado depois que as
leituras em curso terminam, com uma trava de leitura por build.

## Consequência

O estilo em linha fica liberado em todo o guia, mas script continua
restrito à origem, que é o que protege contra injeção. Se o Mermaid
deixar de precisar de estilo em linha, a CSP aperta sem mudar o resto.
Trocar a imagem base é trocar o digest e este ADR.
