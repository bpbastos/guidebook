# Como o guia funciona

O guia do usuário é um arquivo HTML só, com uma pasta de recursos ao
lado. Todas as páginas estão dentro dele.

## Sem servidor e sem internet

O guia não busca nada na rede. Os dados ficam em arquivos da pasta
`assets/`, e as imagens ficam em `media/`.

Por isso o guia abre com dois cliques, direto do disco.

As imagens carregam só quando chegam perto da tela, e o espaço delas já
fica reservado. A página não pula enquanto as capturas chegam.

## Janela e navegação

Um link para o glossário, para referência ou para explicação abre numa
janela. O leitor lê o trecho e volta para onde estava, sem perder o
lugar.

Um link para tarefa ou tutorial troca de página. O guia guarda o ponto
de origem e mostra o botão "Voltar para onde eu estava".

## Busca

A caixa de busca fica no alto do guia. Ela procura nos títulos e no
texto de todas as páginas, seção por seção.

- Acento e maiúscula não contam: "cobranca" acha "cobrança".
- O começo de uma palavra basta: "fech" acha "Fechar".
- Com mais de uma palavra, o resultado precisa ter todas.

Pelo teclado, `/` leva à busca. As setas escolhem o resultado, Enter
abre e Esc limpa. Abrir um resultado também mostra o botão "Voltar para
onde eu estava".

A busca funciona sem internet: o guia monta o índice ao abrir.

## Capturas com marcadores

Uma captura pode ter marcadores numerados por cima. Cada marcador
corresponde a um passo da lista logo abaixo. Clicar num marcador
destaca o passo, e clicar num passo destaca o marcador.

O botão "Visita guiada" amplia a captura e passa de marcador em
marcador. As setas avançam e voltam; Esc encerra. A visita funciona
também no modo apresentação.

Embaixo de cada captura, a lista "Pontos desta tela" diz o mesmo em
texto, para quem não enxerga a imagem. Veja o formato dos arquivos em
[Capturas com marcadores](../reference/02-screenshot-markers.md).

## Vídeo

Uma tarefa pode ter vídeo. O botão "Ver em vídeo" abre o vídeo numa
janela grande, com os controles do navegador. Fechar a janela pausa o
vídeo. No menu, um triângulo marca as tarefas com vídeo.

Veja o formato em [Vídeo por tarefa](../reference/03-task-video.md).

## Apresentação

O botão "Apresentar" mostra a página atual em slides, para treinar uma
equipe numa tela grande. Cada `##` e cada `---` começa um slide novo.

- As setas, PageUp, PageDown e espaço trocam de slide.
- Home e End vão ao primeiro e ao último.
- F liga e desliga a tela cheia; Esc sai.

Links continuam funcionando: o glossário abre por cima do slide, e o
"Voltar para onde eu estava" devolve também o slide. Slide muito
cheio diminui a letra até 60%; depois disso, ele rola.

## Comece por aqui

Cada [perfil](../glossary.md#perfil) tem uma trilha de páginas. O leitor
escolhe o perfil e vê a trilha dele.

## Sem JavaScript

Sem JavaScript, o guia continua legível. As páginas aparecem uma depois
da outra, como um documento longo.
