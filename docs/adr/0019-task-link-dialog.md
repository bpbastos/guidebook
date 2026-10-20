# 0019. Vínculo para tarefa ou tutorial em diálogo

Status: aceita (fase 20).

## Contexto

Um vínculo de apoio para tarefa ou tutorial trocava de página. O leitor
perdia a continuidade do que lia, mesmo com "Voltar para onde eu
estava". Glossário, referência, explicação e bloqueio já abriam em
diálogo.

Para o mecanismo, havia duas alternativas: o mesmo `data-gb="dialog"`
com uma marca de destino, ou um tipo novo.

## Decisão

- Tipo novo, `data-gb="page"`, para o vínculo do Markdown para página ou
  âncora de `tutorials/` ou `tasks/`. O tipo diz de uma vez que o
  diálogo tem outro corte e o botão `open_full_page`. O `href` continua
  `#/<id>[/<âncora>]` e funciona sem JavaScript.
- Com âncora, o diálogo mostra a seção, no corte de antes. Sem âncora,
  mostra o título e o texto antes do primeiro `## `; o link do vídeo não
  conta como texto. Sem texto, entra também a primeira seção.
- O botão é um link `data-gb="nav"` no próprio diálogo, com o `href` do
  destino. Usa o mesmo caminho de qualquer navegação: fecha o diálogo,
  empilha a origem em `backStack` e, na apresentação, segue no slide do
  destino.
- Continuam navegando os vínculos de `index.md`, porta de entrada como o
  menu, e o vínculo para a própria página, que só rola até a seção.
- A cópia no diálogo esconde o botão da visita guiada, em todo diálogo:
  a visita agiria sobre a página de trás, e as setas dela não valem com
  o diálogo aberto. Ficam a captura, os marcadores e a lista de pontos.
  O vídeo abre num segundo diálogo, por cima.

## Consequência

O Markdown não muda. O guia do desenvolvedor não muda. O JavaScript
fica sem teste automático, porque o projeto não tem executor de
navegador. Ele é conferido à mão e na prévia (spec, seção 13.1, itens
8 a 13). O Go testa o tipo de cada vínculo e o botão no HTML.
