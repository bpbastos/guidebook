# Integrar com o e2e do projeto

O e2e do projeto já abre o sistema e passa pelas telas. Ele pode gravar,
no mesmo caminho, as capturas com marcadores e os vídeos das tarefas.
Assim o guia acompanha o sistema sem trabalho à mão.

## Grave as capturas

Em cada passo que o guia mostra, o e2e grava a captura em
`docs/user/media/`. Use PNG ou JPEG.

## Calcule os marcadores

Para cada elemento que o passo cita, o e2e pega a caixa dele na
captura: posição e tamanho, em pixels. O centro da caixa, dividido pelo
tamanho da captura, dá a posição relativa do marcador:

```js
// box e shot em pixels da captura:
// box = { x, y, width, height }, shot = { width, height }
function marker(n, box, shot, text) {
  const round = (v) => Math.round(v * 1000) / 1000;
  return {
    n,
    x: round((box.x + box.width / 2) / shot.width),
    y: round((box.y + box.height / 2) / shot.height),
    text,
  };
}
```

Numa captura de 640 por 400 pixels, um campo com a caixa
`{ x: 40, y: 60, width: 260, height: 40 }` dá `x: 0.266` e `y: 0.2`.

Grave o resultado ao lado da captura, com o mesmo nome e o fim
`.markers.json`. O formato está em
[Capturas com marcadores](../reference/02-screenshot-markers.md).

## Grave os vídeos

O e2e grava um vídeo por tarefa, em `.webm` ou `.mp4`, numa pasta como
`docs/user/media/videos/`. Depois, mede a duração de cada vídeo. Com o
ffprobe, por exemplo:

```sh
ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 register-order.webm
```

A resposta é a duração em segundos, como `3.000000`. Use esse número
em `duration_seconds` no `manifest.json` da pasta. O formato está em
[Vídeo por tarefa](../reference/03-task-video.md).

## Confira no CI

Depois do e2e, rode `guidebook check`. Captura gravada de novo com
outro tamanho, marcador a mais ou vídeo que faltou viram recusa, com o
arquivo e o campo.

## O guidebook não me deixa usar a captura nova

A captura mudou de tamanho e o `.markers.json` ficou para trás. Grave
os dois no mesmo passo do e2e.
