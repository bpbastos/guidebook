# 0020. Vínculo para o glossário inteiro em diálogo

Status: aceita (fase 21).

## Contexto

O vínculo para um termo (`glossary.md#termo`) abria em diálogo, mas o
vínculo para o glossário inteiro, sem âncora, navegava. No modo
apresentação, ele trocava a apresentação toda para o glossário, e quem
lia se perdia, mesmo com "Voltar para onde eu estava". A alternativa
era manter a navegação, como antes.

## Decisão

- Vínculo para `glossary.md` sem âncora abre o glossário inteiro em
  `data-gb="dialog"`, como referência e explicação. O diálogo copia a
  página toda, sem o botão `open_full_page`.
- Continuam navegando os vínculos de `index.md`, porta de entrada como
  o menu, e o vínculo do glossário para ele mesmo.

## Consequência

O Markdown não muda. O guia do desenvolvedor não muda. O JavaScript não
muda: o diálogo já mostrava página inteira para referência e
explicação. O Go testa o tipo do vínculo vindo de tarefa e de
`index.md`.
