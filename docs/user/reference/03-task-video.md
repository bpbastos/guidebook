# Vídeo por tarefa

O e2e do projeto pode gravar um vídeo por tarefa. O guia mostra o botão
"Ver em vídeo (N min)" logo abaixo do título da tarefa.

## Arquivos

O e2e grava os vídeos e um manifesto no mesmo diretório:

```
docs/user/media/videos/register-order.webm
docs/user/media/videos/register-order.png
docs/user/media/videos/manifest.json
```

```json
{
  "version": 1,
  "videos": [
    {
      "task": "tasks/register-order",
      "file": "register-order.webm",
      "duration_seconds": 94.2,
      "poster": "register-order.png"
    }
  ]
}
```

## Campos

| Campo | Regra |
|---|---|
| `version` | sempre `1` |
| `task` | tarefa de `tasks/`, uma vez só no manifesto |
| `file` | nome do vídeo `.webm` ou `.mp4`, no diretório do manifesto |
| `duration_seconds` | duração, maior que 0 |
| `poster` | opcional: imagem PNG ou JPEG mostrada antes de tocar |

O botão arredonda a duração para cima, em minutos.

## No guidebook.yaml

Liste as tarefas que têm vídeo:

```yaml
user:
  videos:
    manifest: docs/user/media/videos/manifest.json
    tasks:
      - tasks/register-order
```

Entrada do manifesto fora dessa lista fica de fora do guia.

## O guidebook não me deixa usar o vídeo

O `check` diz o campo com problema. Os casos mais comuns:

- a tarefa da lista não está no manifesto;
- o arquivo do vídeo não existe, porque o e2e não gravou;
- a extensão não é `.webm` nem `.mp4`.
